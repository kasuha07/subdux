package migrations

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUserPasswordGeneratedMigrationAddsColumnDefaultingFalse(t *testing.T) {
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

	if err := migrateUserPasswordGenerated(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateUserPasswordGenerated(db); err != nil {
		t.Fatal("migration is not idempotent: ", err)
	}
	if !db.Migrator().HasColumn("users", "password_generated") {
		t.Fatal("users.password_generated was not added")
	}
	var saved struct{ PasswordGenerated bool }
	if err := db.Table("users").Select("password_generated").Where("id = 1").Scan(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.PasswordGenerated {
		t.Fatal("existing users must default to password_generated = false")
	}
}
