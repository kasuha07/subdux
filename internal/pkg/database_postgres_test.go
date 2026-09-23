package pkg

import (
	"bytes"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestOpenConfiguredDatabaseDefaultsToSQLite(t *testing.T) {
	t.Setenv(DatabaseURLEnv, "")
	t.Setenv("DATA_PATH", t.TempDir())

	db, err := openConfiguredDatabase()
	if err != nil {
		t.Fatalf("openConfiguredDatabase() error = %v", err)
	}
	if got := db.Dialector.Name(); got != "sqlite" {
		t.Fatalf("database dialect = %q, want sqlite", got)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sqlite database: %v", err)
	}
}

func TestPostgresLogicalBackupRestoresByteaAndSequences(t *testing.T) {
	db := openIsolatedPostgresTestDB(t)
	t.Setenv("JWT_SECRET", "postgres-integration-secret-0123456789")

	user := model.User{
		Username: "backup-" + uuid.NewString(),
		Email:    uuid.NewString() + "@example.com",
		Password: "password-hash",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create backup user: %v", err)
	}
	credential := model.PasskeyCredential{
		UserID:       user.ID,
		Name:         "integration key",
		CredentialID: uuid.NewString(),
		Credential:   []byte{0, 1, 2, 127, 128, 255},
	}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatalf("create passkey credential: %v", err)
	}

	var snapshot bytes.Buffer
	if err := WritePostgresBackup(db, &snapshot); err != nil {
		t.Fatalf("WritePostgresBackup() error = %v", err)
	}

	changed := user
	changed.Email = uuid.NewString() + "@example.com"
	if err := db.Model(&model.User{}).Where("id = ?", user.ID).Update("email", changed.Email).Error; err != nil {
		t.Fatalf("change user before restore: %v", err)
	}
	extra := model.User{
		Username: "after-backup-" + uuid.NewString(),
		Email:    uuid.NewString() + "@example.com",
		Password: "password-hash",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&extra).Error; err != nil {
		t.Fatalf("create user after backup: %v", err)
	}

	if err := RestorePostgresBackup(db, bytes.NewReader(snapshot.Bytes())); err != nil {
		t.Fatalf("RestorePostgresBackup() error = %v", err)
	}

	var restored model.User
	if err := db.First(&restored, user.ID).Error; err != nil {
		t.Fatalf("load restored user: %v", err)
	}
	if restored.Email != user.Email {
		t.Fatalf("restored email = %q, want %q", restored.Email, user.Email)
	}
	var restoredCredential model.PasskeyCredential
	if err := db.Where("user_id = ?", user.ID).First(&restoredCredential).Error; err != nil {
		t.Fatalf("load restored passkey credential: %v", err)
	}
	if !bytes.Equal(restoredCredential.Credential, credential.Credential) {
		t.Fatalf("restored credential = %v, want %v", restoredCredential.Credential, credential.Credential)
	}
	if err := db.First(&model.User{}, extra.ID).Error; err == nil {
		t.Fatalf("user %d created after the snapshot survived restore", extra.ID)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("look up user created after snapshot: %v", err)
	}

	next := model.User{
		Username: "after-restore-" + uuid.NewString(),
		Email:    uuid.NewString() + "@example.com",
		Password: "password-hash",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&next).Error; err != nil {
		t.Fatalf("create user after restore: %v", err)
	}
	if next.ID != user.ID+1 {
		t.Fatalf("next user ID = %d, want %d after restoring sequence", next.ID, user.ID+1)
	}
}

func openIsolatedPostgresTestDB(t *testing.T) *gorm.DB {
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
	schema := "subdux_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	t.Setenv(DatabaseURLEnv, u.String())
	db, err := openConfiguredDatabase()
	if err != nil {
		t.Fatalf("open configured PostgreSQL database: %v", err)
	}
	if !IsPostgres(db) {
		t.Fatalf("configured database dialect = %q, want postgres", db.Dialector.Name())
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("access PostgreSQL application connection: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close PostgreSQL application connection: %v", err)
		}
	})
	return db
}
