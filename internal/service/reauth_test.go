package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

// mutableClock is a test clock whose time can be advanced to exercise TTL logic.
type mutableClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *mutableClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *mutableClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newReauthTestService(t *testing.T) (*ReauthService, model.User, string) {
	t.Helper()
	db := newTestDB(t)

	const password = "s3cret-passphrase"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user := model.User{
		Username: "admin",
		Email:    "admin@example.com",
		Password: string(hash),
		Role:     "admin",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return NewReauthService(db, NewAuthService(db)), user, password
}

func enableReauthTestTOTP(t *testing.T, svc *ReauthService, userID uint, secret string) {
	t.Helper()
	if err := svc.db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
		"totp_enabled": true,
		"totp_secret":  secret,
	}).Error; err != nil {
		t.Fatalf("failed to enable totp: %v", err)
	}
}

func TestReauthVerifyPassword(t *testing.T) {
	svc, user, password := newReauthTestService(t)

	t.Run("correct password mints a usable ticket", func(t *testing.T) {
		ticket, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, password, "")
		if err != nil {
			t.Fatalf("VerifyPassword() error = %v, want nil", err)
		}
		if ticket == "" {
			t.Fatal("VerifyPassword() returned empty ticket")
		}
		if err := svc.Consume(user.ID, ReauthOperationBackup, ticket); err != nil {
			t.Fatalf("Consume() error = %v, want nil", err)
		}
	})

	t.Run("wrong password is rejected", func(t *testing.T) {
		if _, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, "wrong", ""); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("VerifyPassword() error = %v, want ErrReauthRequired", err)
		}
	})

	t.Run("empty password is rejected", func(t *testing.T) {
		if _, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, "", ""); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("VerifyPassword() error = %v, want ErrReauthRequired", err)
		}
	})

	t.Run("unknown operation is rejected", func(t *testing.T) {
		if _, err := svc.VerifyPassword(user.ID, "wipe", password, ""); !errors.Is(err, ErrInvalidReauthOperation) {
			t.Fatalf("VerifyPassword() error = %v, want ErrInvalidReauthOperation", err)
		}
	})

	t.Run("totp-enabled users need a current totp code", func(t *testing.T) {
		const secret = "JBSWY3DPEHPK3PXP"
		enableReauthTestTOTP(t, svc, user.ID, secret)

		if _, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, password, ""); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("VerifyPassword() without code error = %v, want ErrReauthRequired", err)
		}

		if _, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, password, "000000"); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("VerifyPassword() with invalid code error = %v, want ErrReauthRequired", err)
		}

		code, err := totp.GenerateCode(secret, time.Now().UTC())
		if err != nil {
			t.Fatalf("GenerateCode() error = %v, want nil", err)
		}

		ticket, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, password, code)
		if err != nil {
			t.Fatalf("VerifyPassword() with totp error = %v, want nil", err)
		}
		if ticket == "" {
			t.Fatal("VerifyPassword() with totp returned empty ticket")
		}
	})
}

func TestReauthConsume(t *testing.T) {
	svc, user, password := newReauthTestService(t)

	mint := func(op string) string {
		t.Helper()
		ticket, err := svc.VerifyPassword(user.ID, op, password, "")
		if err != nil {
			t.Fatalf("VerifyPassword() error = %v", err)
		}
		return ticket
	}

	t.Run("ticket is single-use", func(t *testing.T) {
		ticket := mint(ReauthOperationBackup)
		if err := svc.Consume(user.ID, ReauthOperationBackup, ticket); err != nil {
			t.Fatalf("first Consume() error = %v, want nil", err)
		}
		if err := svc.Consume(user.ID, ReauthOperationBackup, ticket); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("second Consume() error = %v, want ErrReauthRequired", err)
		}
	})

	t.Run("ticket is operation-scoped", func(t *testing.T) {
		ticket := mint(ReauthOperationBackup)
		if err := svc.Consume(user.ID, ReauthOperationRestore, ticket); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("cross-operation Consume() error = %v, want ErrReauthRequired", err)
		}
		// The mismatched attempt must also have spent the ticket.
		if err := svc.Consume(user.ID, ReauthOperationBackup, ticket); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("post-mismatch Consume() error = %v, want ErrReauthRequired", err)
		}
	})

	t.Run("ticket is user-scoped", func(t *testing.T) {
		ticket := mint(ReauthOperationBackup)
		if err := svc.Consume(user.ID+1, ReauthOperationBackup, ticket); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("cross-user Consume() error = %v, want ErrReauthRequired", err)
		}
	})

	t.Run("empty and unknown tickets are rejected", func(t *testing.T) {
		if err := svc.Consume(user.ID, ReauthOperationBackup, ""); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("empty Consume() error = %v, want ErrReauthRequired", err)
		}
		if err := svc.Consume(user.ID, ReauthOperationBackup, "does-not-exist"); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("unknown Consume() error = %v, want ErrReauthRequired", err)
		}
	})
}

func TestReauthTicketExpiry(t *testing.T) {
	svc, user, password := newReauthTestService(t)

	clock := &mutableClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	restore := pkg.SetClockForTest(clock)
	defer restore()

	ticket, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, password, "")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}

	clock.advance(reauthTicketTTL + time.Second)

	if err := svc.Consume(user.ID, ReauthOperationBackup, ticket); !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("expired Consume() error = %v, want ErrReauthRequired", err)
	}
}

func TestReauthTicketLimitEvictsOldest(t *testing.T) {
	svc, user, _ := newReauthTestService(t)

	clock := &mutableClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	restore := pkg.SetClockForTest(clock)
	defer restore()

	tickets := make([]string, 0, maxReauthTickets+1)
	for i := 0; i < maxReauthTickets+1; i++ {
		ticket, err := svc.mintTicket(user.ID, ReauthOperationBackup)
		if err != nil {
			t.Fatalf("mintTicket() error = %v, want nil", err)
		}
		tickets = append(tickets, ticket)
		clock.advance(time.Millisecond)
	}

	if got := len(svc.tickets); got != maxReauthTickets {
		t.Fatalf("ticket count = %d, want %d", got, maxReauthTickets)
	}
	if err := svc.Consume(user.ID, ReauthOperationBackup, tickets[0]); !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("oldest Consume() error = %v, want ErrReauthRequired", err)
	}
	if err := svc.Consume(user.ID, ReauthOperationBackup, tickets[len(tickets)-1]); err != nil {
		t.Fatalf("newest Consume() error = %v, want nil", err)
	}
}

func TestReauthAvailableMethods(t *testing.T) {
	svc, user, _ := newReauthTestService(t)

	methods, err := svc.AvailableMethods(user.ID, ReauthOperationBackup)
	if err != nil {
		t.Fatalf("AvailableMethods() error = %v", err)
	}
	if !methods.Password {
		t.Fatal("AvailableMethods().Password = false, want true")
	}
	if methods.PasswordRequiresTOTP {
		t.Fatal("AvailableMethods().PasswordRequiresTOTP = true, want false")
	}
	if methods.Passkey {
		t.Fatal("AvailableMethods().Passkey = true, want false (no passkey registered)")
	}

	// Registering a passkey record flips the passkey availability flag.
	if err := svc.db.Create(&model.PasskeyCredential{
		UserID:       user.ID,
		Name:         "test",
		CredentialID: "cred-1",
		Credential:   []byte("{}"),
	}).Error; err != nil {
		t.Fatalf("failed to create passkey: %v", err)
	}

	methods, err = svc.AvailableMethods(user.ID, ReauthOperationBackup)
	if err != nil {
		t.Fatalf("AvailableMethods() error = %v", err)
	}
	if !methods.Passkey {
		t.Fatal("AvailableMethods().Passkey = false, want true")
	}

	const secret = "JBSWY3DPEHPK3PXP"
	enableReauthTestTOTP(t, svc, user.ID, secret)

	methods, err = svc.AvailableMethods(user.ID, ReauthOperationBackup)
	if err != nil {
		t.Fatalf("AvailableMethods() after totp enable error = %v", err)
	}
	if !methods.PasswordRequiresTOTP {
		t.Fatal("AvailableMethods().PasswordRequiresTOTP = false, want true")
	}
}

// seedOIDCEnabled writes the minimum settings that make getOIDCSettings() report
// enabled+configured, so OIDC step-up becomes available.
func seedOIDCEnabled(t *testing.T, svc *ReauthService) {
	t.Helper()
	settings := map[string]string{
		"oidc_enabled":       "true",
		"oidc_issuer_url":    "https://issuer.example.com",
		"oidc_client_id":     "client-id",
		"oidc_client_secret": "client-secret",
		"oidc_redirect_url":  "https://app.example.com/api/auth/oidc/callback",
	}
	for key, value := range settings {
		if err := svc.db.Create(&model.SystemSetting{Key: key, Value: value}).Error; err != nil {
			t.Fatalf("failed to seed setting %q: %v", key, err)
		}
	}
}

func TestReauthAvailableMethodsOIDC(t *testing.T) {
	svc, user, _ := newReauthTestService(t)
	if err := svc.db.AutoMigrate(&model.OIDCConnection{}); err != nil {
		t.Fatalf("failed to migrate oidc connection: %v", err)
	}

	// With OIDC disabled/unconfigured, the factor is not offered even if a
	// connection somehow exists.
	if err := svc.db.Create(&model.OIDCConnection{
		UserID: user.ID, Provider: oidcProviderKey, Subject: "sub-1", Email: user.Email,
	}).Error; err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}
	methods, err := svc.AvailableMethods(user.ID, ReauthOperationBackup)
	if err != nil {
		t.Fatalf("AvailableMethods() error = %v", err)
	}
	if methods.OIDC {
		t.Fatal("AvailableMethods().OIDC = true, want false (provider not enabled)")
	}

	// Enabling and configuring the provider flips it on for a linked user.
	seedOIDCEnabled(t, svc)
	methods, err = svc.AvailableMethods(user.ID, ReauthOperationBackup)
	if err != nil {
		t.Fatalf("AvailableMethods() error = %v", err)
	}
	if !methods.OIDC {
		t.Fatal("AvailableMethods().OIDC = false, want true")
	}

	// A user without a connection is not offered OIDC even when enabled.
	other := model.User{Username: "other", Email: "other@example.com", Password: "x", Role: "user", Status: "active"}
	if err := svc.db.Create(&other).Error; err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}
	methods, err = svc.AvailableMethods(other.ID, ReauthOperationBackup)
	if err != nil {
		t.Fatalf("AvailableMethods() error = %v", err)
	}
	if methods.OIDC {
		t.Fatal("AvailableMethods().OIDC = true for unlinked user, want false")
	}
}

func TestReauthVerifyOIDC(t *testing.T) {
	svc, user, _ := newReauthTestService(t)

	// Seed a valid reauth result session as the OIDC callback would, then verify.
	// A password-only account requires only OIDC-1, so a fresh-grade session is
	// sufficient here; grade enforcement is covered by TestReauthVerifyOIDCGrade.
	mintSession := func(userID uint, operation string) string {
		return svc.auth.storeOIDCResultSession(OIDCSessionResult{
			Purpose:   oidcPurposeReauth,
			UserID:    userID,
			Operation: operation,
			Grade:     OIDCGradeFresh,
		})
	}

	t.Run("valid session mints a usable, operation-scoped ticket", func(t *testing.T) {
		sessionID := mintSession(user.ID, ReauthOperationBackup)
		ticket, err := svc.VerifyOIDC(user.ID, ReauthOperationBackup, sessionID)
		if err != nil {
			t.Fatalf("VerifyOIDC() error = %v, want nil", err)
		}
		if ticket == "" {
			t.Fatal("VerifyOIDC() returned empty ticket")
		}
		if err := svc.Consume(user.ID, ReauthOperationBackup, ticket); err != nil {
			t.Fatalf("Consume() error = %v, want nil", err)
		}
	})

	t.Run("session is single-use", func(t *testing.T) {
		sessionID := mintSession(user.ID, ReauthOperationBackup)
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationBackup, sessionID); err != nil {
			t.Fatalf("first VerifyOIDC() error = %v, want nil", err)
		}
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationBackup, sessionID); err == nil {
			t.Fatal("second VerifyOIDC() error = nil, want non-nil (session spent)")
		}
	})

	t.Run("wrong user is rejected", func(t *testing.T) {
		sessionID := mintSession(user.ID, ReauthOperationBackup)
		if _, err := svc.VerifyOIDC(user.ID+1, ReauthOperationBackup, sessionID); err == nil {
			t.Fatal("cross-user VerifyOIDC() error = nil, want non-nil")
		}
	})

	t.Run("wrong operation is rejected", func(t *testing.T) {
		sessionID := mintSession(user.ID, ReauthOperationBackup)
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationRestore, sessionID); err == nil {
			t.Fatal("cross-operation VerifyOIDC() error = nil, want non-nil")
		}
	})

	t.Run("unknown session is rejected", func(t *testing.T) {
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationBackup, "does-not-exist"); err == nil {
			t.Fatal("unknown VerifyOIDC() error = nil, want non-nil")
		}
	})

	t.Run("unknown operation is rejected before touching the session", func(t *testing.T) {
		if _, err := svc.VerifyOIDC(user.ID, "wipe", "any"); !errors.Is(err, ErrInvalidReauthOperation) {
			t.Fatalf("VerifyOIDC() error = %v, want ErrInvalidReauthOperation", err)
		}
	})
}

func TestFinishOIDCReauthOwnership(t *testing.T) {
	svc, user, _ := newReauthTestService(t)
	if err := svc.db.AutoMigrate(&model.OIDCConnection{}); err != nil {
		t.Fatalf("failed to migrate oidc connection: %v", err)
	}
	auth := svc.auth
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	defer restoreClock()
	startedAt := now.Add(-30 * time.Second)

	// A connection linked to a DIFFERENT user, but the same OIDC subject the
	// callback resolves. The admin must not be able to step up with it.
	other := model.User{Username: "other", Email: "other@example.com", Password: "x", Role: "user", Status: "active"}
	if err := svc.db.Create(&other).Error; err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}
	if err := svc.db.Create(&model.OIDCConnection{
		UserID: other.ID, Provider: oidcProviderKey, Subject: "shared-subject", Email: other.Email,
	}).Error; err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}

	claims := &oidcIdentityClaims{Subject: "shared-subject", Email: other.Email, AuthTime: now.Unix()}
	if _, err := auth.finishOIDCReauth(user.ID, ReauthOperationBackup, claims, startedAt); err == nil {
		t.Fatal("finishOIDCReauth() error = nil for another user's identity, want non-nil")
	}

	// Linking the same subject to the requesting user makes step-up succeed.
	if err := svc.db.Create(&model.OIDCConnection{
		UserID: user.ID, Provider: oidcProviderKey, Subject: "own-subject", Email: user.Email,
	}).Error; err != nil {
		t.Fatalf("failed to create own connection: %v", err)
	}
	ownClaims := &oidcIdentityClaims{Subject: "own-subject", Email: user.Email, AuthTime: now.Unix()}
	result, err := auth.finishOIDCReauth(user.ID, ReauthOperationBackup, ownClaims, startedAt)
	if err != nil {
		t.Fatalf("finishOIDCReauth() error = %v, want nil", err)
	}
	if result.Purpose != oidcPurposeReauth || result.UserID != user.ID || result.Operation != ReauthOperationBackup {
		t.Fatalf("finishOIDCReauth() result = %+v, want reauth/%d/%s", result, user.ID, ReauthOperationBackup)
	}
}

func seedReauthTestPasskey(t *testing.T, svc *ReauthService, userID uint, credID string) {
	t.Helper()
	if err := svc.db.Create(&model.PasskeyCredential{
		UserID:       userID,
		Name:         "test",
		CredentialID: credID,
		Credential:   []byte("{}"),
	}).Error; err != nil {
		t.Fatalf("failed to create passkey: %v", err)
	}
}

// TestReauthPasswordPolicy exercises the reauth matrix's knowledge-factor rule:
// the password path is disabled exactly when a passkey is enrolled without TOTP,
// and returns as password+TOTP when TOTP is also enrolled.
func TestReauthPasswordPolicy(t *testing.T) {
	t.Run("passkey without totp disables the password path", func(t *testing.T) {
		svc, user, password := newReauthTestService(t)
		seedReauthTestPasskey(t, svc, user.ID, "cred-passkey-only")

		methods, err := svc.AvailableMethods(user.ID, ReauthOperationBackup)
		if err != nil {
			t.Fatalf("AvailableMethods() error = %v", err)
		}
		if methods.Password {
			t.Fatal("AvailableMethods().Password = true for passkey-only account, want false")
		}
		if !methods.Passkey {
			t.Fatal("AvailableMethods().Passkey = false, want true")
		}

		if _, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, password, ""); !errors.Is(err, ErrPasswordReauthDisabled) {
			t.Fatalf("VerifyPassword() error = %v, want ErrPasswordReauthDisabled", err)
		}

		methods, err = svc.AvailableMethods(user.ID, ReauthOperationDeletePasskey)
		if err != nil {
			t.Fatalf("AvailableMethods(delete_passkey) error = %v", err)
		}
		if methods.Password {
			t.Fatal("AvailableMethods(delete_passkey).Password = true for passkey-only account, want false")
		}
		if !methods.Passkey {
			t.Fatal("AvailableMethods(delete_passkey).Passkey = false, want true")
		}
		if _, err := svc.VerifyPassword(user.ID, ReauthOperationDeletePasskey, password, ""); !errors.Is(err, ErrPasswordReauthDisabled) {
			t.Fatalf("VerifyPassword(delete_passkey) error = %v, want ErrPasswordReauthDisabled", err)
		}
	})

	t.Run("passkey plus totp restores the password+totp path", func(t *testing.T) {
		svc, user, password := newReauthTestService(t)
		seedReauthTestPasskey(t, svc, user.ID, "cred-passkey-totp")
		const secret = "JBSWY3DPEHPK3PXP"
		enableReauthTestTOTP(t, svc, user.ID, secret)

		methods, err := svc.AvailableMethods(user.ID, ReauthOperationBackup)
		if err != nil {
			t.Fatalf("AvailableMethods() error = %v", err)
		}
		if !methods.Password {
			t.Fatal("AvailableMethods().Password = false for passkey+totp account, want true")
		}
		if !methods.PasswordRequiresTOTP {
			t.Fatal("AvailableMethods().PasswordRequiresTOTP = false, want true")
		}

		// Password alone (no code) is rejected; password+current code succeeds.
		if _, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, password, ""); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("VerifyPassword() without code error = %v, want ErrReauthRequired", err)
		}
		code, err := totp.GenerateCode(secret, time.Now().UTC())
		if err != nil {
			t.Fatalf("GenerateCode() error = %v", err)
		}
		if _, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, password, code); err != nil {
			t.Fatalf("VerifyPassword() with code error = %v, want nil", err)
		}
		if _, err := svc.VerifyPassword(user.ID, ReauthOperationDeletePasskey, password, code); err != nil {
			t.Fatalf("VerifyPassword(delete_passkey) with code error = %v, want nil", err)
		}
	})
}

// TestReauthVerifyOIDCGrade exercises the matrix's OIDC minimum-grade rule: a
// TOTP account needs OIDC-2, a passkey account needs OIDC-3, and a shortfall
// yields ErrOIDCReauthInsufficient without minting a ticket.
func TestReauthVerifyOIDCGrade(t *testing.T) {
	mintSession := func(svc *ReauthService, userID uint, operation string, grade OIDCReauthGrade) string {
		return svc.auth.storeOIDCResultSession(OIDCSessionResult{
			Purpose:   oidcPurposeReauth,
			UserID:    userID,
			Operation: operation,
			Grade:     grade,
		})
	}

	t.Run("totp account rejects OIDC-1 but accepts OIDC-2", func(t *testing.T) {
		svc, user, _ := newReauthTestService(t)
		enableReauthTestTOTP(t, svc, user.ID, "JBSWY3DPEHPK3PXP")

		low := mintSession(svc, user.ID, ReauthOperationBackup, OIDCGradeFresh)
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationBackup, low); !errors.Is(err, ErrOIDCReauthInsufficient) {
			t.Fatalf("VerifyOIDC() OIDC-1 error = %v, want ErrOIDCReauthInsufficient", err)
		}

		ok := mintSession(svc, user.ID, ReauthOperationBackup, OIDCGradeMFA)
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationBackup, ok); err != nil {
			t.Fatalf("VerifyOIDC() OIDC-2 error = %v, want nil", err)
		}
	})

	t.Run("passkey account requires OIDC-3", func(t *testing.T) {
		svc, user, _ := newReauthTestService(t)
		seedReauthTestPasskey(t, svc, user.ID, "cred-oidc-grade")

		mfa := mintSession(svc, user.ID, ReauthOperationBackup, OIDCGradeMFA)
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationBackup, mfa); !errors.Is(err, ErrOIDCReauthInsufficient) {
			t.Fatalf("VerifyOIDC() OIDC-2 error = %v, want ErrOIDCReauthInsufficient", err)
		}

		pr := mintSession(svc, user.ID, ReauthOperationBackup, OIDCGradePhishingResistant)
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationBackup, pr); err != nil {
			t.Fatalf("VerifyOIDC() OIDC-3 error = %v, want nil", err)
		}
	})

	t.Run("disable totp accepts OIDC-2 when passkey and totp are enrolled", func(t *testing.T) {
		svc, user, _ := newReauthTestService(t)
		seedReauthTestPasskey(t, svc, user.ID, "cred-disable-totp-oidc-grade")
		enableReauthTestTOTP(t, svc, user.ID, "JBSWY3DPEHPK3PXP")

		backupMFA := mintSession(svc, user.ID, ReauthOperationBackup, OIDCGradeMFA)
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationBackup, backupMFA); !errors.Is(err, ErrOIDCReauthInsufficient) {
			t.Fatalf("VerifyOIDC() backup OIDC-2 error = %v, want ErrOIDCReauthInsufficient", err)
		}

		disableMFA := mintSession(svc, user.ID, ReauthOperationDisableTOTP, OIDCGradeMFA)
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationDisableTOTP, disableMFA); err != nil {
			t.Fatalf("VerifyOIDC() disable_totp OIDC-2 error = %v, want nil", err)
		}
	})

	t.Run("delete passkey still requires OIDC-3 when passkey and totp are enrolled", func(t *testing.T) {
		svc, user, _ := newReauthTestService(t)
		seedReauthTestPasskey(t, svc, user.ID, "cred-delete-passkey-oidc-grade")
		enableReauthTestTOTP(t, svc, user.ID, "JBSWY3DPEHPK3PXP")

		mfa := mintSession(svc, user.ID, ReauthOperationDeletePasskey, OIDCGradeMFA)
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationDeletePasskey, mfa); !errors.Is(err, ErrOIDCReauthInsufficient) {
			t.Fatalf("VerifyOIDC(delete_passkey) OIDC-2 error = %v, want ErrOIDCReauthInsufficient", err)
		}

		pr := mintSession(svc, user.ID, ReauthOperationDeletePasskey, OIDCGradePhishingResistant)
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationDeletePasskey, pr); err != nil {
			t.Fatalf("VerifyOIDC(delete_passkey) OIDC-3 error = %v, want nil", err)
		}
	})

	t.Run("password-only account accepts OIDC-1", func(t *testing.T) {
		svc, user, _ := newReauthTestService(t)
		s := mintSession(svc, user.ID, ReauthOperationBackup, OIDCGradeFresh)
		if _, err := svc.VerifyOIDC(user.ID, ReauthOperationBackup, s); err != nil {
			t.Fatalf("VerifyOIDC() OIDC-1 error = %v, want nil", err)
		}
	})
}

// TestGradeOIDCReauth covers the acr/amr classification table.
func TestGradeOIDCReauth(t *testing.T) {
	mfaACR := []string{"http://schemas.openid.net/pape/policies/2007/06/multi-factor", "mfa"}
	prACR := []string{"phishing-resistant", "urn:acr:fido"}

	cases := []struct {
		name   string
		claims *oidcIdentityClaims
		want   OIDCReauthGrade
	}{
		{"no evidence stays fresh", &oidcIdentityClaims{}, OIDCGradeFresh},
		{"amr otp is mfa", &oidcIdentityClaims{AMR: []string{"pwd", "otp"}}, OIDCGradeMFA},
		{"amr mfa is mfa", &oidcIdentityClaims{AMR: []string{"mfa"}}, OIDCGradeMFA},
		{"amr fido is phishing-resistant", &oidcIdentityClaims{AMR: []string{"pwd", "fido"}}, OIDCGradePhishingResistant},
		{"amr hwk is phishing-resistant", &oidcIdentityClaims{AMR: []string{"hwk"}}, OIDCGradePhishingResistant},
		{"amr webauthn is phishing-resistant", &oidcIdentityClaims{AMR: []string{"webauthn"}}, OIDCGradePhishingResistant},
		{"configured mfa acr is mfa", &oidcIdentityClaims{ACR: "mfa"}, OIDCGradeMFA},
		{"configured pr acr is phishing-resistant", &oidcIdentityClaims{ACR: "phishing-resistant"}, OIDCGradePhishingResistant},
		{"pr amr beats mfa acr", &oidcIdentityClaims{ACR: "mfa", AMR: []string{"fido"}}, OIDCGradePhishingResistant},
		{"space separated acr matches", &oidcIdentityClaims{ACR: "level1 urn:acr:fido"}, OIDCGradePhishingResistant},
		{"case-insensitive amr", &oidcIdentityClaims{AMR: []string{"FIDO"}}, OIDCGradePhishingResistant},
		{"unrelated acr stays fresh", &oidcIdentityClaims{ACR: "level0"}, OIDCGradeFresh},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := gradeOIDCReauth(tc.claims, mfaACR, prACR); got != tc.want {
				t.Fatalf("gradeOIDCReauth() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestFinishOIDCReauthRequiresFreshLogin(t *testing.T) {
	svc, user, _ := newReauthTestService(t)
	if err := svc.db.AutoMigrate(&model.OIDCConnection{}); err != nil {
		t.Fatalf("failed to migrate oidc connection: %v", err)
	}

	if err := svc.db.Create(&model.OIDCConnection{
		UserID: user.ID, Provider: oidcProviderKey, Subject: "own-subject", Email: user.Email,
	}).Error; err != nil {
		t.Fatalf("failed to create own connection: %v", err)
	}

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	defer restoreClock()
	startedAt := now.Add(-30 * time.Second)

	t.Run("missing auth_time is rejected", func(t *testing.T) {
		claims := &oidcIdentityClaims{Subject: "own-subject", Email: user.Email}
		if _, err := svc.auth.finishOIDCReauth(user.ID, ReauthOperationBackup, claims, startedAt); err == nil {
			t.Fatal("finishOIDCReauth() error = nil, want missing auth_time rejection")
		}
	})

	t.Run("stale auth_time is rejected", func(t *testing.T) {
		claims := &oidcIdentityClaims{
			Subject:  "own-subject",
			Email:    user.Email,
			AuthTime: startedAt.Add(-oidcReauthAuthSkew - time.Second).Unix(),
		}
		if _, err := svc.auth.finishOIDCReauth(user.ID, ReauthOperationBackup, claims, startedAt); err == nil {
			t.Fatal("finishOIDCReauth() error = nil, want stale auth_time rejection")
		}
	})

	t.Run("fresh auth_time is accepted", func(t *testing.T) {
		claims := &oidcIdentityClaims{
			Subject:  "own-subject",
			Email:    user.Email,
			AuthTime: now.Unix(),
		}
		if _, err := svc.auth.finishOIDCReauth(user.ID, ReauthOperationBackup, claims, startedAt); err != nil {
			t.Fatalf("finishOIDCReauth() error = %v, want nil", err)
		}
	})
}
