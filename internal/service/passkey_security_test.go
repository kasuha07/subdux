package service

import (
	"strings"
	"testing"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
)

func TestBuildWebAuthnRequiresConfiguredSiteURL(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate settings: %v", err)
	}

	authService := NewAuthService(db)
	_, err := authService.buildWebAuthn("https://evil.example.com", "evil.example.com", "https")
	if err == nil {
		t.Fatal("buildWebAuthn() error = nil, want site_url configuration error")
	}
	if !strings.Contains(err.Error(), "site_url must be configured") {
		t.Fatalf("buildWebAuthn() error = %q, want site_url configuration error", err.Error())
	}
}

func TestBuildWebAuthnUsesConfiguredSiteURLOnly(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate settings: %v", err)
	}
	seedSystemSetting(t, db, "site_url", "https://app.example.com")
	seedSystemSetting(t, db, "site_name", "Production Subdux")

	authService := NewAuthService(db)
	wa, err := authService.buildWebAuthn("https://evil.example.com", "evil.example.com", "https")
	if err != nil {
		t.Fatalf("buildWebAuthn() error = %v, want nil", err)
	}
	if got, want := wa.Config.RPID, "app.example.com"; got != want {
		t.Fatalf("RPID = %q, want %q", got, want)
	}
	if got, want := len(wa.Config.RPOrigins), 1; got != want {
		t.Fatalf("RPOrigins length = %d, want %d (%v)", got, want, wa.Config.RPOrigins)
	}
	if got, want := wa.Config.RPOrigins[0], "https://app.example.com"; got != want {
		t.Fatalf("RPOrigins[0] = %q, want %q", got, want)
	}
	if got, want := wa.Config.RPDisplayName, "Production Subdux"; got != want {
		t.Fatalf("RPDisplayName = %q, want %q", got, want)
	}
}

func TestBuildWebAuthnNormalizesBareSiteURL(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate settings: %v", err)
	}
	seedSystemSetting(t, db, "site_url", "app.example.com:8443")

	authService := NewAuthService(db)
	wa, err := authService.buildWebAuthn("", "", "")
	if err != nil {
		t.Fatalf("buildWebAuthn() error = %v, want nil", err)
	}
	if got, want := wa.Config.RPID, "app.example.com"; got != want {
		t.Fatalf("RPID = %q, want %q", got, want)
	}
	if got, want := wa.Config.RPOrigins[0], "https://app.example.com:8443"; got != want {
		t.Fatalf("RPOrigins[0] = %q, want %q", got, want)
	}
}

func TestBuildWebAuthnAllowsConfiguredLoopbackSiteURL(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate settings: %v", err)
	}
	seedSystemSetting(t, db, "site_url", "http://localhost:5173")

	authService := NewAuthService(db)
	wa, err := authService.buildWebAuthn("", "", "")
	if err != nil {
		t.Fatalf("buildWebAuthn() error = %v, want nil", err)
	}
	if got, want := wa.Config.RPID, "localhost"; got != want {
		t.Fatalf("RPID = %q, want %q", got, want)
	}
	if got, want := wa.Config.RPOrigins[0], "http://localhost:5173"; got != want {
		t.Fatalf("RPOrigins[0] = %q, want %q", got, want)
	}
}

func TestBuildWebAuthnRejectsPlainHTTPNonLoopbackSiteURL(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate settings: %v", err)
	}
	seedSystemSetting(t, db, "site_url", "http://app.example.com")

	authService := NewAuthService(db)
	_, err := authService.buildWebAuthn("", "", "")
	if err == nil {
		t.Fatal("buildWebAuthn() error = nil, want https requirement error")
	}
	if !strings.Contains(err.Error(), "must use https") {
		t.Fatalf("buildWebAuthn() error = %q, want https requirement error", err.Error())
	}
}

// TestFinishPasskeyReauthRejectsCrossOperation proves the reauth challenge is
// bound to the operation it was started for: a session issued for "backup"
// cannot be completed as "restore" to mint a restore-scoped ticket. The
// operation check runs before any WebAuthn validation, so seeding the session
// directly exercises the guard without a real signed assertion.
func TestFinishPasskeyReauthRejectsCrossOperation(t *testing.T) {
	authService := NewAuthService(nil)

	const userID = uint(7)
	sessionID := authService.storePasskeySession(passkeySession{
		Kind:      passkeySessionKindReauth,
		UserID:    userID,
		Operation: ReauthOperationBackup,
		ExpiresAt: pkg.NowUTC().Add(passkeySessionTTL),
	})

	err := authService.FinishPasskeyReauth(userID, ReauthOperationRestore, sessionID, nil, "", "", "")
	if err == nil {
		t.Fatal("FinishPasskeyReauth() error = nil for cross-operation finish, want non-nil")
	}
	if !strings.Contains(err.Error(), "invalid passkey session") {
		t.Fatalf("FinishPasskeyReauth() error = %q, want invalid passkey session", err.Error())
	}

	// The mismatched attempt must also have spent the single-use session, so a
	// correctly-scoped retry with the same session id fails too.
	if err := authService.FinishPasskeyReauth(userID, ReauthOperationBackup, sessionID, nil, "", "", ""); err == nil {
		t.Fatal("FinishPasskeyReauth() error = nil after session spent, want non-nil")
	}
}
