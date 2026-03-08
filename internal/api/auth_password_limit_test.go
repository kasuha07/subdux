package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/pkg"
)

func TestRegisterRejectsPasswordOver72Bytes(t *testing.T) {
	e := echo.New()
	body := `{"username":"alice","email":"alice@example.com","password":"` + strings.Repeat("a", 73) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := &AuthHandler{}
	if err := handler.Register(c); err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "72 bytes") {
		t.Fatalf("expected error message to mention 72 bytes, got %s", rec.Body.String())
	}
}

func TestRegisterRejectsPasswordUnder8Characters(t *testing.T) {
	e := echo.New()
	body := `{"username":"alice","email":"alice@example.com","password":"short7!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := &AuthHandler{}
	if err := handler.Register(c); err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "at least 8 characters") {
		t.Fatalf("expected error message to mention 8 characters, got %s", rec.Body.String())
	}
}

func TestResetPasswordRejectsPasswordUnder8Characters(t *testing.T) {
	e := echo.New()
	body := `{"email":"alice@example.com","verification_code":"123456","new_password":"short7!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/reset-password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := &AuthHandler{}
	if err := handler.ResetPassword(c); err != nil {
		t.Fatalf("ResetPassword() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "at least 8 characters") {
		t.Fatalf("expected error message to mention 8 characters, got %s", rec.Body.String())
	}
}

func TestChangePasswordRejectsPasswordUnder8Characters(t *testing.T) {
	e := echo.New()
	body := `{"current_password":"current-password","new_password":"short7!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", &jwt.Token{Claims: &pkg.JWTClaims{UserID: 1}})

	handler := &AuthHandler{}
	if err := handler.ChangePassword(c); err != nil {
		t.Fatalf("ChangePassword() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "at least 8 characters") {
		t.Fatalf("expected error message to mention 8 characters, got %s", rec.Body.String())
	}
}
