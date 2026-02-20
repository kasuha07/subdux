package api

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

func isEmojiRune(r rune) bool {
	if r == '\u200D' || r == '\uFE0F' || r == '\uFE0E' {
		return true
	}
	if r >= 0x1F1E0 && r <= 0x1F1FF {
		return true
	}
	if r < 0x00A0 {
		return false
	}
	return unicode.IsGraphic(r) && !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsPunct(r) && !unicode.IsSpace(r)
}

func validateIcon(icon string) bool {
	if icon == "" {
		return true
	}

	if isManagedAssetIcon(icon) {
		return true
	}

	if strings.HasPrefix(icon, "si:") ||
		strings.HasPrefix(icon, "http://") ||
		strings.HasPrefix(icon, "https://") {
		return true
	}
	for _, r := range icon {
		if !isEmojiRune(r) {
			return false
		}
	}
	return true
}

func isManagedAssetIcon(icon string) bool {
	const iconPrefix = "assets/icons/"
	if !strings.HasPrefix(icon, iconPrefix) {
		return false
	}

	filename := strings.TrimPrefix(icon, iconPrefix)
	if filename == "" {
		return false
	}
	if strings.Contains(filename, "/") || strings.Contains(filename, `\`) {
		return false
	}
	if filepath.Base(filename) != filename {
		return false
	}

	return true
}

type SubscriptionHandler struct {
	Service   *service.SubscriptionService
	ERService *service.ExchangeRateService
}

func NewSubscriptionHandler(s *service.SubscriptionService, er *service.ExchangeRateService) *SubscriptionHandler {
	return &SubscriptionHandler{Service: s, ERService: er}
}

func (h *SubscriptionHandler) List(c echo.Context) error {
	userID := getUserID(c)
	subs, err := h.Service.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, subs)
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

	return c.JSON(http.StatusOK, sub)
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
	if input.Amount <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Amount must be positive"})
	}
	if input.BillingCycle == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Billing cycle is required"})
	}
	if !validateIcon(input.Icon) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid icon value"})
	}

	sub, err := h.Service.Create(userID, input)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, sub)
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
	if input.Icon != nil && !validateIcon(*input.Icon) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid icon value"})
	}

	sub, err := h.Service.Update(userID, uint(id), input)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, sub)
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
