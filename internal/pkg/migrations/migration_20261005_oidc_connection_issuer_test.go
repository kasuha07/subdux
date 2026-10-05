package migrations

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

func openLegacyOIDCConnectionDB(t *testing.T, issuerSetting *string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE system_settings (id integer PRIMARY KEY, key text NOT NULL UNIQUE, value text, revision integer NOT NULL DEFAULT 1)`,
		`CREATE TABLE oidc_connections (id integer PRIMARY KEY, user_id integer NOT NULL, provider text NOT NULL, subject text NOT NULL, email text, created_at datetime, updated_at datetime)`,
		`CREATE UNIQUE INDEX idx_oidc_user_provider ON oidc_connections (user_id, provider)`,
		`CREATE UNIQUE INDEX idx_oidc_provider_subject ON oidc_connections (provider, subject)`,
		`INSERT INTO oidc_connections (id, user_id, provider, subject, email) VALUES (1, 10, 'oidc', 'alice', 'alice@example.com')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if issuerSetting != nil {
		if err := db.Exec(`INSERT INTO system_settings (key, value) VALUES ('oidc_issuer_url', ?)`, *issuerSetting).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestOIDCConnectionIssuerMigrationAttributesLinksToConfiguredIssuer(t *testing.T) {
	issuer := "  https://idp.example.com  "
	db := openLegacyOIDCConnectionDB(t, &issuer)

	if err := migrateOIDCConnectionIssuer(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateOIDCConnectionIssuer(db); err != nil {
		t.Fatal("migration is not idempotent: ", err)
	}

	var saved struct {
		Subject string
		Email   string
		Issuer  string
	}
	if err := db.Table("oidc_connections").Where("id = 1").Take(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Issuer != "https://idp.example.com" || saved.Subject != "alice" || saved.Email != "alice@example.com" {
		t.Fatalf("migrated connection = %+v", saved)
	}

	if db.Migrator().HasIndex("oidc_connections", "idx_oidc_provider_subject") {
		t.Fatal("subject-only unique index still exists")
	}
	if !db.Migrator().HasIndex("oidc_connections", "idx_oidc_provider_issuer_subject") {
		t.Fatal("issuer-scoped unique index was not created")
	}

	// The same subject from another issuer is a different identity.
	if err := db.Exec(`INSERT INTO oidc_connections (id, user_id, provider, issuer, subject) VALUES (2, 11, 'oidc', 'https://other.example.com', 'alice')`).Error; err != nil {
		t.Fatalf("same subject under another issuer rejected: %v", err)
	}
	if err := db.Exec(`INSERT INTO oidc_connections (id, user_id, provider, issuer, subject) VALUES (3, 12, 'oidc', 'https://idp.example.com', 'alice')`).Error; err == nil {
		t.Fatal("duplicate issuer+subject accepted")
	}
}

func TestOIDCConnectionIssuerMigrationLeavesLinksUnattributedWithoutIssuer(t *testing.T) {
	db := openLegacyOIDCConnectionDB(t, nil)

	if err := migrateOIDCConnectionIssuer(db); err != nil {
		t.Fatal(err)
	}

	var issuer string
	if err := db.Table("oidc_connections").Select("issuer").Where("id = 1").Scan(&issuer).Error; err != nil {
		t.Fatal(err)
	}
	if issuer != "" {
		t.Fatalf("issuer = %q, want empty", issuer)
	}

	// Configuring an issuer later must not retroactively attribute the link:
	// the migration has already run and nothing else backfills it.
	if err := db.Exec(`INSERT INTO system_settings (key, value) VALUES ('oidc_issuer_url', 'https://idp.example.com')`).Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Table("oidc_connections").Where("issuer = ?", "https://idp.example.com").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unattributed links matched the new issuer: %d", count)
	}
}

func TestOIDCConnectionIssuerMigrationAcceptsCurrentSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "fresh.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.SystemSetting{}, &model.OIDCConnection{}); err != nil {
		t.Fatal(err)
	}
	if err := migrateOIDCConnectionIssuer(db); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "u", Email: "u@example.com", Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.OIDCConnection{UserID: user.ID, Provider: "oidc", Issuer: "https://idp.example.com", Subject: "s"}).Error; err != nil {
		t.Fatalf("runtime model does not match migration: %v", err)
	}
}
