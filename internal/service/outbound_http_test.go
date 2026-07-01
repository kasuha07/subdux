package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
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

func TestSafeOutboundHTTPClientPreservesHTTPProxyForPrivateDNSResults(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	requestCh := make(chan string, 1)
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCh <- req.Host
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer proxyServer.Close()

	seedProxySettings(t, db, "true", "http", proxyServer.URL)

	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		t.Fatalf("lookup should not run for proxied request, got host %q", host)
		return nil, errors.New("unexpected lookup")
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	client := NewSafeOutboundHTTPClient(db, 5*time.Second)
	req, err := http.NewRequest(http.MethodGet, "http://internal.example.com/status", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	resp, err := doNotificationRequest(client, req, nil)
	if err != nil {
		t.Fatalf("doNotificationRequest() error = %v, want nil through proxy", err)
	}
	defer resp.Body.Close()

	select {
	case host := <-requestCh:
		if host != "internal.example.com" {
			t.Fatalf("proxy request host = %q, want internal.example.com", host)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for proxied request")
	}
}

func TestSafeOutboundHTTPClientProxyRoundTripperAppliesSSRFDomainFilter(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	requestCh := make(chan string, 1)
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCh <- req.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()

	seedProxySettings(t, db, "true", "http", proxyServer.URL)
	seedSSRFSettings(t, db, map[string]string{
		ssrfDomainFilterModeKey: "whitelist",
		ssrfDomainFilterListKey: "allowed.example.com",
	})

	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		t.Fatalf("lookup should not run for proxied request, got host %q", host)
		return nil, errors.New("unexpected lookup")
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	client := NewSafeOutboundHTTPClient(db, 5*time.Second)
	req, err := http.NewRequest(http.MethodGet, "http://blocked.example.com/status", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err == nil {
		t.Fatal("client.Do() error = nil, want SSRF domain whitelist rejection")
	}
	if !strings.Contains(err.Error(), "not allowed by ssrf domain whitelist") {
		t.Fatalf("client.Do() error = %q, want SSRF domain whitelist rejection", err.Error())
	}

	select {
	case host := <-requestCh:
		t.Fatalf("proxy received request for %q, want RoundTripper to block before proxy", host)
	default:
	}
}

func TestSafeOutboundDialContextPreservesHTTPProxyConnect(t *testing.T) {
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

	dialContext := NewSafeOutboundDialContext(db, 5*time.Second)
	conn, err := dialContext(context.Background(), "tcp", "smtp.internal.example.com:465")
	if err != nil {
		t.Fatalf("DialContext() failed: %v", err)
	}
	_ = conn.Close()

	select {
	case request := <-requestCh:
		if request != "CONNECT smtp.internal.example.com:465" {
			t.Fatalf("proxy request = %q, want CONNECT smtp.internal.example.com:465", request)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for proxy CONNECT request")
	}
}

func TestSafeOutboundHTTPClientDialsValidatedResolvedIP(t *testing.T) {
	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		if host != "example.com" {
			t.Fatalf("lookup host = %q, want example.com", host)
		}
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	client := NewSafeOutboundHTTPClient(nil, 5*time.Second)
	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err == nil {
		t.Fatal("client.Do() error = nil, want private address validation error")
	}
	if !strings.Contains(err.Error(), "resolves to localhost or private network addresses") {
		t.Fatalf("client.Do() error = %q, want private address validation error", err.Error())
	}
}

func TestSafeOutboundHTTPClientPinsValidatedIPAgainstDNSRebinding(t *testing.T) {
	requestCh := make(chan string, 1)
	dialAddressCh := make(chan string, 1)

	originalLookup := lookupOutboundHostIPs
	lookupCount := 0
	lookupOutboundHostIPs = func(_ context.Context, _ string, lookupHost string) ([]net.IP, error) {
		if lookupHost != "example.com" {
			t.Fatalf("lookup host = %q, want example.com", lookupHost)
		}
		lookupCount++
		if lookupCount > 1 {
			return []net.IP{net.ParseIP("127.0.0.1")}, nil
		}
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	safeDialer := newSafeOutboundDialer(5 * time.Second)
	safeDialer.dialContext = func(_ context.Context, network string, address string) (net.Conn, error) {
		if network != "tcp" {
			return nil, fmt.Errorf("network = %q, want tcp", network)
		}
		dialAddressCh <- address

		clientConn, serverConn := net.Pipe()
		go func() {
			defer serverConn.Close()
			reader := bufio.NewReader(serverConn)
			req, err := http.ReadRequest(reader)
			if err != nil {
				requestCh <- "read error: " + err.Error()
				return
			}
			requestCh <- req.Host
			_, _ = io.WriteString(serverConn, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok")
		}()
		return clientConn, nil
	}

	client := newOutboundHTTPClient(outboundHTTPClientOptions{
		Timeout:      5 * time.Second,
		SecureEgress: true,
		SecureDialer: safeDialer,
	})
	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/path", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do() error = %v, want nil", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	select {
	case dialAddress := <-dialAddressCh:
		if dialAddress != "93.184.216.34:8080" {
			t.Fatalf("dial address = %q, want 93.184.216.34:8080", dialAddress)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pinned dial address")
	}

	select {
	case hostHeader := <-requestCh:
		if hostHeader != "example.com:8080" {
			t.Fatalf("Host header = %q, want example.com:8080", hostHeader)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pinned outbound request")
	}
	if lookupCount != 1 {
		t.Fatalf("lookup count = %d, want 1", lookupCount)
	}
}

func TestDoNotificationRequestNilClientUsesSafeOutboundClient(t *testing.T) {
	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		if host != "example.com" {
			t.Fatalf("lookup host = %q, want example.com", host)
		}
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	resp, err := doNotificationRequest(nil, req, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err == nil {
		t.Fatal("doNotificationRequest() error = nil, want private address validation error")
	}
	if !strings.Contains(err.Error(), "resolves to localhost or private network addresses") {
		t.Fatalf("doNotificationRequest() error = %q, want private address validation error", err.Error())
	}
}

func TestSSRFSettingsDefaultProtection(t *testing.T) {
	settings := defaultAdminSystemSettings()
	if !settings.SSRFProtectionEnabled {
		t.Fatal("SSRFProtectionEnabled = false, want true")
	}
	if settings.SSRFAllowPrivateIP {
		t.Fatal("SSRFAllowPrivateIP = true, want false")
	}
	if settings.SSRFDomainFilterMode != "blacklist" {
		t.Fatalf("SSRFDomainFilterMode = %q, want blacklist", settings.SSRFDomainFilterMode)
	}
	if settings.SSRFIPFilterMode != "blacklist" {
		t.Fatalf("SSRFIPFilterMode = %q, want blacklist", settings.SSRFIPFilterMode)
	}
	if !settings.SSRFFilterResolvedIPs {
		t.Fatal("SSRFFilterResolvedIPs = false, want true")
	}
}

func TestUpdateSettingsNormalizesSSRFProtectionLists(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	disabled := false
	allowPrivate := true
	domainMode := "whitelist"
	domainList := " api.example.com.;example.com,api.example.com "
	ipMode := "blacklist"
	ipList := "10.0.0.2;10.0.0.0/8,10.0.0.2"
	filterResolved := false

	adminService := NewAdminService(db)
	if err := adminService.UpdateSettings(UpdateSettingsInput{
		SSRFProtectionEnabled: &disabled,
		SSRFAllowPrivateIP:    &allowPrivate,
		SSRFDomainFilterMode:  &domainMode,
		SSRFDomainFilterList:  &domainList,
		SSRFIPFilterMode:      &ipMode,
		SSRFIPFilterList:      &ipList,
		SSRFFilterResolvedIPs: &filterResolved,
	}); err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	settings, err := adminService.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if settings.SSRFProtectionEnabled {
		t.Fatal("SSRFProtectionEnabled = true, want false")
	}
	if !settings.SSRFAllowPrivateIP {
		t.Fatal("SSRFAllowPrivateIP = false, want true")
	}
	if settings.SSRFDomainFilterMode != "whitelist" {
		t.Fatalf("SSRFDomainFilterMode = %q, want whitelist", settings.SSRFDomainFilterMode)
	}
	if settings.SSRFDomainFilterList != "api.example.com\nexample.com" {
		t.Fatalf("SSRFDomainFilterList = %q, want normalized list", settings.SSRFDomainFilterList)
	}
	if settings.SSRFIPFilterMode != "blacklist" {
		t.Fatalf("SSRFIPFilterMode = %q, want blacklist", settings.SSRFIPFilterMode)
	}
	if settings.SSRFIPFilterList != "10.0.0.0/8\n10.0.0.2" {
		t.Fatalf("SSRFIPFilterList = %q, want normalized list", settings.SSRFIPFilterList)
	}
	if settings.SSRFFilterResolvedIPs {
		t.Fatal("SSRFFilterResolvedIPs = true, want false")
	}
}

func TestSSRFDomainWhitelistRejectsUnlistedHost(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedSSRFSettings(t, db, map[string]string{
		ssrfDomainFilterModeKey: "whitelist",
		ssrfDomainFilterListKey: "allowed.example.com",
	})

	if err := validateOutboundHost("api.allowed.example.com", "webhook url", db); err != nil {
		t.Fatalf("validateOutboundHost() allowed error = %v, want nil", err)
	}
	err := validateOutboundHost("blocked.example.com", "webhook url", db)
	if err == nil {
		t.Fatal("validateOutboundHost() error = nil, want whitelist rejection")
	}
	if !strings.Contains(err.Error(), "not allowed by ssrf domain whitelist") {
		t.Fatalf("validateOutboundHost() error = %q, want domain whitelist rejection", err.Error())
	}
}

func TestSSRFDomainWhitelistRejectsBareIPWithoutIPWhitelist(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedSSRFSettings(t, db, map[string]string{
		ssrfDomainFilterModeKey: "whitelist",
		ssrfDomainFilterListKey: "allowed.example.com",
	})

	err := validateOutboundHost("93.184.216.34", "webhook url", db)
	if err == nil {
		t.Fatal("validateOutboundHost() error = nil, want bare IP rejected by domain whitelist")
	}
	if !strings.Contains(err.Error(), "not allowed by ssrf domain whitelist") {
		t.Fatalf("validateOutboundHost() error = %q, want domain whitelist rejection", err.Error())
	}
}

func TestSSRFDomainWhitelistAllowsBareIPWithExplicitIPWhitelist(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedSSRFSettings(t, db, map[string]string{
		ssrfDomainFilterModeKey: "whitelist",
		ssrfDomainFilterListKey: "allowed.example.com",
		ssrfIPFilterModeKey:     "whitelist",
		ssrfIPFilterListKey:     "93.184.216.0/24",
	})

	if err := validateOutboundHost("93.184.216.34", "webhook url", db); err != nil {
		t.Fatalf("validateOutboundHost() error = %v, want explicit IP whitelist to allow bare IP", err)
	}
}

func TestSSRFIPBlacklistRejectsResolvedAddress(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedSSRFSettings(t, db, map[string]string{
		ssrfIPFilterModeKey: "blacklist",
		ssrfIPFilterListKey: "93.184.216.0/24",
	})

	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		if host != "example.com" {
			t.Fatalf("lookup host = %q, want example.com", host)
		}
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	err := validateResolvedOutboundHost("example.com", db)
	if err == nil {
		t.Fatal("validateResolvedOutboundHost() error = nil, want IP blacklist rejection")
	}
	if !strings.Contains(err.Error(), "blocked by ssrf ip blacklist") {
		t.Fatalf("validateResolvedOutboundHost() error = %q, want IP blacklist rejection", err.Error())
	}
}

func TestSSRFAllowPrivateIPAllowsRFCPrivateResolvedAddress(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedSSRFSettings(t, db, map[string]string{
		ssrfAllowPrivateIPKey: "true",
	})

	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		if host != "intranet.example.com" {
			t.Fatalf("lookup host = %q, want intranet.example.com", host)
		}
		return []net.IP{net.ParseIP("10.0.0.5")}, nil
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	if err := validateResolvedOutboundHost("intranet.example.com", db); err != nil {
		t.Fatalf("validateResolvedOutboundHost() error = %v, want nil", err)
	}
}

func TestSSRFFilterResolvedIPsCanBeDisabled(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedSSRFSettings(t, db, map[string]string{
		ssrfFilterResolvedIPsKey: "false",
	})

	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		if host != "example.com" {
			t.Fatalf("lookup host = %q, want example.com", host)
		}
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	if err := validateResolvedOutboundHost("example.com", db); err != nil {
		t.Fatalf("validateResolvedOutboundHost() error = %v, want nil when resolved-IP filtering is disabled", err)
	}
}

func TestSSRFProtectionCanBeDisabled(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedSSRFSettings(t, db, map[string]string{
		ssrfProtectionEnabledKey: "false",
	})

	if err := validateOutboundHost("127.0.0.1", "webhook url", db); err != nil {
		t.Fatalf("validateOutboundHost() error = %v, want nil when SSRF protection is disabled", err)
	}
}

func TestAdminServiceTestSSRFAllowsPublicResolvedAddress(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedSSRFSettings(t, db, nil)

	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		if host != "example.com" {
			t.Fatalf("lookup host = %q, want example.com", host)
		}
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	result, err := NewAdminService(db).TestSSRF(SSRFTestInput{Target: "https://Example.com/status"})
	if err != nil {
		t.Fatalf("TestSSRF() error = %v, want nil", err)
	}
	if !result.Allowed {
		t.Fatalf("Allowed = false, want true; reason = %q", result.Reason)
	}
	if result.Host != "example.com" {
		t.Fatalf("Host = %q, want example.com", result.Host)
	}
	if len(result.ResolvedIPs) != 1 || result.ResolvedIPs[0] != "93.184.216.34" {
		t.Fatalf("ResolvedIPs = %#v, want 93.184.216.34", result.ResolvedIPs)
	}
	if !result.ResolvedIPFilterApplied {
		t.Fatal("ResolvedIPFilterApplied = false, want true")
	}
}

func TestAdminServiceTestSSRFReportsBlockedResolvedAddress(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedSSRFSettings(t, db, nil)

	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		if host != "internal.example.com" {
			t.Fatalf("lookup host = %q, want internal.example.com", host)
		}
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	result, err := NewAdminService(db).TestSSRF(SSRFTestInput{Target: "internal.example.com"})
	if err != nil {
		t.Fatalf("TestSSRF() error = %v, want nil", err)
	}
	if result.Allowed {
		t.Fatal("Allowed = true, want false")
	}
	if !strings.Contains(result.Reason, "resolves to localhost or private network addresses") {
		t.Fatalf("Reason = %q, want resolved private-network rejection", result.Reason)
	}
}

func TestAdminServiceTestSSRFProxyModeSkipsLocalDNS(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedSSRFSettings(t, db, nil)
	seedProxySettings(t, db, "true", "http", "http://127.0.0.1:8080")

	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		t.Fatalf("lookup should not run for proxy-mediated SSRF test, got host %q", host)
		return nil, errors.New("unexpected lookup")
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	result, err := NewAdminService(db).TestSSRF(SSRFTestInput{Target: "intranet.example.com"})
	if err != nil {
		t.Fatalf("TestSSRF() error = %v, want nil", err)
	}
	if !result.Allowed {
		t.Fatalf("Allowed = false, want true; reason = %q", result.Reason)
	}
	if !result.ProxyMediated {
		t.Fatal("ProxyMediated = false, want true")
	}
	if result.ResolvedIPFilterApplied {
		t.Fatal("ResolvedIPFilterApplied = true, want false for proxy-mediated test")
	}
	if len(result.ResolvedIPs) != 0 {
		t.Fatalf("ResolvedIPs = %#v, want empty in proxy mode", result.ResolvedIPs)
	}
}

func TestAdminServiceTestSSRFRejectsInvalidTarget(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	_, err := NewAdminService(db).TestSSRF(SSRFTestInput{Target: "ftp://example.com"})
	if !errors.Is(err, ErrInvalidSSRFTestTarget) {
		t.Fatalf("TestSSRF() error = %v, want ErrInvalidSSRFTestTarget", err)
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

func seedSSRFSettings(t *testing.T, db *gorm.DB, overrides map[string]string) {
	t.Helper()

	values := map[string]string{
		ssrfProtectionEnabledKey: "true",
		ssrfAllowPrivateIPKey:    "false",
		ssrfDomainFilterModeKey:  "blacklist",
		ssrfDomainFilterListKey:  "",
		ssrfIPFilterModeKey:      "blacklist",
		ssrfIPFilterListKey:      "",
		ssrfFilterResolvedIPsKey: "true",
	}
	for key, value := range overrides {
		values[key] = value
	}
	for key, value := range values {
		if err := db.Where("key = ?", key).
			Assign(model.SystemSetting{Value: value}).
			FirstOrCreate(&model.SystemSetting{Key: key}).Error; err != nil {
			t.Fatalf("failed to seed ssrf setting %q: %v", key, err)
		}
	}
}
