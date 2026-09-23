package backup

import (
	"archive/zip"
	"bytes"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/pkg/migrations"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPostgresBackupDownloadUsesRestorableJSONArchive(t *testing.T) {
	db := openBackupPostgresTestDB(t)
	t.Setenv("JWT_SECRET", "postgres-backup-test-secret-0123456789")
	t.Setenv("DATA_PATH", t.TempDir())

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

	archivePath, err := NewService(db).BackupDB(false, "")
	if err != nil {
		t.Fatalf("BackupDB() error = %v", err)
	}
	defer os.RemoveAll(filepath.Dir(archivePath))
	if filepath.Ext(archivePath) != ".zip" {
		t.Fatalf("PostgreSQL backup extension = %q, want .zip", filepath.Ext(archivePath))
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open PostgreSQL backup archive: %v", err)
	}
	var hasJSON bool
	var hasSQLite bool
	for _, entry := range archive.File {
		hasJSON = hasJSON || entry.Name == "subdux.json"
		hasSQLite = hasSQLite || entry.Name == "subdux.db"
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close PostgreSQL backup archive: %v", err)
	}
	if !hasJSON || hasSQLite {
		t.Fatalf("PostgreSQL backup entries contain subdux.json=%t and subdux.db=%t", hasJSON, hasSQLite)
	}

	if err := db.Delete(&model.User{}, user.ID).Error; err != nil {
		t.Fatalf("delete user before restore: %v", err)
	}
	result, err := NewService(db).RestoreBackup(archivePath, "")
	if err != nil {
		t.Fatalf("RestoreBackup() error = %v", err)
	}
	if !result.Reopened {
		t.Fatal("PostgreSQL restore did not report the live connection ready")
	}
	var restored model.User
	if err := db.First(&restored, user.ID).Error; err != nil {
		t.Fatalf("load restored user: %v", err)
	}
	if restored.Email != user.Email {
		t.Fatalf("restored email = %q, want %q", restored.Email, user.Email)
	}
}

func TestPostgresRestoreImportsSQLiteBackup(t *testing.T) {
	db := openBackupPostgresTestDB(t)
	t.Setenv("JWT_SECRET", "postgres-sqlite-import-test-secret-0123456789")
	t.Setenv("DATA_PATH", t.TempDir())

	sqlitePath := filepath.Join(t.TempDir(), "legacy-subdux.db")
	sourceDB, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open SQLite backup database: %v", err)
	}
	sourceSQL, err := sourceDB.DB()
	if err != nil {
		t.Fatalf("access SQLite backup connection: %v", err)
	}
	sourceSQL.SetMaxOpenConns(1)
	sourceSQL.SetMaxIdleConns(1)
	if err := sourceDB.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("enable SQLite backup foreign keys: %v", err)
	}
	if err := sourceDB.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		t.Fatalf("enable SQLite backup WAL: %v", err)
	}
	if err := migrations.Run(sourceDB, migrations.SecretCodec{
		Encrypt: pkg.EncryptSystemSettingValue,
		Decrypt: pkg.DecryptSystemSettingValue,
	}); err != nil {
		t.Fatalf("migrate SQLite backup database: %v", err)
	}
	user := model.User{
		Username: "sqlite-import-" + uuid.NewString(),
		Email:    uuid.NewString() + "@example.com",
		Password: "legacy-password-hash",
		Role:     "user",
		Status:   "active",
	}
	if err := sourceDB.Create(&user).Error; err != nil {
		t.Fatalf("create SQLite backup user: %v", err)
	}
	credential := model.PasskeyCredential{
		UserID:       user.ID,
		Name:         "legacy passkey",
		CredentialID: uuid.NewString(),
		Credential:   []byte{0, 1, 2, 127, 128, 255},
	}
	if err := sourceDB.Create(&credential).Error; err != nil {
		t.Fatalf("create SQLite backup passkey: %v", err)
	}
	notifyEnabled := false
	subscription := model.Subscription{
		UserID:           user.ID,
		Name:             "Legacy subscription",
		Amount:           17.25,
		Currency:         "USD",
		Status:           "active",
		RenewalMode:      "auto_renew",
		BillingType:      "recurring",
		NotifyEnabled:    &notifyEnabled,
		NotifyDaysBefore: nil,
	}
	if err := sourceDB.Create(&subscription).Error; err != nil {
		t.Fatalf("create SQLite backup subscription: %v", err)
	}
	if err := sourceSQL.Close(); err != nil {
		t.Fatalf("close SQLite backup database: %v", err)
	}

	existing := model.User{
		Username: "before-import-" + uuid.NewString(),
		Email:    uuid.NewString() + "@example.com",
		Password: "password-hash",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create existing PostgreSQL user: %v", err)
	}

	result, err := NewService(db).RestoreBackup(sqlitePath, "")
	if err != nil {
		t.Fatalf("RestoreBackup(SQLite) error = %v", err)
	}
	if !result.Reopened {
		t.Fatal("SQLite-to-PostgreSQL restore did not report the live connection ready")
	}
	var restored model.User
	if err := db.First(&restored, user.ID).Error; err != nil {
		t.Fatalf("load imported SQLite user: %v", err)
	}
	if restored.Email != user.Email {
		t.Fatalf("imported email = %q, want %q", restored.Email, user.Email)
	}
	var restoredCredential model.PasskeyCredential
	if err := db.Where("user_id = ?", user.ID).First(&restoredCredential).Error; err != nil {
		t.Fatalf("load imported SQLite passkey: %v", err)
	}
	if !bytes.Equal(restoredCredential.Credential, credential.Credential) {
		t.Fatalf("imported credential = %v, want %v", restoredCredential.Credential, credential.Credential)
	}
	var restoredSubscription model.Subscription
	if err := db.First(&restoredSubscription, subscription.ID).Error; err != nil {
		t.Fatalf("load imported SQLite subscription: %v", err)
	}
	if restoredSubscription.Amount != subscription.Amount || restoredSubscription.NotifyEnabled == nil || *restoredSubscription.NotifyEnabled {
		t.Fatalf("imported subscription = amount %v notify enabled %v, want amount %v and false", restoredSubscription.Amount, restoredSubscription.NotifyEnabled, subscription.Amount)
	}
	if err := db.Where("username = ?", existing.Username).First(&model.User{}).Error; err == nil {
		t.Fatalf("pre-import PostgreSQL user %q survived restore", existing.Username)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("look up pre-import user: %v", err)
	}
}

func openBackupPostgresTestDB(t *testing.T) *gorm.DB {
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
	schema := "subdux_backup_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	if err := migrations.Run(db, migrations.SecretCodec{
		Encrypt: pkg.EncryptSystemSettingValue,
		Decrypt: pkg.DecryptSystemSettingValue,
	}); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("migrate isolated PostgreSQL database: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close isolated PostgreSQL database: %v", err)
		}
	})
	return db
}
