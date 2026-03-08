package service

import (
	"context"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestFetchOIDCUserInfoClaimsRejectsLocalhostEndpoint(t *testing.T) {
	_, err := fetchOIDCUserInfoClaims(
		context.Background(),
		nil,
		&oauth2.Token{AccessToken: "token"},
		"http://localhost/userinfo",
	)
	if err == nil {
		t.Fatal("fetchOIDCUserInfoClaims() error = nil, want localhost/private address validation error")
	}
	if !strings.Contains(err.Error(), "must not target localhost or private network addresses") {
		t.Fatalf("fetchOIDCUserInfoClaims() error = %q, want localhost/private address validation error", err.Error())
	}
}
