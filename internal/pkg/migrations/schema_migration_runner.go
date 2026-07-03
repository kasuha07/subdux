package migrations

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const schemaMigrationLockID = 1

func nowUTC() time.Time {
	return time.Now().UTC()
}

func Run(db *gorm.DB) error {
	if err := ensureSchemaMigrationMetadata(db); err != nil {
		return err
	}
	if err := validateSchemaMigrationRecords(db); err != nil {
		return err
	}

	for _, migration := range schemaMigrations {
		if err := runSchemaMigration(db, migration); err != nil {
			return err
		}
	}

	return validateSQLiteForeignKeys(db)
}

func ensureSchemaMigrationMetadata(db *gorm.DB) error {
	if err := db.AutoMigrate(&schemaMigrationRecord{}, &schemaMigrationLock{}); err != nil {
		return fmt.Errorf("auto-migrate schema migration metadata: %w", err)
	}
	return nil
}

func validateSchemaMigrationRecords(db *gorm.DB) error {
	known := make(map[string]schemaMigration, len(schemaMigrations))
	for _, migration := range schemaMigrations {
		known[migration.Name] = migration
	}

	var records []schemaMigrationRecord
	if err := db.Find(&records).Error; err != nil {
		return fmt.Errorf("load schema migration records: %w", err)
	}
	for _, record := range records {
		migration, ok := known[record.Name]
		if !ok {
			return fmt.Errorf("unknown schema migration record %s; this database may be newer than the binary", record.Name)
		}
		if record.Dirty {
			return fmt.Errorf("schema migration %s is marked dirty; restore from backup or repair the migration record before startup", record.Name)
		}
		if record.Checksum != "" && record.Checksum != migration.Checksum {
			return fmt.Errorf("schema migration %s checksum mismatch: database=%s binary=%s", record.Name, record.Checksum, migration.Checksum)
		}
	}
	return nil
}

func runSchemaMigration(db *gorm.DB, migration schemaMigration) error {
	run := func(session *gorm.DB) error {
		return runSchemaMigrationTransaction(session, migration)
	}
	if migration.DisableSQLiteForeignKeys {
		return withSQLiteForeignKeysDisabled(db, run)
	}
	return run(db)
}

func runSchemaMigrationTransaction(db *gorm.DB, migration schemaMigration) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := acquireSchemaMigrationLock(tx); err != nil {
			return fmt.Errorf("acquire schema migration lock: %w", err)
		}

		record, applied, err := loadSchemaMigrationRecord(tx, migration.Name)
		if err != nil {
			return err
		}
		if applied {
			return validateAppliedSchemaMigration(tx, migration, record)
		}

		now := nowUTC()
		record = schemaMigrationRecord{
			Name:      migration.Name,
			Checksum:  migration.Checksum,
			Dirty:     true,
			AppliedAt: now,
		}
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("create dirty migration record %s: %w", migration.Name, err)
		}

		if err := migration.Run(tx); err != nil {
			return fmt.Errorf("apply migration %s: %w", migration.Name, err)
		}

		if err := tx.Model(&schemaMigrationRecord{}).
			Where("name = ?", migration.Name).
			Updates(map[string]interface{}{
				"checksum":   migration.Checksum,
				"dirty":      false,
				"applied_at": nowUTC(),
			}).Error; err != nil {
			return fmt.Errorf("record migration %s: %w", migration.Name, err)
		}
		return nil
	})
}

func acquireSchemaMigrationLock(tx *gorm.DB) error {
	lock := schemaMigrationLock{ID: schemaMigrationLockID, LockedAt: nowUTC()}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"locked_at"}),
	}).Create(&lock).Error
}

func loadSchemaMigrationRecord(tx *gorm.DB, name string) (schemaMigrationRecord, bool, error) {
	var record schemaMigrationRecord
	err := tx.Where("name = ?", name).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return schemaMigrationRecord{}, false, nil
	}
	if err != nil {
		return schemaMigrationRecord{}, false, fmt.Errorf("check migration %s: %w", name, err)
	}
	return record, true, nil
}

func validateAppliedSchemaMigration(tx *gorm.DB, migration schemaMigration, record schemaMigrationRecord) error {
	if record.Dirty {
		return fmt.Errorf("schema migration %s is marked dirty", migration.Name)
	}
	if record.Checksum != "" && record.Checksum != migration.Checksum {
		return fmt.Errorf("schema migration %s checksum mismatch: database=%s binary=%s", migration.Name, record.Checksum, migration.Checksum)
	}
	if record.Checksum == "" {
		if err := tx.Model(&schemaMigrationRecord{}).
			Where("name = ?", migration.Name).
			Update("checksum", migration.Checksum).Error; err != nil {
			return fmt.Errorf("backfill checksum for migration %s: %w", migration.Name, err)
		}
	}
	return nil
}
