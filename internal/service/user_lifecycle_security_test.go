package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func migrateUserLifecycleSecurityTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := db.AutoMigrate(
		&model.User{},
		&model.UserPreference{},
		&model.UserCurrency{},
		&model.Category{},
		&model.PaymentMethod{},
		&model.Subscription{},
		&model.NotificationChannel{},
		&model.NotificationPolicy{},
		&model.NotificationLog{},
		&model.NotificationTemplate{},
		&model.APIKey{},
		&model.RefreshToken{},
		&model.CalendarToken{},
		&model.UserBackupCode{},
		&model.PasskeyCredential{},
		&model.OIDCConnection{},
		&model.EmailVerificationCode{},
	); err != nil {
		t.Fatalf("failed to migrate lifecycle security tables: %v", err)
	}
}

func createLifecycleSecurityUser(t *testing.T, db *gorm.DB, username, email string) model.User {
	t.Helper()

	user := model.User{
		Username: username,
		Email:    email,
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user %s: %v", email, err)
	}
	return user
}

func TestDisabledUserCredentialsAreBlocked(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")

	db := newTestDB(t)
	migrateUserLifecycleSecurityTables(t, db)

	_ = createLifecycleSecurityUser(t, db, "admin", "admin@example.com")
	target := createLifecycleSecurityUser(t, db, "disabled-user", "disabled@example.com")

	adminService := NewAdminService(db)
	authService := NewAuthService(db)
	apiKeyService := NewAPIKeyService(db)
	calendarService := NewCalendarService(db)

	authResp, err := authService.CreateSession(target.ID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	apiKeyResp, err := apiKeyService.Create(target.ID, target.Role, CreateAPIKeyInput{Name: "CLI"})
	if err != nil {
		t.Fatalf("Create() api key error = %v", err)
	}

	calendarToken, err := calendarService.GenerateToken(target.ID, "Personal")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if err := adminService.ChangeUserStatus(target.ID, "disabled"); err != nil {
		t.Fatalf("ChangeUserStatus() error = %v", err)
	}

	var stored model.RefreshToken
	if err := db.Where("user_id = ?", target.ID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load refresh token: %v", err)
	}
	if stored.RevokedAt == nil {
		t.Fatal("disabled user refresh token should be revoked")
	}

	if _, err := authService.RefreshSession(authResp.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("RefreshSession() error = %v, want %v", err, ErrInvalidRefreshToken)
	}

	if _, err := apiKeyService.ValidateKey(apiKeyResp.Key); !errors.Is(err, ErrAPIKeyInvalid) {
		t.Fatalf("ValidateKey() error = %v, want %v", err, ErrAPIKeyInvalid)
	}

	if _, err := calendarService.ValidateToken(calendarToken.Token); err == nil || !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("ValidateToken() error = %v, want invalid token", err)
	}
}

func TestDeleteUserRemovesUserScopedRecordsAndInvalidatesCredentials(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")

	db := newTestDB(t)
	migrateUserLifecycleSecurityTables(t, db)

	_ = createLifecycleSecurityUser(t, db, "admin", "admin@example.com")
	target := createLifecycleSecurityUser(t, db, "delete-user", "delete-user@example.com")

	adminService := NewAdminService(db)
	authService := NewAuthService(db)
	apiKeyService := NewAPIKeyService(db)
	calendarService := NewCalendarService(db)

	preferredCurrency := model.UserPreference{UserID: target.ID, PreferredCurrency: "USD"}
	if err := db.Create(&preferredCurrency).Error; err != nil {
		t.Fatalf("failed to create user preference: %v", err)
	}

	currency := model.UserCurrency{UserID: target.ID, Code: "USD", Symbol: "$", Alias: "US Dollar"}
	if err := db.Create(&currency).Error; err != nil {
		t.Fatalf("failed to create user currency: %v", err)
	}

	category := model.Category{UserID: target.ID, Name: "Video"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	paymentMethod := model.PaymentMethod{UserID: target.ID, Name: "Visa"}
	if err := db.Create(&paymentMethod).Error; err != nil {
		t.Fatalf("failed to create payment method: %v", err)
	}

	nextBillingDate := time.Now().UTC().Add(24 * time.Hour)
	subscription := model.Subscription{
		UserID:          target.ID,
		Name:            "Netflix",
		Amount:          12.99,
		Currency:        "USD",
		Status:          "active",
		RenewalMode:     "auto_renew",
		BillingType:     "recurring",
		NextBillingDate: &nextBillingDate,
		CategoryID:      &category.ID,
		PaymentMethodID: &paymentMethod.ID,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	channelType := "webhook"
	template := model.NotificationTemplate{UserID: target.ID, ChannelType: &channelType, Format: "plaintext", Template: "hello"}
	if err := db.Create(&template).Error; err != nil {
		t.Fatalf("failed to create notification template: %v", err)
	}

	channel := model.NotificationChannel{UserID: target.ID, Type: "webhook", Enabled: true, Config: "{}"}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}

	policy := model.NotificationPolicy{UserID: target.ID, DaysBefore: 3, NotifyOnDueDay: true}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatalf("failed to create notification policy: %v", err)
	}

	logEntry := model.NotificationLog{
		UserID:         target.ID,
		SubscriptionID: subscription.ID,
		ChannelType:    "webhook",
		NotifyDate:     time.Now().UTC(),
		Status:         "sent",
		SentAt:         time.Now().UTC(),
	}
	if err := db.Create(&logEntry).Error; err != nil {
		t.Fatalf("failed to create notification log: %v", err)
	}

	backupCode := model.UserBackupCode{UserID: target.ID, CodeHash: "backup-hash"}
	if err := db.Create(&backupCode).Error; err != nil {
		t.Fatalf("failed to create backup code: %v", err)
	}

	passkey := model.PasskeyCredential{UserID: target.ID, Name: "Laptop", CredentialID: "cred-delete-user", Credential: []byte("credential")}
	if err := db.Create(&passkey).Error; err != nil {
		t.Fatalf("failed to create passkey: %v", err)
	}

	oidc := model.OIDCConnection{UserID: target.ID, Provider: "oidc", Subject: "subject-delete-user", Email: target.Email}
	if err := db.Create(&oidc).Error; err != nil {
		t.Fatalf("failed to create oidc connection: %v", err)
	}

	emailVerificationUserID := target.ID
	verification := model.EmailVerificationCode{
		UserID:    &emailVerificationUserID,
		Email:     target.Email,
		Purpose:   "password_reset",
		CodeHash:  "verification-hash",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := db.Create(&verification).Error; err != nil {
		t.Fatalf("failed to create email verification code: %v", err)
	}

	authResp, err := authService.CreateSession(target.ID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	apiKeyResp, err := apiKeyService.Create(target.ID, target.Role, CreateAPIKeyInput{Name: "CLI"})
	if err != nil {
		t.Fatalf("Create() api key error = %v", err)
	}

	calendarToken, err := calendarService.GenerateToken(target.ID, "Personal")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if err := adminService.DeleteUser(target.ID); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	if err := db.First(&model.User{}, target.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleted user lookup error = %v, want %v", err, gorm.ErrRecordNotFound)
	}

	for _, tc := range []struct {
		name  string
		model interface{}
	}{
		{name: "subscriptions", model: &model.Subscription{}},
		{name: "payment_methods", model: &model.PaymentMethod{}},
		{name: "user_currencies", model: &model.UserCurrency{}},
		{name: "categories", model: &model.Category{}},
		{name: "user_preferences", model: &model.UserPreference{}},
		{name: "user_backup_codes", model: &model.UserBackupCode{}},
		{name: "passkey_credentials", model: &model.PasskeyCredential{}},
		{name: "oidc_connections", model: &model.OIDCConnection{}},
		{name: "email_verification_codes", model: &model.EmailVerificationCode{}},
		{name: "notification_channels", model: &model.NotificationChannel{}},
		{name: "notification_policies", model: &model.NotificationPolicy{}},
		{name: "notification_logs", model: &model.NotificationLog{}},
		{name: "notification_templates", model: &model.NotificationTemplate{}},
		{name: "api_keys", model: &model.APIKey{}},
		{name: "refresh_tokens", model: &model.RefreshToken{}},
		{name: "calendar_tokens", model: &model.CalendarToken{}},
	} {
		var count int64
		if err := db.Model(tc.model).Where("user_id = ?", target.ID).Count(&count).Error; err != nil {
			t.Fatalf("count %s error = %v", tc.name, err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d, want 0", tc.name, count)
		}
	}

	if _, err := authService.RefreshSession(authResp.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("RefreshSession() after delete error = %v, want %v", err, ErrInvalidRefreshToken)
	}

	if _, err := apiKeyService.ValidateKey(apiKeyResp.Key); !errors.Is(err, ErrAPIKeyInvalid) {
		t.Fatalf("ValidateKey() after delete error = %v, want %v", err, ErrAPIKeyInvalid)
	}

	if _, err := calendarService.ValidateToken(calendarToken.Token); err == nil || !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("ValidateToken() after delete error = %v, want invalid token", err)
	}
}
