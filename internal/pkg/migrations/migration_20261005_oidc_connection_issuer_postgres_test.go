package migrations

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openMigrationPostgresTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("SUBDUX_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set SUBDUX_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatalf("SUBDUX_TEST_POSTGRES_DSN must be a postgres:// or postgresql:// URL")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL admin connection: %v", err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatalf("access PostgreSQL admin connection: %v", err)
	}
	schema := "subdux_migration_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec("CREATE SCHEMA \"" + schema + "\"").Error; err != nil {
		_ = adminSQL.Close()
		t.Fatalf("create isolated PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() {
		if err := admin.Exec("DROP SCHEMA IF EXISTS \"" + schema + "\" CASCADE").Error; err != nil {
			t.Errorf("drop isolated PostgreSQL schema: %v", err)
		}
		_ = adminSQL.Close()
	})
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("access isolated PostgreSQL database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func TestOIDCConnectionIssuerMigrationPostgres(t *testing.T) {
	db := openMigrationPostgresTestDB(t)
	for _, statement := range []string{
		`CREATE TABLE system_settings (id bigserial PRIMARY KEY, key text NOT NULL UNIQUE, value text, revision bigint NOT NULL DEFAULT 1)`,
		`CREATE TABLE oidc_connections (id bigserial PRIMARY KEY, user_id bigint NOT NULL, provider varchar(100) NOT NULL, subject varchar(255) NOT NULL, email varchar(255), created_at timestamptz, updated_at timestamptz)`,
		`CREATE UNIQUE INDEX idx_oidc_user_provider ON oidc_connections (user_id, provider)`,
		`CREATE UNIQUE INDEX idx_oidc_provider_subject ON oidc_connections (provider, subject)`,
		`INSERT INTO oidc_connections (user_id, provider, subject, email) VALUES (10, 'oidc', 'alice', 'alice@example.com')`,
		`INSERT INTO system_settings (key, value) VALUES ('oidc_issuer_url', 'https://idp.example.com')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := migrateOIDCConnectionIssuer(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateOIDCConnectionIssuer(db); err != nil {
		t.Fatal("migration is not idempotent: ", err)
	}

	var issuer string
	if err := db.Table("oidc_connections").Select("issuer").Where("subject = ?", "alice").Scan(&issuer).Error; err != nil {
		t.Fatal(err)
	}
	if issuer != "https://idp.example.com" {
		t.Fatalf("issuer = %q, want backfilled configured issuer", issuer)
	}
	if db.Migrator().HasIndex("oidc_connections", "idx_oidc_provider_subject") {
		t.Fatal("subject-only unique index still exists")
	}
	if err := db.Exec(`INSERT INTO oidc_connections (user_id, provider, issuer, subject) VALUES (11, 'oidc', 'https://other.example.com', 'alice')`).Error; err != nil {
		t.Fatalf("same subject under another issuer rejected: %v", err)
	}
	if err := db.Exec(`INSERT INTO oidc_connections (user_id, provider, issuer, subject) VALUES (12, 'oidc', 'https://idp.example.com', 'alice')`).Error; err == nil {
		t.Fatal("duplicate issuer+subject accepted")
	}
}
