package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicAuthStartEndpointsRateLimitedPerIP(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "passkey login start", path: "/api/auth/passkeys/login/start"},
		{name: "oidc login start", path: "/api/auth/oidc/login/start"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newHumanOnlyRouteTestDB(t)
			e := newHumanOnlyRouteTestServer(t, db)

			const limit = 30
			for attempt := 1; attempt <= limit; attempt++ {
				rec := postPublicAuthStart(t, e, tt.path)
				if rec.Code == http.StatusTooManyRequests {
					t.Fatalf("attempt %d rate limited too early: status = %d; body = %s", attempt, rec.Code, rec.Body.String())
				}
			}

			rec := postPublicAuthStart(t, e, tt.path)
			if rec.Code != http.StatusTooManyRequests {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusTooManyRequests, rec.Body.String())
			}
		})
	}
}

func postPublicAuthStart(t *testing.T, e http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, nil)
	req.Header.Set("X-Real-IP", "203.0.113.10")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}
