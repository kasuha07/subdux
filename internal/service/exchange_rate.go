package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExchangeRateService struct {
	DB         *gorm.DB
	httpClient *http.Client
	mu         sync.RWMutex
	cache      map[string]float64
	cacheTime  time.Time
}

func NewExchangeRateService(db *gorm.DB) *ExchangeRateService {
	s := &ExchangeRateService{
		DB: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: make(map[string]float64),
	}
	s.loadCacheFromDB()
	return s
}

func cacheKey(base, target string) string {
	return strings.ToLower(base) + ":" + strings.ToLower(target)
}

func (s *ExchangeRateService) loadCacheFromDB() {
	var rates []model.ExchangeRate
	s.DB.Find(&rates)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range rates {
		s.cache[cacheKey(r.BaseCurrency, r.TargetCurrency)] = r.Rate
	}
	if len(rates) > 0 {
		s.cacheTime = rates[0].FetchedAt
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
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	if from == to {
		return amount
	}

	s.mu.RLock()
	rate, ok := s.cache[cacheKey(from, to)]
	s.mu.RUnlock()

	if ok {
		return amount * rate
	}

	var er model.ExchangeRate
	if err := s.DB.Where("base_currency = ? AND target_currency = ?",
		strings.ToLower(from), strings.ToLower(to)).First(&er).Error; err == nil {
		s.mu.Lock()
		s.cache[cacheKey(from, to)] = er.Rate
		s.mu.Unlock()
		return amount * er.Rate
	}

	return amount
}

func (s *ExchangeRateService) GetRate(base, target string) (float64, bool) {
	base = strings.ToUpper(base)
	target = strings.ToUpper(target)
	if base == target {
		return 1.0, true
	}

	s.mu.RLock()
	rate, ok := s.cache[cacheKey(base, target)]
	s.mu.RUnlock()
	if ok {
		return rate, true
	}

	var er model.ExchangeRate
	if err := s.DB.Where("base_currency = ? AND target_currency = ?",
		strings.ToLower(base), strings.ToLower(target)).First(&er).Error; err == nil {
		return er.Rate, true
	}

	return 0, false
}

type ExchangeRateInfo struct {
	BaseCurrency   string    `json:"base_currency"`
	TargetCurrency string    `json:"target_currency"`
	Rate           float64   `json:"rate"`
	Source         string    `json:"source"`
	FetchedAt      time.Time `json:"fetched_at"`
}

func (s *ExchangeRateService) ListRates(baseCurrency string) ([]ExchangeRateInfo, error) {
	var rates []model.ExchangeRate
	query := s.DB.Order("base_currency ASC, target_currency ASC")
	if baseCurrency != "" {
		query = query.Where("base_currency = ?", strings.ToLower(baseCurrency))
	}
	if err := query.Find(&rates).Error; err != nil {
		return nil, err
	}

	result := make([]ExchangeRateInfo, len(rates))
	for i, r := range rates {
		result[i] = ExchangeRateInfo{
			BaseCurrency:   strings.ToUpper(r.BaseCurrency),
			TargetCurrency: strings.ToUpper(r.TargetCurrency),
			Rate:           r.Rate,
			Source:         r.Source,
			FetchedAt:      r.FetchedAt,
		}
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
	s.DB.Model(&model.ExchangeRate{}).Count(&count)

	var latest model.ExchangeRate
	err := s.DB.Order("fetched_at DESC").First(&latest).Error

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
		apiKey = keySetting.Value
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
				log.Printf("Premium API failed, falling back to free API: %v", err)
				return s.fetchFromFree()
			}
			return nil
		}
		return s.fetchFromFree()
	}
}

var commonCurrencies = []string{"usd", "eur", "gbp", "jpy", "cny", "cad", "aud", "chf", "hkd", "sgd", "krw", "inr", "brl", "mxn", "rub", "twd", "thb", "try", "nzd", "sek", "nok", "dkk", "pln", "czk", "huf", "ils", "php", "myr", "idr", "vnd", "zar"}

func (s *ExchangeRateService) fetchFromFree() error {
	bases := s.getActiveCurrencies()
	now := time.Now()
	var allRates []model.ExchangeRate

	for _, base := range bases {
		rates, err := s.fetchFreeBase(base)
		if err != nil {
			log.Printf("Failed to fetch free rates for %s: %v", base, err)
			continue
		}

		for target, rate := range rates {
			if target == base {
				continue
			}
			allRates = append(allRates, model.ExchangeRate{
				BaseCurrency:   base,
				TargetCurrency: target,
				Rate:           rate,
				Source:         "free",
				FetchedAt:      now,
			})
		}
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
	bases := s.getActiveCurrencies()
	now := time.Now()
	var allRates []model.ExchangeRate

	targets := make(map[string]bool)
	for _, c := range commonCurrencies {
		targets[strings.ToUpper(c)] = true
	}

	for _, base := range bases {
		targetList := make([]string, 0)
		for t := range targets {
			if strings.ToLower(t) != base {
				targetList = append(targetList, t)
			}
		}

		url := fmt.Sprintf("https://api.currencyapi.com/v3/latest?base_currency=%s&currencies=%s",
			strings.ToUpper(base), strings.Join(targetList, ","))

		req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("apikey", apiKey)

		resp, err := s.httpClient.Do(req)
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
			if target == base {
				continue
			}
			allRates = append(allRates, model.ExchangeRate{
				BaseCurrency:   base,
				TargetCurrency: target,
				Rate:           item.Value,
				Source:         "premium",
				FetchedAt:      now,
			})
		}
	}

	if len(allRates) == 0 {
		return fmt.Errorf("no rates fetched from premium API")
	}

	return s.saveRates(allRates)
}

func (s *ExchangeRateService) getActiveCurrencies() []string {
	currencySet := make(map[string]bool)
	for _, c := range commonCurrencies {
		currencySet[c] = true
	}

	var subs []model.Subscription
	s.DB.Select("DISTINCT currency").Where("status = ?", "active").Find(&subs)
	for _, sub := range subs {
		currencySet[strings.ToLower(sub.Currency)] = true
	}

	var prefs []model.UserPreference
	s.DB.Find(&prefs)
	for _, p := range prefs {
		currencySet[strings.ToLower(p.PreferredCurrency)] = true
	}

	result := make([]string, 0, len(currencySet))
	for c := range currencySet {
		result = append(result, c)
	}
	return result
}

func (s *ExchangeRateService) getTargetCurrencies(base string) []string {
	targets := make(map[string]bool)
	for _, c := range commonCurrencies {
		if c != base {
			targets[c] = true
		}
	}

	var subs []model.Subscription
	s.DB.Select("DISTINCT currency").Where("status = ?", "active").Find(&subs)
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
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		for _, r := range rates {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "base_currency"}, {Name: "target_currency"}},
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

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range rates {
		s.cache[cacheKey(r.BaseCurrency, r.TargetCurrency)] = r.Rate
	}
	s.cacheTime = time.Now()

	return nil
}

func (s *ExchangeRateService) httpGet(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
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

func (s *ExchangeRateService) StartBackgroundRefresh(stop <-chan struct{}) {
	go func() {
		if err := s.RefreshRates(); err != nil {
			log.Printf("Initial rate refresh failed: %v", err)
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.RefreshRates(); err != nil {
					log.Printf("Scheduled rate refresh failed: %v", err)
				}
			case <-stop:
				return
			}
		}
	}()
}
