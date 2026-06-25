package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

type codedSQLiteTestError struct {
	code int
	msg  string
}

func (e codedSQLiteTestError) Error() string {
	return e.msg
}

func (e codedSQLiteTestError) Code() int {
	return e.code
}

func TestWriteInternalServerErrorMapsSQLiteBusyToServiceUnavailable(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := fmt.Errorf("write subscription: %w", codedSQLiteTestError{
		code: sqliteBusyPrimaryCode | (3 << 8),
		msg:  "database is locked (773) (SQLITE_BUSY_TIMEOUT)",
	})

	if err := writeInternalServerError(c, err); err != nil {
		t.Fatalf("writeInternalServerError() error = %v", err)
	}

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

func TestWriteInternalServerErrorKeepsGenericErrorsInternal(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/site-info", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := writeInternalServerError(c, errors.New("unexpected storage failure")); err != nil {
		t.Fatalf("writeInternalServerError() error = %v", err)
	}

	if got, want := rec.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if strings.Contains(rec.Body.String(), "database is busy") {
		t.Fatalf("body = %q, should not expose database busy message", rec.Body.String())
	}
}
