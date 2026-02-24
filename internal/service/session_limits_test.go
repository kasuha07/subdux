package service

import (
	"strconv"
	"testing"
)

func TestPasskeySessionStoreLimit(t *testing.T) {
	authService := NewAuthService(nil)

	for i := 0; i < maxPasskeySessions+50; i++ {
		authService.storePasskeySession(passkeySession{
			Kind: passkeySessionKindLogin,
		})
	}

	authService.passkeyMu.Lock()
	defer authService.passkeyMu.Unlock()

	if len(authService.passkeySessions) > maxPasskeySessions {
		t.Fatalf("passkey session count = %d, want <= %d", len(authService.passkeySessions), maxPasskeySessions)
	}
}

func TestOIDCSessionStoreLimit(t *testing.T) {
	authService := NewAuthService(nil)

	for i := 0; i < maxOIDCStateSessions+50; i++ {
		authService.storeOIDCStateSession(strconv.Itoa(i), oidcStateSession{})
	}
	for i := 0; i < maxOIDCResultSession+50; i++ {
		authService.storeOIDCResultSession(OIDCSessionResult{})
	}

	authService.oidcMu.Lock()
	defer authService.oidcMu.Unlock()

	if len(authService.oidcStateSessions) > maxOIDCStateSessions {
		t.Fatalf("oidc state session count = %d, want <= %d", len(authService.oidcStateSessions), maxOIDCStateSessions)
	}
	if len(authService.oidcResultSessions) > maxOIDCResultSession {
		t.Fatalf("oidc result session count = %d, want <= %d", len(authService.oidcResultSessions), maxOIDCResultSession)
	}
}
