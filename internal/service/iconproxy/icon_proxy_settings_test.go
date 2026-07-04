package iconproxy

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	serviceoutbound "github.com/kasuha07/subdux/internal/service/outbound"
	"gorm.io/gorm"
)

type testRoundTripper func(*http.Request) (*http.Response, error)

func (fn testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-iconproxy-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	return db
}

func TestNormalizeDomainWhitelistSortsAndDeduplicates(t *testing.T) {
	input := " icon.horse ;WWW.Google.com\ngoogle.com "
	got, err := NormalizeDomainWhitelist(input)
	if err != nil {
		t.Fatalf("NormalizeDomainWhitelist() error = %v", err)
	}

	want := "google.com\nicon.horse\nwww.google.com"
	if got != want {
		t.Fatalf("NormalizeDomainWhitelist() = %q, want %q", got, want)
	}
}

func TestNormalizeDomainWhitelistRejectsURL(t *testing.T) {
	_, err := NormalizeDomainWhitelist("https://www.google.com")
	if !errors.Is(err, ErrInvalidIconProxyDomainWhitelist) {
		t.Fatalf("NormalizeDomainWhitelist() error = %v, want %v", err, ErrInvalidIconProxyDomainWhitelist)
	}
}

func TestServiceResolveUsesRedirectWhenDisabled(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "icon_proxy_enabled", Value: "false"}).Error; err != nil {
		t.Fatalf("failed to seed icon_proxy_enabled: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "icon_proxy_domain_whitelist", Value: DefaultDomainWhitelist}).Error; err != nil {
		t.Fatalf("failed to seed icon_proxy_domain_whitelist: %v", err)
	}

	svc := NewService(db)
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

func TestServiceResolveRejectsDisallowedUpstreamHost(t *testing.T) {
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

	svc := NewService(db)
	_, err := svc.Resolve("google", "example.com")
	if !errors.Is(err, ErrIconProxyDomainNotAllowed) {
		t.Fatalf("Resolve() error = %v, want %v", err, ErrIconProxyDomainNotAllowed)
	}
}

func TestIsIconProxyDomainAllowedAllowsGoogleRedirectCompat(t *testing.T) {
	if !IsDomainAllowed("t2.gstatic.com", "google.com\nicon.horse") {
		t.Fatal("IsDomainAllowed() should allow *.gstatic.com when google.com is whitelisted")
	}
}

func TestServiceFetchStreamsWhenUpstreamAllowed(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "icon_proxy_enabled", Value: "true"}).Error; err != nil {
		t.Fatalf("failed to seed icon_proxy_enabled: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "icon_proxy_domain_whitelist", Value: DefaultDomainWhitelist}).Error; err != nil {
		t.Fatalf("failed to seed icon_proxy_domain_whitelist: %v", err)
	}

	restoreLookup := serviceoutbound.SetLookupHostIPsForTest(func(_ context.Context, _ string, _ string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("8.8.8.8")}, nil
	})
	defer restoreLookup()

	svc := NewService(db)
	svc.httpClient = &http.Client{
		Transport: testRoundTripper(func(req *http.Request) (*http.Response, error) {
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
