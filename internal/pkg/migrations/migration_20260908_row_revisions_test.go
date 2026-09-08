package migrations

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRowRevisionMigrationPreservesLegacyDataAndIsRepeatable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	tables := []string{"subscriptions", "notification_policies", "notification_channels", "notification_templates", "categories", "payment_methods", "user_currencies", "system_settings"}
	for _, table := range tables {
		if err := db.Exec("CREATE TABLE " + table + " (id INTEGER PRIMARY KEY, value TEXT NOT NULL)").Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec("INSERT INTO " + table + " (id,value) VALUES (1, 'preserved')").Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateRowRevisions(db); err != nil {
		t.Fatal(err)
	}
	for _, table := range tables {
		var row struct {
			Value    string
			Revision uint64
		}
		if err := db.Table(table).Where("id = 1").Take(&row).Error; err != nil {
			t.Fatal(err)
		}
		if row.Value != "preserved" || row.Revision != 1 {
			t.Fatalf("%s: %+v", table, row)
		}
		if err := db.Table(table).Where("id = 1").Update("revision", 7).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec("INSERT INTO " + table + " (id,value) VALUES (2,'new')").Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateRowRevisions(db); err != nil {
		t.Fatal(err)
	}
	for _, table := range tables {
		var revisions []uint64
		if err := db.Table(table).Order("id").Pluck("revision", &revisions).Error; err != nil {
			t.Fatal(err)
		}
		if len(revisions) != 2 || revisions[0] != 7 || revisions[1] != 1 {
			t.Fatalf("%s revisions=%v", table, revisions)
		}
	}
}
