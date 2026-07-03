package migrations

import "time"

const schemaMigrationTableName = "schema_migrations"
const schemaMigrationLockTableName = "schema_migration_locks"

type schemaMigrationRecord struct {
	Name      string    `gorm:"primaryKey;size:191"`
	Checksum  string    `gorm:"not null;size:64;default:''"`
	Dirty     bool      `gorm:"not null;default:false"`
	AppliedAt time.Time `gorm:"not null"`
}

func (schemaMigrationRecord) TableName() string {
	return schemaMigrationTableName
}

type schemaMigrationLock struct {
	ID       uint      `gorm:"primaryKey"`
	LockedAt time.Time `gorm:"not null"`
}

func (schemaMigrationLock) TableName() string {
	return schemaMigrationLockTableName
}
