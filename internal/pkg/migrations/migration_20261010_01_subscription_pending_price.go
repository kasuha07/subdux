package migrations

import "gorm.io/gorm"

// migrateSubscriptionPendingPrice adds the scheduled price change a
// subscription switches to once its introductory or trial price ends. Both
// columns are nullable and need no backfill: existing subscriptions have no
// scheduled change. Fixed SQL keeps this migration independent of future model
// changes; columns that already exist (fresh databases created from the
// current model) are left alone.
func migrateSubscriptionPendingPrice(db *gorm.DB) error {
	if !db.Migrator().HasTable("subscriptions") {
		return nil
	}

	amountType, dateType := "real", "datetime"
	if db.Dialector.Name() == "postgres" {
		amountType, dateType = "decimal", "timestamptz"
	}
	for _, column := range []struct {
		name       string
		columnType string
	}{
		{name: "pending_amount", columnType: amountType},
		{name: "pending_from", columnType: dateType},
	} {
		if db.Migrator().HasColumn("subscriptions", column.name) {
			continue
		}
		if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN " + column.name + " " + column.columnType).Error; err != nil {
			return err
		}
	}
	return nil
}
