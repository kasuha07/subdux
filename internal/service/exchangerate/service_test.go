package exchangerate

import (
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
)

func TestServiceDerivesConversionsFromUSDBaseRates(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.ExchangeRate{}); err != nil {
		t.Fatalf("failed to migrate exchange rate table: %v", err)
	}

	now := time.Now().UTC()
	if err := db.Create(&[]model.ExchangeRate{
		{TargetCurrency: "eur", Rate: 0.8, Source: "test", FetchedAt: now},
		{TargetCurrency: "cny", Rate: 7.2, Source: "test", FetchedAt: now},
	}).Error; err != nil {
		t.Fatalf("failed to seed USD exchange rates: %v", err)
	}

	svc := NewService(db)

	got, ok := svc.Convert(10, "EUR", "CNY")
	if !ok || got != 90 {
		t.Fatalf("Convert(EUR->CNY) = %v, %v; want 90, true", got, ok)
	}
	if _, ok := svc.Convert(10, "EUR", "JPY"); ok {
		t.Fatal("Convert(EUR->JPY) ok = true without a JPY rate")
	}

	rate, ok := svc.GetRate("CNY", "EUR")
	if !ok {
		t.Fatal("GetRate(CNY->EUR) ok = false, want true")
	}
	want := 0.8 / 7.2
	if math.Abs(rate-want) > 1e-12 {
		t.Fatalf("GetRate(CNY->EUR) = %v, want %v", rate, want)
	}
}

func TestListRatesReturnsStoredUSDTargetRates(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.ExchangeRate{}); err != nil {
		t.Fatalf("failed to migrate exchange rate table: %v", err)
	}

	now := time.Now().UTC()
	if err := db.Create(&[]model.ExchangeRate{
		{TargetCurrency: "eur", Rate: 0.8, Source: "test", FetchedAt: now},
		{TargetCurrency: "cny", Rate: 7.2, Source: "test", FetchedAt: now},
	}).Error; err != nil {
		t.Fatalf("failed to seed USD exchange rates: %v", err)
	}

	svc := NewService(db)
	rates, err := svc.ListRates()
	if err != nil {
		t.Fatalf("ListRates() error = %v", err)
	}

	byTarget := make(map[string]float64)
	for _, rate := range rates {
		byTarget[rate.TargetCurrency] = rate.Rate
	}

	if got := byTarget["EUR"]; got != 0.8 {
		t.Fatalf("USD->EUR rate = %v, want 0.8", got)
	}
	if got := byTarget["CNY"]; got != 7.2 {
		t.Fatalf("USD->CNY rate = %v, want 7.2", got)
	}
}

func TestRefreshRatesStoresOnlyUSDBaseRates(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}, &model.ExchangeRate{}); err != nil {
		t.Fatalf("failed to migrate exchange rate tables: %v", err)
	}
	seedSystemSetting(t, db, "currencyapi_key", "secret-api-key")
	seedSystemSetting(t, db, "exchange_rate_source", "premium")

	originalCurrencies := commonCurrencies
	commonCurrencies = []string{"usd", "eur", "cny"}
	defer func() {
		commonCurrencies = originalCurrencies
	}()

	svc := NewService(db)
	svc.httpClient = &http.Client{Transport: exchangeRateTestRoundTripper(func(req *http.Request) (*http.Response, error) {
		if got := req.URL.Query().Get("base_currency"); got != "USD" {
			t.Fatalf("base_currency query = %q, want USD", got)
		}
		if currencies := req.URL.Query().Get("currencies"); strings.Contains(currencies, "USD") {
			t.Fatalf("currencies query = %q, should not include USD", currencies)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":{"EUR":{"code":"EUR","value":0.92},"CNY":{"code":"CNY","value":7.18}}}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}

	if err := svc.RefreshRates(); err != nil {
		t.Fatalf("RefreshRates() error = %v, want nil", err)
	}

	var rows []model.ExchangeRate
	if err := db.Order("target_currency ASC").Find(&rows).Error; err != nil {
		t.Fatalf("load saved exchange rates failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("saved exchange rate rows = %d, want 2", len(rows))
	}
	for _, row := range rows {
		if row.TargetCurrency == "usd" {
			t.Fatal("saved target currency should not be usd")
		}
	}
}
