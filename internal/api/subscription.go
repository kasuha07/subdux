package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/service"
)

type SubscriptionHandler struct {
	Service   *service.SubscriptionService
	ERService *service.ExchangeRateService
}

func NewSubscriptionHandler(s *service.SubscriptionService, er *service.ExchangeRateService) *SubscriptionHandler {
	return &SubscriptionHandler{Service: s, ERService: er}
}

type subscriptionResponse struct {
	ID                uint       `json:"id"`
	Name              string     `json:"name"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	Enabled           bool       `json:"enabled"`
	BillingType       string     `json:"billing_type"`
	RecurrenceType    string     `json:"recurrence_type"`
	IntervalCount     *int       `json:"interval_count"`
	IntervalUnit      string     `json:"interval_unit"`
	BillingAnchorDate *time.Time `json:"billing_anchor_date"`
	MonthlyDay        *int       `json:"monthly_day"`
	YearlyMonth       *int       `json:"yearly_month"`
	YearlyDay         *int       `json:"yearly_day"`
	TrialEnabled      bool       `json:"trial_enabled"`
	TrialStartDate    *time.Time `json:"trial_start_date"`
	TrialEndDate      *time.Time `json:"trial_end_date"`
	NextBillingDate   *time.Time `json:"next_billing_date"`
	Category          string     `json:"category"`
	CategoryID        *uint      `json:"category_id"`
	PaymentMethodID   *uint      `json:"payment_method_id"`
	Icon              string     `json:"icon"`
	URL               string     `json:"url"`
	Notes             string     `json:"notes"`
	CreatedAt         time.Time  `json:"created_at"`
}

func mapSubscriptionResponse(sub model.Subscription) subscriptionResponse {
	return subscriptionResponse{
		ID:                sub.ID,
		Name:              sub.Name,
		Amount:            sub.Amount,
		Currency:          sub.Currency,
		Enabled:           sub.Enabled,
		BillingType:       sub.BillingType,
		RecurrenceType:    sub.RecurrenceType,
		IntervalCount:     sub.IntervalCount,
		IntervalUnit:      sub.IntervalUnit,
		BillingAnchorDate: sub.BillingAnchorDate,
		MonthlyDay:        sub.MonthlyDay,
		YearlyMonth:       sub.YearlyMonth,
		YearlyDay:         sub.YearlyDay,
		TrialEnabled:      sub.TrialEnabled,
		TrialStartDate:    sub.TrialStartDate,
		TrialEndDate:      sub.TrialEndDate,
		NextBillingDate:   sub.NextBillingDate,
		Category:          sub.Category,
		CategoryID:        sub.CategoryID,
		PaymentMethodID:   sub.PaymentMethodID,
		Icon:              sub.Icon,
		URL:               sub.URL,
		Notes:             sub.Notes,
		CreatedAt:         sub.CreatedAt,
	}
}

func mapSubscriptionResponses(subs []model.Subscription) []subscriptionResponse {
	responses := make([]subscriptionResponse, len(subs))
	for i, sub := range subs {
		responses[i] = mapSubscriptionResponse(sub)
	}
	return responses
}

func (h *SubscriptionHandler) List(c echo.Context) error {
	userID := getUserID(c)
	subs, err := h.Service.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, mapSubscriptionResponses(subs))
}

func (h *SubscriptionHandler) GetByID(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	sub, err := h.Service.GetByID(userID, uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Subscription not found"})
	}

	return c.JSON(http.StatusOK, mapSubscriptionResponse(*sub))
}

func (h *SubscriptionHandler) Create(c echo.Context) error {
	userID := getUserID(c)
	var input service.CreateSubscriptionInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	if input.Name == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Name is required"})
	}
	if input.BillingType == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Billing type is required"})
	}
	if input.Amount <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Amount must be positive"})
	}
	if !validateIcon(input.Icon) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid icon value"})
	}

	sub, err := h.Service.Create(userID, input)
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, mapSubscriptionResponse(*sub))
}

func (h *SubscriptionHandler) Update(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	var input service.UpdateSubscriptionInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.Amount != nil && *input.Amount <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Amount must be positive"})
	}
	if input.Icon != nil && !validateIcon(*input.Icon) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid icon value"})
	}

	sub, err := h.Service.Update(userID, uint(id), input)
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, mapSubscriptionResponse(*sub))
}

func (h *SubscriptionHandler) Delete(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	if err := h.Service.Delete(userID, uint(id)); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *SubscriptionHandler) UploadIcon(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	fileHeader, err := c.FormFile("icon")
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "no file provided"})
	}

	maxSize := h.Service.GetMaxIconFileSize()

	src, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to read file"})
	}
	defer src.Close()

	iconPath, err := h.Service.UploadSubscriptionIcon(userID, uint(id), src, fileHeader.Filename, maxSize)
	if err != nil {
		msg := err.Error()
		if msg == "only PNG and JPG images are supported" || msg == "file size exceeds limit" || msg == "subscription not found" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": msg})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": msg})
	}

	return c.JSON(http.StatusOK, echo.Map{"icon": iconPath})
}

func (h *SubscriptionHandler) Dashboard(c echo.Context) error {
	userID := getUserID(c)

	pref, _ := h.ERService.GetUserPreference(userID)
	targetCurrency := pref.PreferredCurrency

	summary, err := h.Service.GetDashboardSummary(userID, targetCurrency, h.ERService)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, summary)
}

func isSubscriptionBadRequestError(message string) bool {
	if message == "payment method not found" {
		return true
	}
	return strings.Contains(message, "required") ||
		strings.Contains(message, "must be") ||
		strings.Contains(message, "invalid date format")
}
