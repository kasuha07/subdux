package migrations

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSubscriptionPendingPriceMigrationAddsNullableColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE subscriptions (id integer PRIMARY KEY, user_id integer NOT NULL, name text NOT NULL, amount real NOT NULL)`,
		`INSERT INTO subscriptions (id, user_id, name, amount) VALUES (1, 10, 'Legacy', 9.99)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := migrateSubscriptionPendingPrice(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateSubscriptionPendingPrice(db); err != nil {
		t.Fatal("migration is not idempotent: ", err)
	}

	for _, column := range []string{"pending_amount", "pending_from"} {
		if !db.Migrator().HasColumn("subscriptions", column) {
			t.Fatalf("subscriptions.%s was not added", column)
		}
	}

	var saved struct {
		Amount        float64
		PendingAmount *float64
		PendingFrom   *string
	}
	if err := db.Raw(`SELECT amount, pending_amount, pending_from FROM subscriptions WHERE id = 1`).Scan(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Amount != 9.99 || saved.PendingAmount != nil || saved.PendingFrom != nil {
		t.Fatalf("existing row = %+v, want amount kept and no scheduled price", saved)
	}
}

func TestSubscriptionPendingPriceMigrationSkipsMissingTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "empty.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateSubscriptionPendingPrice(db); err != nil {
		t.Fatal(err)
	}
}
