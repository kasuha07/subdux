package service

import (
	"context"
	"strings"
	"testing"

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
