package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type AuthHandler struct {
	Service *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{Service: s}
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
