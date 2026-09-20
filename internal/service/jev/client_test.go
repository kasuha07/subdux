package jev

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProviderContractAndFailureFallbacks(t *testing.T) {
	valid := `{"answers":{"category":{"type":"choice","choice":"category_7","confidence":0.95,"probabilities":{"category_7":0.97,"none":0.03}}}}`
	for _, tc := range []struct {
		name, body string
		status     int
		want       bool
	}{
		{"valid", valid, 200, true},
		{"uncertain", strings.Replace(valid, `0.95`, `0.4`, 1), 200, false},
		{"missing confidence", `{"answers":{"category":{"type":"choice","choice":"category_7"}}}`, 200, false},
		{"unknown category", strings.ReplaceAll(valid, "category_7", "category_99"), 200, false},
		{"abstain", `{"answers":{"category":{"type":"choice","choice":"none","confidence":1}}}`, 200, false},
		{"malformed", `<html>error</html>`, 200, false},
		{"oversized", strings.Repeat("x", (64<<10)+1), 200, false},
		{"unauthorized", `secret-in-error`, 401, false},
		{"rate limited", `rate limit`, 429, false},
		{"unavailable", `unavailable`, 503, false},
		{"redirect", `redirect`, 302, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != endpoint || req.Method != http.MethodPost || req.Header.Get("Authorization") != "Bearer fake-key" {
					t.Fatal("wrong provider contract")
				}
				body, _ := io.ReadAll(req.Body)
				if !strings.Contains(string(body), `"website_domain":"spotify.com"`) || strings.Contains(string(body), "private") || strings.Contains(string(body), "token") {
					t.Fatal("URL data leaked")
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})}
			id, err := classifyWithClient(context.Background(), client, "fake-key", SuggestInput{Name: "Spotify", URL: "https://private:private@spotify.com/private?token=private#private"}, []model.Category{{ID: 7, Name: "Music"}})
			if tc.want && (err != nil || id == nil || *id != 7) {
				t.Fatalf("valid classification failed: %v", err)
			}
			if !tc.want && id != nil {
				t.Fatal("invalid classification accepted")
			}
			if err != nil && strings.Contains(err.Error(), "secret-in-error") {
				t.Fatal("provider body leaked")
			}
		})
	}
}

func TestProviderCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if !errors.Is(r.Context().Err(), context.Canceled) {
			t.Fatal("context lost")
		}
		return nil, r.Context().Err()
	})}
	if _, err := classifyWithClient(ctx, client, "key", SuggestInput{Name: "Spotify"}, nil); err == nil {
		t.Fatal("cancellation ignored")
	}
}
