package mcpoauth

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
)

func TestRegistrationFloodCannotMonopolizeClientCapacity(t *testing.T) {
	s, user, granted := setup(t)
	// The setup client holds a grant; it must survive any eviction.
	exchange, _ := authorize(t, s, user, granted, false)
	if _, err := s.Exchange(exchange); err != nil {
		t.Fatal(err)
	}

	// Fill the table to capacity with unused registrations aged past the
	// eviction floor, plus one client with an in-flight authorization request.
	old := time.Now().Add(-orphanClientMinAge - time.Minute)
	pending, err := s.Register(ClientMetadata{ClientName: "pending", RedirectURIs: []string{"http://127.0.0.1/callback"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Begin(authorizeParams(pending.ClientID)); err != nil {
		t.Fatal(err)
	}
	var current int64
	if err := s.db.Model(&model.MCPOAuthClient{}).Count(&current).Error; err != nil {
		t.Fatal(err)
	}
	for i := current; i < maxOAuthClients; i++ {
		if err := s.db.Create(&model.MCPOAuthClient{ID: fmt.Sprintf("sdx_client_flood_%d", i), Name: "flood", RedirectURIs: `["http://127.0.0.1/cb"]`, MetadataAt: old, CreatedAt: old}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := s.db.Model(&model.MCPOAuthClient{}).Where("id IN ?", []string{granted.ClientID, pending.ClientID}).Update("created_at", old.Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}

	// A new legitimate registration still succeeds by evicting an orphan.
	fresh, err := s.Register(ClientMetadata{ClientName: "legit", RedirectURIs: []string{"http://127.0.0.1/callback"}})
	if err != nil {
		t.Fatalf("registration at capacity failed: %v", err)
	}
	var total int64
	if err := s.db.Model(&model.MCPOAuthClient{}).Count(&total).Error; err != nil {
		t.Fatal(err)
	}
	if total != maxOAuthClients {
		t.Fatalf("clients = %d, want %d", total, maxOAuthClients)
	}
	for _, id := range []string{granted.ClientID, pending.ClientID, fresh.ClientID} {
		var n int64
		if err := s.db.Model(&model.MCPOAuthClient{}).Where("id = ?", id).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("client %s was evicted; granted/pending/new clients must be kept", id)
		}
	}
}

func TestRegistrationAtCapacityKeepsRecentClients(t *testing.T) {
	s, _, _ := setup(t)
	now := time.Now()
	var current int64
	if err := s.db.Model(&model.MCPOAuthClient{}).Count(&current).Error; err != nil {
		t.Fatal(err)
	}
	for i := current; i < maxOAuthClients; i++ {
		if err := s.db.Create(&model.MCPOAuthClient{ID: fmt.Sprintf("sdx_client_recent_%d", i), Name: "recent", RedirectURIs: `["http://127.0.0.1/cb"]`, MetadataAt: now, CreatedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
	}
	_, err := s.Register(ClientMetadata{ClientName: "late", RedirectURIs: []string{"http://127.0.0.1/callback"}})
	var protocolErr *ProtocolError
	if !errors.As(err, &protocolErr) || protocolErr.Code != "temporarily_unavailable" {
		t.Fatalf("err = %v, want temporarily_unavailable while every client is newer than the eviction floor", err)
	}
}

func authorizeParams(clientID string) map[string][]string {
	return map[string][]string{"client_id": {clientID}, "redirect_uri": {"http://127.0.0.1:54321/callback"}, "response_type": {"code"},
		"resource": {"https://subdux.example/mcp"}, "scope": {"read"}, "code_challenge": {"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"}, "code_challenge_method": {"S256"}, "state": {"s"}}
}
