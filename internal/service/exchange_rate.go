package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/pkg/logging"
	serviceoutbound "github.com/kasuha07/subdux/internal/service/outbound"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// rateCache holds the in-memory exchange-rate cache behind a pointer so that
// context-scoped service clones (see WithContext) share one cache and one lock
// instead of copying the mutex by value.
type rateCache struct {
	mu    sync.RWMutex
	rates map[string]float64
}

const usdCurrencyCode = "USD"
const usdCurrencyCodeLower = "usd"

func newRateCache() *rateCache {
	return &rateCache{rates: make(map[string]float64)}
}

type ExchangeRateService struct {
	DB         *gorm.DB
	httpClient *http.Client
	cache      *rateCache
}

func NewExchangeRateService(db *gorm.DB) *ExchangeRateService {
	client, err := serviceoutbound.BuildHTTPClientWithTimeout(context.Background(), db, serviceoutbound.PurposeExchangeRate, 30*time.Second)
	if err != nil {
		client = serviceoutbound.NewOutboundHTTPClient(db, 30*time.Second)
	}
	s := &ExchangeRateService{
		DB:         db,
		httpClient: client,
		cache:      newRateCache(),
	}
	s.loadCacheFromDB()
	return s
}

// WithContext returns a shallow copy of the service whose database handle is
// bound to ctx, so GORM cancels in-flight queries when ctx is cancelled (client
// disconnect or write-timeout). The cache pointer and HTTP client are shared
// across clones.
func (s *ExchangeRateService) WithContext(ctx context.Context) *ExchangeRateService {
	clone := *s
	clone.DB = s.DB.WithContext(ctx)
	return &clone
}

func rateCacheKey(target string) string {
	return strings.ToLower(target)
}

func (s *ExchangeRateService) loadCacheFromDB() {
	var rates []model.ExchangeRate
	s.DB.Find(&rates)
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	for _, r := range rates {
		if strings.EqualFold(r.TargetCurrency, usdCurrencyCode) || r.Rate <= 0 {
			continue
		}
		s.cache.rates[rateCacheKey(r.TargetCurrency)] = r.Rate
	}
}

type UpdatePreferenceInput struct {
	PreferredCurrency string `json:"preferred_currency"`
}

func (s *ExchangeRateService) GetUserPreference(userID uint) (*model.UserPreference, error) {
	var pref model.UserPreference
	err := s.DB.Where("user_id = ?", userID).First(&pref).Error
	if err != nil {
		return &model.UserPreference{
			UserID:            userID,
			PreferredCurrency: "USD",
		}, nil
	}
	return &pref, nil
}

func (s *ExchangeRateService) UpdateUserPreference(userID uint, input UpdatePreferenceInput) (*model.UserPreference, error) {
	currency := strings.ToUpper(strings.TrimSpace(input.PreferredCurrency))
	if currency == "" {
		currency = "USD"
	}

	pref := model.UserPreference{
		UserID:            userID,
		PreferredCurrency: currency,
	}

	err := s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"preferred_currency", "updated_at"}),
	}).Create(&pref).Error
	if err != nil {
		return nil, err
	}

	return s.GetUserPreference(userID)
}

func (s *ExchangeRateService) Convert(amount float64, from, to string) float64 {
	from = normalizeCurrencyCode(from)
	to = normalizeCurrencyCode(to)
	if from == to {
		return amount
	}

	rate, ok := s.GetRate(from, to)
	if !ok {
		return amount
	}
	return amount * rate
}

func (s *ExchangeRateService) GetRate(base, target string) (float64, bool) {
	base = normalizeCurrencyCode(base)
	target = normalizeCurrencyCode(target)
	if base == target {
		return 1.0, true
	}

	basePerUSD, ok := s.getUSDBaseRate(base)
	if !ok {
		return 0, false
	}
	targetPerUSD, ok := s.getUSDBaseRate(target)
	if !ok {
		return 0, false
	}

	return targetPerUSD / basePerUSD, true
}

func (s *ExchangeRateService) getUSDBaseRate(currency string) (float64, bool) {
	currency = normalizeCurrencyCode(currency)
	if currency == usdCurrencyCode {
		return 1.0, true
	}

	s.cache.mu.RLock()
	rate, ok := s.cache.rates[rateCacheKey(currency)]
	s.cache.mu.RUnlock()
	if ok && rate > 0 {
		return rate, true
	}

	var er model.ExchangeRate
	if err := s.DB.Where("LOWER(target_currency) = ?",
		strings.ToLower(currency)).First(&er).Error; err == nil && er.Rate > 0 {
		s.cache.mu.Lock()
		s.cache.rates[rateCacheKey(currency)] = er.Rate
		s.cache.mu.Unlock()
		return er.Rate, true
	}

	return 0, false
}

type ExchangeRateInfo struct {
	TargetCurrency string    `json:"target_currency"`
	Rate           float64   `json:"rate"`
	Source         string    `json:"source"`
	FetchedAt      time.Time `json:"fetched_at"`
}

func (s *ExchangeRateService) ListRates() ([]ExchangeRateInfo, error) {
	var rates []model.ExchangeRate
	query := s.DB.Where("LOWER(target_currency) <> ?", usdCurrencyCodeLower).Order("target_currency ASC")
	if err := query.Find(&rates).Error; err != nil {
		return nil, err
	}

	result := make([]ExchangeRateInfo, 0, len(rates))
	for _, r := range rates {
		target := normalizeCurrencyCode(r.TargetCurrency)
		if target == "" || target == usdCurrencyCode || r.Rate <= 0 {
			continue
		}
		result = append(result, ExchangeRateInfo{
			TargetCurrency: target,
			Rate:           r.Rate,
			Source:         r.Source,
			FetchedAt:      r.FetchedAt,
		})
	}
	return result, nil
}

type RateStatus struct {
	LastFetchedAt *time.Time `json:"last_fetched_at"`
	Source        string     `json:"source"`
	RateCount     int64      `json:"rate_count"`
}

func (s *ExchangeRateService) GetStatus() (*RateStatus, error) {
	var count int64
	s.DB.Model(&model.ExchangeRate{}).
		Where("LOWER(target_currency) <> ?", usdCurrencyCodeLower).
		Count(&count)

	var latest model.ExchangeRate
	err := s.DB.
		Where("LOWER(target_currency) <> ?", usdCurrencyCodeLower).
		Order("fetched_at DESC").
		First(&latest).Error

	status := &RateStatus{RateCount: count}
	if err == nil {
		status.LastFetchedAt = &latest.FetchedAt
		status.Source = latest.Source
	}
	return status, nil
}

func (s *ExchangeRateService) RefreshRates() error {
	source := "auto"
	var sourceSetting model.SystemSetting
	if err := s.DB.Where("key = ?", "exchange_rate_source").First(&sourceSetting).Error; err == nil && sourceSetting.Value != "" {
		source = sourceSetting.Value
	}

	var apiKey string
	var keySetting model.SystemSetting
	if err := s.DB.Where("key = ?", "currencyapi_key").First(&keySetting).Error; err == nil {
		decryptedKey, decryptErr := systemsettings.DecryptValueIfNeeded("currencyapi_key", keySetting.Value)
		switch {
		case decryptErr == nil:
			apiKey = strings.TrimSpace(decryptedKey)
		case !pkg.IsSystemSettingEncrypted(keySetting.Value):
			apiKey = strings.TrimSpace(keySetting.Value)
		default:
			return fmt.Errorf("decrypt currency API key: %w", decryptErr)
		}

		if !pkg.IsSystemSettingEncrypted(keySetting.Value) && apiKey != "" {
			encryptedKey, encryptErr := systemsettings.EncryptValueIfNeeded("currencyapi_key", apiKey)
			if encryptErr == nil {
				_ = s.DB.Model(&model.SystemSetting{}).Where("key = ?", "currencyapi_key").Update("value", encryptedKey).Error
			}
		}
	}

	switch source {
	case "free":
		return s.fetchFromFree()
	case "premium":
		if apiKey == "" {
			return fmt.Errorf("premium source selected but no API key configured")
		}
		return s.fetchFromPremium(apiKey)
	default:
		if apiKey != "" {
			if err := s.fetchFromPremium(apiKey); err != nil {
				logging.Warn("premium exchange-rate API failed, falling back to free API",
					slog.Any("error", err))
				return s.fetchFromFree()
			}
			return nil
		}
		return s.fetchFromFree()
	}
}

var commonCurrencies = []string{"usd", "eur", "gbp", "jpy", "cny", "cad", "aud", "chf", "hkd", "sgd", "krw", "inr", "brl", "mxn", "rub", "twd", "thb", "try", "nzd", "sek", "nok", "dkk", "pln", "czk", "huf", "ils", "php", "myr", "idr", "vnd", "zar"}

func (s *ExchangeRateService) fetchFromFree() error {
	base := usdCurrencyCodeLower
	now := pkg.NowUTC()
	var allRates []model.ExchangeRate

	rates, err := s.fetchFreeBase(base)
	if err != nil {
		return fmt.Errorf("fetch free USD exchange rates: %w", err)
	}

	for target, rate := range rates {
		if target == base || rate <= 0 {
			continue
		}
		allRates = append(allRates, model.ExchangeRate{
			TargetCurrency: target,
			Rate:           rate,
			Source:         "free",
			FetchedAt:      now,
		})
	}

	if len(allRates) == 0 {
		return fmt.Errorf("no rates fetched from free API")
	}

	return s.saveRates(allRates)
}

func (s *ExchangeRateService) fetchFreeBase(base string) (map[string]float64, error) {
	url := fmt.Sprintf("https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/%s.min.json", base)
	fallbackURL := fmt.Sprintf("https://latest.currency-api.pages.dev/v1/currencies/%s.min.json", base)

	data, err := s.httpGet(url)
	if err != nil {
		data, err = s.httpGet(fallbackURL)
		if err != nil {
			return nil, fmt.Errorf("both primary and fallback failed for %s: %w", base, err)
		}
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	ratesData, ok := raw[base]
	if !ok {
		return nil, fmt.Errorf("no rates found for base %s", base)
	}

	var rates map[string]float64
	if err := json.Unmarshal(ratesData, &rates); err != nil {
		return nil, fmt.Errorf("unmarshal rates: %w", err)
	}

	filtered := make(map[string]float64)
	targets := s.getTargetCurrencies(base)
	for _, t := range targets {
		if rate, ok := rates[t]; ok {
			filtered[t] = rate
		}
	}

	return filtered, nil
}

func (s *ExchangeRateService) fetchFromPremium(apiKey string) error {
	base := usdCurrencyCodeLower
	now := pkg.NowUTC()
	var allRates []model.ExchangeRate

	targets := s.getTargetCurrencies(base)
	targetList := make([]string, 0, len(targets))
	for _, target := range targets {
		targetList = append(targetList, strings.ToUpper(target))
	}

	url := fmt.Sprintf("https://api.currencyapi.com/v3/latest?base_currency=%s&currencies=%s",
		usdCurrencyCode, strings.Join(targetList, ","))

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("apikey", apiKey)

	resp, err := s.outboundHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("premium API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("premium API returned status %d", resp.StatusCode)
	}

	var result struct {
		Data map[string]struct {
			Code  string  `json:"code"`
			Value float64 `json:"value"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode premium response: %w", err)
	}

	for _, item := range result.Data {
		target := strings.ToLower(item.Code)
		if target == base || item.Value <= 0 {
			continue
		}
		allRates = append(allRates, model.ExchangeRate{
			TargetCurrency: target,
			Rate:           item.Value,
			Source:         "premium",
			FetchedAt:      now,
		})
	}

	if len(allRates) == 0 {
		return fmt.Errorf("no rates fetched from premium API")
	}

	return s.saveRates(allRates)
}

func (s *ExchangeRateService) getTargetCurrencies(base string) []string {
	targets := make(map[string]bool)
	for _, c := range commonCurrencies {
		if c != base {
			targets[c] = true
		}
	}

	var subs []model.Subscription
	s.DB.Select("DISTINCT currency").Where("status = ?", subscriptionservice.StatusActive).Find(&subs)
	for _, sub := range subs {
		c := strings.ToLower(sub.Currency)
		if c != base {
			targets[c] = true
		}
	}

	var prefs []model.UserPreference
	s.DB.Find(&prefs)
	for _, p := range prefs {
		c := strings.ToLower(p.PreferredCurrency)
		if c != base {
			targets[c] = true
		}
	}

	result := make([]string, 0, len(targets))
	for c := range targets {
		result = append(result, c)
	}
	return result
}

func (s *ExchangeRateService) saveRates(rates []model.ExchangeRate) error {
	usdRates := make([]model.ExchangeRate, 0, len(rates))
	for _, r := range rates {
		target := strings.ToLower(strings.TrimSpace(r.TargetCurrency))
		if target == "" || target == usdCurrencyCodeLower || r.Rate <= 0 {
			continue
		}
		r.TargetCurrency = target
		usdRates = append(usdRates, r)
	}

	if len(usdRates) == 0 {
		return fmt.Errorf("no USD-based exchange rates to save")
	}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("LOWER(target_currency) = ?", usdCurrencyCodeLower).
			Delete(&model.ExchangeRate{}).Error; err != nil {
			return err
		}
		for _, r := range usdRates {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "target_currency"}},
				DoUpdates: clause.AssignmentColumns([]string{"rate", "source", "fetched_at", "updated_at"}),
			}).Create(&r).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	for key := range s.cache.rates {
		delete(s.cache.rates, key)
	}
	for _, r := range usdRates {
		s.cache.rates[rateCacheKey(r.TargetCurrency)] = r.Rate
	}

	return nil
}

func normalizeCurrencyCode(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

func (s *ExchangeRateService) httpGet(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.outboundHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var buf [1 << 20]byte
	n := 0
	for {
		nn, err := resp.Body.Read(buf[n:])
		n += nn
		if err != nil {
			break
		}
		if n >= len(buf) {
			return nil, fmt.Errorf("response too large")
		}
	}
	return buf[:n], nil
}

func (s *ExchangeRateService) outboundHTTPClient() *http.Client {
	if s.httpClient != nil {
		return s.httpClient
	}
	client, err := serviceoutbound.BuildHTTPClientWithTimeout(context.Background(), s.DB, serviceoutbound.PurposeExchangeRate, 30*time.Second)
	if err != nil {
		return serviceoutbound.NewOutboundHTTPClient(s.DB, 30*time.Second)
	}
	return client
}

func (s *ExchangeRateService) StartBackgroundRefresh(
	ctx context.Context,
	monitor *BackgroundTaskMonitor,
	wg *sync.WaitGroup,
) {
	const taskKey = "exchange_rate_refresh"
	const refreshInterval = 24 * time.Hour

	if ctx == nil {
		ctx = context.Background()
	}

	if monitor != nil {
		monitor.Register(
			taskKey,
			"Exchange rate refresh",
			"Fetches currency exchange rates and updates the local cache.",
			refreshInterval,
		)
	}

	runRefresh := func(trigger string) {
		run := s.RefreshRates
		if monitor != nil {
			if err := monitor.Run(taskKey, run); err != nil {
				logging.Error("exchange-rate refresh failed",
					slog.String("trigger", trigger), slog.Any("error", err))
			}
			return
		}
		if err := run(); err != nil {
			logging.Error("exchange-rate refresh failed",
				slog.String("trigger", trigger), slog.Any("error", err))
		}
	}

	if wg != nil {
		wg.Add(1)
	}

	go func() {
		if wg != nil {
			defer wg.Done()
		}

		if ctx.Err() != nil {
			return
		}

		runRefresh("initial")

		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				runRefresh("scheduled")
			case <-ctx.Done():
				return
			}
		}
	}()
}
