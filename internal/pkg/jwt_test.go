package pkg

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func newJWTTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-jwt-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func TestInitJWTSecretGeneratesAndPersistsSecretOnFirstBoot(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	jwtSecretFromDB = ""

	db := newJWTTestDB(t)
	if err := InitJWTSecret(db); err != nil {
		t.Fatalf("InitJWTSecret() error = %v, want nil", err)
	}

	if len(jwtSecretFromDB) < minJWTSecretLength {
		t.Fatalf("generated secret length = %d, want at least %d", len(jwtSecretFromDB), minJWTSecretLength)
	}

	var stored model.SystemSetting
	if err := db.Where("key = ?", jwtSecretKey).First(&stored).Error; err != nil {
		t.Fatalf("failed to load persisted JWT secret: %v", err)
	}
	if stored.Value != jwtSecretFromDB {
		t.Fatalf("persisted secret mismatch: got %q, want %q", stored.Value, jwtSecretFromDB)
	}
}

func TestInitJWTSecretRejectsWeakEnvironmentSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "short-secret")
	jwtSecretFromDB = ""

	db := newJWTTestDB(t)
	err := InitJWTSecret(db)
	if err == nil {
		t.Fatal("InitJWTSecret() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "at least") {
		t.Fatalf("InitJWTSecret() error = %q, want length validation error", err.Error())
	}
}

func TestInitJWTSecretRejectsWeakDatabaseSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	jwtSecretFromDB = ""

	db := newJWTTestDB(t)
	if err := db.Create(&model.SystemSetting{Key: jwtSecretKey, Value: "weak"}).Error; err != nil {
		t.Fatalf("failed to seed weak JWT secret: %v", err)
	}

	err := InitJWTSecret(db)
	if err == nil {
		t.Fatal("InitJWTSecret() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "at least") {
		t.Fatalf("InitJWTSecret() error = %q, want length validation error", err.Error())
	}
}
