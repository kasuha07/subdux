package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
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
	Email       string `json:"email"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	TotpEnabled bool   `json:"totp_enabled"`
}

type authResponse struct {
	Token string           `json:"token"`
	User  authUserResponse `json:"user"`
}

type loginResponse struct {
	RequiresTotp bool              `json:"requires_totp"`
	TotpToken    string            `json:"totp_token,omitempty"`
	Token        string            `json:"token,omitempty"`
	User         *authUserResponse `json:"user,omitempty"`
}

func mapAuthUserResponse(user model.User) authUserResponse {
	return authUserResponse{
		ID:          user.ID,
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
		Token:        resp.Token,
		User:         user,
	}
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

	return c.JSON(http.StatusCreated, authResponse{
		Token: resp.Token,
		User:  mapAuthUserResponse(resp.User),
	})
}

func (h *AuthHandler) Me(c echo.Context) error {
	userID := getUserID(c)
	user, err := h.Service.GetUser(userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, mapAuthUserResponse(*user))
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

	return c.JSON(http.StatusOK, mapLoginResponse(resp))
}

func (h *AuthHandler) SetupTOTP(c echo.Context) error {
	userID := getUserID(c)
	result, err := h.TOTPService.GenerateSetup(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) ConfirmTOTP(c echo.Context) error {
	userID := getUserID(c)
	var input struct {
		Code string `json:"code"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.Code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Code is required"})
	}

	backupCodes, err := h.TOTPService.ConfirmSetup(userID, input.Code)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"backup_codes": backupCodes})
}

func (h *AuthHandler) DisableTOTP(c echo.Context) error {
	userID := getUserID(c)
	var input struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.Password == "" || input.Code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Password and code are required"})
	}

	if err := h.TOTPService.Disable(userID, input.Password, input.Code); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "2FA disabled successfully"})
}

type verifyTOTPLoginInput struct {
	TotpToken string `json:"totp_token"`
	Code      string `json:"code"`
}

func (h *AuthHandler) VerifyTOTPLogin(c echo.Context) error {
	var input verifyTOTPLoginInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.TotpToken == "" || input.Code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Token and code are required"})
	}

	userID, err := pkg.ValidateTOTPPendingToken(input.TotpToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid or expired session"})
	}

	if !h.TOTPService.VerifyLogin(userID, input.Code) && !h.TOTPService.VerifyBackupCode(userID, input.Code) {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid code"})
	}

	user, err := h.Service.GetUser(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "User not found"})
	}

	token, err := pkg.GenerateToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to generate token"})
	}

	return c.JSON(http.StatusOK, authResponse{
		Token: token,
		User:  mapAuthUserResponse(*user),
	})
}

type passkeyBeginRegistrationInput struct {
	Name string `json:"name"`
}

func (h *AuthHandler) ListPasskeys(c echo.Context) error {
	userID := getUserID(c)
	passkeys, err := h.Service.ListPasskeys(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to list passkeys"})
	}
	return c.JSON(http.StatusOK, passkeys)
}

func (h *AuthHandler) BeginPasskeyRegistration(c echo.Context) error {
	userID := getUserID(c)
	var input passkeyBeginRegistrationInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	result, err := h.Service.BeginPasskeyRegistration(userID, input.Name, c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme())
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

type passkeyFinishRegistrationInput struct {
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

func (h *AuthHandler) FinishPasskeyRegistration(c echo.Context) error {
	userID := getUserID(c)
	var input passkeyFinishRegistrationInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.SessionID == "" || len(input.Credential) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "session_id and credential are required"})
	}

	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(input.Credential))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid credential payload"})
	}

	passkey, err := h.Service.FinishPasskeyRegistration(userID, input.SessionID, parsedResponse, c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme())
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, passkey)
}

func (h *AuthHandler) DeletePasskey(c echo.Context) error {
	userID := getUserID(c)
	passkeyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid passkey id"})
	}

	if err := h.Service.DeletePasskey(userID, uint(passkeyID)); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "passkey deleted"})
}

func (h *AuthHandler) BeginPasskeyLogin(c echo.Context) error {
	result, err := h.Service.BeginPasskeyLogin(c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme())
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

type passkeyFinishLoginInput struct {
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

func (h *AuthHandler) FinishPasskeyLogin(c echo.Context) error {
	var input passkeyFinishLoginInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.SessionID == "" || len(input.Credential) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "session_id and credential are required"})
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(input.Credential))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid credential payload"})
	}

	resp, err := h.Service.FinishPasskeyLogin(input.SessionID, parsedResponse, c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme())
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, authResponse{
		Token: resp.Token,
		User:  mapAuthUserResponse(resp.User),
	})
}
