package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
)

type AuthHandler struct {
	Service     *service.AuthService
	TOTPService *service.TOTPService
}

func NewAuthHandler(s *service.AuthService, totpSvc *service.TOTPService) *AuthHandler {
	return &AuthHandler{Service: s, TOTPService: totpSvc}
}

func (h *AuthHandler) Register(c echo.Context) error {
	var input service.RegisterInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	if input.Username == "" || input.Email == "" || input.Password == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Username, email and password are required"})
	}

	if len(input.Password) < 6 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Password must be at least 6 characters"})
	}

	resp, err := h.Service.Register(input)
	if err != nil {
		return c.JSON(http.StatusConflict, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, resp)
}

func (h *AuthHandler) Me(c echo.Context) error {
	userID := getUserID(c)
	user, err := h.Service.GetUser(userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) ChangePassword(c echo.Context) error {
	userID := getUserID(c)
	var input service.ChangePasswordInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.CurrentPassword == "" || input.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Current and new passwords are required"})
	}
	if len(input.NewPassword) < 6 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "New password must be at least 6 characters"})
	}
	if err := h.Service.ChangePassword(userID, input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "Password changed successfully"})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var input service.LoginInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	if input.Identifier == "" || input.Password == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Username/email and password are required"})
	}

	resp, err := h.Service.Login(input)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, resp)
}

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

	user, err := h.Service.GetUser(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "User not found"})
	}

	token, err := pkg.GenerateToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to generate token"})
	}

	return c.JSON(http.StatusOK, service.AuthResponse{Token: token, User: *user})
}
