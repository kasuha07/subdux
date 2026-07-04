package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/labstack/echo/v4"
)

type codedSQLiteTestError struct {
	code int
	msg  string
}

func (e codedSQLiteTestError) Error() string { return e.msg }
func (e codedSQLiteTestError) Code() int     { return e.code }

func newAPIErrorContext(method, target string) (echo.Context, *httptest.ResponseRecorder, echo.HTTPErrorHandler) {
	e := echo.New()
	handler := APIErrorHandler(e.HTTPErrorHandler)
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec, handler
}

func TestAPIErrorHandlerMapsSQLiteBusyToServiceUnavailable(t *testing.T) {
	c, rec, handle := newAPIErrorContext(http.MethodPost, "/api/subscriptions")
	err := fmt.Errorf("write subscription: %w", codedSQLiteTestError{
		code: 5 | (3 << 8),
		msg:  "database is locked (773) (SQLITE_BUSY_TIMEOUT)",
	})

	handle(err, c)

	if got, want := rec.Code, http.StatusServiceUnavailable; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if got, want := rec.Header().Get("Retry-After"), "1"; got != want {
		t.Fatalf("Retry-After = %q, want %q", got, want)
	}
	if !strings.Contains(rec.Body.String(), "database is busy, retry later") {
		t.Fatalf("body = %q, want database busy message", rec.Body.String())
	}
}

func TestAPIErrorHandlerKeepsGenericErrorsInternal(t *testing.T) {
	c, rec, handle := newAPIErrorContext(http.MethodGet, "/api/site-info")

	handle(fmt.Errorf("unexpected storage failure"), c)

	if got, want := rec.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if strings.Contains(rec.Body.String(), "database is busy") {
		t.Fatalf("body = %q, should not expose database busy message", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "unexpected storage failure") {
		t.Fatalf("body = %q, should not expose internal error detail", rec.Body.String())
	}
}

func TestAPIErrorHandlerMapsTypedServiceError(t *testing.T) {
	c, rec, handle := newAPIErrorContext(http.MethodPost, "/api/items")

	handlerErr := fmt.Errorf("wrap: %w", serviceerr.New(serviceerr.KindConflict, "service validation failed"))
	handle(handlerErr, c)

	if got, want := rec.Code, http.StatusConflict; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if !strings.Contains(rec.Body.String(), "service validation failed") {
		t.Fatalf("body = %q, want service error message", rec.Body.String())
	}
}
