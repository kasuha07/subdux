package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

// DashboardBootstrapHandler serves the dashboard's first-screen payload in a
// single request. The dashboard previously fanned out to six endpoints in
// parallel, but under SQLite's single-writer queue that parallelism collapses
// into a serial wait and two of those endpoints each reconciled lifecycle and
// re-read the subscriptions table. Aggregating here reconciles once, reads each
// table once, and returns one response.
type DashboardBootstrapHandler struct {
	Subscriptions  *service.SubscriptionService
	ExchangeRates  *service.ExchangeRateService
	Currencies     *service.CurrencyService
	Categories     *service.CategoryService
	PaymentMethods *service.PaymentMethodService
}

func NewDashboardBootstrapHandler(
	subscriptions *service.SubscriptionService,
	exchangeRates *service.ExchangeRateService,
	currencies *service.CurrencyService,
	categories *service.CategoryService,
	paymentMethods *service.PaymentMethodService,
) *DashboardBootstrapHandler {
	return &DashboardBootstrapHandler{
		Subscriptions:  subscriptions,
		ExchangeRates:  exchangeRates,
		Currencies:     currencies,
		Categories:     categories,
		PaymentMethods: paymentMethods,
	}
}

// dashboardBootstrapResponse mirrors the shapes of the individual endpoints it
// replaces (/subscriptions, /dashboard/summary, /categories, /payment-methods,
// /currencies, /preferences/currency) by reusing their response mappers, so the
// frontend can consume it without new types.
type dashboardBootstrapResponse struct {
	Subscriptions     []subscriptionResponse    `json:"subscriptions"`
	Summary           *service.DashboardSummary `json:"summary"`
	Categories        []categoryResponse        `json:"categories"`
	PaymentMethods    []paymentMethodResponse   `json:"payment_methods"`
	Currencies        []userCurrencyResponse    `json:"currencies"`
	PreferredCurrency string                    `json:"preferred_currency"`
}

func (h *DashboardBootstrapHandler) Get(c echo.Context) error {
	userID := getUserID(c)
	ctx := c.Request().Context()

	erService := h.ExchangeRates.WithContext(ctx)
	pref, err := erService.GetUserPreference(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}

	subs, summary, err := h.Subscriptions.WithContext(ctx).
		SubscriptionsWithSummary(userID, pref.PreferredCurrency, erService)
	if err != nil {
		return writeInternalServerError(c, err)
	}

	categories, err := h.Categories.WithContext(ctx).List(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}

	paymentMethods, err := h.PaymentMethods.WithContext(ctx).List(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}

	currencies, err := h.Currencies.WithContext(ctx).List(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, dashboardBootstrapResponse{
		Subscriptions:     mapSubscriptionResponses(subs),
		Summary:           summary,
		Categories:        mapCategoryResponses(categories),
		PaymentMethods:    mapPaymentMethodResponses(paymentMethods),
		Currencies:        mapUserCurrencyResponses(currencies),
		PreferredCurrency: pref.PreferredCurrency,
	})
}
