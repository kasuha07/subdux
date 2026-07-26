package pkg

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

func TestReopenDatabaseSwapsPool(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "reopen-database-test-key")
	// JWT_SECRET is deliberately left unset so InitJWTSecret resolves the
	// secret from the database and the reopen has something to rehydrate.
	t.Setenv("JWT_SECRET", "")
	restoreGlobalJWTSecret(t)

	// The replacement database is built first so its JWT secret is not the one
	// left in the package global when the live database is opened below.
	replacementPath := filepath.Join(t.TempDir(), "subdux.db")
	replacementDB, err := openSQLiteDatabase(replacementPath)
	if err != nil {
		t.Fatalf("openSQLiteDatabase(replacement) error = %v", err)
	}
	if err := replacementDB.Create(&model.SystemSetting{Key: "restore_marker", Value: "replacement"}).Error; err != nil {
		t.Fatalf("seed replacement marker error = %v", err)
	}
	if err := InitJWTSecret(replacementDB); err != nil {
		t.Fatalf("InitJWTSecret(replacement) error = %v", err)
	}
	replacementSecret := string(GetJWTSecret())
	replacementSQLDB, err := replacementDB.DB()
	if err != nil {
		t.Fatalf("replacementDB.DB() error = %v", err)
	}
	if err := replacementSQLDB.Close(); err != nil {
		t.Fatalf("close replacement pool error = %v", err)
	}

	dataPath := t.TempDir()
	t.Setenv("DATA_PATH", dataPath)
	dbPath := filepath.Join(dataPath, "subdux.db")
	db, err := openSQLiteDatabase(dbPath)
	if err != nil {
		t.Fatalf("openSQLiteDatabase() error = %v", err)
	}
	if err := InitJWTSecret(db); err != nil {
		t.Fatalf("InitJWTSecret() error = %v", err)
	}
	if originalSecret := string(GetJWTSecret()); originalSecret == replacementSecret {
		t.Fatal("the two databases generated the same JWT secret; the test cannot detect a stale secret")
	}

	// Mirror what RestoreBackup does: close the live pool, then swap the file.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close live pool error = %v", err)
	}
	copyFileForTest(t, replacementPath, dbPath)

	// Restore runs against a WithContext session, not the root handle; the swap
	// still has to reach every holder of the shared *gorm.DB.
	if err := ReopenDatabase(db.WithContext(context.Background())); err != nil {
		t.Fatalf("ReopenDatabase() error = %v", err)
	}

	var marker model.SystemSetting
	if err := db.Where("key = ?", "restore_marker").First(&marker).Error; err != nil {
		t.Fatalf("read restore marker error = %v", err)
	}
	if marker.Value != "replacement" {
		t.Fatalf("restore_marker = %q, want %q", marker.Value, "replacement")
	}

	// Create goes through gorm's default transaction, which reaches BeginTx on
	// the pool wrapper. A missing BeginTx only fails here, never at compile time.
	if err := db.Create(&model.SystemSetting{Key: "written_after_reopen", Value: "ok"}).Error; err != nil {
		t.Fatalf("write after reopen error = %v", err)
	}

	reopenedSQLDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() after reopen error = %v", err)
	}
	if reopenedSQLDB == sqlDB {
		t.Fatal("db.DB() still returns the closed pool after reopen")
	}
	if err := reopenedSQLDB.Ping(); err != nil {
		t.Fatalf("ping after reopen error = %v", err)
	}
	t.Cleanup(func() { _ = reopenedSQLDB.Close() })

	if got := string(GetJWTSecret()); got != replacementSecret {
		t.Fatal("JWT secret was not reloaded from the replacement database")
	}
}

func TestReopenDatabaseWithoutWrapper(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "raw.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	if err := ReopenDatabase(db); !errors.Is(err, ErrDatabaseNotReopenable) {
		t.Fatalf("ReopenDatabase() error = %v, want ErrDatabaseNotReopenable", err)
	}
}

// restoreGlobalJWTSecret keeps InitJWTSecret's package-level side effect from
// leaking into the other tests in this package.
func restoreGlobalJWTSecret(t *testing.T) {
	t.Helper()

	previous := jwtSecretFromDB
	t.Cleanup(func() { jwtSecretFromDB = previous })
}

func copyFileForTest(t *testing.T, sourcePath string, targetPath string) {
	t.Helper()

	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatalf("open %q error = %v", sourcePath, err)
	}
	defer source.Close()

	target, err := os.Create(targetPath)
	if err != nil {
		t.Fatalf("create %q error = %v", targetPath, err)
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		t.Fatalf("copy %q to %q error = %v", sourcePath, targetPath, err)
	}
}
