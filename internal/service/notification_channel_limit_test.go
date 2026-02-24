package service

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func newNotificationChannelLimitTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-notification-channel-limit-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.NotificationChannel{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func createNotificationChannelLimitTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	user := model.User{
		Username: "notify-channel-limit-user",
		Email:    "notify-channel-limit@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

func createNotificationChannelLimitTestChannel(t *testing.T, db *gorm.DB, userID uint, enabled bool) model.NotificationChannel {
	t.Helper()

	channel := model.NotificationChannel{
		UserID:  userID,
		Type:    "webhook",
		Enabled: enabled,
		Config:  `{"url":"https://example.com/webhook"}`,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("failed to create test channel: %v", err)
	}
	return channel
}

func TestCreateChannelRejectsEnabledWhenLimitExceeded(t *testing.T) {
	db := newNotificationChannelLimitTestDB(t)
	user := createNotificationChannelLimitTestUser(t, db)
	service := NewNotificationService(db, nil, nil)

	for i := 0; i < maxEnabledNotificationChannels; i++ {
		createNotificationChannelLimitTestChannel(t, db, user.ID, true)
	}

	_, err := service.CreateChannel(user.ID, CreateChannelInput{
		Type:    "webhook",
		Enabled: true,
		Config:  `{"url":"https://example.com/webhook"}`,
	})
	if err == nil {
		t.Fatal("CreateChannel() error = nil, want enabled channel limit error")
	}
	if !strings.Contains(err.Error(), "you can enable at most 3 notification channels") {
		t.Fatalf("CreateChannel() error = %q, want enabled channel limit message", err.Error())
	}
}

func TestCreateChannelAllowsDisabledWhenLimitReached(t *testing.T) {
	db := newNotificationChannelLimitTestDB(t)
	user := createNotificationChannelLimitTestUser(t, db)
	service := NewNotificationService(db, nil, nil)

	for i := 0; i < maxEnabledNotificationChannels; i++ {
		createNotificationChannelLimitTestChannel(t, db, user.ID, true)
	}

	channel, err := service.CreateChannel(user.ID, CreateChannelInput{
		Type:    "webhook",
		Enabled: false,
		Config:  `{"url":"https://example.com/webhook"}`,
	})
	if err != nil {
		t.Fatalf("CreateChannel() error = %v, want nil", err)
	}
	if channel.Enabled {
		t.Fatal("CreateChannel() created enabled channel, want disabled")
	}
}

func TestUpdateChannelRejectsDisabledToEnabledWhenLimitExceeded(t *testing.T) {
	db := newNotificationChannelLimitTestDB(t)
	user := createNotificationChannelLimitTestUser(t, db)
	service := NewNotificationService(db, nil, nil)

	for i := 0; i < maxEnabledNotificationChannels; i++ {
		createNotificationChannelLimitTestChannel(t, db, user.ID, true)
	}
	disabled := createNotificationChannelLimitTestChannel(t, db, user.ID, false)

	enable := true
	_, err := service.UpdateChannel(user.ID, disabled.ID, UpdateChannelInput{
		Enabled: &enable,
	})
	if err == nil {
		t.Fatal("UpdateChannel() error = nil, want enabled channel limit error")
	}
	if !strings.Contains(err.Error(), "you can enable at most 3 notification channels") {
		t.Fatalf("UpdateChannel() error = %q, want enabled channel limit message", err.Error())
	}
}

func TestUpdateChannelAllowsConfigEditWhenLimitReached(t *testing.T) {
	db := newNotificationChannelLimitTestDB(t)
	user := createNotificationChannelLimitTestUser(t, db)
	service := NewNotificationService(db, nil, nil)

	channel := createNotificationChannelLimitTestChannel(t, db, user.ID, true)
	for i := 1; i < maxEnabledNotificationChannels; i++ {
		createNotificationChannelLimitTestChannel(t, db, user.ID, true)
	}

	config := `{"url":"https://example.com/updated"}`
	updated, err := service.UpdateChannel(user.ID, channel.ID, UpdateChannelInput{
		Config: &config,
	})
	if err != nil {
		t.Fatalf("UpdateChannel() error = %v, want nil", err)
	}
	if updated.Config != config {
		t.Fatalf("UpdateChannel() config = %q, want %q", updated.Config, config)
	}
}
