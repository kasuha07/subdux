package reauth

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/servicetest"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const testOIDCProviderKey = "oidc"

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return servicetest.NewDB(t)
}

func createTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()
	return servicetest.CreateUser(t, db)
}

type testAuthenticator struct {
	db           *gorm.DB
	mu           *sync.Mutex
	oidcSessions map[string]testOIDCSession
}

type testOIDCSession struct {
	userID    uint
	operation string
	grade     OIDCReauthGrade
}

func newTestAuthenticator(db *gorm.DB) *testAuthenticator {
	return &testAuthenticator{
		db:           db,
		mu:           &sync.Mutex{},
		oidcSessions: make(map[string]testOIDCSession),
	}
}

func (a *testAuthenticator) WithContext(ctx context.Context) Authenticator {
	clone := *a
	clone.db = a.db.WithContext(ctx)
	return &clone
}

func (a *testAuthenticator) FactorState(userID uint) (FactorState, error) {
	var user model.User
	if err := a.db.Select("id", "password", "totp_enabled").First(&user, userID).Error; err != nil {
		return FactorState{}, err
	}
	var passkeyCount int64
	if err := a.db.Model(&model.PasskeyCredential{}).Where("user_id = ?", userID).Count(&passkeyCount).Error; err != nil {
		return FactorState{}, err
	}
	hasOIDC, err := a.hasOIDCConnection(userID)
	if err != nil {
		return FactorState{}, err
	}
	hasOIDC = hasOIDC && a.oidcProviderEnabled()
	return FactorState{
		HasPassword: user.Password != "",
		HasTOTP:     user.TotpEnabled,
		HasPasskey:  passkeyCount > 0,
		HasOIDC:     hasOIDC,
	}, nil
}

func (a *testAuthenticator) VerifyPassword(userID uint, password string, code string, requireTOTP bool) error {
	var user model.User
	if err := a.db.First(&user, userID).Error; err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return errors.New("invalid password")
	}
	if requireTOTP {
		if code == "" || user.TotpSecret == nil || !totp.Validate(code, *user.TotpSecret) {
			return errors.New("invalid totp")
		}
	}
	return nil
}

func (a *testAuthenticator) BeginPasskeyReauth(userID uint, operation string, origin string, host string, scheme string) (*PasskeyBeginResult, error) {
	return &PasskeyBeginResult{SessionID: uuid.NewString(), Options: map[string]any{}}, nil
}

func (a *testAuthenticator) FinishPasskeyReauth(userID uint, operation string, sessionID string, parsedResponse *protocol.ParsedCredentialAssertionData, origin string, host string, scheme string) error {
	return nil
}

func (a *testAuthenticator) BeginOIDCReauth(userID uint, operation string) (*OIDCStartResult, error) {
	return &OIDCStartResult{AuthorizationURL: "https://issuer.example.com/auth"}, nil
}

func (a *testAuthenticator) ConsumeOIDCReauthResult(sessionID string, userID uint, operation string) (OIDCReauthGrade, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	session, ok := a.oidcSessions[sessionID]
	if !ok {
		return 0, errors.New("invalid oidc reauth session")
	}
	delete(a.oidcSessions, sessionID)
	if session.userID != userID || session.operation != operation {
		return 0, errors.New("invalid oidc reauth session")
	}
	return session.grade, nil
}

func (a *testAuthenticator) HasOIDCConnection(userID uint) (bool, error) {
	return a.hasOIDCConnection(userID)
}

func (a *testAuthenticator) hasOIDCConnection(userID uint) (bool, error) {
	var count int64
	if err := a.db.Model(&model.OIDCConnection{}).
		Where("provider = ? AND user_id = ?", testOIDCProviderKey, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (a *testAuthenticator) oidcProviderEnabled() bool {
	var count int64
	err := a.db.Model(&model.SystemSetting{}).
		Where("key = ? AND value = ?", "oidc_enabled", "true").
		Count(&count).Error
	return err == nil && count > 0
}

func (a *testAuthenticator) mintOIDCSession(userID uint, operation string, grade OIDCReauthGrade) string {
	sessionID := uuid.NewString()
	a.mu.Lock()
	defer a.mu.Unlock()
	a.oidcSessions[sessionID] = testOIDCSession{userID: userID, operation: operation, grade: grade}
	return sessionID
}
