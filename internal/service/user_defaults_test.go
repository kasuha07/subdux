package service

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	notificationservice "github.com/kasuha07/subdux/internal/service/notification"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.SystemSetting{},
		&model.EmailVerificationCode{},
		&model.RefreshToken{},
		&model.PasskeyCredential{},
		&model.UserBackupCode{},
		&model.UserPreference{},
		&model.UserCurrency{},
		&model.Category{},
		&model.PaymentMethod{},
		&model.Subscription{},
		&model.SubscriptionEvent{},
		&model.SubscriptionActionSnooze{},
		&model.NotificationLog{},
		&model.NotificationTemplate{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func createTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	user := model.User{
		Username: "tester",
		Email:    "tester@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

func TestSeedUserDefaultsIsIdempotent(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)

	if err := serviceutil.SeedUserDefaults(db, user.ID); err != nil {
		t.Fatalf("first seed failed: %v", err)
	}
	if err := serviceutil.SeedUserDefaults(db, user.ID); err != nil {
		t.Fatalf("second seed failed: %v", err)
	}

	var categoryCount int64
	if err := db.Model(&model.Category{}).Where("user_id = ?", user.ID).Count(&categoryCount).Error; err != nil {
		t.Fatalf("count categories failed: %v", err)
	}
	if categoryCount != 5 {
		t.Fatalf("category count = %d, want %d", categoryCount, 5)
	}

	var paymentMethodCount int64
	if err := db.Model(&model.PaymentMethod{}).Where("user_id = ?", user.ID).Count(&paymentMethodCount).Error; err != nil {
		t.Fatalf("count payment methods failed: %v", err)
	}
	if paymentMethodCount != 3 {
		t.Fatalf("payment method count = %d, want %d", paymentMethodCount, 3)
	}

	var currencyCount int64
	if err := db.Model(&model.UserCurrency{}).Where("user_id = ?", user.ID).Count(&currencyCount).Error; err != nil {
		t.Fatalf("count currencies failed: %v", err)
	}
	if currencyCount != 4 {
		t.Fatalf("currency count = %d, want %d", currencyCount, 4)
	}

	var preference model.UserPreference
	if err := db.Where("user_id = ?", user.ID).First(&preference).Error; err != nil {
		t.Fatalf("query preference failed: %v", err)
	}
	if preference.PreferredCurrency != "CNY" {
		t.Fatalf("preferred currency = %q, want %q", preference.PreferredCurrency, "CNY")
	}

	var templates []model.NotificationTemplate
	if err := db.Where("user_id = ?", user.ID).Order("id ASC").Find(&templates).Error; err != nil {
		t.Fatalf("query notification templates failed: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("notification template count = %d, want 1", len(templates))
	}
	if templates[0].ChannelType != nil {
		t.Fatal("default notification template channel_type should be nil")
	}
	if templates[0].Format != "plaintext" {
		t.Fatalf("default notification template format = %q, want %q", templates[0].Format, "plaintext")
	}
	if templates[0].Template == "" {
		t.Fatal("default notification template should not be empty")
	}

	validator := notificationservice.NewTemplateValidator()
	if err := validator.ValidateFormat(templates[0].Format); err != nil {
		t.Fatalf("default notification template format is invalid: %v", err)
	}
	if err := validator.ValidateTemplate(templates[0].Template); err != nil {
		t.Fatalf("default notification template is invalid: %v", err)
	}
}

func TestCategoryAndPaymentMethodCustomizeFlags(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	if err := serviceutil.SeedUserDefaults(db, user.ID); err != nil {
		t.Fatalf("seed defaults failed: %v", err)
	}

	categoryService := catalogservice.NewCategoryService(db)
	paymentMethodService := catalogservice.NewPaymentMethodService(db)

	customCategory, err := categoryService.Create(user.ID, catalogservice.CreateCategoryInput{
		Name:         "My Custom Category",
		DisplayOrder: 99,
	})
	if err != nil {
		t.Fatalf("create custom category failed: %v", err)
	}
	if customCategory.SystemKey != nil {
		t.Fatal("custom category system_key should be nil")
	}
	if !customCategory.NameCustomized {
		t.Fatal("custom category name_customized should be true")
	}

	var seededCategory model.Category
	if err := db.Where("user_id = ? AND system_key = ?", user.ID, "video").First(&seededCategory).Error; err != nil {
		t.Fatalf("query seeded category failed: %v", err)
	}
	renamedCategory := "Streaming"
	updatedCategory, err := categoryService.Update(user.ID, seededCategory.ID, catalogservice.UpdateCategoryInput{Name: &renamedCategory})
	if err != nil {
		t.Fatalf("rename seeded category failed: %v", err)
	}
	if !updatedCategory.NameCustomized {
		t.Fatal("renamed seeded category name_customized should be true")
	}
	if updatedCategory.Name != renamedCategory {
		t.Fatalf("renamed category name = %q, want %q", updatedCategory.Name, renamedCategory)
	}

	customMethod, err := paymentMethodService.Create(user.ID, catalogservice.CreatePaymentMethodInput{
		Name:      "My Wallet",
		SortOrder: 99,
	})
	if err != nil {
		t.Fatalf("create custom payment method failed: %v", err)
	}
	if customMethod.SystemKey != nil {
		t.Fatal("custom payment method system_key should be nil")
	}
	if !customMethod.NameCustomized {
		t.Fatal("custom payment method name_customized should be true")
	}

	var seededMethod model.PaymentMethod
	if err := db.Where("user_id = ? AND system_key = ?", user.ID, "alipay").First(&seededMethod).Error; err != nil {
		t.Fatalf("query seeded payment method failed: %v", err)
	}
	renamedMethod := "My Card"
	updatedMethod, err := paymentMethodService.Update(user.ID, seededMethod.ID, catalogservice.UpdatePaymentMethodInput{Name: &renamedMethod})
	if err != nil {
		t.Fatalf("rename seeded payment method failed: %v", err)
	}
	if !updatedMethod.NameCustomized {
		t.Fatal("renamed seeded payment method name_customized should be true")
	}
	if updatedMethod.Name != renamedMethod {
		t.Fatalf("renamed payment method name = %q, want %q", updatedMethod.Name, renamedMethod)
	}
}
