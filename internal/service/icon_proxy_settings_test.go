package service

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/shiroha/subdux/internal/model"
)

func TestUpdateSettingsIconProxyDomainWhitelistNormalization(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	svc := NewAdminService(db)
	input := " icon.horse ;WWW.Google.com\ngoogle.com "
	if err := svc.UpdateSettings(UpdateSettingsInput{
		IconProxyDomainWhitelist: &input,
	}); err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	settings, err := svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}

	want := "google.com\nicon.horse\nwww.google.com"
	if settings.IconProxyDomainWhitelist != want {
		t.Fatalf("IconProxyDomainWhitelist = %q, want %q", settings.IconProxyDomainWhitelist, want)
	}
}

func TestUpdateSettingsIconProxyDomainWhitelistValidationError(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	svc := NewAdminService(db)
	input := "https://www.google.com"
	err := svc.UpdateSettings(UpdateSettingsInput{
		IconProxyDomainWhitelist: &input,
	})
	if !errors.Is(err, ErrInvalidIconProxyDomainWhitelist) {
		t.Fatalf("UpdateSettings() error = %v, want %v", err, ErrInvalidIconProxyDomainWhitelist)
	}
}

func TestIconProxyServiceResolveUsesRedirectWhenDisabled(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "icon_proxy_enabled", Value: "false"}).Error; err != nil {
		t.Fatalf("failed to seed icon_proxy_enabled: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "icon_proxy_domain_whitelist", Value: defaultIconProxyDomainWhitelist}).Error; err != nil {
		t.Fatalf("failed to seed icon_proxy_domain_whitelist: %v", err)
	}

	svc := NewIconProxyService(db)
	resolution, err := svc.Resolve("google", "example.com")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if resolution.Proxy {
		t.Fatal("Resolve() should disable backend proxy mode when setting is false")
	}
	if got, want := resolution.UpstreamHost, "www.google.com"; got != want {
		t.Fatalf("UpstreamHost = %q, want %q", got, want)
	}
	if got, want := resolution.UpstreamURL, "https://www.google.com/s2/favicons?domain=example.com&sz=64"; got != want {
		t.Fatalf("UpstreamURL = %q, want %q", got, want)
	}
}

func TestIconProxyServiceResolveRejectsDisallowedUpstreamHost(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "icon_proxy_enabled", Value: "true"}).Error; err != nil {
		t.Fatalf("failed to seed icon_proxy_enabled: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "icon_proxy_domain_whitelist", Value: "icon.horse"}).Error; err != nil {
		t.Fatalf("failed to seed icon_proxy_domain_whitelist: %v", err)
	}

	svc := NewIconProxyService(db)
	_, err := svc.Resolve("google", "example.com")
	if !errors.Is(err, ErrIconProxyDomainNotAllowed) {
		t.Fatalf("Resolve() error = %v, want %v", err, ErrIconProxyDomainNotAllowed)
	}
}

func TestIsIconProxyDomainAllowedAllowsGoogleRedirectCompat(t *testing.T) {
	if !isIconProxyDomainAllowed("t2.gstatic.com", "google.com\nicon.horse") {
		t.Fatal("isIconProxyDomainAllowed() should allow *.gstatic.com when google.com is whitelisted")
	}
}

func TestIconProxyServiceFetchStreamsWhenUpstreamAllowed(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "icon_proxy_enabled", Value: "true"}).Error; err != nil {
		t.Fatalf("failed to seed icon_proxy_enabled: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "icon_proxy_domain_whitelist", Value: defaultIconProxyDomainWhitelist}).Error; err != nil {
		t.Fatalf("failed to seed icon_proxy_domain_whitelist: %v", err)
	}

	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, _ string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("8.8.8.8")}, nil
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	svc := NewIconProxyService(db)
	svc.httpClient = &http.Client{
		Transport: notificationTestRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"image/png"},
				},
				Body: io.NopCloser(strings.NewReader("pngdata")),
			}, nil
		}),
	}

	resolution, err := svc.Resolve("google", "example.com")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	resp, err := svc.Fetch(context.Background(), resolution)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if got := string(body); got != "pngdata" {
		t.Fatalf("body = %q, want %q", got, "pngdata")
	}
}
