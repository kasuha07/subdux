package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/service"
)

type AuthHandler struct {
	Service     *service.AuthService
	TOTPService *service.TOTPService
}

func NewAuthHandler(s *service.AuthService, totpSvc *service.TOTPService) *AuthHandler {
	return &AuthHandler{Service: s, TOTPService: totpSvc}
}

type authUserResponse struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	TotpEnabled bool   `json:"totp_enabled"`
}

type authResponse struct {
	Token        string           `json:"token"`
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	User         authUserResponse `json:"user"`
}

type loginResponse struct {
	RequiresTotp bool              `json:"requires_totp"`
	TotpToken    string            `json:"totp_token,omitempty"`
	Token        string            `json:"token,omitempty"`
	AccessToken  string            `json:"access_token,omitempty"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	User         *authUserResponse `json:"user,omitempty"`
}

func mapAuthUserResponse(user model.User) authUserResponse {
	return authUserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Role:        user.Role,
		Status:      user.Status,
		TotpEnabled: user.TotpEnabled,
	}
}

func mapLoginResponse(resp *service.LoginResponse) loginResponse {
	var user *authUserResponse
	if resp.User != nil {
		mapped := mapAuthUserResponse(*resp.User)
		user = &mapped
	}

	return loginResponse{
		RequiresTotp: resp.RequiresTotp,
		TotpToken:    resp.TotpToken,
		Token:        resp.AccessToken,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		User:         user,
	}
}

func mapAuthResponse(resp *service.AuthResponse) authResponse {
	return authResponse{
		Token:        resp.AccessToken,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		User:         mapAuthUserResponse(resp.User),
	}
}

func authServiceErrorStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrRegistrationDisabled):
		return http.StatusForbidden
	case errors.Is(err, service.ErrEmailDomainNotAllowed):
		return http.StatusForbidden
	case errors.Is(err, service.ErrEmailAlreadyRegistered), errors.Is(err, service.ErrUsernameAlreadyTaken):
		return http.StatusConflict
	case errors.Is(err, service.ErrVerificationCodeTooFrequent):
		return http.StatusTooManyRequests
	case errors.Is(err, service.ErrUserNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrRegistrationEmailVerificationDisabled),
		errors.Is(err, service.ErrVerificationCodeRequired),
		errors.Is(err, service.ErrVerificationCodeInvalid),
		errors.Is(err, service.ErrVerificationCodeTooManyAttempts),
		errors.Is(err, service.ErrInvalidEmail),
		errors.Is(err, service.ErrCurrentPasswordIncorrect),
		errors.Is(err, service.ErrNewEmailSameAsCurrent),
		errors.Is(err, service.ErrPasswordTooLong),
		errors.Is(err, service.ErrSMTPUnavailable):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeAuthServiceError(c echo.Context, err error) error {
	status := authServiceErrorStatus(err)
	if status == http.StatusInternalServerError {
		return c.JSON(status, echo.Map{"error": "internal server error"})
	}
	return c.JSON(status, echo.Map{"error": err.Error()})
}
