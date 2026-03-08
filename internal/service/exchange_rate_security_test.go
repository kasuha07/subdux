package service

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

type exchangeRateTestRoundTripper func(*http.Request) (*http.Response, error)

func (fn exchangeRateTestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestRefreshRatesDecryptsEncryptedCurrencyAPIKey(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}, &model.ExchangeRate{}); err != nil {
		t.Fatalf("failed to migrate exchange rate security tables: %v", err)
	}

	encryptedKey, err := encryptSystemSettingValueIfNeeded("currencyapi_key", "secret-api-key")
	if err != nil {
		t.Fatalf("encryptSystemSettingValueIfNeeded() error = %v, want nil", err)
	}
	seedSystemSetting(t, db, "currencyapi_key", encryptedKey)
	seedSystemSetting(t, db, "exchange_rate_source", "premium")

	originalCurrencies := commonCurrencies
	commonCurrencies = []string{"usd", "eur"}
	defer func() {
		commonCurrencies = originalCurrencies
	}()

	callCount := 0
	svc := NewExchangeRateService(db)
	svc.httpClient = &http.Client{Transport: exchangeRateTestRoundTripper(func(req *http.Request) (*http.Response, error) {
		callCount++
		if got := req.Header.Get("apikey"); got != "secret-api-key" {
			t.Fatalf("apikey header = %q, want %q", got, "secret-api-key")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":{"EUR":{"code":"EUR","value":0.92}}}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}

	if err := svc.RefreshRates(); err != nil {
		t.Fatalf("RefreshRates() error = %v, want nil", err)
	}
	if callCount == 0 {
		t.Fatal("RefreshRates() made no premium API requests")
	}
}

func TestRefreshRatesMigratesLegacyPlaintextCurrencyAPIKey(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}, &model.ExchangeRate{}); err != nil {
		t.Fatalf("failed to migrate exchange rate security tables: %v", err)
	}

	seedSystemSetting(t, db, "currencyapi_key", "legacy-secret")
	seedSystemSetting(t, db, "exchange_rate_source", "premium")

	originalCurrencies := commonCurrencies
	commonCurrencies = []string{"usd", "eur"}
	defer func() {
		commonCurrencies = originalCurrencies
	}()

	svc := NewExchangeRateService(db)
	svc.httpClient = &http.Client{Transport: exchangeRateTestRoundTripper(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("apikey"); got != "legacy-secret" {
			t.Fatalf("apikey header = %q, want %q", got, "legacy-secret")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":{"EUR":{"code":"EUR","value":0.92}}}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}

	if err := svc.RefreshRates(); err != nil {
		t.Fatalf("RefreshRates() error = %v, want nil", err)
	}

	var stored model.SystemSetting
	if err := db.Where("key = ?", "currencyapi_key").First(&stored).Error; err != nil {
		t.Fatalf("loading migrated currencyapi_key failed: %v", err)
	}
	if stored.Value == "legacy-secret" {
		t.Fatal("currencyapi_key should be migrated away from plaintext storage")
	}
	if !strings.HasPrefix(stored.Value, "enc:v1:") {
		t.Fatalf("currencyapi_key value = %q, want encrypted prefix", stored.Value)
	}
}

func seedSystemSetting(t *testing.T, db *gorm.DB, key string, value string) {
	t.Helper()
	if err := db.Where("key = ?", key).
		Assign(model.SystemSetting{Value: value}).
		FirstOrCreate(&model.SystemSetting{Key: key}).Error; err != nil {
		t.Fatalf("failed to seed setting %q: %v", key, err)
	}
}
