package service

import (
	"context"
	"errors"
	"sync"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Reauth ("step-up") re-verifies that the human behind an already-authenticated
// session is present before a sensitive operation runs. A reauth method
// (password, passkey, or OIDC) is verified here and, on success, a short-lived,
// single-use, operation-scoped ticket is minted. The password method can itself
// require a second factor: when the user has TOTP enabled, password reauth is
// upgraded to password plus a current TOTP code. The sensitive endpoint then
// only has to Consume the ticket — it never needs to know which factor path was
// used.
//
// Tickets and passkey challenge sessions are held in memory, matching the
// existing passkey/OIDC session stores. This is correct for a single-process
// deployment; a multi-instance deployment would need shared storage for these
// (a pre-existing property of the passkey/OIDC flows, not unique to reauth).

// ErrInvalidReauthOperation is returned when a caller supplies an unknown
// operation identifier.
var ErrInvalidReauthOperation = errors.New("invalid reauth operation")

// ErrReauthRequired is returned by Consume when no valid ticket backs the
// request (missing, expired, already used, or scoped to another user or
// operation). It deliberately does not distinguish these cases.
var ErrReauthRequired = errors.New("re-authentication required")

// ErrPasswordReauthDisabled is returned when the password (knowledge) factor is
// not an accepted reauth method for the account. This is the case when the user
// has a passkey enrolled but no TOTP: per policy a passkey account must step up
// with the passkey itself (or, if TOTP is also enrolled, password+TOTP). The
// error is distinct from ErrReauthRequired so the caller can steer the user to a
// stronger factor rather than implying a wrong password.
var ErrPasswordReauthDisabled = errors.New("password re-authentication is not available for this account; use a passkey")

// ErrOIDCReauthInsufficient is returned when an OIDC step-up succeeds but the
// provider login did not prove a strong enough assurance level for the account's
// enrolled factors (e.g. a TOTP account needs OIDC-2 MFA, a passkey account needs
// OIDC-3 phishing-resistant). The user can fall back to another accepted method.
var ErrOIDCReauthInsufficient = errors.New("your provider sign-in did not prove strong enough authentication for this account; use a passkey or another method")

// ReauthService verifies a re-authentication factor and manages the resulting
// tickets. Passkey verification is delegated to AuthService, which owns the
// WebAuthn machinery; password verification is done here against the user's
// bcrypt hash.
type ReauthService struct {
	db   *gorm.DB
	auth *AuthService

	mu      *sync.Mutex
	tickets map[string]reauthTicket
}

func NewReauthService(db *gorm.DB, auth *AuthService) *ReauthService {
	return &ReauthService{
		db:      db,
		auth:    auth,
		mu:      &sync.Mutex{},
		tickets: make(map[string]reauthTicket),
	}
}

// WithContext binds the database handle (and the delegated AuthService) to ctx.
// The in-memory ticket store and its lock are shared via pointers, so the clone
// sees the same tickets as the parent.
func (s *ReauthService) WithContext(ctx context.Context) *ReauthService {
	clone := *s
	clone.db = withContext(s.db, ctx)
	if s.auth != nil {
		clone.auth = s.auth.WithContext(ctx)
	}
	return &clone
}

// VerifyPassword checks the user's account password and, when TOTP is enabled on
// the account, also requires a current authenticator code. It fails with
// ErrPasswordReauthDisabled when the knowledge factor is not an accepted method
// for the account (a passkey enrolled without TOTP). On success it mints a ticket
// for the given operation.
func (s *ReauthService) VerifyPassword(userID uint, operation string, password string, code string) (string, error) {
	if !IsValidReauthOperation(operation) {
		return "", ErrInvalidReauthOperation
	}
	if password == "" {
		return "", ErrReauthRequired
	}

	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", ErrReauthRequired
	}

	hasPasskey, err := s.auth.HasPasskeys(userID)
	if err != nil {
		return "", err
	}
	policy := reauthPolicyFor(operation, reauthFactorAvailability{
		hasTOTP:    user.TotpEnabled,
		hasPasskey: hasPasskey,
	})
	if !policy.PasswordAllowed {
		return "", ErrPasswordReauthDisabled
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return "", ErrReauthRequired
	}
	if policy.PasswordRequiresTOTP {
		if code == "" || user.TotpSecret == nil || !totp.Validate(code, *user.TotpSecret) {
			return "", ErrReauthRequired
		}
	}

	return s.mintTicket(userID, operation)
}

// BeginPasskey starts a user-scoped passkey assertion for the operation. The
// operation is validated here; the challenge itself is issued by AuthService.
func (s *ReauthService) BeginPasskey(userID uint, operation string, origin string, host string, scheme string) (*PasskeyBeginResult, error) {
	if !IsValidReauthOperation(operation) {
		return nil, ErrInvalidReauthOperation
	}
	return s.auth.BeginPasskeyReauth(userID, operation, origin, host, scheme)
}

// FinishPasskey validates a passkey assertion for the user and, on success,
// mints a ticket for the operation.
func (s *ReauthService) FinishPasskey(userID uint, operation string, sessionID string, parsedResponse *protocol.ParsedCredentialAssertionData, origin string, host string, scheme string) (string, error) {
	if !IsValidReauthOperation(operation) {
		return "", ErrInvalidReauthOperation
	}
	if err := s.auth.FinishPasskeyReauth(userID, operation, sessionID, parsedResponse, origin, host, scheme); err != nil {
		return "", err
	}
	return s.mintTicket(userID, operation)
}

// BeginOIDC starts an OIDC step-up for the operation, returning the provider
// authorization URL the client opens (in a popup) to authenticate. The operation
// is validated here and carried through the OIDC state session.
func (s *ReauthService) BeginOIDC(userID uint, operation string) (*OIDCStartResult, error) {
	if !IsValidReauthOperation(operation) {
		return nil, ErrInvalidReauthOperation
	}
	return s.auth.BeginOIDCReauth(userID, operation)
}

// VerifyOIDC completes an OIDC step-up: it spends the single-use reauth result
// session produced by the OIDC callback (bound to this user and operation),
// enforces that the provider login proved a strong enough assurance level for the
// account's enrolled factors, and on success mints a ticket. Mirrors
// FinishPasskey — the sensitive endpoints never learn which factor was used.
func (s *ReauthService) VerifyOIDC(userID uint, operation string, sessionID string) (string, error) {
	if !IsValidReauthOperation(operation) {
		return "", ErrInvalidReauthOperation
	}

	grade, err := s.auth.ConsumeOIDCReauthResult(sessionID, userID, operation)
	if err != nil {
		return "", err
	}

	var user model.User
	if err := s.db.Select("id", "totp_enabled").First(&user, userID).Error; err != nil {
		return "", ErrReauthRequired
	}
	hasPasskey, err := s.auth.HasPasskeys(userID)
	if err != nil {
		return "", err
	}
	policy := reauthPolicyFor(operation, reauthFactorAvailability{
		hasTOTP:    user.TotpEnabled,
		hasPasskey: hasPasskey,
	})
	if grade < policy.RequiredOIDCGrade {
		return "", ErrOIDCReauthInsufficient
	}

	return s.mintTicket(userID, operation)
}
