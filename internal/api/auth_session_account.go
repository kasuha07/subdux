package api

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"

	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

func (h *AuthHandler) Me(c echo.Context) error {
	userID := getUserID(c)
	user, err := h.Service.WithContext(c.Request().Context()).GetUser(userID)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorFunc(http.StatusNotFound, func(error) bool { return true }),
		)
	}
	return c.JSON(http.StatusOK, mapAuthUserResponse(*user))
}

func (h *AuthHandler) SendEmailChangeVerificationCode(c echo.Context) error {
	userID := getUserID(c)
	var input struct {
		NewEmail string `json:"new_email"`
	}
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}

	input.NewEmail = strings.TrimSpace(input.NewEmail)
	if input.NewEmail == "" {
		return writeError(c, http.StatusBadRequest, "New email is required")
	}
	if _, err := mail.ParseAddress(input.NewEmail); err != nil {
		return writeError(c, http.StatusBadRequest, "Invalid email")
	}

	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		userID,
		servicereauth.ReauthOperationChangeEmail,
		c.Request().Header.Get(reauthTicketHeader),
	); err != nil {
		return writeReauthError(c, err)
	}

	if err := h.Service.WithContext(c.Request().Context()).SendEmailChangeVerificationCode(userID, input.NewEmail); err != nil {
		return writeAuthServiceError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "verification code sent"})
}

func (h *AuthHandler) ConfirmEmailChange(c echo.Context) error {
	userID := getUserID(c)
	var input struct {
		NewEmail         string `json:"new_email"`
		VerificationCode string `json:"verification_code"`
	}
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}

	input.NewEmail = strings.TrimSpace(input.NewEmail)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)
	if input.NewEmail == "" || input.VerificationCode == "" {
		return writeError(c, http.StatusBadRequest, "New email and verification code are required")
	}
	if _, err := mail.ParseAddress(input.NewEmail); err != nil {
		return writeError(c, http.StatusBadRequest, "Invalid email")
	}

	resp, err := h.Service.WithContext(c.Request().Context()).ConfirmEmailChange(userID, input.NewEmail, input.VerificationCode)
	if err != nil {
		return writeAuthServiceError(c, err)
	}

	return writeAuthSuccess(c, http.StatusOK, resp)
}

func (h *AuthHandler) Login(c echo.Context) error {
	var input serviceauth.LoginInput
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}

	if input.Identifier == "" || input.Password == "" {
		return writeError(c, http.StatusBadRequest, "Username/email and password are required")
	}

	resp, err := h.Service.WithContext(c.Request().Context()).Login(input)
	if err != nil {
		clearRefreshTokenCookie(c)
		return writeServiceError(c, err,
			serviceErrorFunc(http.StatusUnauthorized, func(error) bool { return true }),
		)
	}

	return writeLoginSuccess(c, http.StatusOK, resp)
}

func (h *AuthHandler) RefreshSession(c echo.Context) error {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !bindOptionalJSON(c, &input, "Invalid request body") {
		return nil
	}

	input.RefreshToken = strings.TrimSpace(input.RefreshToken)
	if input.RefreshToken == "" {
		input.RefreshToken = getCookieValue(c, refreshTokenCookieName)
	}
	if input.RefreshToken == "" {
		clearRefreshTokenCookie(c)
		return writeError(c, http.StatusBadRequest, "refresh token is required")
	}

	resp, err := h.Service.WithContext(c.Request().Context()).RefreshSession(input.RefreshToken)
	if err != nil {
		if errors.Is(err, serviceauth.ErrInvalidRefreshToken) {
			clearRefreshTokenCookie(c)
			return writeServiceError(c, err,
				serviceError(http.StatusUnauthorized, serviceauth.ErrInvalidRefreshToken),
			)
		}
		return writeError(c, http.StatusInternalServerError, "failed to refresh session")
	}

	return writeAuthSuccess(c, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c echo.Context) error {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !bindOptionalJSON(c, &input, "Invalid request body") {
		return nil
	}

	input.RefreshToken = strings.TrimSpace(input.RefreshToken)
	if input.RefreshToken == "" {
		input.RefreshToken = getCookieValue(c, refreshTokenCookieName)
	}

	if err := h.Service.WithContext(c.Request().Context()).Logout(input.RefreshToken); err != nil {
		return writeInternalServerError(c, err)
	}

	clearRefreshTokenCookie(c)
	return c.NoContent(http.StatusNoContent)
}

func (h *AuthHandler) LogoutAll(c echo.Context) error {
	if err := h.Service.WithContext(c.Request().Context()).LogoutAll(getUserID(c)); err != nil {
		return writeInternalServerError(c, err)
	}

	clearRefreshTokenCookie(c)
	return c.NoContent(http.StatusNoContent)
}
