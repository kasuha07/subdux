package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	adminservice "github.com/kasuha07/subdux/internal/service/admin"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func newAdminSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-admin-settings-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings: %v", err)
	}
	return db
}

func TestUpdateSettingsRejectsInvalidEmailDomainWhitelist(t *testing.T) {
	e := echo.New()
	body := `{"email_domain_whitelist":"http://example.com"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: adminservice.NewService(db)}
	if err := handler.UpdateSettings(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateSettingsRejectsTooLongEmailDomainWhitelist(t *testing.T) {
	e := echo.New()
	domains := make([]string, 0, 9)
	for i := 0; i < 9; i++ {
		domains = append(domains, strings.Repeat("a", 59)+string(rune('a'+i))+".example.com")
	}
	body := `{"email_domain_whitelist":"` + strings.Join(domains, ";") + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: adminservice.NewService(db)}
	if err := handler.UpdateSettings(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateSettingsRejectsInvalidIconProxyDomainWhitelist(t *testing.T) {
	e := echo.New()
	body := `{"icon_proxy_domain_whitelist":"https://www.google.com"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: adminservice.NewService(db)}
	if err := handler.UpdateSettings(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateSettingsRejectsInvalidSSRFFilterMode(t *testing.T) {
	e := echo.New()
	body := `{"ssrf_domain_filter_mode":"deny"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: adminservice.NewService(db)}
	if err := handler.UpdateSettings(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateSettingsRejectsInvalidSSRFIPFilterList(t *testing.T) {
	e := echo.New()
	body := `{"ssrf_ip_filter_list":"10.0.0.0/99"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: adminservice.NewService(db)}
	if err := handler.UpdateSettings(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateSettingsRejectsInvalidSystemProxyURL(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "scheme mismatch",
			body: `{"system_proxy_type":"http","system_proxy_url":"socks5://proxy.example.com:1080"}`,
			want: "system proxy url must start with http://",
		},
		{
			name: "invalid port",
			body: `{"system_proxy_type":"http","system_proxy_url":"http://proxy.example.com:99999"}`,
			want: "system proxy url port is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			db := newAdminSettingsTestDB(t)
			handler := &AdminHandler{Service: adminservice.NewService(db)}
			if err := handler.UpdateSettings(c); err != nil {
				APIErrorHandler(e.HTTPErrorHandler)(err, c)
			}

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.want) {
				t.Fatalf("body = %s, want message containing %q", rec.Body.String(), tt.want)
			}
		})
	}
}

func TestAdminHandlerTestSSRFReturnsPolicyResult(t *testing.T) {
	e := echo.New()
	body := `{"target":"127.0.0.1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/settings/ssrf/test", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: adminservice.NewService(db)}
	if err := handler.TestSSRF(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var result adminservice.SSRFTestResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Allowed {
		t.Fatal("Allowed = true, want false for loopback target")
	}
	if result.Host != "127.0.0.1" {
		t.Fatalf("Host = %q, want 127.0.0.1", result.Host)
	}
	if !strings.Contains(result.Reason, "localhost or private network addresses") {
		t.Fatalf("Reason = %q, want restricted target reason", result.Reason)
	}
}

func TestAdminHandlerTestSSRFRejectsInvalidTarget(t *testing.T) {
	e := echo.New()
	body := `{"target":"http://exa mple.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/settings/ssrf/test", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: adminservice.NewService(db)}
	if err := handler.TestSSRF(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestAdminHandlerTestSMTPRejectsInvalidRuntimeConfig(t *testing.T) {
	e, c, rec := newAdminSMTPTestContext(`{"recipient_email":"admin@example.com"}`)

	db := newAdminSettingsTestDB(t)
	mustUpsertSystemSettings(t, db, map[string]string{
		"smtp_enabled": "true",
	})

	handler := &AdminHandler{Service: adminservice.NewService(db)}
	if err := handler.TestSMTP(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "smtp host is required") {
		t.Fatalf("body = %s, want smtp host error", rec.Body.String())
	}
}

func TestAdminHandlerTestSMTPReturnsDecryptErrorAsBadRequest(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "admin-smtp-test-settings-key")

	e, c, rec := newAdminSMTPTestContext(`{"recipient_email":"admin@example.com"}`)

	db := newAdminSettingsTestDB(t)
	mustUpsertSystemSettings(t, db, map[string]string{
		"smtp_enabled":    "true",
		"smtp_host":       "smtp.example.com",
		"smtp_port":       "587",
		"smtp_username":   "mailer",
		"smtp_password":   "enc:v1:not-valid-base64",
		"smtp_from_email": "noreply@example.com",
	})

	handler := &AdminHandler{Service: adminservice.NewService(db)}
	if err := handler.TestSMTP(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "failed to decrypt smtp settings") {
		t.Fatalf("body = %s, want decrypt error", rec.Body.String())
	}
}

func TestAdminHandlerTestSMTPReturnsSendErrorAsBadRequest(t *testing.T) {
	host, port, cleanup := startSMTPServerWithoutStartTLS(t)
	defer cleanup()

	e, c, rec := newAdminSMTPTestContext(`{"recipient_email":"admin@example.com"}`)

	db := newAdminSettingsTestDB(t)
	mustUpsertSystemSettings(t, db, map[string]string{
		"smtp_enabled":    "true",
		"smtp_host":       host,
		"smtp_port":       strconv.Itoa(port),
		"smtp_from_email": "noreply@example.com",
	})

	handler := &AdminHandler{Service: adminservice.NewService(db)}
	if err := handler.TestSMTP(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "smtp server does not support STARTTLS") {
		t.Fatalf("body = %s, want STARTTLS error", rec.Body.String())
	}
}

func TestAdminHandlerTestSMTPPreservesRateLimitStatus(t *testing.T) {
	e, c, rec := newAdminSMTPTestContext(`{"recipient_email":"admin@example.com"}`)

	db := newAdminSettingsTestDB(t)
	mustUpsertSystemSettings(t, db, map[string]string{
		"smtp_enabled":                    "true",
		"smtp_host":                       "smtp.example.com",
		"smtp_port":                       "587",
		"smtp_from_email":                 "noreply@example.com",
		"smtp_rate_limit_seconds":         "60",
		"smtp_rate_limit_last_attempt_at": pkg.NowUTC().Format(time.RFC3339Nano),
	})

	handler := &AdminHandler{Service: adminservice.NewService(db)}
	if err := handler.TestSMTP(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusTooManyRequests, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "smtp send rate limit exceeded") {
		t.Fatalf("body = %s, want rate limit error", rec.Body.String())
	}
}

func TestAdminSSRFTestRouteRequiresAdminRole(t *testing.T) {
	t.Setenv("JWT_SECRET", "admin-ssrf-test-jwt-secret-0123456789")
	db := newAdminSettingsTestDB(t)
	if err := pkg.InitJWTSecret(db); err != nil {
		t.Fatalf("failed to initialize jwt secret: %v", err)
	}

	e := echo.New()
	SetupRoutes(context.Background(), e, db, serviceutil.NewBackgroundTaskMonitor())

	token, err := pkg.GenerateAccessToken(1, "alice", "alice@example.com", "user")
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/settings/ssrf/test", strings.NewReader(`{"target":"127.0.0.1"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestUpdateSettingsRejectsTooLongIconProxyDomainWhitelist(t *testing.T) {
	e := echo.New()
	domains := make([]string, 0, 9)
	for i := 0; i < 9; i++ {
		domains = append(domains, strings.Repeat("a", 59)+string(rune('a'+i))+".example.com")
	}
	body := `{"icon_proxy_domain_whitelist":"` + strings.Join(domains, ";") + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: adminservice.NewService(db)}
	if err := handler.UpdateSettings(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func newAdminSMTPTestContext(body string) (*echo.Echo, echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/settings/smtp/test", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", &jwt.Token{
		Claims: &pkg.JWTClaims{
			UserID:   1,
			Role:     "admin",
			AuthType: pkg.AuthTypeUser,
		},
	})
	return e, c, rec
}

func mustUpsertSystemSettings(t *testing.T, db *gorm.DB, values map[string]string) {
	t.Helper()

	for key, value := range values {
		if err := db.Where("key = ?", key).
			Assign(model.SystemSetting{Value: value}).
			FirstOrCreate(&model.SystemSetting{Key: key}).Error; err != nil {
			t.Fatalf("failed to seed setting %q: %v", key, err)
		}
	}
}

func startSMTPServerWithoutStartTLS(t *testing.T) (string, int, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen for smtp test server: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		if _, err := rw.WriteString("220 localhost ESMTP test server\r\n"); err != nil {
			return
		}
		if err := rw.Flush(); err != nil {
			return
		}

		for {
			line, err := rw.ReadString('\n')
			if err != nil {
				return
			}

			command := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
				if _, err := rw.WriteString("250-localhost\r\n250 AUTH PLAIN LOGIN\r\n"); err != nil {
					return
				}
				if err := rw.Flush(); err != nil {
					return
				}
			case strings.HasPrefix(command, "QUIT"):
				_, _ = rw.WriteString("221 bye\r\n")
				_ = rw.Flush()
				return
			default:
				_, _ = rw.WriteString("250 ok\r\n")
				_ = rw.Flush()
			}
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	cleanup := func() {
		_ = ln.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out shutting down smtp test server on %s", fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port))
		}
	}

	return addr.IP.String(), addr.Port, cleanup
}
