package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
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

func TestReauthVerifyPassword(t *testing.T) {
	svc, user, password := newReauthTestService(t)

	t.Run("correct password mints a usable ticket", func(t *testing.T) {
		ticket, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, password)
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
		if _, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, "wrong"); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("VerifyPassword() error = %v, want ErrReauthRequired", err)
		}
	})

	t.Run("empty password is rejected", func(t *testing.T) {
		if _, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, ""); !errors.Is(err, ErrReauthRequired) {
			t.Fatalf("VerifyPassword() error = %v, want ErrReauthRequired", err)
		}
	})

	t.Run("unknown operation is rejected", func(t *testing.T) {
		if _, err := svc.VerifyPassword(user.ID, "wipe", password); !errors.Is(err, ErrInvalidReauthOperation) {
			t.Fatalf("VerifyPassword() error = %v, want ErrInvalidReauthOperation", err)
		}
	})
}

func TestReauthConsume(t *testing.T) {
	svc, user, password := newReauthTestService(t)

	mint := func(op string) string {
		t.Helper()
		ticket, err := svc.VerifyPassword(user.ID, op, password)
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

	ticket, err := svc.VerifyPassword(user.ID, ReauthOperationBackup, password)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}

	clock.advance(reauthTicketTTL + time.Second)

	if err := svc.Consume(user.ID, ReauthOperationBackup, ticket); !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("expired Consume() error = %v, want ErrReauthRequired", err)
	}
}

func TestReauthAvailableMethods(t *testing.T) {
	svc, user, _ := newReauthTestService(t)

	methods, err := svc.AvailableMethods(user.ID)
	if err != nil {
		t.Fatalf("AvailableMethods() error = %v", err)
	}
	if !methods.Password {
		t.Fatal("AvailableMethods().Password = false, want true")
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

	methods, err = svc.AvailableMethods(user.ID)
	if err != nil {
		t.Fatalf("AvailableMethods() error = %v", err)
	}
	if !methods.Passkey {
		t.Fatal("AvailableMethods().Passkey = false, want true")
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
	methods, err := svc.AvailableMethods(user.ID)
	if err != nil {
		t.Fatalf("AvailableMethods() error = %v", err)
	}
	if methods.OIDC {
		t.Fatal("AvailableMethods().OIDC = true, want false (provider not enabled)")
	}

	// Enabling and configuring the provider flips it on for a linked user.
	seedOIDCEnabled(t, svc)
	methods, err = svc.AvailableMethods(user.ID)
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
	methods, err = svc.AvailableMethods(other.ID)
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
	mintSession := func(userID uint, operation string) string {
		return svc.auth.storeOIDCResultSession(OIDCSessionResult{
			Purpose:   oidcPurposeReauth,
			UserID:    userID,
			Operation: operation,
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

	claims := &oidcIdentityClaims{Subject: "shared-subject", Email: other.Email}
	if _, err := auth.finishOIDCReauth(user.ID, ReauthOperationBackup, claims); err == nil {
		t.Fatal("finishOIDCReauth() error = nil for another user's identity, want non-nil")
	}

	// Linking the same subject to the requesting user makes step-up succeed.
	if err := svc.db.Create(&model.OIDCConnection{
		UserID: user.ID, Provider: oidcProviderKey, Subject: "own-subject", Email: user.Email,
	}).Error; err != nil {
		t.Fatalf("failed to create own connection: %v", err)
	}
	ownClaims := &oidcIdentityClaims{Subject: "own-subject", Email: user.Email}
	result, err := auth.finishOIDCReauth(user.ID, ReauthOperationBackup, ownClaims)
	if err != nil {
		t.Fatalf("finishOIDCReauth() error = %v, want nil", err)
	}
	if result.Purpose != oidcPurposeReauth || result.UserID != user.ID || result.Operation != ReauthOperationBackup {
		t.Fatalf("finishOIDCReauth() result = %+v, want reauth/%d/%s", result, user.ID, ReauthOperationBackup)
	}
}
