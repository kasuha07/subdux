package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
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
