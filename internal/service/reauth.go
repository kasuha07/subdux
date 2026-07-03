package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
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

// Operation identifiers scope a ticket to a single sensitive action so a ticket
// minted for one operation cannot authorize another.
const (
	ReauthOperationBackup         = "backup"
	ReauthOperationBackupSchedule = "backup_schedule"
	ReauthOperationRestore        = "restore"
	ReauthOperationChangeEmail    = "change_email"
	ReauthOperationAddPasskey     = "add_passkey"
	ReauthOperationDeletePasskey  = "delete_passkey"
	ReauthOperationEnableTOTP     = "enable_totp"
	ReauthOperationDisableTOTP    = "disable_totp"
	ReauthOperationConnectOIDC    = "connect_oidc"
	ReauthOperationCreateAPIKey   = "create_api_key"
	ReauthOperationChangeUserRole = "change_user_role"
	ReauthOperationExportRedacted = "export_redacted"
	ReauthOperationExportSecrets  = "export_secrets"
	ReauthOperationImportSubdux   = "import_subdux"
	ReauthOperationImportWallos   = "import_wallos"
)

const (
	reauthTicketTTL   = 5 * time.Minute
	maxReauthTickets  = 256
	reauthTicketBytes = 32
)

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

// ReauthMethods reports which factors a user can use to re-authenticate, after
// applying the account's step-up policy (reauthPolicyFor). Password is offered
// only when the knowledge factor is an accepted method for the account — it is
// withheld from passkey accounts that have no TOTP, which must step up with the
// passkey itself. PasswordRequiresTOTP is set when the password path is offered
// but must be accompanied by a current TOTP code. Passkey is offered when the
// user has one registered. OIDC is offered when the provider is enabled and the
// user has a linked OIDC identity; the grade the provider must prove is enforced
// at verification time (VerifyOIDC), not advertised here.
type ReauthMethods struct {
	Password             bool `json:"password"`
	PasswordRequiresTOTP bool `json:"password_requires_totp"`
	Passkey              bool `json:"passkey"`
	OIDC                 bool `json:"oidc"`
}

// reauthPolicy captures the accepted step-up methods for an account and
// operation given its enrolled factors. It is the single source of truth for the
// default reauth matrix:
//
//	enrolled factors            password        passkey   OIDC min grade
//	password                    yes             -         OIDC-1
//	password+TOTP               yes (+TOTP)     -         OIDC-2
//	password+passkey            no              yes       OIDC-3
//	password+TOTP+passkey       yes (+TOTP)     yes       OIDC-3
//
// The knowledge (password) factor is disabled exactly when a passkey is enrolled
// without TOTP: such an account must use its passkey (a stronger, phishing-
// resistant factor) rather than a bare password. When TOTP is also enrolled the
// password path returns as password+TOTP, matching the passkey's strength.
//
// Disabling TOTP has one operation-specific exception: when both TOTP and a
// passkey are enrolled, OIDC-2 remains an accepted fallback even though OIDC-3 is
// preferred. This lets a user remove the TOTP factor after proving either a
// passkey/OIDC-3 path or the still-current MFA path (password+TOTP or OIDC-2).
//
// Deleting a passkey deliberately uses the normal passkey-account policy and is
// scoped only to the user and delete_passkey operation, not to the credential
// being deleted. The user may prove presence with any registered passkey; the
// target passkey ID is authorized by the deletion endpoint's ownership check.
type reauthPolicy struct {
	passwordAllowed      bool
	passwordRequiresTOTP bool
	requiredOIDCGrade    OIDCReauthGrade
}

func reauthPolicyFor(operation string, hasTOTP, hasPasskey bool) reauthPolicy {
	requiredGrade := OIDCGradeFresh
	if hasPasskey {
		requiredGrade = OIDCGradePhishingResistant
	} else if hasTOTP {
		requiredGrade = OIDCGradeMFA
	}
	if operation == ReauthOperationDisableTOTP && hasTOTP {
		requiredGrade = OIDCGradeMFA
	}
	return reauthPolicy{
		passwordAllowed:      !(hasPasskey && !hasTOTP),
		passwordRequiresTOTP: hasTOTP,
		requiredOIDCGrade:    requiredGrade,
	}
}

type reauthTicket struct {
	userID    uint
	operation string
	expiresAt time.Time
	createdAt time.Time
}

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

// IsValidReauthOperation reports whether operation is a known reauth operation.
// It is the single source of truth for the set of valid operations.
func IsValidReauthOperation(operation string) bool {
	switch operation {
	case ReauthOperationBackup,
		ReauthOperationBackupSchedule,
		ReauthOperationRestore,
		ReauthOperationChangeEmail,
		ReauthOperationAddPasskey,
		ReauthOperationDeletePasskey,
		ReauthOperationEnableTOTP,
		ReauthOperationDisableTOTP,
		ReauthOperationConnectOIDC,
		ReauthOperationCreateAPIKey,
		ReauthOperationChangeUserRole,
		ReauthOperationExportRedacted,
		ReauthOperationExportSecrets,
		ReauthOperationImportSubdux,
		ReauthOperationImportWallos:
		return true
	default:
		return false
	}
}

// AvailableMethods reports the factors the user can present for reauth, after
// applying the account's step-up policy.
func (s *ReauthService) AvailableMethods(userID uint, operation string) (ReauthMethods, error) {
	if !IsValidReauthOperation(operation) {
		return ReauthMethods{}, ErrInvalidReauthOperation
	}

	var user model.User
	if err := s.db.Select("id", "totp_enabled").First(&user, userID).Error; err != nil {
		return ReauthMethods{}, err
	}
	hasPasskey, err := s.auth.HasPasskeys(userID)
	if err != nil {
		return ReauthMethods{}, err
	}
	hasOIDC, err := s.auth.CanReauthWithOIDC(userID)
	if err != nil {
		return ReauthMethods{}, err
	}

	policy := reauthPolicyFor(operation, user.TotpEnabled, hasPasskey)
	return ReauthMethods{
		Password:             policy.passwordAllowed,
		PasswordRequiresTOTP: policy.passwordAllowed && policy.passwordRequiresTOTP,
		Passkey:              hasPasskey,
		OIDC:                 hasOIDC,
	}, nil
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
	policy := reauthPolicyFor(operation, user.TotpEnabled, hasPasskey)
	if !policy.passwordAllowed {
		return "", ErrPasswordReauthDisabled
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return "", ErrReauthRequired
	}
	if policy.passwordRequiresTOTP {
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
	policy := reauthPolicyFor(operation, user.TotpEnabled, hasPasskey)
	if grade < policy.requiredOIDCGrade {
		return "", ErrOIDCReauthInsufficient
	}

	return s.mintTicket(userID, operation)
}

// Consume validates and atomically spends a ticket. A ticket is valid only for
// the same user and operation it was minted for, and only once.
func (s *ReauthService) Consume(userID uint, operation string, ticket string) error {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return ErrReauthRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()

	entry, ok := s.tickets[ticket]
	if !ok {
		return ErrReauthRequired
	}
	// Single-use: remove regardless of whether it matches, so a leaked ticket
	// cannot be probed against multiple users/operations.
	delete(s.tickets, ticket)

	if entry.userID != userID || entry.operation != operation {
		return ErrReauthRequired
	}
	if pkg.NowUTC().After(entry.expiresAt) {
		return ErrReauthRequired
	}
	return nil
}

func (s *ReauthService) mintTicket(userID uint, operation string) (string, error) {
	// generateSecureToken returns URL-safe base64 with no padding.
	ticket, err := generateSecureToken(reauthTicketBytes)
	if err != nil {
		return "", err
	}

	now := pkg.NowUTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()
	s.enforceLimitLocked()

	s.tickets[ticket] = reauthTicket{
		userID:    userID,
		operation: operation,
		expiresAt: now.Add(reauthTicketTTL),
		createdAt: now,
	}
	return ticket, nil
}

func (s *ReauthService) cleanupLocked() {
	now := pkg.NowUTC()
	for ticket, entry := range s.tickets {
		if now.After(entry.expiresAt) {
			delete(s.tickets, ticket)
		}
	}
}

func (s *ReauthService) enforceLimitLocked() {
	overflow := len(s.tickets) - maxReauthTickets + 1
	if overflow <= 0 {
		return
	}
	for i := 0; i < overflow; i++ {
		oldestTicket := ""
		var oldestTime time.Time
		for ticket, entry := range s.tickets {
			if oldestTicket == "" || entry.createdAt.Before(oldestTime) {
				oldestTicket = ticket
				oldestTime = entry.createdAt
			}
		}
		if oldestTicket == "" {
			return
		}
		delete(s.tickets, oldestTicket)
	}
}
