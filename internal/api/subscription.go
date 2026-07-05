package api

import (
	"errors"
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/contract"
	"github.com/kasuha07/subdux/internal/api/httpx"
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

type subscriptionResponse = contract.SubscriptionResponse

type subscriptionDetailResponse struct {
	Subscription     subscriptionResponse                                     `json:"subscription"`
	Timeline         []subscriptionservice.SubscriptionDetailEvent            `json:"timeline"`
	PriceHistory     []subscriptionservice.SubscriptionDetailPriceHistoryItem `json:"price_history"`
	NotificationLogs []subscriptionservice.SubscriptionDetailNotificationLog  `json:"notification_logs"`
	UpcomingCharges  []subscriptionservice.SubscriptionDetailUpcomingCharge   `json:"upcoming_charges"`
	Calendar         subscriptionservice.SubscriptionDetailCalendar           `json:"calendar"`
}

func mapSubscriptionResponse(sub model.Subscription) subscriptionResponse {
	return contract.MapSubscriptionResponse(sub)
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
	return contract.MapSubscriptionResponses(subs)
}

func (h *SubscriptionHandler) List(c echo.Context) error {
	userID := apimw.From(c).UserID
	subs, err := h.Service.WithContext(c.Request().Context()).List(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapSubscriptionResponses(subs))
}

func (h *SubscriptionHandler) GetByID(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}

	sub, err := h.Service.WithContext(c.Request().Context()).GetByID(userID, uint(id))
	if err != nil {
		return httpx.WriteError(c, http.StatusNotFound, "subscription_not_found")
	}

	return c.JSON(http.StatusOK, mapSubscriptionResponse(*sub))
}

func (h *SubscriptionHandler) GetDetail(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}

	detail, err := h.Service.WithContext(c.Request().Context()).GetDetail(userID, uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.WriteError(c, http.StatusNotFound, "subscription_not_found")
		}
		return err
	}

	return c.JSON(http.StatusOK, mapSubscriptionDetailResponse(*detail))
}

func (h *SubscriptionHandler) Create(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input subscriptionservice.CreateSubscriptionInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	if input.Name == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "name_is_required")
	}
	if input.Amount < 0 {
		return httpx.WriteError(c, http.StatusBadRequest, "amount_must_not_be_negative")
	}
	if !validateSubscriptionIcon(input.Icon) {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid_icon_value")
	}

	sub, err := h.Service.WithContext(c.Request().Context()).Create(userID, input)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, mapSubscriptionResponse(*sub))
}

func (h *SubscriptionHandler) Update(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}

	var input subscriptionservice.UpdateSubscriptionInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	if input.Amount != nil && *input.Amount < 0 {
		return httpx.WriteError(c, http.StatusBadRequest, "amount_must_not_be_negative")
	}
	if input.Icon != nil && !validateSubscriptionIcon(*input.Icon) {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid_icon_value")
	}

	sub, err := h.Service.WithContext(c.Request().Context()).Update(userID, uint(id), input)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, mapSubscriptionResponse(*sub))
}

func (h *SubscriptionHandler) Delete(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).Delete(userID, uint(id)); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *SubscriptionHandler) MarkRenewed(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}

	sub, err := h.Service.WithContext(c.Request().Context()).MarkManualRenewed(userID, uint(id))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, mapSubscriptionResponse(*sub))
}

// Reconcile persists any due lifecycle transitions for the caller's
// subscriptions on demand, then returns the refreshed list. Reads advance
// lifecycle in memory only; this endpoint is the explicit repair entry that
// forces those transitions to disk without waiting for the background sweep.
func (h *SubscriptionHandler) Reconcile(c echo.Context) error {
	userID := apimw.From(c).UserID
	svc := h.Service.WithContext(c.Request().Context())

	if err := svc.ReconcileUserLifecycle(userID); err != nil {
		return err
	}

	subs, err := svc.List(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapSubscriptionResponses(subs))
}

func (h *SubscriptionHandler) ActionCenter(c echo.Context) error {
	userID := apimw.From(c).UserID
	center, err := h.Service.WithContext(c.Request().Context()).GetActionCenter(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, center)
}

func (h *SubscriptionHandler) SnoozeAction(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input subscriptionservice.SnoozeSubscriptionActionInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	snooze, err := h.Service.WithContext(c.Request().Context()).SnoozeAction(userID, input)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, snooze)
}

func (h *SubscriptionHandler) UploadIcon(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}

	svc := h.Service.WithContext(c.Request().Context())

	fileHeader, err := c.FormFile("icon")
	if err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "no_file_provided")
	}

	maxSize := svc.GetMaxIconFileSize()

	src, err := fileHeader.Open()
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed_to_read_file")
	}
	defer src.Close()

	iconPath, err := svc.UploadSubscriptionIcon(userID, uint(id), src, fileHeader.Filename, maxSize)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{"icon": iconPath})
}

func (h *SubscriptionHandler) Dashboard(c echo.Context) error {
	userID := apimw.From(c).UserID
	ctx := c.Request().Context()
	erService := h.ERService.WithContext(ctx)

	pref, _ := erService.GetUserPreference(userID)
	targetCurrency := pref.PreferredCurrency

	summary, err := h.Service.WithContext(ctx).GetDashboardSummary(userID, targetCurrency, erService)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, summary)
}

func (h *SubscriptionHandler) AnalyticsReport(c echo.Context) error {
	userID := apimw.From(c).UserID
	ctx := c.Request().Context()
	erService := h.ERService.WithContext(ctx)

	pref, _ := erService.GetUserPreference(userID)
	targetCurrency := pref.PreferredCurrency

	report, err := h.Service.WithContext(ctx).GetAnalyticsReport(userID, targetCurrency, erService)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, report)
}
