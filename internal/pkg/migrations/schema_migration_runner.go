package migrations

import (
	"errors"
	"fmt"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const schemaMigrationLockID = 1

// SecretCodec gives migrations read/write access to values that are encrypted
// at rest. It is injected rather than imported because the package that owns
// the encryption key (internal/pkg) already imports this one, so a direct
// import would be a cycle.
//
// Encrypt and Decrypt must be inverses over the same envelope used by the
// running application; otherwise a migration that rewrites a secret would leave
// it unreadable.
type SecretCodec struct {
	Encrypt func(string) (string, error)
	Decrypt func(string) (string, error)
}

// errSecretCodecUnavailable guards the migrations that touch secrets. Failing
// loudly is required: silently skipping the rewrite would leave a secret either
// unmigrated or, worse, written back in plaintext.
var errSecretCodecUnavailable = errors.New("schema migration requires a secret codec but none was provided")

func (c SecretCodec) decrypt(value string) (string, error) {
	if c.Decrypt == nil {
		return "", errSecretCodecUnavailable
	}
	return c.Decrypt(value)
}

func (c SecretCodec) encrypt(value string) (string, error) {
	if c.Encrypt == nil {
		return "", errSecretCodecUnavailable
	}
	return c.Encrypt(value)
}

// activeSecretCodec is set for the duration of a Run so individual migration
// funcs keep the plain func(db) signature the registry is built around.
var activeSecretCodec SecretCodec

func nowUTC() time.Time {
	return time.Now().UTC()
}

func Run(db *gorm.DB, codec SecretCodec) error {
	activeSecretCodec = codec
	defer func() { activeSecretCodec = SecretCodec{} }()

	if db.Dialector.Name() == "postgres" {
		if err := preparePostgresMigrationDependencies(db); err != nil {
			return err
		}
	}

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

	return validateForeignKeys(db)
}

// preparePostgresMigrationDependencies creates the parent tables needed by
// the historical bootstrap migration. PostgreSQL requires referenced tables
// to exist when it creates a foreign key; the original SQLite bootstrap order
// creates subscriptions before categories and payment methods.
func preparePostgresMigrationDependencies(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.User{}, &model.Category{}, &model.PaymentMethod{}); err != nil {
		return fmt.Errorf("prepare PostgreSQL migration dependencies: %w", err)
	}
	return nil
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
	if migration.DisableSQLiteForeignKeys && db.Dialector.Name() == "postgres" {
		// The published migration rebuilds tables to add SQLite-only constraints;
		// PostgreSQL gets those model constraints when the tables are created.
		migration.Run = func(*gorm.DB) error { return nil }
	}
	run := func(session *gorm.DB) error {
		return runSchemaMigrationTransaction(session, migration)
	}
	if migration.DisableSQLiteForeignKeys && db.Dialector.Name() == "sqlite" {
		_, applied, err := loadSchemaMigrationRecord(db, migration.Name)
		if err != nil {
			return err
		}
		if applied {
			return run(db)
		}
	}
	if migration.DisableSQLiteForeignKeys && db.Dialector.Name() == "sqlite" {
		return withSQLiteForeignKeysDisabled(db, run)
	}
	return run(db)
}

func validateForeignKeys(db *gorm.DB) error {
	switch db.Dialector.Name() {
	case "sqlite":
		return validateSQLiteForeignKeys(db)
	case "postgres":
		var violation struct {
			RelationName   string
			ForeignKeyName string
		}
		result := db.Raw(`
			SELECT conrelid::regclass::text AS relation_name, conname AS foreign_key_name
			FROM pg_constraint
			WHERE contype = 'f'
			  AND NOT convalidated
			  AND connamespace = (SELECT oid FROM pg_namespace WHERE nspname = current_schema())
			LIMIT 1
		`).Scan(&violation)
		if result.Error != nil {
			return fmt.Errorf("validate PostgreSQL foreign keys: %w", result.Error)
		}
		if result.RowsAffected > 0 {
			return fmt.Errorf("PostgreSQL foreign key %s on %s is not validated", violation.ForeignKeyName, violation.RelationName)
		}
		return nil
	default:
		return fmt.Errorf("unsupported database dialect %q", db.Dialector.Name())
	}
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
