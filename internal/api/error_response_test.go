package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

type codedSQLiteTestError struct {
	code int
	msg  string
}

var errServiceValidation = errors.New("service validation failed")

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

func TestParseUintParamWritesBadRequest(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/items/not-a-number", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("not-a-number")

	if _, ok := parseUintParam(c, "id", "invalid item id"); ok {
		t.Fatal("parseUintParam() ok = true, want false")
	}

	if got, want := rec.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if !strings.Contains(rec.Body.String(), "invalid item id") {
		t.Fatalf("body = %q, want invalid item id", rec.Body.String())
	}
}

func TestBindOptionalJSONAllowsEmptyBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if ok := bindOptionalJSON(c, &input, "Invalid request body"); !ok {
		t.Fatal("bindOptionalJSON() ok = false, want true")
	}
}

func TestBindLimitedJSONMapsTooLargeRequest(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/import/subdux", strings.NewReader(`{"data":[]}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	req.Body = http.MaxBytesReader(rec, req.Body, 1)
	c := e.NewContext(req, rec)

	var input struct {
		Data []string `json:"data"`
	}
	if ok := bindLimitedJSON(c, &input, "import file is too large", "invalid JSON"); ok {
		t.Fatal("bindLimitedJSON() ok = true, want false")
	}

	if got, want := rec.Code, http.StatusRequestEntityTooLarge; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if !strings.Contains(rec.Body.String(), "import file is too large") {
		t.Fatalf("body = %q, want import size error", rec.Body.String())
	}
}

func TestWriteServiceErrorUsesConfiguredStatus(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/items", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := writeServiceError(c, fmt.Errorf("wrap: %w", errServiceValidation), serviceError(http.StatusConflict, errServiceValidation)); err != nil {
		t.Fatalf("writeServiceError() error = %v", err)
	}

	if got, want := rec.Code, http.StatusConflict; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if !strings.Contains(rec.Body.String(), errServiceValidation.Error()) {
		t.Fatalf("body = %q, want service error", rec.Body.String())
	}
}

func TestParseUintParamAcceptsUintMax(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/items/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.FormatUint(uint64(^uint(0)), 10))

	id, ok := parseUintParam(c, "id")
	if !ok {
		t.Fatal("parseUintParam() ok = false, want true")
	}
	if got, want := id, ^uint(0); got != want {
		t.Fatalf("id = %d, want %d", got, want)
	}
}
