package subscription

import (
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/pkg/migrations"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func TestPostgresLifecycleSweepSkipsLockedRowAndReconcilesItLater(t *testing.T) {
	db := openSubscriptionPostgresTestDB(t)
	now := time.Date(2026, time.September, 24, 12, 0, 0, 0, time.UTC)
	overdue := now.AddDate(0, 0, -2)
	user := model.User{Username: "sweep-user", Email: "sweep@example.com", Password: "hash", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	subs := []model.Subscription{
		{UserID: user.ID, Name: "locked due", Amount: 1, Currency: "USD", Enabled: true, Status: subscriptionStatusActive, RenewalMode: renewalModeManualRenew, BillingType: billingTypeRecurring, NextBillingDate: &overdue},
		{UserID: user.ID, Name: "unlocked due", Amount: 1, Currency: "USD", Enabled: true, Status: subscriptionStatusActive, RenewalMode: renewalModeManualRenew, BillingType: billingTypeRecurring, NextBillingDate: &overdue},
	}
	if err := db.Create(&subs).Error; err != nil {
		t.Fatalf("create due subscriptions: %v", err)
	}

	locker := db.Begin()
	if locker.Error != nil {
		t.Fatalf("begin row-locking transaction: %v", locker.Error)
	}
	defer locker.Rollback()
	var locked model.Subscription
	if err := locker.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, subs[0].ID).Error; err != nil {
		t.Fatalf("lock first subscription: %v", err)
	}

	service := NewService(db)
	if err := service.reconcileDueLifecyclesPostgres(now); err != nil {
		t.Fatalf("sweep while first subscription is locked: %v", err)
	}
	var afterFirstSweep []model.Subscription
	if err := db.Order("id ASC").Find(&afterFirstSweep).Error; err != nil {
		t.Fatalf("reload subscriptions after first sweep: %v", err)
	}
	if len(afterFirstSweep) != 2 || afterFirstSweep[0].Status != subscriptionStatusActive || afterFirstSweep[1].Status != subscriptionStatusEnded {
		t.Fatalf("statuses after locked sweep = %+v, want active then ended", afterFirstSweep)
	}
	if afterFirstSweep[0].Revision != 1 || afterFirstSweep[1].Revision != 2 {
		t.Fatalf("revisions after locked sweep = %d, %d; want 1, 2", afterFirstSweep[0].Revision, afterFirstSweep[1].Revision)
	}

	if err := locker.Rollback().Error; err != nil {
		t.Fatalf("release first subscription lock: %v", err)
	}
	if err := service.reconcileDueLifecyclesPostgres(now); err != nil {
		t.Fatalf("sweep after releasing lock: %v", err)
	}
	var afterSecondSweep []model.Subscription
	if err := db.Order("id ASC").Find(&afterSecondSweep).Error; err != nil {
		t.Fatalf("reload subscriptions after second sweep: %v", err)
	}
	if len(afterSecondSweep) != 2 || afterSecondSweep[0].Status != subscriptionStatusEnded || afterSecondSweep[1].Status != subscriptionStatusEnded {
		t.Fatalf("statuses after unlocked sweep = %+v, want both ended", afterSecondSweep)
	}
	if afterSecondSweep[0].Revision != 2 || afterSecondSweep[1].Revision != 2 {
		t.Fatalf("revisions after unlocked sweep = %d, %d; want both 2", afterSecondSweep[0].Revision, afterSecondSweep[1].Revision)
	}
}

func openSubscriptionPostgresTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("SUBDUX_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set SUBDUX_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatalf("SUBDUX_TEST_POSTGRES_DSN must be a postgres:// or postgresql:// URL")
	}

	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL admin connection: %v", err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatalf("access PostgreSQL admin connection: %v", err)
	}
	schema := "subdux_sweep_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec("CREATE SCHEMA \"" + schema + "\"").Error; err != nil {
		_ = adminSQL.Close()
		t.Fatalf("create isolated PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() {
		if err := admin.Exec("DROP SCHEMA IF EXISTS \"" + schema + "\" CASCADE").Error; err != nil {
			t.Errorf("drop isolated PostgreSQL schema: %v", err)
		}
		_ = adminSQL.Close()
	})

	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("access isolated PostgreSQL database: %v", err)
	}
	t.Setenv("JWT_SECRET", "postgres-integration-secret-0123456789")
	if err := migrations.Run(db, migrations.SecretCodec{
		Encrypt: pkg.EncryptSystemSettingValue,
		Decrypt: pkg.DecryptSystemSettingValue,
	}); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("migrate isolated PostgreSQL database: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close isolated PostgreSQL database: %v", err)
		}
	})
	return db
}
