package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const (
	oidcPurposeLogin   = "login"
	oidcPurposeConnect = "connect"
	oidcProviderKey    = "oidc"

	defaultOIDCScopes    = "openid profile email"
	oidcStateSessionTTL  = 10 * time.Minute
	oidcResultSessionTTL = 3 * time.Minute
)

type OIDCPublicConfig struct {
	Enabled        bool   `json:"enabled"`
	ProviderName   string `json:"provider_name"`
	AutoCreateUser bool   `json:"auto_create_user"`
}

type OIDCStartResult struct {
	AuthorizationURL string `json:"authorization_url"`
}

type OIDCConnectionInfo struct {
	ID        uint      `json:"id"`
	Provider  string    `json:"provider"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OIDCCallbackResult struct {
	Purpose   string `json:"purpose"`
	SessionID string `json:"session_id"`
}

type OIDCSessionResult struct {
	Purpose    string              `json:"purpose"`
	Token      string              `json:"token,omitempty"`
	User       *model.User         `json:"user,omitempty"`
	Connected  bool                `json:"connected,omitempty"`
	Connection *OIDCConnectionInfo `json:"connection,omitempty"`
	Error      string              `json:"error,omitempty"`
}

type oidcStateSession struct {
	Purpose      string
	UserID       uint
	CodeVerifier string
	Nonce        string
	ExpiresAt    time.Time
}

type oidcResultSession struct {
	Result    OIDCSessionResult
	ExpiresAt time.Time
}

type oidcSettings struct {
	Enabled        bool
	ProviderName   string
	IssuerURL      string
	ClientID       string
	ClientSecret   string
	RedirectURL    string
	Scopes         []string
	AutoCreateUser bool
	AuthURL        string
	TokenURL       string
	UserinfoURL    string
	Audience       string
	Resource       string
	ExtraAuth      map[string]string
}

func (s oidcSettings) isConfigured() bool {
	return s.IssuerURL != "" && s.ClientID != "" && s.ClientSecret != "" && s.RedirectURL != ""
}

type oidcIdentityClaims struct {
	Subject           string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	Nonce             string `json:"nonce"`
}

func (s *AuthService) GetOIDCPublicConfig() *OIDCPublicConfig {
	settings := s.getOIDCSettings()

	return &OIDCPublicConfig{
		Enabled:        settings.Enabled && settings.isConfigured(),
		ProviderName:   settings.ProviderName,
		AutoCreateUser: settings.AutoCreateUser,
	}
}

func (s *AuthService) BeginOIDCLogin() (*OIDCStartResult, error) {
	settings := s.getOIDCSettings()
	if !settings.Enabled || !settings.isConfigured() {
		return nil, errors.New("oidc login is not available")
	}

	authorizationURL, err := s.buildOIDCAuthorizationURL(settings, oidcPurposeLogin, 0)
	if err != nil {
		return nil, err
	}

	return &OIDCStartResult{AuthorizationURL: authorizationURL}, nil
}

func (s *AuthService) BeginOIDCConnect(userID uint) (*OIDCStartResult, error) {
	settings := s.getOIDCSettings()
	if !settings.Enabled || !settings.isConfigured() {
		return nil, errors.New("oidc login is not available")
	}

	authorizationURL, err := s.buildOIDCAuthorizationURL(settings, oidcPurposeConnect, userID)
	if err != nil {
		return nil, err
	}

	return &OIDCStartResult{AuthorizationURL: authorizationURL}, nil
}

func (s *AuthService) HandleOIDCCallback(state string, code string, providerError string, providerErrorDescription string) (*OIDCCallbackResult, error) {
	purpose := oidcPurposeLogin
	trimmedState := strings.TrimSpace(state)
	if trimmedState == "" {
		return s.createOIDCCallbackErrorResult(purpose, "missing OIDC state")
	}

	session, err := s.takeOIDCStateSession(trimmedState)
	if err != nil {
		return s.createOIDCCallbackErrorResult(purpose, err.Error())
	}

	purpose = session.Purpose
	if providerError != "" {
		message := strings.TrimSpace(providerErrorDescription)
		if message == "" {
			message = providerError
		}
		return s.createOIDCCallbackErrorResult(purpose, fmt.Sprintf("oidc authorization failed: %s", message))
	}

	if strings.TrimSpace(code) == "" {
		return s.createOIDCCallbackErrorResult(purpose, "missing OIDC authorization code")
	}

	settings := s.getOIDCSettings()
	if !settings.Enabled || !settings.isConfigured() {
		return s.createOIDCCallbackErrorResult(purpose, "oidc login is not available")
	}

	claims, err := s.resolveOIDCIdentity(settings, code, session.CodeVerifier, session.Nonce)
	if err != nil {
		return s.createOIDCCallbackErrorResult(purpose, err.Error())
	}

	var result OIDCSessionResult
	if purpose == oidcPurposeConnect {
		result, err = s.finishOIDCConnect(session.UserID, claims)
	} else {
		result, err = s.finishOIDCLogin(settings, claims)
	}
	if err != nil {
		return s.createOIDCCallbackErrorResult(purpose, err.Error())
	}

	sessionID := s.storeOIDCResultSession(result)
	return &OIDCCallbackResult{Purpose: purpose, SessionID: sessionID}, nil
}

func (s *AuthService) ConsumeOIDCSessionResult(sessionID string) (*OIDCSessionResult, error) {
	result, err := s.takeOIDCResultSession(strings.TrimSpace(sessionID))
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *AuthService) ListOIDCConnections(userID uint) ([]OIDCConnectionInfo, error) {
	var records []model.OIDCConnection
	if err := s.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	result := make([]OIDCConnectionInfo, 0, len(records))
	for _, record := range records {
		result = append(result, mapOIDCConnectionInfo(record))
	}
	return result, nil
}

func (s *AuthService) DeleteOIDCConnection(userID uint, connectionID uint) error {
	result := s.DB.Where("id = ? AND user_id = ?", connectionID, userID).Delete(&model.OIDCConnection{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("oidc connection not found")
	}
	return nil
}

func (s *AuthService) buildOIDCAuthorizationURL(settings oidcSettings, purpose string, userID uint) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, settings.IssuerURL)
	if err != nil {
		return "", errors.New("failed to initialize oidc provider")
	}

	endpoint := provider.Endpoint()
	if strings.TrimSpace(settings.AuthURL) != "" {
		endpoint.AuthURL = strings.TrimSpace(settings.AuthURL)
	}
	if strings.TrimSpace(settings.TokenURL) != "" {
		endpoint.TokenURL = strings.TrimSpace(settings.TokenURL)
	}

	oauthConfig := oauth2.Config{
		ClientID:     settings.ClientID,
		ClientSecret: settings.ClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  settings.RedirectURL,
		Scopes:       settings.Scopes,
	}

	state := uuid.NewString()
	nonce, err := generateSecureToken(24)
	if err != nil {
		return "", errors.New("failed to create oidc session")
	}

	codeVerifier := oauth2.GenerateVerifier()
	s.storeOIDCStateSession(state, oidcStateSession{
		Purpose:      purpose,
		UserID:       userID,
		CodeVerifier: codeVerifier,
		Nonce:        nonce,
		ExpiresAt:    time.Now().Add(oidcStateSessionTTL),
	})

	authOptions := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOnline,
		oauth2.S256ChallengeOption(codeVerifier),
		oauth2.SetAuthURLParam("nonce", nonce),
	}
	if settings.Audience != "" {
		authOptions = append(authOptions, oauth2.SetAuthURLParam("audience", settings.Audience))
	}
	if settings.Resource != "" {
		authOptions = append(authOptions, oauth2.SetAuthURLParam("resource", settings.Resource))
	}
	for key, value := range settings.ExtraAuth {
		authOptions = append(authOptions, oauth2.SetAuthURLParam(key, value))
	}

	authorizationURL := oauthConfig.AuthCodeURL(
		state,
		authOptions...,
	)

	return authorizationURL, nil
}

func (s *AuthService) resolveOIDCIdentity(settings oidcSettings, code string, codeVerifier string, expectedNonce string) (*oidcIdentityClaims, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, settings.IssuerURL)
	if err != nil {
		return nil, errors.New("failed to initialize oidc provider")
	}

	endpoint := provider.Endpoint()
	if strings.TrimSpace(settings.AuthURL) != "" {
		endpoint.AuthURL = strings.TrimSpace(settings.AuthURL)
	}
	if strings.TrimSpace(settings.TokenURL) != "" {
		endpoint.TokenURL = strings.TrimSpace(settings.TokenURL)
	}

	oauthConfig := oauth2.Config{
		ClientID:     settings.ClientID,
		ClientSecret: settings.ClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  settings.RedirectURL,
		Scopes:       settings.Scopes,
	}

	tokenOptions := []oauth2.AuthCodeOption{oauth2.VerifierOption(codeVerifier)}
	if settings.Audience != "" {
		tokenOptions = append(tokenOptions, oauth2.SetAuthURLParam("audience", settings.Audience))
	}
	if settings.Resource != "" {
		tokenOptions = append(tokenOptions, oauth2.SetAuthURLParam("resource", settings.Resource))
	}

	oauthToken, err := oauthConfig.Exchange(ctx, code, tokenOptions...)
	if err != nil {
		return nil, errors.New("failed to exchange oidc authorization code")
	}

	rawIDToken, ok := oauthToken.Extra("id_token").(string)
	if !ok || strings.TrimSpace(rawIDToken) == "" {
		return nil, errors.New("oidc provider did not return id_token")
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: settings.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, errors.New("failed to verify oidc identity")
	}

	var claims oidcIdentityClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, errors.New("failed to parse oidc identity")
	}

	if expectedNonce != "" {
		if claims.Nonce == "" {
			return nil, errors.New("oidc nonce is missing")
		}
		if claims.Nonce != expectedNonce {
			return nil, errors.New("oidc nonce does not match")
		}
	}

	claims.Subject = strings.TrimSpace(claims.Subject)
	if claims.Subject == "" {
		return nil, errors.New("oidc subject is missing")
	}

	needsUserInfo := strings.TrimSpace(claims.Email) == "" ||
		strings.TrimSpace(claims.PreferredUsername) == "" ||
		strings.TrimSpace(claims.Name) == ""
	if needsUserInfo {
		userInfoClaims, userInfoErr := fetchOIDCUserInfoClaims(ctx, provider, oauthToken, settings.UserinfoURL)
		if userInfoErr == nil && userInfoClaims != nil {
			if userInfoClaims.Subject != "" && userInfoClaims.Subject != claims.Subject {
				return nil, errors.New("oidc subject mismatch")
			}

			if strings.TrimSpace(claims.Email) == "" {
				claims.Email = strings.TrimSpace(userInfoClaims.Email)
			}
			if strings.TrimSpace(claims.PreferredUsername) == "" {
				claims.PreferredUsername = strings.TrimSpace(userInfoClaims.PreferredUsername)
			}
			if strings.TrimSpace(claims.Name) == "" {
				claims.Name = strings.TrimSpace(userInfoClaims.Name)
			}
		}
	}

	return &claims, nil
}

func (s *AuthService) finishOIDCLogin(settings oidcSettings, claims *oidcIdentityClaims) (OIDCSessionResult, error) {
	var connection model.OIDCConnection
	err := s.DB.Where("provider = ? AND subject = ?", oidcProviderKey, claims.Subject).First(&connection).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return OIDCSessionResult{}, errors.New("failed to load oidc connection")
	}

	var user *model.User
	if err == nil {
		user, err = s.GetUser(connection.UserID)
		if err != nil {
			return OIDCSessionResult{}, errors.New("linked user not found")
		}

		if user.Status == "disabled" {
			return OIDCSessionResult{}, errors.New("account is disabled")
		}

		email := strings.TrimSpace(claims.Email)
		if email != "" && email != connection.Email {
			_ = s.DB.Model(&connection).Update("email", email).Error
		}
	} else {
		if !settings.AutoCreateUser {
			return OIDCSessionResult{}, errors.New("oidc account is not linked")
		}

		user, err = s.createOIDCUser(claims)
		if err != nil {
			return OIDCSessionResult{}, err
		}
	}

	token, err := pkg.GenerateToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		return OIDCSessionResult{}, err
	}

	return OIDCSessionResult{
		Purpose: oidcPurposeLogin,
		Token:   token,
		User:    user,
	}, nil
}

func (s *AuthService) finishOIDCConnect(userID uint, claims *oidcIdentityClaims) (OIDCSessionResult, error) {
	if userID == 0 {
		return OIDCSessionResult{}, errors.New("invalid oidc connect session")
	}

	user, err := s.GetUser(userID)
	if err != nil {
		return OIDCSessionResult{}, err
	}
	if user.Status == "disabled" {
		return OIDCSessionResult{}, errors.New("account is disabled")
	}

	email := strings.TrimSpace(claims.Email)
	var connection model.OIDCConnection
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var existingBySubject model.OIDCConnection
		if err := tx.Where("provider = ? AND subject = ?", oidcProviderKey, claims.Subject).First(&existingBySubject).Error; err == nil {
			if existingBySubject.UserID != userID {
				return errors.New("this oidc account is already linked to another user")
			}

			if email != "" && email != existingBySubject.Email {
				if err := tx.Model(&existingBySubject).Update("email", email).Error; err != nil {
					return err
				}
				existingBySubject.Email = email
			}
			connection = existingBySubject
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var existingForUser model.OIDCConnection
		if err := tx.Where("provider = ? AND user_id = ?", oidcProviderKey, userID).First(&existingForUser).Error; err == nil {
			return errors.New("you have already connected another oidc account")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		connection = model.OIDCConnection{
			UserID:   userID,
			Provider: oidcProviderKey,
			Subject:  claims.Subject,
			Email:    email,
		}
		if err := tx.Create(&connection).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return OIDCSessionResult{}, err
	}

	mapped := mapOIDCConnectionInfo(connection)
	return OIDCSessionResult{
		Purpose:    oidcPurposeConnect,
		Connected:  true,
		Connection: &mapped,
	}, nil
}

func (s *AuthService) createOIDCUser(claims *oidcIdentityClaims) (*model.User, error) {
	email := strings.TrimSpace(claims.Email)
	if email == "" {
		return nil, errors.New("oidc provider did not return an email")
	}

	var existing model.User
	if err := s.DB.Where("email = ?", email).First(&existing).Error; err == nil {
		return nil, errors.New("email already registered, connect oidc from account settings")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	usernameSeed := claims.PreferredUsername
	if strings.TrimSpace(usernameSeed) == "" {
		localPart := email
		if at := strings.Index(localPart, "@"); at > 0 {
			localPart = localPart[:at]
		}
		usernameSeed = localPart
	}
	if strings.TrimSpace(usernameSeed) == "" {
		usernameSeed = claims.Name
	}
	if strings.TrimSpace(usernameSeed) == "" {
		usernameSeed = "user"
	}

	username, err := s.allocateOIDCUsername(usernameSeed)
	if err != nil {
		return nil, err
	}

	randomPassword, err := generateSecureToken(24)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := model.User{
		Username: username,
		Email:    email,
		Password: string(hash),
		Role:     "user",
		Status:   "active",
	}

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var existingByEmail model.User
		if err := tx.Where("email = ?", email).First(&existingByEmail).Error; err == nil {
			return errors.New("email already registered, connect oidc from account settings")
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		if err := SeedUserDefaults(tx, user.ID); err != nil {
			return err
		}

		connection := model.OIDCConnection{
			UserID:   user.ID,
			Provider: oidcProviderKey,
			Subject:  claims.Subject,
			Email:    email,
		}
		if err := tx.Create(&connection).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *AuthService) allocateOIDCUsername(seed string) (string, error) {
	base := normalizeOIDCUsername(seed)
	if base == "" {
		base = "user"
	}

	for i := 0; i < 1000; i++ {
		candidate := base
		if i > 0 {
			suffix := strconv.Itoa(i + 1)
			maxBaseLen := 255 - len(suffix)
			if len(candidate) > maxBaseLen {
				candidate = candidate[:maxBaseLen]
			}
			candidate += suffix
		}

		var count int64
		if err := s.DB.Model(&model.User{}).Where("username = ?", candidate).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}

	return "", errors.New("failed to allocate username")
}

func (s *AuthService) createOIDCCallbackErrorResult(purpose string, message string) (*OIDCCallbackResult, error) {
	if purpose != oidcPurposeConnect {
		purpose = oidcPurposeLogin
	}

	sessionID := s.storeOIDCResultSession(OIDCSessionResult{
		Purpose: purpose,
		Error:   message,
	})

	return &OIDCCallbackResult{Purpose: purpose, SessionID: sessionID}, nil
}

func (s *AuthService) getOIDCSettings() oidcSettings {
	scopes := parseOIDCScopes(s.getSetting("oidc_scopes"))
	if len(scopes) == 0 {
		scopes = parseOIDCScopes(defaultOIDCScopes)
	}

	providerName := strings.TrimSpace(s.getSetting("oidc_provider_name"))
	if providerName == "" {
		providerName = "OIDC"
	}

	return oidcSettings{
		Enabled:        parseBoolSetting(s.getSetting("oidc_enabled")),
		ProviderName:   providerName,
		IssuerURL:      strings.TrimSpace(s.getSetting("oidc_issuer_url")),
		ClientID:       strings.TrimSpace(s.getSetting("oidc_client_id")),
		ClientSecret:   strings.TrimSpace(s.getSetting("oidc_client_secret")),
		RedirectURL:    strings.TrimSpace(s.getSetting("oidc_redirect_url")),
		Scopes:         scopes,
		AutoCreateUser: parseBoolSetting(s.getSetting("oidc_auto_create_user")),
		AuthURL:        strings.TrimSpace(s.getSetting("oidc_authorization_endpoint")),
		TokenURL:       strings.TrimSpace(s.getSetting("oidc_token_endpoint")),
		UserinfoURL:    strings.TrimSpace(s.getSetting("oidc_userinfo_endpoint")),
		Audience:       strings.TrimSpace(s.getSetting("oidc_audience")),
		Resource:       strings.TrimSpace(s.getSetting("oidc_resource")),
		ExtraAuth:      parseOIDCExtraAuthParams(s.getSetting("oidc_extra_auth_params")),
	}
}

func (s *AuthService) storeOIDCStateSession(state string, session oidcStateSession) {
	s.oidcMu.Lock()
	defer s.oidcMu.Unlock()

	s.cleanupOIDCSessionsLocked()
	s.oidcStateSessions[state] = session
}

func (s *AuthService) takeOIDCStateSession(state string) (oidcStateSession, error) {
	s.oidcMu.Lock()
	defer s.oidcMu.Unlock()

	s.cleanupOIDCSessionsLocked()

	session, exists := s.oidcStateSessions[state]
	if !exists {
		return oidcStateSession{}, errors.New("invalid or expired oidc session")
	}
	delete(s.oidcStateSessions, state)

	if time.Now().After(session.ExpiresAt) {
		return oidcStateSession{}, errors.New("oidc session expired")
	}

	return session, nil
}

func (s *AuthService) storeOIDCResultSession(result OIDCSessionResult) string {
	s.oidcMu.Lock()
	defer s.oidcMu.Unlock()

	s.cleanupOIDCSessionsLocked()

	sessionID := uuid.NewString()
	s.oidcResultSessions[sessionID] = oidcResultSession{
		Result:    result,
		ExpiresAt: time.Now().Add(oidcResultSessionTTL),
	}

	return sessionID
}

func (s *AuthService) takeOIDCResultSession(sessionID string) (OIDCSessionResult, error) {
	s.oidcMu.Lock()
	defer s.oidcMu.Unlock()

	s.cleanupOIDCSessionsLocked()

	session, exists := s.oidcResultSessions[sessionID]
	if !exists {
		return OIDCSessionResult{}, errors.New("invalid or expired oidc result session")
	}
	delete(s.oidcResultSessions, sessionID)

	if time.Now().After(session.ExpiresAt) {
		return OIDCSessionResult{}, errors.New("oidc result session expired")
	}

	return session.Result, nil
}

func (s *AuthService) cleanupOIDCSessionsLocked() {
	now := time.Now()

	for state, session := range s.oidcStateSessions {
		if now.After(session.ExpiresAt) {
			delete(s.oidcStateSessions, state)
		}
	}

	for sessionID, session := range s.oidcResultSessions {
		if now.After(session.ExpiresAt) {
			delete(s.oidcResultSessions, sessionID)
		}
	}
}

func mapOIDCConnectionInfo(record model.OIDCConnection) OIDCConnectionInfo {
	return OIDCConnectionInfo{
		ID:        record.ID,
		Provider:  record.Provider,
		Email:     record.Email,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func parseBoolSetting(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

func parseOIDCScopes(raw string) []string {
	value := strings.ReplaceAll(strings.TrimSpace(raw), ",", " ")
	if value == "" {
		return nil
	}

	parts := strings.Fields(value)
	if len(parts) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(parts))
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		scope := strings.TrimSpace(part)
		if scope == "" {
			continue
		}
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}

	if len(scopes) == 0 {
		return nil
	}

	hasOpenID := false
	for _, scope := range scopes {
		if scope == "openid" {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID {
		scopes = append([]string{"openid"}, scopes...)
	}

	return scopes
}

func parseOIDCExtraAuthParams(raw string) map[string]string {
	parsed := make(map[string]string)
	value := strings.TrimSpace(raw)
	if value == "" {
		return parsed
	}
	value = strings.TrimPrefix(value, "?")

	query, err := url.ParseQuery(value)
	if err != nil {
		return parsed
	}

	disallowed := map[string]struct{}{
		"scope":                 {},
		"state":                 {},
		"nonce":                 {},
		"redirect_uri":          {},
		"client_id":             {},
		"response_type":         {},
		"code_challenge":        {},
		"code_challenge_method": {},
	}

	for key, values := range query {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		if _, blocked := disallowed[trimmedKey]; blocked {
			continue
		}
		if len(values) == 0 {
			continue
		}

		trimmedValue := strings.TrimSpace(values[len(values)-1])
		if trimmedValue == "" {
			continue
		}
		parsed[trimmedKey] = trimmedValue
	}

	return parsed
}

func fetchOIDCUserInfoClaims(ctx context.Context, provider *oidc.Provider, oauthToken *oauth2.Token, userInfoEndpoint string) (*oidcIdentityClaims, error) {
	if strings.TrimSpace(userInfoEndpoint) == "" {
		userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(oauthToken))
		if err != nil {
			return nil, err
		}

		var claims oidcIdentityClaims
		if err := userInfo.Claims(&claims); err != nil {
			return nil, err
		}
		return &claims, nil
	}

	if strings.TrimSpace(oauthToken.AccessToken) == "" {
		return nil, errors.New("oidc access token is missing")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+oauthToken.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("oidc userinfo endpoint returned %d", resp.StatusCode)
	}

	var claims oidcIdentityClaims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

func normalizeOIDCUsername(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(trimmed))
	pendingSeparator := false

	for _, ch := range trimmed {
		switch {
		case ch >= 'a' && ch <= 'z':
			if pendingSeparator && builder.Len() > 0 {
				builder.WriteByte('_')
			}
			builder.WriteRune(ch)
			pendingSeparator = false
		case ch >= '0' && ch <= '9':
			if pendingSeparator && builder.Len() > 0 {
				builder.WriteByte('_')
			}
			builder.WriteRune(ch)
			pendingSeparator = false
		case ch == '_' || ch == '-' || ch == '.' || ch == ' ':
			pendingSeparator = true
		}

		if builder.Len() >= 32 {
			break
		}
	}

	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "user"
	}
	return result
}

func generateSecureToken(byteLen int) (string, error) {
	if byteLen <= 0 {
		byteLen = 16
	}

	buffer := make([]byte, byteLen)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
