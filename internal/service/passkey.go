package service

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

const passkeySessionTTL = 5 * time.Minute

type passkeySessionKind string

const (
	passkeySessionKindRegistration passkeySessionKind = "registration"
	passkeySessionKindLogin        passkeySessionKind = "login"
)

type passkeySession struct {
	Kind      passkeySessionKind
	UserID    uint
	Name      string
	Data      webauthn.SessionData
	ExpiresAt time.Time
}

type PasskeyCredentialInfo struct {
	ID           uint       `json:"id"`
	Name         string     `json:"name"`
	CredentialID string     `json:"credential_id"`
	LastUsedAt   *time.Time `json:"last_used_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type PasskeyBeginResult struct {
	SessionID string      `json:"session_id"`
	Options   interface{} `json:"options"`
}

type webAuthnUser struct {
	account     model.User
	displayName string
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte {
	return []byte(strconv.FormatUint(uint64(u.account.ID), 10))
}

func (u *webAuthnUser) WebAuthnName() string {
	return u.account.Username
}

func (u *webAuthnUser) WebAuthnDisplayName() string {
	return u.displayName
}

func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

func (u *webAuthnUser) WebAuthnIcon() string {
	return ""
}

func (s *AuthService) ListPasskeys(userID uint) ([]PasskeyCredentialInfo, error) {
	var records []model.PasskeyCredential
	if err := s.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	result := make([]PasskeyCredentialInfo, 0, len(records))
	for _, record := range records {
		result = append(result, PasskeyCredentialInfo{
			ID:           record.ID,
			Name:         record.Name,
			CredentialID: record.CredentialID,
			LastUsedAt:   record.LastUsedAt,
			CreatedAt:    record.CreatedAt,
		})
	}
	return result, nil
}

func (s *AuthService) BeginPasskeyRegistration(userID uint, name string, origin string, host string, scheme string) (*PasskeyBeginResult, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}

	wa, err := s.buildWebAuthn(origin, host, scheme)
	if err != nil {
		return nil, err
	}

	waUser, err := s.getWebAuthnUser(*user)
	if err != nil {
		return nil, err
	}

	options, sessionData, err := wa.BeginRegistration(waUser)
	if err != nil {
		return nil, err
	}

	options.Response.AuthenticatorSelection = protocol.AuthenticatorSelection{
		ResidentKey:      protocol.ResidentKeyRequirementPreferred,
		UserVerification: protocol.VerificationPreferred,
	}
	options.Response.Attestation = protocol.PreferNoAttestation

	label := strings.TrimSpace(name)
	if label == "" {
		label = "Passkey"
	}

	sessionID := s.storePasskeySession(passkeySession{
		Kind:      passkeySessionKindRegistration,
		UserID:    userID,
		Name:      label,
		Data:      *sessionData,
		ExpiresAt: time.Now().Add(passkeySessionTTL),
	})

	return &PasskeyBeginResult{
		SessionID: sessionID,
		Options:   options,
	}, nil
}

func (s *AuthService) FinishPasskeyRegistration(userID uint, sessionID string, parsedResponse *protocol.ParsedCredentialCreationData, origin string, host string, scheme string) (*PasskeyCredentialInfo, error) {
	session, err := s.takePasskeySession(sessionID, passkeySessionKindRegistration)
	if err != nil {
		return nil, err
	}

	if session.UserID != userID {
		return nil, errors.New("invalid passkey session")
	}

	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}

	wa, err := s.buildWebAuthn(origin, host, scheme)
	if err != nil {
		return nil, err
	}

	waUser, err := s.getWebAuthnUser(*user)
	if err != nil {
		return nil, err
	}

	credential, err := wa.CreateCredential(waUser, session.Data, parsedResponse)
	if err != nil {
		return nil, errors.New("failed to register passkey")
	}

	payload, err := json.Marshal(credential)
	if err != nil {
		return nil, err
	}

	record := model.PasskeyCredential{
		UserID:       userID,
		Name:         session.Name,
		CredentialID: encodeCredentialID(credential.ID),
		Credential:   payload,
	}

	if err := s.DB.Create(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errors.New("passkey already registered")
		}
		return nil, err
	}

	return &PasskeyCredentialInfo{
		ID:           record.ID,
		Name:         record.Name,
		CredentialID: record.CredentialID,
		LastUsedAt:   record.LastUsedAt,
		CreatedAt:    record.CreatedAt,
	}, nil
}

func (s *AuthService) BeginPasskeyLogin(origin string, host string, scheme string) (*PasskeyBeginResult, error) {
	wa, err := s.buildWebAuthn(origin, host, scheme)
	if err != nil {
		return nil, err
	}

	options, sessionData, err := wa.BeginDiscoverableLogin()
	if err != nil {
		return nil, err
	}
	options.Response.UserVerification = protocol.VerificationPreferred

	sessionID := s.storePasskeySession(passkeySession{
		Kind:      passkeySessionKindLogin,
		Data:      *sessionData,
		ExpiresAt: time.Now().Add(passkeySessionTTL),
	})

	return &PasskeyBeginResult{
		SessionID: sessionID,
		Options:   options,
	}, nil
}

func (s *AuthService) FinishPasskeyLogin(sessionID string, parsedResponse *protocol.ParsedCredentialAssertionData, origin string, host string, scheme string) (*AuthResponse, error) {
	session, err := s.takePasskeySession(sessionID, passkeySessionKindLogin)
	if err != nil {
		return nil, err
	}

	wa, err := s.buildWebAuthn(origin, host, scheme)
	if err != nil {
		return nil, err
	}

	waUser, credential, err := wa.ValidatePasskeyLogin(s.discoverableUserHandler, session.Data, parsedResponse)
	if err != nil {
		return nil, errors.New("passkey verification failed")
	}

	resolved, ok := waUser.(*webAuthnUser)
	if !ok {
		return nil, errors.New("failed to resolve passkey user")
	}
	user := resolved.account

	now := time.Now()
	payload, marshalErr := json.Marshal(credential)
	if marshalErr == nil {
		_ = s.DB.Model(&model.PasskeyCredential{}).
			Where("user_id = ? AND credential_id = ?", user.ID, encodeCredentialID(credential.ID)).
			Updates(map[string]interface{}{
				"credential":   payload,
				"last_used_at": &now,
			}).Error
	}

	token, err := pkg.GenerateToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) DeletePasskey(userID uint, passkeyID uint) error {
	result := s.DB.Where("id = ? AND user_id = ?", passkeyID, userID).Delete(&model.PasskeyCredential{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("passkey not found")
	}
	return nil
}

func (s *AuthService) getWebAuthnUser(user model.User) (*webAuthnUser, error) {
	records, err := s.getPasskeyRecords(user.ID)
	if err != nil {
		return nil, err
	}

	credentials := make([]webauthn.Credential, 0, len(records))
	for _, record := range records {
		var credential webauthn.Credential
		if err := json.Unmarshal(record.Credential, &credential); err != nil {
			return nil, fmt.Errorf("failed to decode passkey: %w", err)
		}
		credentials = append(credentials, credential)
	}

	displayName := user.Username
	if strings.TrimSpace(user.Email) != "" {
		displayName = user.Email
	}

	return &webAuthnUser{
		account:     user,
		displayName: displayName,
		credentials: credentials,
	}, nil
}

func (s *AuthService) discoverableUserHandler(rawID, userHandle []byte) (webauthn.User, error) {
	user, err := s.resolvePasskeyUser(rawID, userHandle)
	if err != nil {
		return nil, err
	}

	return s.getWebAuthnUser(*user)
}

func (s *AuthService) resolvePasskeyUser(rawID, userHandle []byte) (*model.User, error) {
	if len(userHandle) > 0 {
		if userID, err := strconv.ParseUint(string(userHandle), 10, 64); err == nil && userID > 0 {
			user, userErr := s.GetUser(uint(userID))
			if userErr != nil {
				return nil, errors.New("user not found")
			}
			if user.Status == "disabled" {
				return nil, errors.New("account is disabled")
			}
			return user, nil
		}
	}

	if len(rawID) == 0 {
		return nil, errors.New("invalid passkey user")
	}

	var passkey model.PasskeyCredential
	if err := s.DB.Where("credential_id = ?", encodeCredentialID(rawID)).First(&passkey).Error; err != nil {
		return nil, errors.New("invalid passkey user")
	}

	user, err := s.GetUser(passkey.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if user.Status == "disabled" {
		return nil, errors.New("account is disabled")
	}
	return user, nil
}

func (s *AuthService) getPasskeyRecords(userID uint) ([]model.PasskeyCredential, error) {
	var records []model.PasskeyCredential
	if err := s.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (s *AuthService) buildWebAuthn(origin string, host string, scheme string) (*webauthn.WebAuthn, error) {
	siteName := s.getSetting("site_name")
	if siteName == "" {
		siteName = "Subdux"
	}

	siteURL := normalizeSiteURL(s.getSetting("site_url"))

	origins := make(map[string]struct{})
	rpID := ""

	if siteURL != "" {
		parsed, _ := url.Parse(siteURL)
		rpID = parsed.Hostname()
		origins[siteURL] = struct{}{}
	}

	if origin != "" {
		if parsed, err := url.Parse(origin); err == nil && parsed.Hostname() != "" {
			if rpID == "" {
				rpID = parsed.Hostname()
			}
			origins[fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)] = struct{}{}
		}
	}

	if rpID == "" {
		hostname := extractHostName(host)
		if hostname == "" {
			return nil, errors.New("failed to determine relying party id")
		}
		rpID = hostname
	}

	if len(origins) == 0 {
		if scheme == "" {
			scheme = "http"
		}
		if host == "" {
			host = rpID
		}
		origins[fmt.Sprintf("%s://%s", scheme, host)] = struct{}{}
	}

	originList := make([]string, 0, len(origins))
	for value := range origins {
		originList = append(originList, value)
	}
	sort.Strings(originList)

	config := &webauthn.Config{
		RPDisplayName: siteName,
		RPID:          rpID,
		RPOrigins:     originList,
	}

	return webauthn.New(config)
}

func (s *AuthService) getSetting(key string) string {
	var setting model.SystemSetting
	if err := s.DB.Where("key = ?", key).First(&setting).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(setting.Value)
}

func normalizeSiteURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Hostname() == "" {
		return ""
	}

	scheme := parsed.Scheme
	if scheme == "" {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s", scheme, parsed.Host)
}

func extractHostName(host string) string {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return ""
	}

	if strings.Contains(trimmed, ",") {
		parts := strings.Split(trimmed, ",")
		trimmed = strings.TrimSpace(parts[0])
	}

	if parsed, err := url.Parse(trimmed); err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}

	if h, _, err := net.SplitHostPort(trimmed); err == nil {
		return h
	}

	return strings.TrimSpace(trimmed)
}

func encodeCredentialID(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func (s *AuthService) storePasskeySession(session passkeySession) string {
	s.passkeyMu.Lock()
	defer s.passkeyMu.Unlock()

	s.cleanupPasskeySessionsLocked()

	sessionID := uuid.NewString()
	s.passkeySessions[sessionID] = session
	return sessionID
}

func (s *AuthService) takePasskeySession(sessionID string, expected passkeySessionKind) (passkeySession, error) {
	s.passkeyMu.Lock()
	defer s.passkeyMu.Unlock()

	s.cleanupPasskeySessionsLocked()

	session, exists := s.passkeySessions[sessionID]
	if !exists {
		return passkeySession{}, errors.New("invalid or expired passkey session")
	}

	delete(s.passkeySessions, sessionID)

	if session.Kind != expected {
		return passkeySession{}, errors.New("invalid passkey session")
	}

	if time.Now().After(session.ExpiresAt) {
		return passkeySession{}, errors.New("passkey session expired")
	}

	return session, nil
}

func (s *AuthService) cleanupPasskeySessionsLocked() {
	now := time.Now()
	for sessionID, session := range s.passkeySessions {
		if now.After(session.ExpiresAt) {
			delete(s.passkeySessions, sessionID)
		}
	}
}
