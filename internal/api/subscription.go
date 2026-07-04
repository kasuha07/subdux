package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/kasuha07/subdux/internal/api/apicontract"
	"github.com/kasuha07/subdux/internal/model"
	exchangerate "github.com/kasuha07/subdux/internal/service/exchangerate"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type SubscriptionHandler struct {
	Service   *subscriptionservice.Service
	ERService *exchangerate.Service
}

func NewSubscriptionHandler(s *subscriptionservice.Service, er *exchangerate.Service) *SubscriptionHandler {
	return &SubscriptionHandler{Service: s, ERService: er}
}

type subscriptionResponse = apicontract.SubscriptionResponse

type subscriptionDetailResponse struct {
	Subscription     subscriptionResponse                                     `json:"subscription"`
	Timeline         []subscriptionservice.SubscriptionDetailEvent            `json:"timeline"`
	PriceHistory     []subscriptionservice.SubscriptionDetailPriceHistoryItem `json:"price_history"`
	NotificationLogs []subscriptionservice.SubscriptionDetailNotificationLog  `json:"notification_logs"`
	UpcomingCharges  []subscriptionservice.SubscriptionDetailUpcomingCharge   `json:"upcoming_charges"`
	Calendar         subscriptionservice.SubscriptionDetailCalendar           `json:"calendar"`
}

func mapSubscriptionResponse(sub model.Subscription) subscriptionResponse {
	return apicontract.MapSubscriptionResponse(sub)
}

func mapSubscriptionDetailResponse(detail subscriptionservice.SubscriptionDetail) subscriptionDetailResponse {
	return subscriptionDetailResponse{
		Subscription:     mapSubscriptionResponse(detail.Subscription),
		Timeline:         detail.Timeline,
		PriceHistory:     detail.PriceHistory,
		NotificationLogs: detail.NotificationLogs,
		UpcomingCharges:  detail.UpcomingCharges,
		Calendar:         detail.Calendar,
	}
}

func mapSubscriptionResponses(subs []model.Subscription) []subscriptionResponse {
	return apicontract.MapSubscriptionResponses(subs)
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
	id, ok := parseUintParam(c, "id", "Invalid ID")
	if !ok {
		return nil
	}

	sub, err := h.Service.WithContext(c.Request().Context()).GetByID(userID, uint(id))
	if err != nil {
		return writeError(c, http.StatusNotFound, "Subscription not found")
	}

	return c.JSON(http.StatusOK, mapSubscriptionResponse(*sub))
}

func (h *SubscriptionHandler) GetDetail(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "Invalid ID")
	if !ok {
		return nil
	}

	detail, err := h.Service.WithContext(c.Request().Context()).GetDetail(userID, uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return writeError(c, http.StatusNotFound, "Subscription not found")
		}
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, mapSubscriptionDetailResponse(*detail))
}

func (h *SubscriptionHandler) Create(c echo.Context) error {
	userID := getUserID(c)
	var input subscriptionservice.CreateSubscriptionInput
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}

	if input.Name == "" {
		return writeError(c, http.StatusBadRequest, "Name is required")
	}
	if input.Amount < 0 {
		return writeError(c, http.StatusBadRequest, "Amount must not be negative")
	}
	if !validateSubscriptionIcon(input.Icon) {
		return writeError(c, http.StatusBadRequest, "Invalid icon value")
	}

	sub, err := h.Service.WithContext(c.Request().Context()).Create(userID, input)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorFunc(http.StatusBadRequest, func(err error) bool {
				return isSubscriptionBadRequestError(err.Error())
			}),
		)
	}

	return c.JSON(http.StatusCreated, mapSubscriptionResponse(*sub))
}

func (h *SubscriptionHandler) Update(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "Invalid ID")
	if !ok {
		return nil
	}

	var input subscriptionservice.UpdateSubscriptionInput
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}
	if input.Amount != nil && *input.Amount < 0 {
		return writeError(c, http.StatusBadRequest, "Amount must not be negative")
	}
	if input.Icon != nil && !validateSubscriptionIcon(*input.Icon) {
		return writeError(c, http.StatusBadRequest, "Invalid icon value")
	}

	sub, err := h.Service.WithContext(c.Request().Context()).Update(userID, uint(id), input)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorFunc(http.StatusBadRequest, func(err error) bool {
				return isSubscriptionBadRequestError(err.Error())
			}),
		)
	}

	return c.JSON(http.StatusOK, mapSubscriptionResponse(*sub))
}

func (h *SubscriptionHandler) Delete(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "Invalid ID")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).Delete(userID, uint(id)); err != nil {
		return writeServiceError(c, err,
			serviceErrorFunc(http.StatusBadRequest, func(err error) bool {
				return isSubscriptionBadRequestError(err.Error())
			}),
		)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *SubscriptionHandler) MarkRenewed(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "Invalid ID")
	if !ok {
		return nil
	}

	sub, err := h.Service.WithContext(c.Request().Context()).MarkManualRenewed(userID, uint(id))
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorFunc(http.StatusBadRequest, func(err error) bool {
				return isSubscriptionBadRequestError(err.Error())
			}),
		)
	}

	return c.JSON(http.StatusOK, mapSubscriptionResponse(*sub))
}

// Reconcile persists any due lifecycle transitions for the caller's
// subscriptions on demand, then returns the refreshed list. Reads advance
// lifecycle in memory only; this endpoint is the explicit repair entry that
// forces those transitions to disk without waiting for the background sweep.
func (h *SubscriptionHandler) Reconcile(c echo.Context) error {
	userID := getUserID(c)
	svc := h.Service.WithContext(c.Request().Context())

	if err := svc.ReconcileUserLifecycle(userID); err != nil {
		return writeInternalServerError(c, err)
	}

	subs, err := svc.List(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, mapSubscriptionResponses(subs))
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
	var input subscriptionservice.SnoozeSubscriptionActionInput
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}

	snooze, err := h.Service.WithContext(c.Request().Context()).SnoozeAction(userID, input)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusNotFound, "subscription not found"),
			serviceErrorFunc(http.StatusBadRequest, func(err error) bool {
				message := err.Error()
				return isSubscriptionBadRequestError(message) ||
					strings.Contains(message, "action key") ||
					strings.Contains(message, "snooze")
			}),
		)
	}

	return c.JSON(http.StatusOK, snooze)
}

func (h *SubscriptionHandler) UploadIcon(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "Invalid ID")
	if !ok {
		return nil
	}

	svc := h.Service.WithContext(c.Request().Context())

	fileHeader, err := c.FormFile("icon")
	if err != nil {
		return writeError(c, http.StatusBadRequest, "no file provided")
	}

	maxSize := svc.GetMaxIconFileSize()

	src, err := fileHeader.Open()
	if err != nil {
		return writeError(c, http.StatusInternalServerError, "failed to read file")
	}
	defer src.Close()

	iconPath, err := svc.UploadSubscriptionIcon(userID, uint(id), src, fileHeader.Filename, maxSize)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorFunc(http.StatusForbidden, isIconUploadForbiddenError),
			serviceErrorFunc(http.StatusBadRequest, isIconUploadBadRequestError),
		)
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
