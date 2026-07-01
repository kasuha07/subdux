package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Reauth ("step-up") re-verifies that the human behind an already-authenticated
// session is present before a sensitive operation runs. A single factor
// (password or passkey) is verified here and, on success, a short-lived,
// single-use, operation-scoped ticket is minted. The sensitive endpoint then
// only has to Consume the ticket — it never needs to know which factor was
// used. Adding a new factor later (e.g. TOTP) means adding one verifier that
// mints a ticket; the sensitive endpoints stay untouched.
//
// Tickets and passkey challenge sessions are held in memory, matching the
// existing passkey/OIDC session stores. This is correct for a single-process
// deployment; a multi-instance deployment would need shared storage for these
// (a pre-existing property of the passkey/OIDC flows, not unique to reauth).

// Operation identifiers scope a ticket to a single sensitive action so a ticket
// minted for one operation cannot authorize another.
const (
	ReauthOperationBackup  = "backup"
	ReauthOperationRestore = "restore"
)

const (
	reauthTicketTTL   = 5 * time.Minute
	maxReauthTickets  = 1024
	reauthTicketBytes = 32
)

// ErrInvalidReauthOperation is returned when a caller supplies an unknown
// operation identifier.
var ErrInvalidReauthOperation = errors.New("invalid reauth operation")

// ErrReauthRequired is returned by Consume when no valid ticket backs the
// request (missing, expired, already used, or scoped to another user or
// operation). It deliberately does not distinguish these cases.
var ErrReauthRequired = errors.New("re-authentication required")

// ReauthMethods reports which factors a user can use to re-authenticate.
// Password is always offered; a wrong or unusable password (e.g. an OIDC-only
// account) simply fails verification. Passkey is offered when the user has one
// registered. OIDC is offered when the provider is enabled/configured and the
// user has a linked OIDC identity — the step-up factor for OIDC-only admins who
// never learned their randomly-generated password.
type ReauthMethods struct {
	Password bool `json:"password"`
	Passkey  bool `json:"passkey"`
	OIDC     bool `json:"oidc"`
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

func isValidReauthOperation(operation string) bool {
	switch operation {
	case ReauthOperationBackup, ReauthOperationRestore:
		return true
	default:
		return false
	}
}

// AvailableMethods reports the factors the user can present for reauth.
func (s *ReauthService) AvailableMethods(userID uint) (ReauthMethods, error) {
	hasPasskey, err := s.auth.HasPasskeys(userID)
	if err != nil {
		return ReauthMethods{}, err
	}
	hasOIDC, err := s.auth.CanReauthWithOIDC(userID)
	if err != nil {
		return ReauthMethods{}, err
	}
	return ReauthMethods{Password: true, Passkey: hasPasskey, OIDC: hasOIDC}, nil
}

// VerifyPassword checks the user's account password and, on success, mints a
// ticket for the given operation.
func (s *ReauthService) VerifyPassword(userID uint, operation string, password string) (string, error) {
	if !isValidReauthOperation(operation) {
		return "", ErrInvalidReauthOperation
	}
	if password == "" {
		return "", ErrReauthRequired
	}

	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", ErrReauthRequired
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return "", ErrReauthRequired
	}

	return s.mintTicket(userID, operation)
}

// BeginPasskey starts a user-scoped passkey assertion for the operation. The
// operation is validated here; the challenge itself is issued by AuthService.
func (s *ReauthService) BeginPasskey(userID uint, operation string, origin string, host string, scheme string) (*PasskeyBeginResult, error) {
	if !isValidReauthOperation(operation) {
		return nil, ErrInvalidReauthOperation
	}
	return s.auth.BeginPasskeyReauth(userID, operation, origin, host, scheme)
}

// FinishPasskey validates a passkey assertion for the user and, on success,
// mints a ticket for the operation.
func (s *ReauthService) FinishPasskey(userID uint, operation string, sessionID string, parsedResponse *protocol.ParsedCredentialAssertionData, origin string, host string, scheme string) (string, error) {
	if !isValidReauthOperation(operation) {
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
	if !isValidReauthOperation(operation) {
		return nil, ErrInvalidReauthOperation
	}
	return s.auth.BeginOIDCReauth(userID, operation)
}

// VerifyOIDC completes an OIDC step-up: it spends the single-use reauth result
// session produced by the OIDC callback (bound to this user and operation) and,
// on success, mints a ticket. Mirrors FinishPasskey — the sensitive endpoints
// never learn which factor was used.
func (s *ReauthService) VerifyOIDC(userID uint, operation string, sessionID string) (string, error) {
	if !isValidReauthOperation(operation) {
		return "", ErrInvalidReauthOperation
	}
	if err := s.auth.ConsumeOIDCReauthResult(sessionID, userID, operation); err != nil {
		return "", err
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
