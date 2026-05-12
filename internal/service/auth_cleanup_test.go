package service

import (
	"context"
	"testing"
	"time"
)

func TestStartSessionCleanupLoopRemovesExpiredSessions(t *testing.T) {
	authService := NewAuthService(nil)
	authService.sessionCleanupInterval = 5 * time.Millisecond

	now := time.Now().UTC()

	authService.passkeyMu.Lock()
	authService.passkeySessions["expired-passkey"] = passkeySession{ExpiresAt: now.Add(-time.Minute)}
	authService.passkeyMu.Unlock()

	authService.oidcMu.Lock()
	authService.oidcStateSessions["expired-state"] = oidcStateSession{ExpiresAt: now.Add(-time.Minute)}
	authService.oidcResultSessions["expired-result"] = oidcResultSession{ExpiresAt: now.Add(-time.Minute)}
	authService.oidcMu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	authService.StartSessionCleanupLoop(ctx)

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		authService.passkeyMu.Lock()
		passkeyCount := len(authService.passkeySessions)
		authService.passkeyMu.Unlock()

		authService.oidcMu.Lock()
		stateCount := len(authService.oidcStateSessions)
		resultCount := len(authService.oidcResultSessions)
		authService.oidcMu.Unlock()

		if passkeyCount == 0 && stateCount == 0 && resultCount == 0 {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	authService.passkeyMu.Lock()
	defer authService.passkeyMu.Unlock()
	authService.oidcMu.Lock()
	defer authService.oidcMu.Unlock()

	t.Fatalf(
		"expired sessions were not fully cleaned up: passkeys=%d oidc_state=%d oidc_result=%d",
		len(authService.passkeySessions),
		len(authService.oidcStateSessions),
		len(authService.oidcResultSessions),
	)
}
