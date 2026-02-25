package service

import (
	"errors"
	"testing"

	"github.com/shiroha/subdux/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func TestSendRegistrationVerificationCodeBlockedByEmailDomainWhitelist(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}, &model.EmailVerificationCode{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	if err := db.Create(&model.SystemSetting{Key: "registration_enabled", Value: "true"}).Error; err != nil {
		t.Fatalf("failed to seed registration_enabled: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "registration_email_verification_enabled", Value: "true"}).Error; err != nil {
		t.Fatalf("failed to seed registration_email_verification_enabled: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "email_domain_whitelist", Value: "example.com"}).Error; err != nil {
		t.Fatalf("failed to seed email_domain_whitelist: %v", err)
	}

	svc := NewAuthService(db)
	err := svc.SendRegistrationVerificationCode("user@blocked.net")
	if !errors.Is(err, ErrEmailDomainNotAllowed) {
		t.Fatalf("SendRegistrationVerificationCode() error = %v, want %v", err, ErrEmailDomainNotAllowed)
	}
}

func TestSendEmailChangeVerificationCodeBlockedByEmailDomainWhitelist(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}, &model.EmailVerificationCode{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	password := "strong-password"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate password hash: %v", err)
	}

	user := model.User{
		Username: "alice",
		Email:    "alice@example.com",
		Password: string(hash),
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "email_domain_whitelist", Value: "example.com"}).Error; err != nil {
		t.Fatalf("failed to seed email_domain_whitelist: %v", err)
	}

	svc := NewAuthService(db)
	err = svc.SendEmailChangeVerificationCode(user.ID, "new@blocked.net", password)
	if !errors.Is(err, ErrEmailDomainNotAllowed) {
		t.Fatalf("SendEmailChangeVerificationCode() error = %v, want %v", err, ErrEmailDomainNotAllowed)
	}
}

func TestConfirmEmailChangeBlockedByEmailDomainWhitelist(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}, &model.EmailVerificationCode{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	user := model.User{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "email_domain_whitelist", Value: "example.com"}).Error; err != nil {
		t.Fatalf("failed to seed email_domain_whitelist: %v", err)
	}

	svc := NewAuthService(db)
	_, err := svc.ConfirmEmailChange(user.ID, "new@blocked.net", "123456")
	if !errors.Is(err, ErrEmailDomainNotAllowed) {
		t.Fatalf("ConfirmEmailChange() error = %v, want %v", err, ErrEmailDomainNotAllowed)
	}
}

func TestRegisterWithVerificationEnabledBlockedByEmailDomainWhitelist(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}, &model.EmailVerificationCode{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	if err := db.Create(&model.SystemSetting{Key: "registration_enabled", Value: "true"}).Error; err != nil {
		t.Fatalf("failed to seed registration_enabled: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "registration_email_verification_enabled", Value: "true"}).Error; err != nil {
		t.Fatalf("failed to seed registration_email_verification_enabled: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "email_domain_whitelist", Value: "example.com"}).Error; err != nil {
		t.Fatalf("failed to seed email_domain_whitelist: %v", err)
	}

	svc := NewAuthService(db)
	_, err := svc.Register(RegisterInput{
		Username:         "new-user",
		Email:            "new@blocked.net",
		Password:         "password123",
		VerificationCode: "123456",
	})
	if !errors.Is(err, ErrEmailDomainNotAllowed) {
		t.Fatalf("Register() error = %v, want %v", err, ErrEmailDomainNotAllowed)
	}
}

func TestRegisterWithVerificationDisabledStillBlockedByEmailDomainWhitelist(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	if err := db.Create(&model.SystemSetting{Key: "registration_enabled", Value: "true"}).Error; err != nil {
		t.Fatalf("failed to seed registration_enabled: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "registration_email_verification_enabled", Value: "false"}).Error; err != nil {
		t.Fatalf("failed to seed registration_email_verification_enabled: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "email_domain_whitelist", Value: "example.com"}).Error; err != nil {
		t.Fatalf("failed to seed email_domain_whitelist: %v", err)
	}

	svc := NewAuthService(db)
	_, err := svc.Register(RegisterInput{
		Username: "new-user",
		Email:    "new@blocked.net",
		Password: "password123",
	})
	if !errors.Is(err, ErrEmailDomainNotAllowed) {
		t.Fatalf("Register() error = %v, want %v", err, ErrEmailDomainNotAllowed)
	}
}
