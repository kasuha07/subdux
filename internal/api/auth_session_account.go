package api

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

func (h *AuthHandler) Me(c echo.Context) error {
	userID := getUserID(c)
	user, err := h.Service.GetUser(userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, mapAuthUserResponse(*user))
}

func (h *AuthHandler) SendEmailChangeVerificationCode(c echo.Context) error {
	userID := getUserID(c)
	var input struct {
		NewEmail string `json:"new_email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	input.NewEmail = strings.TrimSpace(input.NewEmail)
	if input.NewEmail == "" || input.Password == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "New email and password are required"})
	}
	if _, err := mail.ParseAddress(input.NewEmail); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid email"})
	}

	if err := h.Service.SendEmailChangeVerificationCode(userID, input.NewEmail, input.Password); err != nil {
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
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	input.NewEmail = strings.TrimSpace(input.NewEmail)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)
	if input.NewEmail == "" || input.VerificationCode == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "New email and verification code are required"})
	}
	if _, err := mail.ParseAddress(input.NewEmail); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid email"})
	}

	resp, err := h.Service.ConfirmEmailChange(userID, input.NewEmail, input.VerificationCode)
	if err != nil {
		return writeAuthServiceError(c, err)
	}

	return c.JSON(http.StatusOK, mapAuthResponse(resp))
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

	return c.JSON(http.StatusOK, mapLoginResponse(resp))
}

func (h *AuthHandler) RefreshSession(c echo.Context) error {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	input.RefreshToken = strings.TrimSpace(input.RefreshToken)
	if input.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "refresh token is required"})
	}

	resp, err := h.Service.RefreshSession(input.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid refresh token"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to refresh session"})
	}

	return c.JSON(http.StatusOK, mapAuthResponse(resp))
}
