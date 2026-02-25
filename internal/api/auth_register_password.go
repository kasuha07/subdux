package api

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

func (h *AuthHandler) Register(c echo.Context) error {
	var input service.RegisterInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)

	if input.Username == "" || input.Email == "" || input.Password == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Username, email and password are required"})
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid email"})
	}

	if len(input.Password) < 6 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Password must be at least 6 characters"})
	}
	if len([]byte(input.Password)) > 72 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Password must not exceed 72 bytes"})
	}

	resp, err := h.Service.Register(input)
	if err != nil {
		return writeAuthServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, mapAuthResponse(resp))
}

func (h *AuthHandler) GetRegistrationConfig(c echo.Context) error {
	config, err := h.Service.GetRegistrationConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to load registration config"})
	}
	return c.JSON(http.StatusOK, config)
}

func (h *AuthHandler) SendRegisterVerificationCode(c echo.Context) error {
	var input struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Email is required"})
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid email"})
	}

	if err := h.Service.SendRegistrationVerificationCode(email); err != nil {
		return writeAuthServiceError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "verification code sent"})
}

func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var input struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Email is required"})
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid email"})
	}

	if err := h.Service.RequestPasswordReset(email); err != nil {
		return writeAuthServiceError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "if the account exists, a verification code has been sent"})
}

func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var input struct {
		Email            string `json:"email"`
		VerificationCode string `json:"verification_code"`
		NewPassword      string `json:"new_password"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	input.Email = strings.TrimSpace(input.Email)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)

	if input.Email == "" || input.VerificationCode == "" || input.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Email, verification code and new password are required"})
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid email"})
	}
	if len(input.NewPassword) < 6 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "New password must be at least 6 characters"})
	}
	if len([]byte(input.NewPassword)) > 72 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "New password must not exceed 72 bytes"})
	}

	if err := h.Service.ResetPassword(input.Email, input.VerificationCode, input.NewPassword); err != nil {
		return writeAuthServiceError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "password reset successfully"})
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
	if len([]byte(input.NewPassword)) > 72 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "New password must not exceed 72 bytes"})
	}
	if err := h.Service.ChangePassword(userID, input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "Password changed successfully"})
}
