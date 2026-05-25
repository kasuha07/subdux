package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func TestUpdateSettingsEncryptsSystemProxyURLAndDoesNotExposeValue(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	adminService := NewAdminService(db)
	enabled := true
	proxyType := "socks5"
	proxyURL := "socks5://user:pass@proxy.example.com:1080"
	if err := adminService.UpdateSettings(UpdateSettingsInput{
		SystemProxyEnabled: &enabled,
		SystemProxyType:    &proxyType,
		SystemProxyURL:     &proxyURL,
	}); err != nil {
		t.Fatalf("UpdateSettings() failed: %v", err)
	}

	var stored model.SystemSetting
	if err := db.Where("key = ?", "system_proxy_url").First(&stored).Error; err != nil {
		t.Fatalf("failed to read stored proxy url: %v", err)
	}
	if stored.Value == proxyURL {
		t.Fatal("system proxy url should not be stored in plaintext")
	}
	if !strings.HasPrefix(stored.Value, "enc:v1:") {
		t.Fatalf("expected encrypted proxy url prefix, got %q", stored.Value)
	}

	settings, err := adminService.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() failed: %v", err)
	}
	if !settings.SystemProxyEnabled {
		t.Fatal("SystemProxyEnabled = false, want true")
	}
	if settings.SystemProxyType != "socks5" {
		t.Fatalf("SystemProxyType = %q, want socks5", settings.SystemProxyType)
	}
	if !settings.SystemProxyURLSet {
		t.Fatal("SystemProxyURLSet = false, want true")
	}
}

func TestUpdateSettingsRejectsEnabledSystemProxyWithoutURL(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	adminService := NewAdminService(db)
	enabled := true
	if err := adminService.UpdateSettings(UpdateSettingsInput{SystemProxyEnabled: &enabled}); !errors.Is(err, ErrInvalidSystemProxyURL) {
		t.Fatalf("UpdateSettings() error = %v, want ErrInvalidSystemProxyURL", err)
	}
}

func TestUpdateSettingsAllowsDisabledSystemProxyTypeChangeWithoutURL(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	adminService := NewAdminService(db)
	disabled := false
	proxyType := "socks5"
	if err := adminService.UpdateSettings(UpdateSettingsInput{
		SystemProxyEnabled: &disabled,
		SystemProxyType:    &proxyType,
	}); err != nil {
		t.Fatalf("UpdateSettings() error = %v, want nil", err)
	}
}

func TestNewOutboundHTTPClientUsesHTTPProxySetting(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedProxySettings(t, db, "true", "http", "http://proxy.example.com:8080")

	client := NewOutboundHTTPClient(db, 5*time.Second)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy == nil {
		t.Fatal("transport.Proxy is nil, want configured HTTP proxy")
	}

	req, err := http.NewRequest(http.MethodGet, "https://example.com/path", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() failed: %v", err)
	}
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy() failed: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://proxy.example.com:8080" {
		t.Fatalf("proxy url = %v, want http://proxy.example.com:8080", proxyURL)
	}
}

func TestNewOutboundHTTPClientUsesSOCKS5Dialer(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedProxySettings(t, db, "true", "socks5", "socks5://proxy.example.com:1080")

	client := NewOutboundHTTPClient(db, 5*time.Second)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("transport.Proxy is not nil for socks5 proxy")
	}
	if transport.DialContext == nil {
		t.Fatal("transport.DialContext is nil, want socks5 dialer")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := transport.DialContext(ctx, "tcp", "example.com:443"); !errors.Is(err, context.Canceled) {
		t.Fatalf("DialContext() error = %v, want context.Canceled", err)
	}
}

func TestOutboundDialContextUsesHTTPProxyConnect(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	defer listener.Close()

	requestCh := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		req, err := http.ReadRequest(reader)
		if err != nil {
			requestCh <- "read error: " + err.Error()
			return
		}
		requestCh <- fmt.Sprintf("%s %s", req.Method, req.Host)
		_, _ = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	}()

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedProxySettings(t, db, "true", "http", "http://"+listener.Addr().String())

	dialContext := NewOutboundDialContext(db, 5*time.Second)
	conn, err := dialContext(context.Background(), "tcp", "smtp.example.com:465")
	if err != nil {
		t.Fatalf("DialContext() failed: %v", err)
	}
	_ = conn.Close()

	select {
	case request := <-requestCh:
		if request != "CONNECT smtp.example.com:465" {
			t.Fatalf("proxy request = %q, want CONNECT smtp.example.com:465", request)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for proxy CONNECT request")
	}
}

func seedProxySettings(t *testing.T, db *gorm.DB, enabled string, proxyType string, proxyURL string) {
	t.Helper()

	encryptedProxyURL, err := encryptSystemSettingValueIfNeeded("system_proxy_url", proxyURL)
	if err != nil {
		t.Fatalf("failed to encrypt proxy url: %v", err)
	}

	entries := []model.SystemSetting{
		{Key: "system_proxy_enabled", Value: enabled},
		{Key: "system_proxy_type", Value: proxyType},
		{Key: "system_proxy_url", Value: encryptedProxyURL},
	}
	for _, entry := range entries {
		if err := db.Where("key = ?", entry.Key).
			Assign(model.SystemSetting{Value: entry.Value}).
			FirstOrCreate(&model.SystemSetting{Key: entry.Key}).Error; err != nil {
			t.Fatalf("failed to seed proxy setting %q: %v", entry.Key, err)
		}
	}
}
