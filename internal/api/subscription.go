package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/service"
	"gorm.io/gorm"
)

type SubscriptionHandler struct {
	Service   *service.SubscriptionService
	ERService *service.ExchangeRateService
}

func NewSubscriptionHandler(s *service.SubscriptionService, er *service.ExchangeRateService) *SubscriptionHandler {
	return &SubscriptionHandler{Service: s, ERService: er}
}

type subscriptionResponse struct {
	ID               uint      `json:"id"`
	Name             string    `json:"name"`
	Amount           float64   `json:"amount"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	RenewalMode      string    `json:"renewal_mode"`
	EndsAt           *string   `json:"ends_at"`
	BillingType      string    `json:"billing_type"`
	RecurrenceType   string    `json:"recurrence_type"`
	IntervalCount    *int      `json:"interval_count"`
	IntervalUnit     string    `json:"interval_unit"`
	MonthlyDay       *int      `json:"monthly_day"`
	YearlyMonth      *int      `json:"yearly_month"`
	YearlyDay        *int      `json:"yearly_day"`
	NextBillingDate  *string   `json:"next_billing_date"`
	Category         string    `json:"category"`
	CategoryID       *uint     `json:"category_id"`
	PaymentMethodID  *uint     `json:"payment_method_id"`
	NotifyEnabled    *bool     `json:"notify_enabled"`
	NotifyDaysBefore *int      `json:"notify_days_before"`
	Icon             string    `json:"icon"`
	URL              string    `json:"url"`
	Notes            string    `json:"notes"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type subscriptionDetailResponse struct {
	Subscription     subscriptionResponse                         `json:"subscription"`
	Timeline         []service.SubscriptionDetailEvent            `json:"timeline"`
	PriceHistory     []service.SubscriptionDetailPriceHistoryItem `json:"price_history"`
	NotificationLogs []service.SubscriptionDetailNotificationLog  `json:"notification_logs"`
	UpcomingCharges  []service.SubscriptionDetailUpcomingCharge   `json:"upcoming_charges"`
	Calendar         service.SubscriptionDetailCalendar           `json:"calendar"`
}

func mapSubscriptionResponse(sub model.Subscription) subscriptionResponse {
	return subscriptionResponse{
		ID:               sub.ID,
		Name:             sub.Name,
		Amount:           sub.Amount,
		Currency:         sub.Currency,
		Status:           sub.Status,
		RenewalMode:      sub.RenewalMode,
		EndsAt:           formatDateOnly(sub.EndsAt),
		BillingType:      sub.BillingType,
		RecurrenceType:   sub.RecurrenceType,
		IntervalCount:    sub.IntervalCount,
		IntervalUnit:     sub.IntervalUnit,
		MonthlyDay:       sub.MonthlyDay,
		YearlyMonth:      sub.YearlyMonth,
		YearlyDay:        sub.YearlyDay,
		NextBillingDate:  formatDateOnly(sub.NextBillingDate),
		Category:         sub.Category,
		CategoryID:       sub.CategoryID,
		PaymentMethodID:  sub.PaymentMethodID,
		NotifyEnabled:    sub.NotifyEnabled,
		NotifyDaysBefore: sub.NotifyDaysBefore,
		Icon:             sub.Icon,
		URL:              sub.URL,
		Notes:            sub.Notes,
		CreatedAt:        sub.CreatedAt,
		UpdatedAt:        sub.UpdatedAt,
	}
}

func mapSubscriptionDetailResponse(detail service.SubscriptionDetail) subscriptionDetailResponse {
	return subscriptionDetailResponse{
		Subscription:     mapSubscriptionResponse(detail.Subscription),
		Timeline:         detail.Timeline,
		PriceHistory:     detail.PriceHistory,
		NotificationLogs: detail.NotificationLogs,
		UpcomingCharges:  detail.UpcomingCharges,
		Calendar:         detail.Calendar,
	}
}

func formatDateOnly(value *time.Time) *string {
	if value == nil {
		return nil
	}

	formatted := value.Format("2006-01-02")
	return &formatted
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
	subs, err := h.Service.WithContext(c.Request().Context()).List(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, mapSubscriptionResponses(subs))
}

func (h *SubscriptionHandler) GetByID(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	sub, err := h.Service.WithContext(c.Request().Context()).GetByID(userID, uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Subscription not found"})
	}

	return c.JSON(http.StatusOK, mapSubscriptionResponse(*sub))
}

func (h *SubscriptionHandler) GetDetail(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	detail, err := h.Service.WithContext(c.Request().Context()).GetDetail(userID, uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "Subscription not found"})
		}
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, mapSubscriptionDetailResponse(*detail))
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
	if input.Amount < 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Amount must not be negative"})
	}
	if !validateSubscriptionIcon(input.Icon) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid icon value"})
	}

	sub, err := h.Service.WithContext(c.Request().Context()).Create(userID, input)
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
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
	if input.Amount != nil && *input.Amount < 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Amount must not be negative"})
	}
	if input.Icon != nil && !validateSubscriptionIcon(*input.Icon) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid icon value"})
	}

	sub, err := h.Service.WithContext(c.Request().Context()).Update(userID, uint(id), input)
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, mapSubscriptionResponse(*sub))
}

func (h *SubscriptionHandler) Delete(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	if err := h.Service.WithContext(c.Request().Context()).Delete(userID, uint(id)); err != nil {
		if isSubscriptionBadRequestError(err.Error()) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *SubscriptionHandler) MarkRenewed(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	sub, err := h.Service.WithContext(c.Request().Context()).MarkManualRenewed(userID, uint(id))
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, mapSubscriptionResponse(*sub))
}

func (h *SubscriptionHandler) ActionCenter(c echo.Context) error {
	userID := getUserID(c)
	center, err := h.Service.WithContext(c.Request().Context()).GetActionCenter(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, center)
}

func (h *SubscriptionHandler) SnoozeAction(c echo.Context) error {
	userID := getUserID(c)
	var input service.SnoozeSubscriptionActionInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	snooze, err := h.Service.WithContext(c.Request().Context()).SnoozeAction(userID, input)
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) || strings.Contains(err.Error(), "action key") || strings.Contains(err.Error(), "snooze") {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		if err.Error() == "subscription not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, snooze)
}

func (h *SubscriptionHandler) UploadIcon(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	svc := h.Service.WithContext(c.Request().Context())

	fileHeader, err := c.FormFile("icon")
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "no file provided"})
	}

	maxSize := svc.GetMaxIconFileSize()

	src, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to read file"})
	}
	defer src.Close()

	iconPath, err := svc.UploadSubscriptionIcon(userID, uint(id), src, fileHeader.Filename, maxSize)
	if err != nil {
		if isIconUploadForbiddenError(err) {
			return c.JSON(http.StatusForbidden, echo.Map{"error": err.Error()})
		}
		if isIconUploadBadRequestError(err) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"icon": iconPath})
}

func (h *SubscriptionHandler) Dashboard(c echo.Context) error {
	userID := getUserID(c)
	ctx := c.Request().Context()
	erService := h.ERService.WithContext(ctx)

	pref, _ := erService.GetUserPreference(userID)
	targetCurrency := pref.PreferredCurrency

	summary, err := h.Service.WithContext(ctx).GetDashboardSummary(userID, targetCurrency, erService)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, summary)
}

func (h *SubscriptionHandler) AnalyticsReport(c echo.Context) error {
	userID := getUserID(c)
	ctx := c.Request().Context()
	erService := h.ERService.WithContext(ctx)

	pref, _ := erService.GetUserPreference(userID)
	targetCurrency := pref.PreferredCurrency

	report, err := h.Service.WithContext(ctx).GetAnalyticsReport(userID, targetCurrency, erService)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, report)
}

func isSubscriptionBadRequestError(message string) bool {
	if message == "payment method not found" || message == "category not found" {
		return true
	}
	return strings.Contains(message, "required") ||
		strings.Contains(message, "must be") ||
		strings.Contains(message, "invalid date format") ||
		strings.Contains(message, "invalid subscription url") ||
		strings.Contains(message, "no longer supported") ||
		strings.Contains(message, "read-only") ||
		strings.Contains(message, "only ")
}
