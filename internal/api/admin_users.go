package api

import (
	"net/http"
	"time"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	adminservice "github.com/kasuha07/subdux/internal/service/admin"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

type adminUserResponse struct {
	ID                uint      `json:"id"`
	Username          string    `json:"username"`
	Email             string    `json:"email"`
	Role              string    `json:"role"`
	Status            string    `json:"status"`
	TotpEnabled       bool      `json:"totp_enabled"`
	PasskeyCount      int64     `json:"passkey_count"`
	CreatedAt         time.Time `json:"created_at"`
	SubscriptionCount int64     `json:"subscription_count"`
}

func mapAdminUserResponse(user adminservice.AdminUserListItem) adminUserResponse {
	return adminUserResponse{
		ID:                user.ID,
		Username:          user.Username,
		Email:             user.Email,
		Role:              user.Role,
		Status:            user.Status,
		TotpEnabled:       user.TotpEnabled,
		PasskeyCount:      user.PasskeyCount,
		CreatedAt:         user.CreatedAt,
		SubscriptionCount: user.SubscriptionCount,
	}
}

func mapAdminUserResponses(users []adminservice.AdminUserListItem) []adminUserResponse {
	responses := make([]adminUserResponse, len(users))
	for i, user := range users {
		responses[i] = mapAdminUserResponse(user)
	}
	return responses
}

func (h *AdminHandler) ListUsers(c echo.Context) error {
	users, err := h.Service.WithContext(c.Request().Context()).ListUsers()
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to list users")
	}
	return c.JSON(http.StatusOK, mapAdminUserResponses(users))
}

func (h *AdminHandler) CreateUser(c echo.Context) error {
	var input adminservice.CreateUserInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}

	if input.Username == "" || input.Email == "" || input.Password == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "username, email and password are required")
	}

	if len(input.Password) < 8 {
		return httpx.WriteError(c, http.StatusBadRequest, "password must be at least 8 characters")
	}
	if err := serviceauth.ValidateBcryptPasswordLength(input.Password); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "password must not exceed 72 bytes")
	}

	if operation, ok := servicereauth.OperationForCreateUserRole(input.Role); ok {
		if err := h.Reauth.WithContext(c.Request().Context()).Consume(
			apimw.From(c).UserID,
			operation,
			apimw.ReauthTicketFromRequest(c),
		); err != nil {
			return apimw.WriteReauthError(c, err)
		}
	}

	user, err := h.Service.WithContext(c.Request().Context()).CreateUser(input)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, mapAdminUserResponse(adminservice.AdminUserListItem{User: *user}))
}

func (h *AdminHandler) ChangeUserRole(c echo.Context) error {
	userID, ok := httpx.ParseUintParam(c, "id", "invalid user id")
	if !ok {
		return nil
	}

	var input adminservice.ChangeRoleInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}

	if input.Role != "admin" && input.Role != "user" {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid role")
	}

	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		apimw.From(c).UserID,
		servicereauth.ReauthOperationChangeUserRole,
		apimw.ReauthTicketFromRequest(c),
	); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	if err := h.Service.WithContext(c.Request().Context()).ChangeUserRole(uint(userID), input.Role); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "role updated"})
}

func (h *AdminHandler) ChangeUserStatus(c echo.Context) error {
	userID, ok := httpx.ParseUintParam(c, "id", "invalid user id")
	if !ok {
		return nil
	}

	var input adminservice.ChangeStatusInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).ChangeUserStatus(uint(userID), input.Status); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "status updated"})
}

func (h *AdminHandler) DisableUserTOTP(c echo.Context) error {
	userID, ok := httpx.ParseUintParam(c, "id", "invalid user id")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).DisableUserTOTP(uint(userID)); err != nil {
		return writeAdminCredentialResetError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "two-factor authentication disabled"})
}

func (h *AdminHandler) DisableUserPasskeys(c echo.Context) error {
	userID, ok := httpx.ParseUintParam(c, "id", "invalid user id")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).DisableUserPasskeys(uint(userID)); err != nil {
		return writeAdminCredentialResetError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "passkeys disabled"})
}

func writeAdminCredentialResetError(c echo.Context, err error) error {
	return err
}

func (h *AdminHandler) DeleteUser(c echo.Context) error {
	userID, ok := httpx.ParseUintParam(c, "id", "invalid user id")
	if !ok {
		return nil
	}

	currentUserID := apimw.From(c).UserID
	if currentUserID == uint(userID) {
		return httpx.WriteError(c, http.StatusBadRequest, "cannot delete yourself")
	}

	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		currentUserID,
		servicereauth.ReauthOperationDeleteUser,
		apimw.ReauthTicketFromRequest(c),
	); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	if err := h.Service.WithContext(c.Request().Context()).DeleteUser(uint(userID)); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "user deleted"})
}
