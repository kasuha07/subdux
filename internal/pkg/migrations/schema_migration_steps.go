package migrations

import (
	"fmt"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

func migrateSubscriptionEventsSchema(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.SubscriptionEvent{},
		&model.SubscriptionActionSnooze{},
	)
}

func migrateNotificationOutboxLeases(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.BackgroundTaskLease{},
		&model.NotificationOutbox{},
		&model.NotificationLog{},
	)
}

func migrateAPIKeyKindAndAudit(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.APIKey{}, &model.AuditEvent{}); err != nil {
		return err
	}
	return db.Model(&model.APIKey{}).
		Where("key_kind IS NULL OR TRIM(key_kind) = ''").
		Update("key_kind", "api_integration").Error
}

func migrateManualRenewDailyNotificationPolicy(db *gorm.DB) error {
	return db.AutoMigrate(&model.NotificationPolicy{})
}

func migrateMCPIdempotencyKeys(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.MCPIdempotencyKey{}); err != nil {
		return err
	}
	// Idempotency correctness depends on this unique index actually existing:
	// it is the backstop that prevents two concurrent retries from both
	// inserting a record for the same key. Fail loudly at startup rather than
	// silently running with a weakened guarantee.
	if !db.Migrator().HasIndex(&model.MCPIdempotencyKey{}, "idx_mcp_idempotency_user_key") {
		return fmt.Errorf("expected unique index idx_mcp_idempotency_user_key was not created")
	}
	return nil
}

// migratePerformanceCompositeIndexes adds composite indexes that cover the hot
// read queries (lifecycle reconcile, dashboard summary, action center, price
// history, annual-growth baseline). Under SQLite's single-writer queue an
// unindexed scan directly lengthens the serialized queue, so these indexes are
// pure throughput wins with no behavioral change. AutoMigrate creates only the
// indexes that are missing; the HasIndex assertions fail loudly if a composite
// index the hot paths rely on did not materialize.
func migratePerformanceCompositeIndexes(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.Subscription{},
		&model.SubscriptionEvent{},
		&model.NotificationLog{},
	); err != nil {
		return err
	}

	required := []struct {
		model interface{}
		name  string
	}{
		{&model.Subscription{}, "idx_subscriptions_user_status_billing"},
		{&model.Subscription{}, "idx_subscriptions_user_next_billing"},
		{&model.SubscriptionEvent{}, "idx_subscription_events_user_sub_created"},
		{&model.NotificationLog{}, "idx_notification_logs_user_status_sent"},
		{&model.NotificationLog{}, "idx_notification_logs_user_sub_channel_sent"},
	}
	for _, index := range required {
		if !db.Migrator().HasIndex(index.model, index.name) {
			return fmt.Errorf("expected index %s was not created", index.name)
		}
	}
	return nil
}

func migrateUSDBaseExchangeRates(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.ExchangeRate{}) {
		return db.AutoMigrate(&model.ExchangeRate{})
	}

	var hasBaseCurrencyColumn int
	if err := db.Raw(
		"SELECT COUNT(1) FROM pragma_table_info(?) WHERE name = ?",
		"exchange_rates",
		"base_currency",
	).Scan(&hasBaseCurrencyColumn).Error; err != nil {
		return err
	}
	if hasBaseCurrencyColumn == 0 {
		return db.AutoMigrate(&model.ExchangeRate{})
	}

	type legacyExchangeRate struct {
		ID             uint
		BaseCurrency   string
		TargetCurrency string
		Rate           float64
		Source         string
		FetchedAt      time.Time
		CreatedAt      time.Time
		UpdatedAt      time.Time
	}

	var existing []legacyExchangeRate
	if err := db.Raw(`
		SELECT id, base_currency, target_currency, rate, source, fetched_at, created_at, updated_at
		FROM exchange_rates
	`).Scan(&existing).Error; err != nil {
		return err
	}

	type selectedRate struct {
		rate      legacyExchangeRate
		fetchedAt time.Time
	}
	usdRates := make(map[string]selectedRate)
	for _, rate := range existing {
		base := normalizeMigrationCurrency(rate.BaseCurrency)
		target := normalizeMigrationCurrency(rate.TargetCurrency)
		if rate.Rate <= 0 || base == "" || target == "" || base == target {
			continue
		}

		usdRate := rate
		switch {
		case base == "USD":
			usdRate.TargetCurrency = strings.ToLower(target)
		case target == "USD":
			usdRate.TargetCurrency = strings.ToLower(base)
			usdRate.Rate = 1 / rate.Rate
		default:
			continue
		}

		key := usdRate.TargetCurrency
		selected, exists := usdRates[key]
		if !exists || rate.FetchedAt.After(selected.fetchedAt) {
			usdRate.ID = 0
			usdRates[key] = selectedRate{rate: usdRate, fetchedAt: rate.FetchedAt}
		}
	}

	for _, rate := range existing {
		base := normalizeMigrationCurrency(rate.BaseCurrency)
		target := normalizeMigrationCurrency(rate.TargetCurrency)
		if rate.Rate <= 0 || base == "" || target == "" || base == target || base == "USD" || target == "USD" {
			continue
		}
		baseRate, ok := usdRates[strings.ToLower(base)]
		if !ok || baseRate.rate.Rate <= 0 {
			continue
		}

		usdRate := rate
		usdRate.ID = 0
		usdRate.TargetCurrency = strings.ToLower(target)
		usdRate.Rate = baseRate.rate.Rate * rate.Rate

		key := usdRate.TargetCurrency
		selected, exists := usdRates[key]
		if !exists || rate.FetchedAt.After(selected.fetchedAt) {
			usdRates[key] = selectedRate{rate: usdRate, fetchedAt: rate.FetchedAt}
		}
	}

	if err := db.Migrator().DropTable(&model.ExchangeRate{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&model.ExchangeRate{}); err != nil {
		return err
	}
	for _, selected := range usdRates {
		rate := model.ExchangeRate{
			TargetCurrency: selected.rate.TargetCurrency,
			Rate:           selected.rate.Rate,
			Source:         selected.rate.Source,
			FetchedAt:      selected.rate.FetchedAt,
			CreatedAt:      selected.rate.CreatedAt,
			UpdatedAt:      selected.rate.UpdatedAt,
		}
		if err := db.Create(&rate).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeMigrationCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}
