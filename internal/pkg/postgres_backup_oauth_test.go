package pkg

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

func TestPostgresBackupPreservesMCPOAuthState(t *testing.T) {
	for _, source := range []string{"postgres", "sqlite"} {
		t.Run(source, func(t *testing.T) {
			db := openIsolatedPostgresTestDB(t)
			t.Setenv("JWT_SECRET", "postgres-oauth-backup-test-secret-0123456789")
			sourceDB, restore := postgresBackupRoundTripSource(t, db, source)
			user := model.User{Username: "oauth-owner", Email: "oauth@example.com", Password: "hash", Role: "user", Status: "active"}
			if err := sourceDB.Create(&user).Error; err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC().Truncate(time.Microsecond)
			client := model.MCPOAuthClient{ID: "backup-client", Name: "Backup agent", RedirectURIs: `["https://client.example/callback"]`, MetadataAt: now}
			if err := sourceDB.Create(&client).Error; err != nil {
				t.Fatal(err)
			}
			grant := model.MCPOAuthGrant{UserID: user.ID, ClientID: client.ID, ClientName: client.Name, Scopes: "read write offline_access", Resource: "https://subdux.example/mcp", ExpiresAt: now.Add(30 * 24 * time.Hour), LastUsedAt: &now}
			if err := sourceDB.Create(&grant).Error; err != nil {
				t.Fatal(err)
			}
			codeHash := strings.Repeat("a", 64)
			request := model.MCPOAuthRequest{ID: strings.Repeat("b", 64), ClientID: client.ID, ClientName: client.Name, UserID: &user.ID, GrantID: &grant.ID,
				RedirectURI: "https://client.example/callback", Scopes: grant.Scopes, Resource: grant.Resource, State: "client-state", CodeChallenge: strings.Repeat("c", 43),
				CodeHash: &codeHash, DecidedAt: &now, ConsumedAt: &now, ExpiresAt: now.Add(time.Minute)}
			if err := sourceDB.Create(&request).Error; err != nil {
				t.Fatal(err)
			}
			tokens := []model.MCPOAuthToken{
				{Hash: strings.Repeat("d", 64), GrantID: grant.ID, Kind: "access", ExpiresAt: now.Add(15 * time.Minute)},
				{Hash: strings.Repeat("e", 64), GrantID: grant.ID, Kind: "refresh", ExpiresAt: grant.ExpiresAt},
				{Hash: strings.Repeat("f", 64), GrantID: grant.ID, Kind: "refresh", ExpiresAt: grant.ExpiresAt, ConsumedAt: &now},
			}
			if err := sourceDB.Create(&tokens).Error; err != nil {
				t.Fatal(err)
			}

			restore()
			var restoredClient model.MCPOAuthClient
			if err := db.Where("id = ?", client.ID).Take(&restoredClient).Error; err != nil {
				t.Fatalf("load restored OAuth client: %v", err)
			}
			if restoredClient.Name != client.Name || restoredClient.RedirectURIs != client.RedirectURIs || !restoredClient.MetadataAt.Equal(client.MetadataAt) {
				t.Fatal("restored OAuth client metadata changed")
			}
			var restoredGrant model.MCPOAuthGrant
			if err := db.First(&restoredGrant, grant.ID).Error; err != nil {
				t.Fatalf("load restored OAuth grant: %v", err)
			}
			if restoredGrant.UserID != user.ID || restoredGrant.ClientID != client.ID || restoredGrant.Scopes != grant.Scopes || restoredGrant.Resource != grant.Resource ||
				!restoredGrant.ExpiresAt.Equal(grant.ExpiresAt) || restoredGrant.LastUsedAt == nil || !restoredGrant.LastUsedAt.Equal(now) {
				t.Fatal("restored OAuth grant binding, scopes or lifetime changed")
			}
			var restoredRequest model.MCPOAuthRequest
			if err := db.Where("id = ?", request.ID).Take(&restoredRequest).Error; err != nil {
				t.Fatalf("load restored OAuth request: %v", err)
			}
			if restoredRequest.UserID == nil || *restoredRequest.UserID != user.ID || restoredRequest.GrantID == nil || *restoredRequest.GrantID != grant.ID ||
				restoredRequest.CodeHash == nil || *restoredRequest.CodeHash != codeHash || restoredRequest.ConsumedAt == nil || !restoredRequest.ConsumedAt.Equal(now) {
				t.Fatal("restored authorization code lost its consumption or grant binding")
			}
			for _, token := range tokens {
				var restoredToken model.MCPOAuthToken
				if err := db.Where("hash = ?", token.Hash).Take(&restoredToken).Error; err != nil {
					t.Fatalf("load restored OAuth token: %v", err)
				}
				if restoredToken.GrantID != grant.ID || restoredToken.Kind != token.Kind || !restoredToken.ExpiresAt.Equal(token.ExpiresAt) ||
					(restoredToken.ConsumedAt == nil) != (token.ConsumedAt == nil) {
					t.Fatal("restored OAuth token binding, expiry or replay marker changed")
				}
			}
			grant.ID = 0
			if err := db.Create(&grant).Error; err != nil {
				t.Fatalf("create OAuth grant after restore: %v", err)
			}
			if grant.ID != restoredGrant.ID+1 {
				t.Fatalf("restored grant sequence: got %d, want %d", grant.ID, restoredGrant.ID+1)
			}
		})
	}
}

func TestPostgresRestorePreservesLegacyLoginIdentifiers(t *testing.T) {
	for _, source := range []string{"postgres", "sqlite"} {
		t.Run(source, func(t *testing.T) {
			db := openIsolatedPostgresTestDB(t)
			t.Setenv("JWT_SECRET", "postgres-legacy-backup-test-secret-0123456789")
			sourceDB, restore := postgresBackupRoundTripSource(t, db, source)
			if source == "postgres" {
				if err := sourceDB.Exec("ALTER TABLE users DISABLE TRIGGER trg_users_login_identifiers").Error; err != nil {
					t.Fatal(err)
				}
			}
			users := []model.User{
				{Username: "legacy@example.com", Email: "first@example.com", Password: "hash", Role: "user", Status: "active"},
				{Username: "second@example.com", Email: "legacy@example.com", Password: "hash", Role: "user", Status: "active"},
			}
			if err := sourceDB.Create(&users).Error; err != nil {
				t.Fatal(err)
			}
			if source == "postgres" {
				if err := sourceDB.Exec("ALTER TABLE users ENABLE TRIGGER trg_users_login_identifiers").Error; err != nil {
					t.Fatal(err)
				}
			}
			restore()
			for _, user := range users {
				var restored model.User
				if err := db.First(&restored, user.ID).Error; err != nil {
					t.Fatal(err)
				}
				if restored.Username != user.Username || restored.Email != user.Email {
					t.Fatal("restore changed a legacy login identifier")
				}
			}
			assertPostgresLoginIdentifierTrigger(t, db)

			// A failed restore must roll back data and trigger changes. Ordinary
			// uniqueness constraints remain active while importing legacy rows.
			var snapshot bytes.Buffer
			if err := WritePostgresBackup(db, &snapshot); err != nil {
				t.Fatal(err)
			}
			var data struct {
				Format string                     `json:"format"`
				Tables map[string]json.RawMessage `json:"tables"`
			}
			if err := json.Unmarshal(snapshot.Bytes(), &data); err != nil {
				t.Fatal(err)
			}
			var rows []map[string]any
			if err := json.Unmarshal(data.Tables["users"], &rows); err != nil {
				t.Fatal(err)
			}
			rows[0]["email"], rows[1]["email"] = "duplicate@example.com", "DUPLICATE@example.com"
			encodedRows, err := json.Marshal(rows)
			if err != nil {
				t.Fatal(err)
			}
			data.Tables["users"] = encodedRows
			invalid, err := json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}
			err = RestorePostgresBackup(db, bytes.NewReader(invalid))
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
				t.Fatalf("case-folded duplicate restore error = %v, want uniqueness rejection", err)
			}
			for _, user := range users {
				var restored model.User
				if err := db.First(&restored, user.ID).Error; err != nil || restored.Email != user.Email {
					t.Fatalf("failed restore changed existing user: %v", err)
				}
			}
			assertPostgresLoginIdentifierTrigger(t, db)
		})
	}
}

func assertPostgresLoginIdentifierTrigger(t *testing.T, db *gorm.DB) {
	t.Helper()
	var enabled string
	if err := db.Raw("SELECT tgenabled FROM pg_trigger WHERE tgrelid = 'users'::regclass AND tgname = 'trg_users_login_identifiers'").Scan(&enabled).Error; err != nil {
		t.Fatal(err)
	}
	if enabled != "O" {
		t.Fatalf("login identifier trigger enabled state = %q, want O", enabled)
	}
	conflict := model.User{Username: "new-user", Email: "SECOND@EXAMPLE.COM", Password: "hash", Role: "user", Status: "active"}
	err := db.Create(&conflict).Error
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != "users_email_username_collision" {
		t.Fatalf("post-restore identifier claim = %v, want cross-field rejection", err)
	}
}

func postgresBackupRoundTripSource(t *testing.T, target *gorm.DB, source string) (*gorm.DB, func()) {
	t.Helper()
	if source == "postgres" {
		return target, func() {
			var snapshot bytes.Buffer
			if err := WritePostgresBackup(target, &snapshot); err != nil {
				t.Fatalf("write PostgreSQL backup: %v", err)
			}
			if err := RestorePostgresBackup(target, bytes.NewReader(snapshot.Bytes())); err != nil {
				t.Fatalf("restore PostgreSQL backup: %v", err)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "source.db")
	db, err := openSQLiteDatabase(path)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db, func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatal(err)
		}
		if err := ImportSQLiteBackupToPostgres(target, path); err != nil {
			t.Fatalf("import SQLite backup into PostgreSQL: %v", err)
		}
	}
}
