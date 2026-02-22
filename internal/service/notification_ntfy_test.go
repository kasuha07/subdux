package service

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/shiroha/subdux/internal/model"
)

func TestSendNtfyUsesSubscriptionURLAsClickHeader(t *testing.T) {
	const subscriptionURL = "https://subscription.example.com/manage"
	const configClickURL = "https://channel.example.com/fallback"

	var gotClick string
	var gotXClick string

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		gotClick = r.Header.Get("Click")
		gotXClick = r.Header.Get("X-Click")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})
	defer func() {
		http.DefaultTransport = originalTransport
	}()

	config, err := json.Marshal(map[string]string{
		"url":   "https://ntfy.example.com",
		"topic": "subdux-test",
		"click": configClickURL,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	svc := &NotificationService{}
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

	var gotClick string
	var gotXClick string

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		gotClick = r.Header.Get("Click")
		gotXClick = r.Header.Get("X-Click")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})
	defer func() {
		http.DefaultTransport = originalTransport
	}()

	config, err := json.Marshal(map[string]string{
		"url":   "https://ntfy.example.com",
		"topic": "subdux-test",
		"click": configClickURL,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	svc := &NotificationService{}
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

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
