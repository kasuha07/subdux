package service

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func TestSendNtfyUsesSubscriptionURLAsClickHeader(t *testing.T) {
	const subscriptionURL = "https://subscription.example.com/manage"
	const configClickURL = "https://channel.example.com/fallback"

	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		if host != "ntfy.example.com" {
			return nil, errors.New("unexpected lookup host")
		}
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	var gotClick string
	var gotXClick string

	db, proxyServer := newNtfyProxyTestDB(t, func(r *http.Request) {
		gotClick = r.Header.Get("Click")
		gotXClick = r.Header.Get("X-Click")
	})
	defer proxyServer.Close()

	config, err := json.Marshal(map[string]string{
		"url":   "http://ntfy.example.com",
		"topic": "subdux-test",
		"click": configClickURL,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	svc := &NotificationService{DB: db}
	channel := model.NotificationChannel{
		Type:   "ntfy",
		Config: string(config),
	}

	if err := svc.sendNtfy(channel, "subscription reminder", subscriptionURL); err != nil {
		t.Fatalf("sendNtfy() error = %v", err)
	}

	if gotClick != subscriptionURL {
		t.Fatalf("Click header = %q, want %q", gotClick, subscriptionURL)
	}
	if gotXClick != subscriptionURL {
		t.Fatalf("X-Click header = %q, want %q", gotXClick, subscriptionURL)
	}
}

func TestSendNtfyFallsBackToConfigClickHeaderWhenSubscriptionURLMissing(t *testing.T) {
	const configClickURL = "https://channel.example.com/fallback"

	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		if host != "ntfy.example.com" {
			return nil, errors.New("unexpected lookup host")
		}
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	var gotClick string
	var gotXClick string

	db, proxyServer := newNtfyProxyTestDB(t, func(r *http.Request) {
		gotClick = r.Header.Get("Click")
		gotXClick = r.Header.Get("X-Click")
	})
	defer proxyServer.Close()

	config, err := json.Marshal(map[string]string{
		"url":   "http://ntfy.example.com",
		"topic": "subdux-test",
		"click": configClickURL,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	svc := &NotificationService{DB: db}
	channel := model.NotificationChannel{
		Type:   "ntfy",
		Config: string(config),
	}

	if err := svc.sendNtfy(channel, "subscription reminder", ""); err != nil {
		t.Fatalf("sendNtfy() error = %v", err)
	}

	if gotClick != configClickURL {
		t.Fatalf("Click header = %q, want %q", gotClick, configClickURL)
	}
	if gotXClick != configClickURL {
		t.Fatalf("X-Click header = %q, want %q", gotXClick, configClickURL)
	}
}

func newNtfyProxyTestDB(t *testing.T, inspect func(*http.Request)) (*gorm.DB, *httptest.Server) {
	t.Helper()
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "ntfy-proxy-test-key")

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if inspect != nil {
			inspect(r)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		proxyServer.Close()
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	seedProxySettings(t, db, "true", "http", proxyServer.URL)
	return db, proxyServer
}
