package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

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
	Username    string `json:"username"`
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
		Token:        resp.Token,
		User:         user,
	}
}

func authServiceErrorStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrRegistrationDisabled):
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

func (h *AuthHandler) Register(c echo.Context) error {
	var input service.RegisterInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)

	if input.Username == "" || input.Email == "" || input.Password == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Username, email and password are required"})
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid email"})
	}

	if len(input.Password) < 6 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Password must be at least 6 characters"})
	}

	resp, err := h.Service.Register(input)
	if err != nil {
		return writeAuthServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, authResponse{
		Token: resp.Token,
		User:  mapAuthUserResponse(resp.User),
	})
}

func (h *AuthHandler) GetRegistrationConfig(c echo.Context) error {
	config, err := h.Service.GetRegistrationConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to load registration config"})
	}
	return c.JSON(http.StatusOK, config)
}

func (h *AuthHandler) SendRegisterVerificationCode(c echo.Context) error {
	var input struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Email is required"})
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid email"})
	}

	if err := h.Service.SendRegistrationVerificationCode(email); err != nil {
		return writeAuthServiceError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "verification code sent"})
}

func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var input struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Email is required"})
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid email"})
	}

	if err := h.Service.RequestPasswordReset(email); err != nil {
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
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	input.Email = strings.TrimSpace(input.Email)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)

	if input.Email == "" || input.VerificationCode == "" || input.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Email, verification code and new password are required"})
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid email"})
	}
	if len(input.NewPassword) < 6 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "New password must be at least 6 characters"})
	}

	if err := h.Service.ResetPassword(input.Email, input.VerificationCode, input.NewPassword); err != nil {
		return writeAuthServiceError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "password reset successfully"})
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

	return c.JSON(http.StatusOK, authResponse{
		Token: resp.Token,
		User:  mapAuthUserResponse(resp.User),
	})
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

type oidcSessionResponse struct {
	Purpose    string                      `json:"purpose"`
	Token      string                      `json:"token,omitempty"`
	User       *authUserResponse           `json:"user,omitempty"`
	Connected  bool                        `json:"connected,omitempty"`
	Connection *service.OIDCConnectionInfo `json:"connection,omitempty"`
	Error      string                      `json:"error,omitempty"`
}

func mapOIDCSessionResponse(result *service.OIDCSessionResult) oidcSessionResponse {
	var user *authUserResponse
	if result.User != nil {
		mapped := mapAuthUserResponse(*result.User)
		user = &mapped
	}

	return oidcSessionResponse{
		Purpose:    result.Purpose,
		Token:      result.Token,
		User:       user,
		Connected:  result.Connected,
		Connection: result.Connection,
		Error:      result.Error,
	}
}

func (h *AuthHandler) GetOIDCConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, h.Service.GetOIDCPublicConfig())
}

func (h *AuthHandler) BeginOIDCLogin(c echo.Context) error {
	result, err := h.Service.BeginOIDCLogin()
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) BeginOIDCConnect(c echo.Context) error {
	userID := getUserID(c)
	result, err := h.Service.BeginOIDCConnect(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) OIDCCallback(c echo.Context) error {
	callbackResult, err := h.Service.HandleOIDCCallback(
		c.QueryParam("state"),
		c.QueryParam("code"),
		c.QueryParam("error"),
		c.QueryParam("error_description"),
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to process oidc callback"})
	}

	redirectPath := "/login"
	if callbackResult.Purpose == "connect" {
		redirectPath = "/settings"
	}

	query := url.Values{}
	query.Set("oidc_action", callbackResult.Purpose)
	query.Set("oidc_session", callbackResult.SessionID)
	return c.Redirect(http.StatusFound, redirectPath+"?"+query.Encode())
}

func (h *AuthHandler) GetOIDCSession(c echo.Context) error {
	sessionID := c.Param("id")
	result, err := h.Service.ConsumeOIDCSessionResult(sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, mapOIDCSessionResponse(result))
}

func (h *AuthHandler) ListOIDCConnections(c echo.Context) error {
	userID := getUserID(c)
	connections, err := h.Service.ListOIDCConnections(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to list oidc connections"})
	}

	return c.JSON(http.StatusOK, connections)
}

func (h *AuthHandler) DeleteOIDCConnection(c echo.Context) error {
	userID := getUserID(c)
	connectionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid oidc connection id"})
	}

	if err := h.Service.DeleteOIDCConnection(userID, uint(connectionID)); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "oidc connection deleted"})
}
