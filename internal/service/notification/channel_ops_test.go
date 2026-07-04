package notification

import (
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

func TestTestChannelDeliveryConfigErrorIsBadRequest(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "notification-test-channel-error-key")

	db := newTestDB(t)
	user := createTestUser(t, db)

	encryptedConfig, err := encryptNotificationChannelConfig(`not-json`)
	if err != nil {
		t.Fatalf("failed to encrypt notification channel config: %v", err)
	}

	channel := model.NotificationChannel{
		UserID:  user.ID,
		Type:    "smtp",
		Enabled: true,
		Config:  encryptedConfig,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}

	template := model.NotificationTemplate{
		UserID:   user.ID,
		Format:   "plaintext",
		Template: "Test {{.SubscriptionName}}",
	}
	if err := db.Create(&template).Error; err != nil {
		t.Fatalf("failed to create notification template: %v", err)
	}

	svc := NewService(db, NewNotificationTemplateService(db, NewTemplateValidator()), NewTemplateRenderer(NewTemplateValidator()))
	err = svc.TestChannel(user.ID, channel.ID)
	if err == nil {
		t.Fatal("TestChannel() error = nil, want invalid smtp config")
	}
	if kind, ok := serviceerr.KindOf(err); !ok || kind != serviceerr.KindInvalid {
		t.Fatalf("TestChannel() error kind = %v (ok=%t), want KindInvalid; err = %v", kind, ok, err)
	}
	if !strings.Contains(err.Error(), "invalid smtp config") {
		t.Fatalf("TestChannel() error = %q, want invalid smtp config message", err.Error())
	}
}

func TestTestChannelMissingTemplateErrorIsBadRequest(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "notification-test-channel-render-key")

	db := newTestDB(t)
	user := createTestUser(t, db)

	encryptedConfig, err := encryptNotificationChannelConfig(`{"host":"smtp.example.com","from_email":"from@example.com","to_email":"to@example.com"}`)
	if err != nil {
		t.Fatalf("failed to encrypt notification channel config: %v", err)
	}

	channel := model.NotificationChannel{
		UserID:  user.ID,
		Type:    "smtp",
		Enabled: true,
		Config:  encryptedConfig,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}

	svc := NewService(db, NewNotificationTemplateService(db, NewTemplateValidator()), NewTemplateRenderer(NewTemplateValidator()))
	err = svc.TestChannel(user.ID, channel.ID)
	if err == nil {
		t.Fatal("TestChannel() error = nil, want missing template error")
	}
	if kind, ok := serviceerr.KindOf(err); !ok || kind != serviceerr.KindInvalid {
		t.Fatalf("TestChannel() error kind = %v (ok=%t), want KindInvalid; err = %v", kind, ok, err)
	}
	if !strings.Contains(err.Error(), "failed to render notification message") {
		t.Fatalf("TestChannel() error = %q, want render failure message", err.Error())
	}
}
