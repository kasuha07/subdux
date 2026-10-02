package migrations

import (
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/servicetest"
)

func TestMCPOAuthMigrationPreservesExistingDataAndCascades(t *testing.T) {
	db := servicetest.NewDB(t)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatal(err)
	}
	// Reproduce old schemas, without the additive OAuth audit columns.
	if err := db.Exec(`CREATE TABLE audit_events (event_id text PRIMARY KEY, key_id integer NOT NULL, key_kind text NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE mcp_idempotency_keys (id integer PRIMARY KEY, user_id integer NOT NULL, key_id integer NOT NULL, response text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO audit_events VALUES ('existing', 7, 'mcp_client')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO mcp_idempotency_keys VALUES (1, 1, 7, '{"existing":true}')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateMCPOAuth(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateMCPOAuth(db); err != nil {
		t.Fatal("migration is not idempotent", err)
	}
	var saved struct {
		KeyID        uint
		OAuthGrantID *uint
	}
	if err := db.Table("audit_events").Where("event_id = ?", "existing").Take(&saved).Error; err != nil || saved.KeyID != 7 || saved.OAuthGrantID != nil {
		t.Fatal("old audit changed", err)
	}
	var response string
	if err := db.Table("mcp_idempotency_keys").Select("response").Where("id = 1").Scan(&response).Error; err != nil || response != `{"existing":true}` {
		t.Fatal("old idempotency result changed", err)
	}
	user := servicetest.CreateUser(t, db)
	grant := model.MCPOAuthGrant{UserID: user.ID, ClientID: "client", ClientName: "Agent", Scopes: "read", Resource: "https://example.com/mcp", ExpiresAt: time.Now().Add(time.Hour)}
	if err := db.Create(&grant).Error; err != nil {
		t.Fatal("runtime model does not match migration", err)
	}
	if err := db.Create(&model.MCPOAuthToken{Hash: "hash", GrantID: grant.ID, Kind: "access", ExpiresAt: time.Now().Add(time.Minute)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.MCPOAuthRequest{ID: "request", UserID: &user.ID, ExpiresAt: time.Now().Add(time.Minute)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&user).Error; err != nil {
		t.Fatal(err)
	}
	for _, table := range []interface{}{&model.MCPOAuthRequest{}, &model.MCPOAuthGrant{}, &model.MCPOAuthToken{}} {
		var count int64
		if err := db.Model(table).Count(&count).Error; err != nil || count != 0 {
			t.Fatal("user deletion left OAuth credentials behind", err)
		}
	}
}
