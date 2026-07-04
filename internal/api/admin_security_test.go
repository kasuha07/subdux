package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCreateUserRejectsPasswordOver72Bytes(t *testing.T) {
	e := echo.New()
	body := `{"username":"alice","email":"alice@example.com","password":"` + strings.Repeat("a", 73) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := &AdminHandler{}
	if err := handler.CreateUser(c); err != nil {
		t.Fatalf("CreateUser() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "72 bytes") {
		t.Fatalf("expected error message to mention 72 bytes, got %s", rec.Body.String())
	}
}

func TestCreateUserRejectsPasswordUnder8Characters(t *testing.T) {
	e := echo.New()
	body := `{"username":"alice","email":"alice@example.com","password":"short7!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := &AdminHandler{}
	if err := handler.CreateUser(c); err != nil {
		t.Fatalf("CreateUser() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "at least 8 characters") {
		t.Fatalf("expected error message to mention 8 characters, got %s", rec.Body.String())
	}
}

func TestCreateUserDuplicateEmailAndUsernameReturnBadRequest(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	tests := []struct {
		name     string
		username string
		email    string
		wantBody string
	}{
		{
			name:     "duplicate email",
			username: "duplicate-email",
			email:    admin.Email,
			wantBody: "email already registered",
		},
		{
			name:     "duplicate username",
			username: admin.Username,
			email:    "duplicate-username@example.com",
			wantBody: "username already taken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postAdminUser(t, e, token, tt.username, tt.email, "user", "")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body = %s, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
