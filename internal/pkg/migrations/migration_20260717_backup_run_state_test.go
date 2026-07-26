package migrations

import (
	"reflect"
	"testing"
)

func TestMigrateBackupRunStateUsesFixedSchemas(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := db.AutoMigrate(&backupDestinationMigration20260713{}); err != nil {
		t.Fatalf("migrate backup destinations v1: %v", err)
	}

	if err := migrateBackupRunState(db); err != nil {
		t.Fatalf("migrateBackupRunState() error = %v", err)
	}

	var destinationColumns []string
	if err := db.Raw("SELECT name FROM pragma_table_info(?) ORDER BY cid", "backup_destinations").Pluck("name", &destinationColumns).Error; err != nil {
		t.Fatalf("inspect backup_destinations schema: %v", err)
	}
	wantDestinationColumns := []string{
		"id",
		"revision",
		"type",
		"enabled",
		"config",
		"last_run_at",
		"last_status",
		"last_error",
		"sort_order",
		"created_at",
		"updated_at",
		"last_retention_status",
		"last_retention_error",
		"last_bookkeeping_status",
		"last_bookkeeping_error",
	}
	if !reflect.DeepEqual(destinationColumns, wantDestinationColumns) {
		t.Fatalf("backup_destinations columns = %v, want %v", destinationColumns, wantDestinationColumns)
	}

	for _, table := range []string{"backup_runs", "backup_run_destinations"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("table %s does not exist", table)
		}
	}
	if !db.Migrator().HasIndex(&backupRunDestinationMigration20260717{}, "idx_backup_run_destinations") {
		t.Fatal("backup_run_destinations unique index is missing")
	}
}
