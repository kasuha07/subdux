package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
)

func (h *AuthHandler) SetupTOTP(c echo.Context) error {
	userID := getUserID(c)
	result, err := h.TOTPService.GenerateSetup(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) ConfirmTOTP(c echo.Context) error {
	userID := getUserID(c)
	var input struct {
		Code string `json:"code"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.Code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Code is required"})
	}

	backupCodes, err := h.TOTPService.ConfirmSetup(userID, input.Code)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"backup_codes": backupCodes})
}

func (h *AuthHandler) DisableTOTP(c echo.Context) error {
	userID := getUserID(c)
	var input struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.Password == "" || input.Code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Password and code are required"})
	}

	if err := h.TOTPService.Disable(userID, input.Password, input.Code); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "2FA disabled successfully"})
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
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid or expired session"})
	}

	if !h.TOTPService.VerifyLogin(userID, input.Code) && !h.TOTPService.VerifyBackupCode(userID, input.Code) {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid code"})
	}

	resp, err := h.Service.CreateSession(userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid or expired session"})
		}
		if strings.Contains(strings.ToLower(err.Error()), "disabled") {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "account is disabled"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create session"})
	}

	return c.JSON(http.StatusOK, mapAuthResponse(resp))
}
