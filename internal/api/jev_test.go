package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/jev"
)

func TestJevSettingsHTTPContract(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createHumanOnlyRouteTestUser(t, db)
	token, err := pkg.GenerateAccessToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		t.Fatal(err)
	}
	e := newHumanOnlyRouteTestServer(t, db)
	request := func(method, path, body string, auth bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if auth {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	for _, path := range []string{"/api/jev/settings", "/api/jev/category-suggestion"} {
		method := http.MethodGet
		if strings.Contains(path, "suggestion") {
			method = http.MethodPost
		}
		if rec := request(method, path, `{}`, false); rec.Code != 401 {
			t.Fatalf("anonymous %s = %d", path, rec.Code)
		}
	}
	rec := request(http.MethodGet, "/api/jev/settings", "", true)
	if rec.Code != 200 {
		t.Fatalf("GET = %d", rec.Code)
	}
	var settings jev.Settings
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Enabled || settings.APIKeyConfigured {
		t.Fatal("unexpected default opt-in")
	}
	rec = request(http.MethodPut, "/api/jev/settings", `{"enabled":true,"api_key":"http-test-secret","revision":0}`, true)
	if rec.Code != 200 || strings.Contains(rec.Body.String(), "http-test-secret") || strings.Contains(rec.Body.String(), "enc:v1:") {
		t.Fatalf("unsafe settings response: %d", rec.Code)
	}
	rec = request(http.MethodPut, "/api/jev/settings", `{"enabled":false,"revision":0}`, true)
	if rec.Code != 409 {
		t.Fatalf("stale write = %d", rec.Code)
	}
	rec = request(http.MethodPost, "/api/jev/category-suggestion", `{"name":"Spotify","url":"https://spotify.com"}`, true)
	// No categories: no paid call and a stable nullable result.
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"category_id":null}` {
		t.Fatalf("empty taxonomy = %d %s", rec.Code, rec.Body.String())
	}
	rec = request(http.MethodPut, "/api/jev/settings", `{"enabled":true,"remove_api_key":true,"revision":1}`, true)
	if rec.Code != 200 {
		t.Fatalf("remove = %d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Enabled || settings.APIKeyConfigured {
		t.Fatal("removal did not disable")
	}
	for _, body := range []string{`{`, strings.Repeat("x", (8<<10)+1)} {
		rec = request(http.MethodPost, "/api/jev/category-suggestion", body, true)
		if rec.Code != 400 && rec.Code != 413 {
			t.Fatalf("invalid body = %d", rec.Code)
		}
	}
}
