package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
)

func TestPendingTOTPRejectedByAllProtectedGroups(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	e := newHumanOnlyRouteTestServer(t, db)
	user := createReauthTestAdmin(t, db)
	token, err := pkg.GenerateTOTPPendingToken(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/subscriptions", "/api/auth/passkeys", "/api/admin/users", "/api/reauth/password"} {
		method := http.MethodGet
		if path == "/api/reauth/password" {
			method = http.MethodPost
		}
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: got %d, want 401: %s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestExistingAccessTokenUsesCurrentAccountState(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	e := newHumanOnlyRouteTestServer(t, db)
	user := createReauthTestAdmin(t, db)
	token, err := pkg.GenerateAccessToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		t.Fatal(err)
	}
	check := func(path string, want int) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s: got %d, want %d: %s", path, rec.Code, want, rec.Body.String())
		}
	}
	check("/api/admin/users", 200)
	if err := db.Model(&user).Update("role", "user").Error; err != nil {
		t.Fatal(err)
	}
	check("/api/admin/users", 403)
	check("/api/subscriptions", 200)
	if err := db.Model(&user).Update("status", "disabled").Error; err != nil {
		t.Fatal(err)
	}
	check("/api/admin/users", 401)
	check("/api/auth/passkeys", 401)
	check("/api/subscriptions", 401)
	if err := db.Delete(&model.User{}, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	check("/api/subscriptions", 401)
}

func TestLoginPaddingDoesNotBypassAccountLimit(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	e := newHumanOnlyRouteTestServer(t, db)
	for attempt := 0; attempt < 15; attempt++ {
		body := `{"identifier":"missing","password":"wrong"}`
		if attempt >= 10 {
			body = `{"Identifier":"missing","password":"wrong","padding":"` + strings.Repeat("x", 9000) + `"}`
		}
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		// Vary the actual peer to isolate the account limiter from IP limiting.
		req.RemoteAddr = fmt.Sprintf("203.0.113.%d:1234", attempt+1)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", attempt+1))
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		want := 401
		if attempt >= 10 {
			want = 429
		}
		if rec.Code != want {
			t.Fatalf("attempt %d: got %d, want %d: %s", attempt, rec.Code, want, rec.Body.String())
		}
	}
}

func TestLoginForwardedHeadersDoNotBypassIPLimit(t *testing.T) {
	e := newHumanOnlyRouteTestServer(t, newHumanOnlyRouteTestDB(t))
	for attempt := 0; attempt < 35; attempt++ {
		body := fmt.Sprintf(`{"identifier":"missing-%d","password":"wrong"}`, attempt)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", attempt+1))
		req.Header.Set("X-Real-IP", fmt.Sprintf("198.51.100.%d", attempt+1))
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		want := 401
		if attempt >= 30 {
			want = 429
		}
		if rec.Code != want {
			t.Fatalf("attempt %d: got %d, want %d", attempt, rec.Code, want)
		}
	}
}

func TestOversizedLoginNeverReachesCredentials(t *testing.T) {
	e := newHumanOnlyRouteTestServer(t, newHumanOnlyRouteTestDB(t))
	for _, length := range []int64{-1, 1, 140000} {
		body := `{"identifier":"missing","password":"wrong","padding":"` + strings.Repeat("x", 140000) + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.ContentLength = length
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != 400 && rec.Code != 413 {
			t.Fatalf("content length %d: got %d", length, rec.Code)
		}
	}
}
