package pkg

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func newSubscriptionLifecycleMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-subscription-lifecycle-migration-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&model.Subscription{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func TestBackfillSubscriptionLifecycleFieldsRewritesLegacyBuyoutRecords(t *testing.T) {
	db := newSubscriptionLifecycleMigrationTestDB(t)

	nextBillingDate := time.Date(2025, time.January, 15, 9, 0, 0, 0, time.UTC)
	sub := model.Subscription{
		UserID:          1,
		Name:            "Legacy buyout",
		Amount:          99,
		Currency:        "USD",
		Enabled:         true,
		BillingType:     "one_time",
		NextBillingDate: &nextBillingDate,
		CreatedAt:       time.Date(2024, time.December, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2025, time.January, 16, 12, 0, 0, 0, time.UTC),
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}

	if err := backfillSubscriptionLifecycleFields(db); err != nil {
		t.Fatalf("backfillSubscriptionLifecycleFields() error = %v", err)
	}

	var migrated model.Subscription
	if err := db.First(&migrated, sub.ID).Error; err != nil {
		t.Fatalf("failed to reload subscription: %v", err)
	}

	if got, want := migrated.BillingType, subscriptionBillingTypeRecurring; got != want {
		t.Fatalf("billing_type = %q, want %q", got, want)
	}
	if got, want := migrated.Status, subscriptionStatusEnded; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if got, want := migrated.RenewalMode, subscriptionRenewalModeCancelEnd; got != want {
		t.Fatalf("renewal_mode = %q, want %q", got, want)
	}
	if migrated.Enabled {
		t.Fatal("enabled = true, want false")
	}
	if migrated.EndsAt == nil {
		t.Fatal("ends_at = nil, want derived final boundary")
	}
	if got, want := migrated.EndsAt.Format("2006-01-02"), "2025-01-15"; got != want {
		t.Fatalf("ends_at = %s, want %s", got, want)
	}
	if migrated.RecurrenceType != "" {
		t.Fatalf("recurrence_type = %q, want empty", migrated.RecurrenceType)
	}
	if migrated.IntervalCount != nil {
		t.Fatalf("interval_count = %v, want nil", *migrated.IntervalCount)
	}
	if migrated.IntervalUnit != "" {
		t.Fatalf("interval_unit = %q, want empty", migrated.IntervalUnit)
	}
	if migrated.MonthlyDay != nil || migrated.YearlyMonth != nil || migrated.YearlyDay != nil {
		t.Fatal("legacy buyout schedule fields should be cleared")
	}
}
