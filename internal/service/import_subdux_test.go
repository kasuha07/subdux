package service

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func newImportTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-import-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.UserPreference{},
		&model.UserCurrency{},
		&model.Category{},
		&model.PaymentMethod{},
		&model.Subscription{},
		&model.NotificationChannel{},
		&model.NotificationTemplate{},
		&model.NotificationPolicy{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func ptrString(value string) *string {
	return &value
}

func ptrInt(value int) *int {
	return &value
}

func ptrUint(value uint) *uint {
	return &value
}

func ptrBool(value bool) *bool {
	return &value
}

func sampleSubduxImportData() SubduxImportData {
	nextBilling := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	return SubduxImportData{
		Currencies: []model.UserCurrency{
			{Code: "USD", Symbol: "$", Alias: "US Dollar", SortOrder: 1},
		},
		Categories: []model.Category{
			{ID: 101, Name: "Video", SystemKey: ptrString("video"), DisplayOrder: 1},
		},
		PaymentMethods: []model.PaymentMethod{
			{ID: 201, Name: "Visa", SystemKey: ptrString("visa"), SortOrder: 1},
		},
		Subscriptions: []model.Subscription{
			{
				Name:            "Netflix",
				Amount:          15.99,
				Currency:        "USD",
				Enabled:         true,
				BillingType:     "recurring",
				RecurrenceType:  "interval",
				IntervalCount:   ptrInt(1),
				IntervalUnit:    "month",
				NextBillingDate: &nextBilling,
				Category:        "Video",
				CategoryID:      ptrUint(101),
				PaymentMethodID: ptrUint(201),
				NotifyEnabled:   ptrBool(true),
			},
		},
		Preference: &model.UserPreference{PreferredCurrency: "USD"},
		Notifications: SubduxNotificationImportData{
			Channels: []model.NotificationChannel{
				{Type: "webhook", Enabled: true, Config: "{\n  \"url\": \"https://example.com/hook\"\n}"},
			},
			Templates: []model.NotificationTemplate{
				{ChannelType: nil, Format: "plaintext", Template: "{{.SubscriptionName}} due soon"},
			},
			Policy: &model.NotificationPolicy{DaysBefore: 2, NotifyOnDueDay: true},
		},
	}
}

func TestImportFromSubduxPreviewDoesNotWrite(t *testing.T) {
	db := newImportTestDB(t)
	user := createTestUser(t, db)
	svc := NewImportService(db)

	resp, err := svc.ImportFromSubdux(user.ID, sampleSubduxImportData(), false)
	if err != nil {
		t.Fatalf("preview import failed: %v", err)
	}
	if resp.Preview == nil {
		t.Fatal("preview response should contain preview payload")
	}

	assertTableCount := func(table any, expected int64) {
		t.Helper()
		var count int64
		if err := db.Model(table).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
			t.Fatalf("failed to count records: %v", err)
		}
		if count != expected {
			t.Fatalf("record count = %d, want %d", count, expected)
		}
	}

	assertTableCount(&model.UserCurrency{}, 0)
	assertTableCount(&model.Category{}, 0)
	assertTableCount(&model.PaymentMethod{}, 0)
	assertTableCount(&model.Subscription{}, 0)
	assertTableCount(&model.NotificationChannel{}, 0)
	assertTableCount(&model.NotificationTemplate{}, 0)

	var prefCount int64
	if err := db.Model(&model.UserPreference{}).Where("user_id = ?", user.ID).Count(&prefCount).Error; err != nil {
		t.Fatalf("failed to count preferences: %v", err)
	}
	if prefCount != 0 {
		t.Fatalf("preference count = %d, want 0", prefCount)
	}

	var policyCount int64
	if err := db.Model(&model.NotificationPolicy{}).Where("user_id = ?", user.ID).Count(&policyCount).Error; err != nil {
		t.Fatalf("failed to count policies: %v", err)
	}
	if policyCount != 0 {
		t.Fatalf("policy count = %d, want 0", policyCount)
	}
}

func TestImportFromSubduxConfirmImportsAndUpdates(t *testing.T) {
	db := newImportTestDB(t)
	user := createTestUser(t, db)
	svc := NewImportService(db)

	if err := db.Create(&model.UserPreference{UserID: user.ID, PreferredCurrency: "EUR"}).Error; err != nil {
		t.Fatalf("failed to seed preference: %v", err)
	}
	if err := db.Create(&model.NotificationPolicy{UserID: user.ID, DaysBefore: 7, NotifyOnDueDay: false}).Error; err != nil {
		t.Fatalf("failed to seed policy: %v", err)
	}

	resp, err := svc.ImportFromSubdux(user.ID, sampleSubduxImportData(), true)
	if err != nil {
		t.Fatalf("confirm import failed: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("confirm response should contain result payload")
	}
	if resp.Result.Imported == 0 {
		t.Fatal("expected imported count to be greater than zero")
	}

	assertCount := func(table any, expected int64) {
		t.Helper()
		var count int64
		if err := db.Model(table).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
			t.Fatalf("failed to count records: %v", err)
		}
		if count != expected {
			t.Fatalf("count = %d, want %d", count, expected)
		}
	}

	assertCount(&model.UserCurrency{}, 1)
	assertCount(&model.Category{}, 1)
	assertCount(&model.PaymentMethod{}, 1)
	assertCount(&model.Subscription{}, 1)
	assertCount(&model.NotificationChannel{}, 1)
	assertCount(&model.NotificationTemplate{}, 1)
	assertCount(&model.UserPreference{}, 1)
	assertCount(&model.NotificationPolicy{}, 1)

	var pref model.UserPreference
	if err := db.Where("user_id = ?", user.ID).First(&pref).Error; err != nil {
		t.Fatalf("failed to query preference: %v", err)
	}
	if pref.PreferredCurrency != "USD" {
		t.Fatalf("preferred currency = %q, want %q", pref.PreferredCurrency, "USD")
	}

	var policy model.NotificationPolicy
	if err := db.Where("user_id = ?", user.ID).First(&policy).Error; err != nil {
		t.Fatalf("failed to query policy: %v", err)
	}
	if policy.DaysBefore != 2 || !policy.NotifyOnDueDay {
		t.Fatalf("policy = (%d, %v), want (2, true)", policy.DaysBefore, policy.NotifyOnDueDay)
	}
}

func TestImportFromSubduxReimportIsIdempotent(t *testing.T) {
	db := newImportTestDB(t)
	user := createTestUser(t, db)
	svc := NewImportService(db)
	data := sampleSubduxImportData()

	if _, err := svc.ImportFromSubdux(user.ID, data, true); err != nil {
		t.Fatalf("first import failed: %v", err)
	}
	second, err := svc.ImportFromSubdux(user.ID, data, true)
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}

	if second.Result == nil {
		t.Fatal("second import result should not be nil")
	}
	if second.Result.Imported != 0 {
		t.Fatalf("second import imported = %d, want 0", second.Result.Imported)
	}
	if second.Result.Skipped == 0 {
		t.Fatal("second import skipped should be greater than zero")
	}

	assertCount := func(table any) {
		t.Helper()
		var count int64
		if err := db.Model(table).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
			t.Fatalf("failed to count records: %v", err)
		}
		if count != 1 {
			t.Fatalf("count = %d, want 1", count)
		}
	}

	assertCount(&model.UserCurrency{})
	assertCount(&model.Category{})
	assertCount(&model.PaymentMethod{})
	assertCount(&model.Subscription{})
	assertCount(&model.NotificationChannel{})
	assertCount(&model.NotificationTemplate{})
	assertCount(&model.UserPreference{})
	assertCount(&model.NotificationPolicy{})
}

func TestImportFromSubduxInvalidFormat(t *testing.T) {
	db := newImportTestDB(t)
	user := createTestUser(t, db)
	svc := NewImportService(db)

	_, err := svc.ImportFromSubdux(user.ID, SubduxImportData{}, false)
	if !errors.Is(err, ErrInvalidSubduxImportFormat) {
		t.Fatalf("expected ErrInvalidSubduxImportFormat, got %v", err)
	}
}

func TestImportFromSubduxTooLarge(t *testing.T) {
	db := newImportTestDB(t)
	user := createTestUser(t, db)
	svc := NewImportService(db)

	data := SubduxImportData{
		Currencies:     make([]model.UserCurrency, maxSubduxImportItemsPerCollection+1),
		Categories:     []model.Category{},
		PaymentMethods: []model.PaymentMethod{},
		Subscriptions:  []model.Subscription{},
	}

	_, err := svc.ImportFromSubdux(user.ID, data, false)
	if !errors.Is(err, ErrSubduxImportTooLarge) {
		t.Fatalf("expected ErrSubduxImportTooLarge, got %v", err)
	}
}

func TestImportFromSubduxSkipsInvalidChannelType(t *testing.T) {
	db := newImportTestDB(t)
	user := createTestUser(t, db)
	svc := NewImportService(db)

	data := sampleSubduxImportData()
	data.Notifications.Channels = append(data.Notifications.Channels, model.NotificationChannel{
		Type:   "invalid-channel",
		Config: `{"foo":"bar"}`,
	})

	resp, err := svc.ImportFromSubdux(user.ID, data, true)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("result should not be nil")
	}

	var channelCount int64
	if err := db.Model(&model.NotificationChannel{}).Where("user_id = ?", user.ID).Count(&channelCount).Error; err != nil {
		t.Fatalf("failed to count channels: %v", err)
	}
	if channelCount != 1 {
		t.Fatalf("channel count = %d, want 1", channelCount)
	}
	if resp.Result.Skipped == 0 {
		t.Fatal("expected skipped count to be greater than zero for invalid channel")
	}
}

func TestImportFromSubduxChannelCanonicalIdempotent(t *testing.T) {
	db := newImportTestDB(t)
	user := createTestUser(t, db)
	svc := NewImportService(db)

	if err := db.Create(&model.NotificationChannel{
		UserID:  user.ID,
		Type:    "webhook",
		Enabled: true,
		Config: `{
  "method": "POST",
  "url": "https://example.com/hook"
}`,
	}).Error; err != nil {
		t.Fatalf("failed to seed channel: %v", err)
	}

	data := sampleSubduxImportData()
	data.Notifications.Channels = []model.NotificationChannel{
		{
			Type:    "webhook",
			Enabled: true,
			Config:  `{"url":"https://example.com/hook","method":"POST"}`,
		},
	}

	resp, err := svc.ImportFromSubdux(user.ID, data, true)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("result should not be nil")
	}

	var channelCount int64
	if err := db.Model(&model.NotificationChannel{}).Where("user_id = ? AND type = ?", user.ID, "webhook").Count(&channelCount).Error; err != nil {
		t.Fatalf("failed to count channels: %v", err)
	}
	if channelCount != 1 {
		t.Fatalf("channel count = %d, want 1", channelCount)
	}
}

func TestImportFromWallosTooLarge(t *testing.T) {
	db := newImportTestDB(t)
	user := createTestUser(t, db)
	svc := NewImportService(db)

	data := make([]WallosSubscription, maxWallosImportItems+1)
	_, err := svc.ImportFromWallos(user.ID, data, false)
	if !errors.Is(err, ErrWallosImportTooLarge) {
		t.Fatalf("expected ErrWallosImportTooLarge, got %v", err)
	}
}
