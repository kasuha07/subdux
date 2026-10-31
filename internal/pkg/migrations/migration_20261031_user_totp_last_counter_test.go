package migrations

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUserTOTPLastCounterMigrationAddsColumnDefaultingZero(t *testing.T) {
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

	if err := migrateUserTOTPLastCounter(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateUserTOTPLastCounter(db); err != nil {
		t.Fatal("migration is not idempotent: ", err)
	}
	var saved struct{ TotpLastCounter int64 }
	if err := db.Table("users").Select("totp_last_counter").Where("id = 1").Scan(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.TotpLastCounter != 0 {
		t.Fatalf("existing users totp_last_counter = %d, want 0", saved.TotpLastCounter)
	}
}
