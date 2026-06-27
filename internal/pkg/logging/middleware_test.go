package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// newTestEcho wires the RequestLogger middleware onto a fresh Echo instance and
// captures its JSON output in buf.
func newTestEcho(t *testing.T, buf *bytes.Buffer) *echo.Echo {
	t.Helper()
	Configure(Options{Level: slog.LevelDebug, Format: FormatJSON, Output: buf})

	e := echo.New()
	e.Use(RequestLogger())
	return e
}

func decodeRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &record); err != nil {
		t.Fatalf("log output is not valid JSON: %v\nraw: %s", err, buf.String())
	}
	return record
}

func TestRequestLoggerEmitsStructuredRecord(t *testing.T) {
	var buf bytes.Buffer
	e := newTestEcho(t, &buf)
	e.GET("/api/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ping?password=secret&page=2", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	record := decodeRecord(t, &buf)

	if record["msg"] != "http request" {
		t.Fatalf("msg = %v, want %q", record["msg"], "http request")
	}
	if record["level"] != "INFO" {
		t.Fatalf("level = %v, want INFO for 200", record["level"])
	}
	if record["method"] != http.MethodGet {
		t.Fatalf("method = %v, want GET", record["method"])
	}
	if record["status"].(float64) != float64(http.StatusOK) {
		t.Fatalf("status = %v, want 200", record["status"])
	}
	uri, _ := record["uri"].(string)
	if !bytes.Contains([]byte(uri), []byte("page=2")) {
		t.Fatalf("uri = %q, want it to contain page=2", uri)
	}
	if bytes.Contains([]byte(uri), []byte("secret")) {
		t.Fatalf("uri = %q leaked sensitive query value", uri)
	}
	if _, ok := record["request_id"]; !ok {
		t.Fatal("expected request_id on the record")
	}
}

func TestRequestLoggerGeneratesRequestIDHeader(t *testing.T) {
	var buf bytes.Buffer
	e := newTestEcho(t, &buf)
	e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get(HeaderRequestID); got == "" {
		t.Fatal("expected a generated X-Request-ID response header")
	}
}

func TestRequestLoggerReusesInboundRequestID(t *testing.T) {
	var buf bytes.Buffer
	e := newTestEcho(t, &buf)

	var seenInHandler string
	e.GET("/", func(c echo.Context) error {
		seenInHandler = RequestIDFromContext(c.Request().Context())
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderRequestID, "inbound-id-42")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get(HeaderRequestID); got != "inbound-id-42" {
		t.Fatalf("response X-Request-ID = %q, want inbound-id-42", got)
	}
	if seenInHandler != "inbound-id-42" {
		t.Fatalf("handler saw request id %q, want inbound-id-42", seenInHandler)
	}

	record := decodeRecord(t, &buf)
	if record["request_id"] != "inbound-id-42" {
		t.Fatalf("log request_id = %v, want inbound-id-42", record["request_id"])
	}
}

func TestRequestLoggerInjectsContextLogger(t *testing.T) {
	var buf bytes.Buffer
	e := newTestEcho(t, &buf)

	e.GET("/work", func(c echo.Context) error {
		// A handler logging through the context logger should inherit the
		// request_id automatically.
		FromContext(c.Request().Context()).Info("handler work done")
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/work", nil)
	req.Header.Set(HeaderRequestID, "ctx-id-7")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Two records: the handler's line and the middleware's summary. Both must
	// carry the request id.
	out := buf.String()
	if bytes.Count([]byte(out), []byte("ctx-id-7")) < 2 {
		t.Fatalf("expected request id on both handler and summary records, got: %s", out)
	}
}

func TestLevelForStatus(t *testing.T) {
	tests := []struct {
		status int
		want   slog.Level
	}{
		{status: http.StatusOK, want: slog.LevelInfo},
		{status: http.StatusFound, want: slog.LevelInfo},
		{status: http.StatusBadRequest, want: slog.LevelWarn},
		{status: http.StatusNotFound, want: slog.LevelWarn},
		{status: http.StatusInternalServerError, want: slog.LevelError},
		{status: http.StatusBadGateway, want: slog.LevelError},
	}

	for _, tt := range tests {
		if got := levelForStatus(tt.status); got != tt.want {
			t.Fatalf("levelForStatus(%d) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestRequestLoggerMapsErrorStatusToErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	e := newTestEcho(t, &buf)
	e.GET("/boom", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusInternalServerError, "kaboom")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	record := decodeRecord(t, &buf)
	if record["level"] != "ERROR" {
		t.Fatalf("level = %v, want ERROR for 500 response", record["level"])
	}
	if record["status"].(float64) != float64(http.StatusInternalServerError) {
		t.Fatalf("status = %v, want 500", record["status"])
	}
}
