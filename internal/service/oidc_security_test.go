package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestFetchOIDCUserInfoClaimsAllowsAdminConfiguredLocalEndpoint(t *testing.T) {
	client := &http.Client{Transport: notificationTestRoundTripper(func(req *http.Request) (*http.Response, error) {
		if got := req.URL.String(); got != "http://127.0.0.1/userinfo" {
			t.Fatalf("userinfo URL = %q, want local admin-configured endpoint", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"sub":"user-1","email":"user@example.com"}`)),
			Request:    req,
		}, nil
	})}

	claims, err := fetchOIDCUserInfoClaims(context.Background(), nil, &oauth2.Token{AccessToken: "token"}, "http://127.0.0.1/userinfo", client)
	if err != nil {
		t.Fatalf("fetchOIDCUserInfoClaims() error = %v, want nil", err)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("Subject = %q, want user-1", claims.Subject)
	}
}
