package auth

import "testing"

func TestInvalidOIDCCallbacksDoNotAllocateOrEvictResultSessions(t *testing.T) {
	authService := NewService(newTestDB(t))

	legit := authService.storeOIDCResultSession(OIDCSessionResult{Purpose: oidcPurposeLogin, Token: "legit"})

	for i := 0; i < maxOIDCResultSession*2; i++ {
		state := ""
		if i%2 == 1 {
			state = "never-issued-state"
		}
		result, err := authService.HandleOIDCCallback(state, "", "code", "", "")
		if err != nil {
			t.Fatalf("HandleOIDCCallback() error = %v", err)
		}
		if result.SessionID != "" {
			t.Fatalf("invalid callback allocated result session %q", result.SessionID)
		}
		want := "missing_oidc_state"
		if state != "" {
			want = "invalid_or_expired_oidc_session"
		}
		if result.ErrorCode != want || result.Purpose != oidcPurposeLogin {
			t.Fatalf("result = %+v, want login purpose with %s", result, want)
		}
	}

	authService.oidcMu.Lock()
	pooled := len(authService.oidcResultSessions)
	authService.oidcMu.Unlock()
	if pooled != 1 {
		t.Fatalf("result sessions = %d, want only the legitimate one", pooled)
	}

	consumed, err := authService.ConsumeOIDCSessionResult(legit)
	if err != nil {
		t.Fatalf("legitimate result was evicted: %v", err)
	}
	if consumed.Token != "legit" {
		t.Fatalf("consumed = %+v, want legitimate result", consumed)
	}
}
