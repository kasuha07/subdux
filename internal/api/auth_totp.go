package api

import (
	"errors"
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	"github.com/kasuha07/subdux/internal/pkg"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

func (h *AuthHandler) SetupTOTP(c echo.Context) error {
	userID := apimw.From(c).UserID
	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		userID,
		servicereauth.ReauthOperationEnableTOTP,
		apimw.ReauthTicketFromRequest(c),
	); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	result, err := h.TOTPService.WithContext(c.Request().Context()).BeginSetup(userID)
	if err != nil {
		return writeTOTPServiceError(c, err)
	}
	return c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) ConfirmTOTP(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input struct {
		SessionID string `json:"session_id"`
		Code      string `json:"code"`
	}
	if !httpx.BindJSON(c, &input, "Invalid request body") {
		return nil
	}
	if input.SessionID == "" || input.Code == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "session_id and code are required")
	}

	backupCodes, err := h.TOTPService.WithContext(c.Request().Context()).ConfirmSetup(userID, input.SessionID, input.Code)
	if err != nil {
		return writeTOTPServiceError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"backup_codes": backupCodes})
}

func (h *AuthHandler) DisableTOTP(c echo.Context) error {
	userID := apimw.From(c).UserID
	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		userID,
		servicereauth.ReauthOperationDisableTOTP,
		apimw.ReauthTicketFromRequest(c),
	); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	if err := h.TOTPService.WithContext(c.Request().Context()).Disable(userID); err != nil {
		return writeTOTPServiceError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "2FA disabled successfully"})
}

func writeTOTPServiceError(c echo.Context, err error) error {
	return err
}

type verifyTOTPLoginInput struct {
	TotpToken string `json:"totp_token"`
	Code      string `json:"code"`
}

func (h *AuthHandler) VerifyTOTPLogin(c echo.Context) error {
	var input verifyTOTPLoginInput
	if !httpx.BindJSON(c, &input, "Invalid request body") {
		return nil
	}
	if input.TotpToken == "" || input.Code == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "Token and code are required")
	}

	userID, err := pkg.ValidateTOTPPendingToken(input.TotpToken)
	if err != nil {
		apimw.ClearRefreshTokenCookie(c)
		return httpx.WriteError(c, http.StatusUnauthorized, "Invalid or expired session")
	}

	ctx := c.Request().Context()
	totpSvc := h.TOTPService.WithContext(ctx)
	if !totpSvc.VerifyLogin(userID, input.Code) && !totpSvc.VerifyBackupCode(userID, input.Code) {
		apimw.ClearRefreshTokenCookie(c)
		return httpx.WriteError(c, http.StatusUnauthorized, "Invalid code")
	}

	resp, err := h.Service.WithContext(ctx).CreateSession(userID)
	if err != nil {
		if errors.Is(err, serviceauth.ErrUserNotFound) {
			apimw.ClearRefreshTokenCookie(c)
			return httpx.WriteError(c, http.StatusUnauthorized, "Invalid or expired session")
		}
		if errors.Is(err, serviceauth.ErrAccountDisabled) {
			apimw.ClearRefreshTokenCookie(c)
			return httpx.WriteError(c, http.StatusUnauthorized, "account is disabled")
		}
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to create session")
	}

	return writeAuthSuccess(c, http.StatusOK, resp)
}
