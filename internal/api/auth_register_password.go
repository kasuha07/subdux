package api

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	"github.com/labstack/echo/v4"
)

func (h *AuthHandler) Register(c echo.Context) error {
	var input serviceauth.RegisterInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)

	if input.Username == "" || input.Email == "" || input.Password == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "username_email_and_password_are_required")
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid_email")
	}

	if len(input.Password) < 8 {
		return httpx.WriteError(c, http.StatusBadRequest, "password_must_be_at_least_8_characters")
	}
	if err := serviceauth.ValidateBcryptPasswordLength(input.Password); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "password_must_not_exceed_72_bytes")
	}

	resp, err := h.Service.WithContext(c.Request().Context()).Register(input)
	if err != nil {
		return err
	}

	return writeAuthSuccess(c, http.StatusCreated, resp)
}

func (h *AuthHandler) GetRegistrationConfig(c echo.Context) error {
	config, err := h.Service.WithContext(c.Request().Context()).GetRegistrationConfig()
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed_to_load_registration_config")
	}
	return c.JSON(http.StatusOK, config)
}

func (h *AuthHandler) SendRegisterVerificationCode(c echo.Context) error {
	var input struct {
		Email string `json:"email"`
	}
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "email_is_required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid_email")
	}

	if err := h.Service.WithContext(c.Request().Context()).SendRegistrationVerificationCode(email); err != nil {
		return err
	}

	return httpx.WriteMessage(c, http.StatusOK, "verification_code_sent")
}

func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var input struct {
		Email string `json:"email"`
	}
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "email_is_required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid_email")
	}

	if err := h.Service.WithContext(c.Request().Context()).RequestPasswordReset(email); err != nil {
		return err
	}

	return httpx.WriteMessage(c, http.StatusOK, "if_the_account_exists_a_verification_code_has_been_sent")
}

func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var input struct {
		Email            string `json:"email"`
		VerificationCode string `json:"verification_code"`
		NewPassword      string `json:"new_password"`
	}
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	input.Email = strings.TrimSpace(input.Email)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)

	if input.Email == "" || input.VerificationCode == "" || input.NewPassword == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "email_verification_code_and_new_password_are_required")
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid_email")
	}
	if len(input.NewPassword) < 8 {
		return httpx.WriteError(c, http.StatusBadRequest, "new_password_must_be_at_least_8_characters")
	}
	if err := serviceauth.ValidateBcryptPasswordLength(input.NewPassword); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "new_password_must_not_exceed_72_bytes")
	}

	if err := h.Service.WithContext(c.Request().Context()).ResetPassword(input.Email, input.VerificationCode, input.NewPassword); err != nil {
		return err
	}

	return httpx.WriteMessage(c, http.StatusOK, "password_reset_successfully")
}

func (h *AuthHandler) ChangePassword(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input serviceauth.ChangePasswordInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	if input.CurrentPassword == "" || input.NewPassword == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "current_and_new_passwords_are_required")
	}
	if len(input.NewPassword) < 8 {
		return httpx.WriteError(c, http.StatusBadRequest, "new_password_must_be_at_least_8_characters")
	}
	if err := serviceauth.ValidateBcryptPasswordLength(input.NewPassword); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "new_password_must_not_exceed_72_bytes")
	}
	if err := h.Service.WithContext(c.Request().Context()).ChangePassword(userID, input); err != nil {
		return err
	}
	return httpx.WriteMessage(c, http.StatusOK, "password_changed_successfully")
}
