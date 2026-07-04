package api

import (
	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/model"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	Service     *serviceauth.Service
	TOTPService *serviceauth.TOTPService
	Reauth      *servicereauth.Service
}

func NewAuthHandler(s *serviceauth.Service, totpSvc *serviceauth.TOTPService, reauth *servicereauth.Service) *AuthHandler {
	return &AuthHandler{Service: s, TOTPService: totpSvc, Reauth: reauth}
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
	Token       string           `json:"token"`
	AccessToken string           `json:"access_token"`
	User        authUserResponse `json:"user"`
}

type loginResponse struct {
	RequiresTotp bool              `json:"requires_totp"`
	TotpToken    string            `json:"totp_token,omitempty"`
	Token        string            `json:"token,omitempty"`
	AccessToken  string            `json:"access_token,omitempty"`
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

func mapLoginResponse(resp *serviceauth.LoginResponse) loginResponse {
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
		User:         user,
	}
}

func mapAuthResponse(resp *serviceauth.AuthResponse) authResponse {
	return authResponse{
		Token:       resp.AccessToken,
		AccessToken: resp.AccessToken,
		User:        mapAuthUserResponse(resp.User),
	}
}

func writeAuthSuccess(c echo.Context, status int, resp *serviceauth.AuthResponse) error {
	apimw.SetRefreshTokenCookie(c, resp.RefreshToken)
	return c.JSON(status, mapAuthResponse(resp))
}

func writeLoginSuccess(c echo.Context, status int, resp *serviceauth.LoginResponse) error {
	apimw.SetRefreshTokenCookie(c, resp.RefreshToken)
	return c.JSON(status, mapLoginResponse(resp))
}
