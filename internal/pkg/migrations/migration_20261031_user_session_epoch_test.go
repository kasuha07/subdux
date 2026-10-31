package migrations

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUserSessionEpochMigrationAddsColumnDefaultingZero(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE users (id integer PRIMARY KEY, username text NOT NULL, email text NOT NULL, password text NOT NULL)`,
		`INSERT INTO users (id, username, email, password) VALUES (1, 'legacy', 'legacy@example.com', 'hash')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := migrateUserSessionEpoch(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateUserSessionEpoch(db); err != nil {
		t.Fatal("migration is not idempotent: ", err)
	}
	var saved struct{ SessionEpoch uint64 }
	if err := db.Table("users").Select("session_epoch").Where("id = 1").Scan(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.SessionEpoch != 0 {
		t.Fatalf("existing users session_epoch = %d, want 0 so pre-upgrade tokens stay valid", saved.SessionEpoch)
	}
}
