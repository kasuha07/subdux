package service

import (
	"errors"
	"testing"

	"github.com/shiroha/subdux/internal/model"
)

func TestAPIKeyCreateRequiresValidKind(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.APIKey{}); err != nil {
		t.Fatalf("failed to migrate api keys: %v", err)
	}
	user := createTestUser(t, db)
	svc := NewAPIKeyService(db)

	if _, err := svc.Create(user.ID, user.Role, CreateAPIKeyInput{Name: "No kind"}); !errors.Is(err, ErrAPIKeyKindRequired) {
		t.Fatalf("Create() error = %v, want %v", err, ErrAPIKeyKindRequired)
	}
	if _, err := svc.Create(user.ID, user.Role, CreateAPIKeyInput{Name: "Bad kind", KeyKind: "legacy"}); !errors.Is(err, ErrAPIKeyKindInvalid) {
		t.Fatalf("Create() error = %v, want %v", err, ErrAPIKeyKindInvalid)
	}
}

func TestAPIKeyCreateNormalizesScopesAndRejectsWriteOnly(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.APIKey{}); err != nil {
		t.Fatalf("failed to migrate api keys: %v", err)
	}
	user := createTestUser(t, db)
	svc := NewAPIKeyService(db)

	resp, err := svc.Create(user.ID, user.Role, CreateAPIKeyInput{
		Name:    "Default read",
		KeyKind: APIKeyKindMCPClient,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got, want := resp.APIKey.Scopes, APIKeyScopeRead; got != want {
		t.Fatalf("scopes = %q, want %q", got, want)
	}

	if _, err := svc.Create(user.ID, user.Role, CreateAPIKeyInput{
		Name:    "Writer",
		KeyKind: APIKeyKindMCPClient,
		Scopes:  []string{APIKeyScopeWrite},
	}); !errors.Is(err, ErrAPIKeyScopeInvalid) {
		t.Fatalf("Create() error = %v, want %v", err, ErrAPIKeyScopeInvalid)
	}
}

func TestAPIKeyValidateReturnsKindAndMigratesEmptyKindToAPIIntegration(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.APIKey{}); err != nil {
		t.Fatalf("failed to migrate api keys: %v", err)
	}
	user := createTestUser(t, db)
	svc := NewAPIKeyService(db)

	resp, err := svc.Create(user.ID, user.Role, CreateAPIKeyInput{
		Name:    "MCP",
		KeyKind: APIKeyKindMCPClient,
		Scopes:  []string{APIKeyScopeRead, APIKeyScopeWrite},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	principal, err := svc.ValidateKey(resp.Key)
	if err != nil {
		t.Fatalf("ValidateKey() error = %v", err)
	}
	if principal.KeyKind != APIKeyKindMCPClient {
		t.Fatalf("KeyKind = %q, want %q", principal.KeyKind, APIKeyKindMCPClient)
	}

	if err := db.Model(&model.APIKey{}).Where("id = ?", resp.APIKey.ID).Update("key_kind", "").Error; err != nil {
		t.Fatalf("failed to clear key kind: %v", err)
	}
	principal, err = svc.ValidateKey(resp.Key)
	if err != nil {
		t.Fatalf("ValidateKey() error = %v", err)
	}
	if principal.KeyKind != APIKeyKindAPIIntegration {
		t.Fatalf("KeyKind = %q, want %q", principal.KeyKind, APIKeyKindAPIIntegration)
	}
}
