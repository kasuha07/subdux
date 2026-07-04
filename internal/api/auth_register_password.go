package api

import (
	"net/http"
	"net/mail"
	"strings"

	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	"github.com/labstack/echo/v4"
)

func (h *AuthHandler) Register(c echo.Context) error {
	var input serviceauth.RegisterInput
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)

	if input.Username == "" || input.Email == "" || input.Password == "" {
		return writeError(c, http.StatusBadRequest, "Username, email and password are required")
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return writeError(c, http.StatusBadRequest, "Invalid email")
	}

	if len(input.Password) < 8 {
		return writeError(c, http.StatusBadRequest, "Password must be at least 8 characters")
	}
	if err := serviceauth.ValidateBcryptPasswordLength(input.Password); err != nil {
		return writeError(c, http.StatusBadRequest, "Password must not exceed 72 bytes")
	}

	resp, err := h.Service.WithContext(c.Request().Context()).Register(input)
	if err != nil {
		return writeAuthServiceError(c, err)
	}

	return writeAuthSuccess(c, http.StatusCreated, resp)
}

func (h *AuthHandler) GetRegistrationConfig(c echo.Context) error {
	config, err := h.Service.WithContext(c.Request().Context()).GetRegistrationConfig()
	if err != nil {
		return writeError(c, http.StatusInternalServerError, "failed to load registration config")
	}
	return c.JSON(http.StatusOK, config)
}

func (h *AuthHandler) SendRegisterVerificationCode(c echo.Context) error {
	var input struct {
		Email string `json:"email"`
	}
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		return writeError(c, http.StatusBadRequest, "Email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return writeError(c, http.StatusBadRequest, "Invalid email")
	}

	if err := h.Service.WithContext(c.Request().Context()).SendRegistrationVerificationCode(email); err != nil {
		return writeAuthServiceError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "verification code sent"})
}

func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var input struct {
		Email string `json:"email"`
	}
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		return writeError(c, http.StatusBadRequest, "Email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return writeError(c, http.StatusBadRequest, "Invalid email")
	}

	if err := h.Service.WithContext(c.Request().Context()).RequestPasswordReset(email); err != nil {
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
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}

	input.Email = strings.TrimSpace(input.Email)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)

	if input.Email == "" || input.VerificationCode == "" || input.NewPassword == "" {
		return writeError(c, http.StatusBadRequest, "Email, verification code and new password are required")
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return writeError(c, http.StatusBadRequest, "Invalid email")
	}
	if len(input.NewPassword) < 8 {
		return writeError(c, http.StatusBadRequest, "New password must be at least 8 characters")
	}
	if err := serviceauth.ValidateBcryptPasswordLength(input.NewPassword); err != nil {
		return writeError(c, http.StatusBadRequest, "New password must not exceed 72 bytes")
	}

	if err := h.Service.WithContext(c.Request().Context()).ResetPassword(input.Email, input.VerificationCode, input.NewPassword); err != nil {
		return writeAuthServiceError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "password reset successfully"})
}

func (h *AuthHandler) ChangePassword(c echo.Context) error {
	userID := getUserID(c)
	var input serviceauth.ChangePasswordInput
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}
	if input.CurrentPassword == "" || input.NewPassword == "" {
		return writeError(c, http.StatusBadRequest, "Current and new passwords are required")
	}
	if len(input.NewPassword) < 8 {
		return writeError(c, http.StatusBadRequest, "New password must be at least 8 characters")
	}
	if err := serviceauth.ValidateBcryptPasswordLength(input.NewPassword); err != nil {
		return writeError(c, http.StatusBadRequest, "New password must not exceed 72 bytes")
	}
	if err := h.Service.WithContext(c.Request().Context()).ChangePassword(userID, input); err != nil {
		return writeServiceError(c, err,
			serviceErrorFunc(http.StatusBadRequest, func(error) bool { return true }),
		)
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "Password changed successfully"})
}
