package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	exchangerate "github.com/kasuha07/subdux/internal/service/exchangerate"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"github.com/labstack/echo/v4"
)

// DashboardBootstrapHandler serves the dashboard's first-screen payload in a
// single request. The dashboard previously fanned out to six endpoints in
// parallel, but under SQLite's single-writer queue that parallelism collapses
// into a serial wait and two of those endpoints each reconciled lifecycle and
// re-read the subscriptions table. Aggregating here reconciles once, reads each
// table once, and returns one response.
type DashboardBootstrapHandler struct {
	Subscriptions  *subscriptionservice.Service
	ExchangeRates  *exchangerate.Service
	Currencies     *catalogservice.CurrencyService
	Categories     *catalogservice.CategoryService
	PaymentMethods *catalogservice.PaymentMethodService
}

func NewDashboardBootstrapHandler(
	subscriptions *subscriptionservice.Service,
	exchangeRates *exchangerate.Service,
	currencies *catalogservice.CurrencyService,
	categories *catalogservice.CategoryService,
	paymentMethods *catalogservice.PaymentMethodService,
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
	Subscriptions     []subscriptionResponse                `json:"subscriptions"`
	Summary           *subscriptionservice.DashboardSummary `json:"summary"`
	Categories        []categoryResponse                    `json:"categories"`
	PaymentMethods    []paymentMethodResponse               `json:"payment_methods"`
	Currencies        []userCurrencyResponse                `json:"currencies"`
	PreferredCurrency string                                `json:"preferred_currency"`
}

func (h *DashboardBootstrapHandler) Get(c echo.Context) error {
	userID := apimw.From(c).UserID
	ctx := c.Request().Context()

	erService := h.ExchangeRates.WithContext(ctx)
	pref, err := erService.GetUserPreference(userID)
	if err != nil {
		return err
	}

	subs, summary, err := h.Subscriptions.WithContext(ctx).
		SubscriptionsWithSummary(userID, pref.PreferredCurrency, erService)
	if err != nil {
		return err
	}

	categories, err := h.Categories.WithContext(ctx).List(userID)
	if err != nil {
		return err
	}

	paymentMethods, err := h.PaymentMethods.WithContext(ctx).List(userID)
	if err != nil {
		return err
	}

	currencies, err := h.Currencies.WithContext(ctx).List(userID)
	if err != nil {
		return err
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
