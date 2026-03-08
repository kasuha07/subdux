package service

import (
	"context"
	"net"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestFetchOIDCUserInfoClaimsRejectsPrivateEndpoint(t *testing.T) {
	originalLookup := lookupOutboundHostIPs
	lookupOutboundHostIPs = func(_ context.Context, _ string, host string) ([]net.IP, error) {
		if host != "127.0.0.1" {
			t.Fatalf("lookup host = %q, want 127.0.0.1", host)
		}
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	}
	defer func() {
		lookupOutboundHostIPs = originalLookup
	}()

	_, err := fetchOIDCUserInfoClaims(context.Background(), nil, &oauth2.Token{AccessToken: "token"}, "http://127.0.0.1/userinfo")
	if err == nil {
		t.Fatal("fetchOIDCUserInfoClaims() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "must not target localhost or private network addresses") {
		t.Fatalf("fetchOIDCUserInfoClaims() error = %q, want localhost/private address validation error", err.Error())
	}
}
