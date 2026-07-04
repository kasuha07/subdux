package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/kasuha07/subdux/internal/pkg"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

func (h *AuthHandler) SetupTOTP(c echo.Context) error {
	userID := getUserID(c)
	var input struct {
		ReauthTicket string `json:"reauth_ticket"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if err := h.Reauth.WithContext(c.Request().Context()).Consume(userID, servicereauth.ReauthOperationEnableTOTP, input.ReauthTicket); err != nil {
		return writeReauthError(c, err)
	}

	result, err := h.TOTPService.WithContext(c.Request().Context()).BeginSetup(userID)
	if err != nil {
		return writeTOTPServiceError(c, err)
	}
	return c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) ConfirmTOTP(c echo.Context) error {
	userID := getUserID(c)
	var input struct {
		SessionID string `json:"session_id"`
		Code      string `json:"code"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.SessionID == "" || input.Code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "session_id and code are required"})
	}

	backupCodes, err := h.TOTPService.WithContext(c.Request().Context()).ConfirmSetup(userID, input.SessionID, input.Code)
	if err != nil {
		return writeTOTPServiceError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"backup_codes": backupCodes})
}

func (h *AuthHandler) DisableTOTP(c echo.Context) error {
	userID := getUserID(c)
	var input struct {
		ReauthTicket string `json:"reauth_ticket"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if err := h.Reauth.WithContext(c.Request().Context()).Consume(userID, servicereauth.ReauthOperationDisableTOTP, input.ReauthTicket); err != nil {
		return writeReauthError(c, err)
	}

	if err := h.TOTPService.WithContext(c.Request().Context()).Disable(userID); err != nil {
		return writeTOTPServiceError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "2FA disabled successfully"})
}

func writeTOTPServiceError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, serviceauth.ErrTOTPAlreadyEnabled),
		errors.Is(err, serviceauth.ErrTOTPSetupExpired),
		errors.Is(err, serviceauth.ErrTOTPInvalidCode),
		errors.Is(err, serviceauth.ErrTOTPInvalidPassword),
		errors.Is(err, serviceauth.ErrTOTPInvalidAuthCode):
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	case errors.Is(err, serviceauth.ErrUserNotFound):
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	default:
		return writeInternalServerError(c, err)
	}
}

type verifyTOTPLoginInput struct {
	TotpToken string `json:"totp_token"`
	Code      string `json:"code"`
}

func (h *AuthHandler) VerifyTOTPLogin(c echo.Context) error {
	var input verifyTOTPLoginInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.TotpToken == "" || input.Code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Token and code are required"})
	}

	userID, err := pkg.ValidateTOTPPendingToken(input.TotpToken)
	if err != nil {
		clearRefreshTokenCookie(c)
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid or expired session"})
	}

	ctx := c.Request().Context()
	totpSvc := h.TOTPService.WithContext(ctx)
	if !totpSvc.VerifyLogin(userID, input.Code) && !totpSvc.VerifyBackupCode(userID, input.Code) {
		clearRefreshTokenCookie(c)
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid code"})
	}

	resp, err := h.Service.WithContext(ctx).CreateSession(userID)
	if err != nil {
		if errors.Is(err, serviceauth.ErrUserNotFound) {
			clearRefreshTokenCookie(c)
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid or expired session"})
		}
		if strings.Contains(strings.ToLower(err.Error()), "disabled") {
			clearRefreshTokenCookie(c)
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "account is disabled"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create session"})
	}

	return writeAuthSuccess(c, http.StatusOK, resp)
}
