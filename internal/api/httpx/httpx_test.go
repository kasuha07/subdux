package httpx

import (
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

func (e codedSQLiteTestError) Error() string { return e.msg }
func (e codedSQLiteTestError) Code() int     { return e.code }

func TestIsTransientSQLiteBusyError(t *testing.T) {
	busy := fmt.Errorf("write subscription: %w", codedSQLiteTestError{
		code: sqliteBusyPrimaryCode | (3 << 8),
		msg:  "database is locked (773) (SQLITE_BUSY_TIMEOUT)",
	})
	if !IsTransientSQLiteBusyError(busy) {
		t.Fatal("expected coded SQLITE_BUSY error to be transient")
	}
	if IsTransientSQLiteBusyError(fmt.Errorf("unexpected storage failure")) {
		t.Fatal("generic error should not be treated as transient busy")
	}
	if IsTransientSQLiteBusyError(nil) {
		t.Fatal("nil error should not be treated as transient busy")
	}
}

func TestIsRequestTooLargeError(t *testing.T) {
	if !IsRequestTooLargeError(&http.MaxBytesError{Limit: 1}) {
		t.Fatal("expected MaxBytesError to be detected")
	}
	if IsRequestTooLargeError(nil) {
		t.Fatal("nil error should not be request-too-large")
	}
}

func TestParseUintParamWritesBadRequest(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/items/not-a-number", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("not-a-number")

	if _, ok := ParseUintParam(c, "id", "invalid item id"); ok {
		t.Fatal("ParseUintParam() ok = true, want false")
	}
	if got, want := rec.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if !strings.Contains(rec.Body.String(), "invalid item id") {
		t.Fatalf("body = %q, want invalid item id", rec.Body.String())
	}
}

func TestParseUintParamAcceptsUintMax(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/items/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.FormatUint(uint64(^uint(0)), 10))

	id, ok := ParseUintParam(c, "id")
	if !ok {
		t.Fatal("ParseUintParam() ok = false, want true")
	}
	if got, want := id, ^uint(0); got != want {
		t.Fatalf("id = %d, want %d", got, want)
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
	if ok := BindOptionalJSON(c, &input, "Invalid request body"); !ok {
		t.Fatal("BindOptionalJSON() ok = false, want true")
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
	if ok := BindLimitedJSON(c, &input, "import file is too large", "invalid JSON"); ok {
		t.Fatal("BindLimitedJSON() ok = true, want false")
	}
	if got, want := rec.Code, http.StatusRequestEntityTooLarge; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if !strings.Contains(rec.Body.String(), "import file is too large") {
		t.Fatalf("body = %q, want import size error", rec.Body.String())
	}
}
