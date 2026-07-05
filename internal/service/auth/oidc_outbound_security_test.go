package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"golang.org/x/oauth2"
)

func TestFetchOIDCUserInfoClaimsRejectsNonHTTPUserinfoEndpoint(t *testing.T) {
	_, err := fetchOIDCUserInfoClaims(
		context.Background(),
		nil,
		&oauth2.Token{AccessToken: "token"},
		"ftp://localhost/userinfo",
		nil,
	)
	if err == nil {
		t.Fatal("fetchOIDCUserInfoClaims() error = nil, want URL scheme validation error")
	}
	if !strings.Contains(err.Error(), "must start with http:// or https://") {
		t.Fatalf("fetchOIDCUserInfoClaims() error = %q, want URL scheme validation error", err.Error())
	}
}

func TestFetchOIDCUserInfoClaimsReturnsParameterizedStatusError(t *testing.T) {
	client := &http.Client{Transport: notificationTestRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Request:    req,
		}, nil
	})}

	_, err := fetchOIDCUserInfoClaims(
		context.Background(),
		nil,
		&oauth2.Token{AccessToken: "token"},
		"https://issuer.example.com/userinfo",
		client,
	)
	if err == nil {
		t.Fatal("fetchOIDCUserInfoClaims() error = nil, want non-2xx error")
	}

	var typed *serviceerr.Error
	if !errors.As(err, &typed) || typed == nil {
		t.Fatalf("fetchOIDCUserInfoClaims() error = %T, want *serviceerr.Error", err)
	}
	if typed.Code != "oidc_userinfo_endpoint_returned_status" {
		t.Fatalf("Code = %q, want oidc_userinfo_endpoint_returned_status", typed.Code)
	}
	if got := typed.Params["status"]; got != http.StatusNotFound {
		t.Fatalf("Params[status] = %v, want %d", got, http.StatusNotFound)
	}
	if typed.Error() != "oidc userinfo endpoint returned 404" {
		t.Fatalf("Error() = %q, want oidc userinfo endpoint returned 404", typed.Error())
	}
}
