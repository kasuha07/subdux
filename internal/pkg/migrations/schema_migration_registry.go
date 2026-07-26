package migrations

import (
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

type schemaMigration struct {
	Name                     string
	Checksum                 string
	Run                      func(db *gorm.DB) error
	DisableSQLiteForeignKeys bool
	Destructive              bool
	DiscardPolicy            string
}

var schemaMigrations = []schemaMigration{
	{Name: "20260512_01_create_missing_tables", Checksum: "245c0c8016fb52e36d24e5fc475208d011aa694f0c8ea5b74310e4fad99a2e57", Run: createMissingTables},
	{Name: "20260512_02_subscription_lifecycle_backfill", Checksum: "5808302cc9f5c723cf7fdc0a57abc88e6655195ab37172c9d11ee9e2d1ee9ef8", Run: backfillSubscriptionLifecycleFields},
	{Name: "20260512_03_sqlite_integrity_hardening", Checksum: "3fc886fe568666161a3198031cb174a76f63aa5fc0537de256b24d3faa4a68cc", Run: migrateSQLiteIntegrityHardening, DisableSQLiteForeignKeys: true},
	{Name: "20260512_04_auto_migrate_latest_schema", Checksum: "4eb2d348b78dc3c8216fb3d3e0d61754c48a231ea8effd23174f879399cba720", Run: autoMigrate20260512ApplicationSchema},
	{
		Name:          "20260525_00_subscription_event_orphans",
		Checksum:      "c8be95386abc60e4a6c4a9e3debd55ca05f39ac9879bfec2e5e678be96f8a544",
		Run:           cleanupSubscriptionEventOrphans,
		Destructive:   true,
		DiscardPolicy: "Preserve subscription event rows where possible. Delete events whose user_id has no parent user, and clear subscription_id for events that reference a missing or different-user subscription so SQLite foreign key constraints can be enforced.",
	},
	{Name: "20260525_01_subscription_events", Checksum: "b562049e926f70372b47f45cd680dc13fa21c78b9113741a1ab472ee537aeb74", Run: migrateSubscriptionEventsSchema},
	{Name: "20260527_01_subscription_action_snoozes", Checksum: "21b028f8c331b01ccad0b0c9f8c90c913bbf6228fa9138afeb937b04aac8056c", Run: migrateSubscriptionEventsSchema},
	{Name: "20260622_01_notification_outbox_leases", Checksum: "774a578619b78de5464eb0f0a2ba3a90b0a792b03f615f930f15cd8a57f325d0", Run: migrateNotificationOutboxLeases},
	{Name: "20260623_01_api_key_kind_and_audit", Checksum: "bf185c37f9c1889ac9b2eb12e593d4481b5df5b53367be061900306501bf530e", Run: migrateAPIKeyKindAndAudit},
	{Name: "20260628_01_manual_renew_daily_notifications", Checksum: "56b5d9babd3f0ad8b0971aa5f1a7761970027102ad012bc4adb00caa4c056500", Run: migrateManualRenewDailyNotificationPolicy},
	{Name: "20260628_02_mcp_idempotency_keys", Checksum: "d0198cc337bda90a1531f6e419f7a538cb506d24af6b1e4686f96ebe29c6762d", Run: migrateMCPIdempotencyKeys},
	{Name: "20260628_03_performance_composite_indexes", Checksum: "271bab2eac3f658250896444c658f5bb5a730cd57f2eed77cfb7fd3e9bbfbc42", Run: migratePerformanceCompositeIndexes},
	{
		Name:          "20260703_01_usd_base_exchange_rates",
		Checksum:      "8444b4c52d92a7ff1ed722104a330c7be64a2f9e4828bead6ae133d479dcaa52",
		Run:           migrateUSDBaseExchangeRates,
		Destructive:   true,
		DiscardPolicy: "Rebuild exchange_rates as USD-base only. Preserve direct USD pairs and rates derivable through an existing USD pair; discard invalid rows, self-pairs, and cross-rates that cannot be expressed from available USD data.",
	},
	{Name: "20260707_01_notification_quiet_hours", Checksum: "8f4f38eefaf226063a55efab892d433c722df441a319553c8ca6cff7ab402e74", Run: migrateNotificationQuietHours},
	{Name: "20260713_01_backup_destinations", Checksum: "79a58cf376923d88b3640d5590ddf0635a8332997cf84de4004ee002dc57b55f", Run: migrateBackupDestinations},
	{Name: "20260717_01_backup_run_state", Checksum: "e02814fe9c2d974ebfa66222f0bdae6dcf2c2e9b396440dacea29b31466b5ef2", Run: migrateBackupRunState},
	{
		Name:          "20260726_01_backup_per_destination_schedule",
		Checksum:      "5f5862fccea6706c59887670a1a380b7379744edea91695862f86f7d7bf03947",
		Run:           migrateBackupPerDestinationSchedule,
		Destructive:   true,
		DiscardPolicy: "Fold the global backup schedule into every destination's config (time of day, include assets, archive encryption and its password), preserving the old effective state by disabling destinations when the global schedule was off, then delete the retired backup_schedule_enabled, backup_time_of_day, backup_include_assets, backup_encrypt_enabled, backup_encryption_password, backup_last_run_at, backup_last_status and backup_last_error settings rows.",
	},
}

func autoMigrate20260512ApplicationSchema(db *gorm.DB) error {
	return db.AutoMigrate(migration20260512ApplicationModels()...)
}

// migration20260512ApplicationModels is the model list for the historical
// 20260512 bootstrap migrations. Do not append new application models here;
// add a dedicated migration with fixed structs or explicit SQL instead.
func migration20260512ApplicationModels() []interface{} {
	return []interface{}{
		&model.User{},
		&model.EmailVerificationCode{},
		&model.Subscription{},
		&model.SystemSetting{},
		&model.UserPreference{},
		&model.UserCurrency{},
		&model.UserBackupCode{},
		&model.PasskeyCredential{},
		&model.OIDCConnection{},
		&model.Category{},
		&model.PaymentMethod{},
		&model.NotificationChannel{},
		&model.NotificationPolicy{},
		&model.NotificationLog{},
		&model.NotificationTemplate{},
		&model.APIKey{},
		&model.RefreshToken{},
		&model.CalendarToken{},
		&model.AuditEvent{},
		&model.MCPIdempotencyKey{},
	}
}
