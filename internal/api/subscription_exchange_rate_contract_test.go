package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	exchangerate "github.com/kasuha07/subdux/internal/service/exchangerate"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"github.com/labstack/echo/v4"
)

func TestSubscriptionReadEndpointsExposeExchangeRateUnavailable(t *testing.T) {
	db := newBootstrapTestDB(t)
	user := model.User{
		Username: "exchange-contract",
		Email:    "exchange-contract@example.com",
		Password: "x",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	monthly := 1
	subscriptions := subscriptionservice.NewService(db)
	if _, err := subscriptions.Create(user.ID, subscriptionservice.CreateSubscriptionInput{
		Name:            "Euro Subscription",
		Amount:          12,
		Currency:        "EUR",
		Status:          "active",
		RenewalMode:     "auto_renew",
		BillingType:     "recurring",
		RecurrenceType:  "interval",
		IntervalCount:   &monthly,
		IntervalUnit:    "month",
		NextBillingDate: "2026-08-15",
	}); err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	exchangeRates := exchangerate.NewService(db)
	handler := NewSubscriptionHandler(subscriptions, exchangeRates)
	bootstrapHandler := NewDashboardBootstrapHandler(
		subscriptions,
		exchangeRates,
		catalogservice.NewCurrencyService(db),
		catalogservice.NewCategoryService(db),
		catalogservice.NewPaymentMethodService(db),
	)
	tests := []struct {
		name string
		path string
		call func(echo.Context) error
	}{
		{name: "dashboard summary", path: "/api/dashboard/summary", call: handler.Dashboard},
		{name: "dashboard bootstrap", path: "/api/dashboard/bootstrap", call: bootstrapHandler.Get},
		{name: "analytics report", path: "/api/reports/analytics", call: handler.AnalyticsReport},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.HTTPErrorHandler = APIErrorHandler(e.HTTPErrorHandler)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			c := authedContext(e, rec, req, user.ID)

			if err := tt.call(c); err != nil {
				e.HTTPErrorHandler(err, c)
			}

			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
			}
			if !hasErrorCode(rec.Body.String(), subscriptionservice.ErrExchangeRateUnavailable.Code) {
				t.Fatalf(
					"body = %s, want error_code %q",
					rec.Body.String(),
					subscriptionservice.ErrExchangeRateUnavailable.Code,
				)
			}
		})
	}
}
