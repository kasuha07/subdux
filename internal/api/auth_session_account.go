package api

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

func (h *AuthHandler) Me(c echo.Context) error {
	userID := apimw.From(c).UserID
	user, err := h.Service.WithContext(c.Request().Context()).GetUser(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapAuthUserResponse(*user))
}

func (h *AuthHandler) SendEmailChangeVerificationCode(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input struct {
		NewEmail string `json:"new_email"`
	}
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	input.NewEmail = strings.TrimSpace(input.NewEmail)
	if input.NewEmail == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "new_email_is_required")
	}
	if _, err := mail.ParseAddress(input.NewEmail); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid_email")
	}

	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		userID,
		servicereauth.ReauthOperationChangeEmail,
		c.Request().Header.Get(apimw.ReauthTicketHeader),
	); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	if err := h.Service.WithContext(c.Request().Context()).SendEmailChangeVerificationCode(userID, input.NewEmail); err != nil {
		return err
	}

	return httpx.WriteMessage(c, http.StatusOK, "verification_code_sent")
}

func (h *AuthHandler) ConfirmEmailChange(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input struct {
		NewEmail         string `json:"new_email"`
		VerificationCode string `json:"verification_code"`
	}
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	input.NewEmail = strings.TrimSpace(input.NewEmail)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)
	if input.NewEmail == "" || input.VerificationCode == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "new_email_and_verification_code_are_required")
	}
	if _, err := mail.ParseAddress(input.NewEmail); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid_email")
	}

	resp, err := h.Service.WithContext(c.Request().Context()).ConfirmEmailChange(userID, input.NewEmail, input.VerificationCode)
	if err != nil {
		return err
	}

	return writeAuthSuccess(c, http.StatusOK, resp)
}

func (h *AuthHandler) Login(c echo.Context) error {
	var input serviceauth.LoginInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	if input.Identifier == "" || input.Password == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "username_or_email_and_password_are_required")
	}

	resp, err := h.Service.WithContext(c.Request().Context()).Login(input)
	if err != nil {
		apimw.ClearRefreshTokenCookie(c)
		// Login deliberately collapses every failure (bad credentials, disabled
		// account, lookup error) to a single 401 so the response never reveals
		// which account exists or why sign-in failed.
		return httpx.WriteErrorFrom(c, http.StatusUnauthorized, err)
	}

	return writeLoginSuccess(c, http.StatusOK, resp)
}

func (h *AuthHandler) RefreshSession(c echo.Context) error {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !httpx.BindOptionalJSON(c, &input, "invalid_request_body") {
		return nil
	}

	input.RefreshToken = strings.TrimSpace(input.RefreshToken)
	if input.RefreshToken == "" {
		input.RefreshToken = apimw.GetCookieValue(c, apimw.RefreshTokenCookieName)
	}
	if input.RefreshToken == "" {
		apimw.ClearRefreshTokenCookie(c)
		return httpx.WriteError(c, http.StatusBadRequest, "refresh_token_is_required")
	}

	resp, err := h.Service.WithContext(c.Request().Context()).RefreshSession(input.RefreshToken)
	if err != nil {
		if errors.Is(err, serviceauth.ErrInvalidRefreshToken) {
			apimw.ClearRefreshTokenCookie(c)
			return httpx.WriteErrorFrom(c, http.StatusUnauthorized, err)
		}
		return httpx.WriteError(c, http.StatusInternalServerError, "failed_to_refresh_session")
	}

	return writeAuthSuccess(c, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c echo.Context) error {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !httpx.BindOptionalJSON(c, &input, "invalid_request_body") {
		return nil
	}

	input.RefreshToken = strings.TrimSpace(input.RefreshToken)
	if input.RefreshToken == "" {
		input.RefreshToken = apimw.GetCookieValue(c, apimw.RefreshTokenCookieName)
	}

	if err := h.Service.WithContext(c.Request().Context()).Logout(input.RefreshToken); err != nil {
		return err
	}

	apimw.ClearRefreshTokenCookie(c)
	return c.NoContent(http.StatusNoContent)
}

func (h *AuthHandler) LogoutAll(c echo.Context) error {
	if err := h.Service.WithContext(c.Request().Context()).LogoutAll(apimw.From(c).UserID); err != nil {
		return err
	}

	apimw.ClearRefreshTokenCookie(c)
	return c.NoContent(http.StatusNoContent)
}
