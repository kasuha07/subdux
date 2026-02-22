package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func newNotificationTemplatePreviewTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-notification-template-preview-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.Subscription{},
		&model.Category{},
		&model.PaymentMethod{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func TestPreviewTemplateUsesFirstSubscriptionData(t *testing.T) {
	db := newNotificationTemplatePreviewTestDB(t)
	svc := NewNotificationTemplateService(db, NewTemplateValidator())

	user := model.User{
		Username: "preview-user",
		Email:    "preview@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	category := model.Category{
		UserID: user.ID,
		Name:   "Streaming",
	}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	paymentMethod := model.PaymentMethod{
		UserID: user.ID,
		Name:   "Visa",
	}
	if err := db.Create(&paymentMethod).Error; err != nil {
		t.Fatalf("failed to create payment method: %v", err)
	}

	billingDate := time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	subscription := model.Subscription{
		UserID:          user.ID,
		Name:            "Spotify Family",
		Amount:          19.99,
		Currency:        "USD",
		CategoryID:      &category.ID,
		PaymentMethodID: &paymentMethod.ID,
		URL:             "https://spotify.com",
		Notes:           "annual promo",
		NextBillingDate: &billingDate,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	laterSubscription := model.Subscription{
		UserID:   user.ID,
		Name:     "Should Not Be Used",
		Amount:   99.99,
		Currency: "USD",
	}
	if err := db.Create(&laterSubscription).Error; err != nil {
		t.Fatalf("failed to create later subscription: %v", err)
	}

	input := CreateTemplateInput{
		Format:   "plaintext",
		Template: "{{.SubscriptionName}}|{{.Amount}}|{{.Currency}}|{{.Category}}|{{.PaymentMethod}}|{{.URL}}|{{.Remark}}|{{.UserEmail}}|{{.BillingDate}}",
	}

	preview, err := svc.PreviewTemplate(user.ID, input)
	if err != nil {
		t.Fatalf("PreviewTemplate() error = %v", err)
	}

	const want = "Spotify Family|19.99|USD|Streaming|Visa|https://spotify.com|annual promo|preview@example.com|2026-04-01"
	if preview != want {
		t.Fatalf("PreviewTemplate() preview = %q, want %q", preview, want)
	}
}

func TestPreviewTemplateFallsBackToSampleDataWhenNoSubscription(t *testing.T) {
	db := newNotificationTemplatePreviewTestDB(t)
	svc := NewNotificationTemplateService(db, NewTemplateValidator())

	user := model.User{
		Username: "no-sub-user",
		Email:    "nosub@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	input := CreateTemplateInput{
		Format:   "plaintext",
		Template: "{{.SubscriptionName}}|{{.Amount}}|{{.Currency}}|{{.Category}}|{{.PaymentMethod}}|{{.URL}}|{{.Remark}}|{{.UserEmail}}|{{.BillingDate}}",
	}

	preview, err := svc.PreviewTemplate(user.ID, input)
	if err != nil {
		t.Fatalf("PreviewTemplate() error = %v", err)
	}

	const want = "Netflix Premium|15.99|USD|Entertainment|Credit Card|https://www.netflix.com|Family plan|user@example.com|2026-03-15"
	if preview != want {
		t.Fatalf("PreviewTemplate() preview = %q, want %q", preview, want)
	}
}
