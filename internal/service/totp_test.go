package service

import (
	"errors"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func newTOTPTestService(t *testing.T) (*TOTPService, model.User) {
	t.Helper()

	db := newTestDB(t)
	user := model.User{
		Username: "totp-user",
		Email:    "totp-user@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return NewTOTPService(db), user
}

func TestTOTPBeginSetupAndConfirm(t *testing.T) {
	svc, user := newTOTPTestService(t)

	setup, err := svc.BeginSetup(user.ID)
	if err != nil {
		t.Fatalf("BeginSetup() error = %v, want nil", err)
	}
	if setup.SessionID == "" {
		t.Fatal("BeginSetup() returned empty session id")
	}
	if setup.Secret == "" || setup.OtpauthURI == "" {
		t.Fatalf("BeginSetup() returned incomplete setup: %+v", setup)
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateCode() error = %v, want nil", err)
	}

	backupCodes, err := svc.ConfirmSetup(user.ID, setup.SessionID, code)
	if err != nil {
		t.Fatalf("ConfirmSetup() error = %v, want nil", err)
	}
	if len(backupCodes) != 8 {
		t.Fatalf("ConfirmSetup() backup code count = %d, want 8", len(backupCodes))
	}

	var stored model.User
	if err := svc.DB.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if !stored.TotpEnabled {
		t.Fatal("stored.TotpEnabled = false, want true")
	}
	if stored.TotpSecret == nil || *stored.TotpSecret == "" {
		t.Fatal("stored.TotpSecret is empty, want secret persisted")
	}
	if stored.TotpTempSecret != nil {
		t.Fatalf("stored.TotpTempSecret = %v, want nil", *stored.TotpTempSecret)
	}

	if _, err := svc.ConfirmSetup(user.ID, setup.SessionID, code); !errors.Is(err, ErrTOTPSetupExpired) {
		t.Fatalf("reused ConfirmSetup() error = %v, want ErrTOTPSetupExpired", err)
	}
}

func TestTOTPConfirmSetupRollsBackWhenBackupCodeInsertFails(t *testing.T) {
	svc, user := newTOTPTestService(t)

	setup, err := svc.BeginSetup(user.ID)
	if err != nil {
		t.Fatalf("BeginSetup() error = %v, want nil", err)
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateCode() error = %v, want nil", err)
	}

	createCalls := 0
	callbackName := "test:fail_second_backup_code_create"
	if err := svc.DB.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "user_backup_codes" {
			return
		}
		createCalls++
		if createCalls == 2 {
			tx.AddError(errors.New("forced backup code create failure"))
		}
	}); err != nil {
		t.Fatalf("register callback failed: %v", err)
	}

	if _, err := svc.ConfirmSetup(user.ID, setup.SessionID, code); err == nil {
		t.Fatal("ConfirmSetup() error = nil, want forced backup code create failure")
	}

	var stored model.User
	if err := svc.DB.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("failed to reload user after rollback: %v", err)
	}
	if stored.TotpEnabled {
		t.Fatal("stored.TotpEnabled = true after rollback, want false")
	}
	if stored.TotpSecret != nil {
		t.Fatalf("stored.TotpSecret = %q after rollback, want nil", *stored.TotpSecret)
	}

	var backupCodeCount int64
	if err := svc.DB.Model(&model.UserBackupCode{}).Where("user_id = ?", user.ID).Count(&backupCodeCount).Error; err != nil {
		t.Fatalf("count backup codes after rollback failed: %v", err)
	}
	if backupCodeCount != 0 {
		t.Fatalf("backup code count after rollback = %d, want 0", backupCodeCount)
	}

	if err := svc.DB.Callback().Create().Remove(callbackName); err != nil {
		t.Fatalf("remove callback failed: %v", err)
	}

	backupCodes, err := svc.ConfirmSetup(user.ID, setup.SessionID, code)
	if err != nil {
		t.Fatalf("ConfirmSetup() retry error = %v, want nil", err)
	}
	if len(backupCodes) != 8 {
		t.Fatalf("ConfirmSetup() retry backup code count = %d, want 8", len(backupCodes))
	}
}

func TestTOTPBeginSetupReplacesOlderSession(t *testing.T) {
	svc, user := newTOTPTestService(t)

	first, err := svc.BeginSetup(user.ID)
	if err != nil {
		t.Fatalf("first BeginSetup() error = %v, want nil", err)
	}
	second, err := svc.BeginSetup(user.ID)
	if err != nil {
		t.Fatalf("second BeginSetup() error = %v, want nil", err)
	}
	if first.SessionID == second.SessionID {
		t.Fatalf("session ids should differ, got %q", first.SessionID)
	}

	firstCode, err := totp.GenerateCode(first.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateCode(first) error = %v, want nil", err)
	}
	if _, err := svc.ConfirmSetup(user.ID, first.SessionID, firstCode); !errors.Is(err, ErrTOTPSetupExpired) {
		t.Fatalf("ConfirmSetup(first) error = %v, want ErrTOTPSetupExpired", err)
	}

	secondCode, err := totp.GenerateCode(second.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateCode(second) error = %v, want nil", err)
	}
	if _, err := svc.ConfirmSetup(user.ID, second.SessionID, secondCode); err != nil {
		t.Fatalf("ConfirmSetup(second) error = %v, want nil", err)
	}
}

func TestTOTPSetupSessionExpiry(t *testing.T) {
	svc, user := newTOTPTestService(t)

	clock := &mutableClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	restore := pkg.SetClockForTest(clock)
	defer restore()

	setup, err := svc.BeginSetup(user.ID)
	if err != nil {
		t.Fatalf("BeginSetup() error = %v, want nil", err)
	}

	clock.advance(totpSetupSessionTTL + time.Second)

	if _, err := svc.ConfirmSetup(user.ID, setup.SessionID, "000000"); !errors.Is(err, ErrTOTPSetupExpired) {
		t.Fatalf("expired ConfirmSetup() error = %v, want ErrTOTPSetupExpired", err)
	}
}

func TestTOTPBeginSetupRejectsAlreadyEnabled(t *testing.T) {
	svc, user := newTOTPTestService(t)

	secret := "SECRET123"
	if err := svc.DB.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"totp_enabled": true,
		"totp_secret":  secret,
	}).Error; err != nil {
		t.Fatalf("failed to enable totp: %v", err)
	}

	if _, err := svc.BeginSetup(user.ID); !errors.Is(err, ErrTOTPAlreadyEnabled) {
		t.Fatalf("BeginSetup() error = %v, want ErrTOTPAlreadyEnabled", err)
	}
}

func TestTOTPDisableClearsFactorsAfterCallerReauthenticates(t *testing.T) {
	svc, user := newTOTPTestService(t)

	setup, err := svc.BeginSetup(user.ID)
	if err != nil {
		t.Fatalf("BeginSetup() error = %v, want nil", err)
	}
	const secret = "JBSWY3DPEHPK3PXP"
	if err := svc.DB.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"totp_enabled": true,
		"totp_secret":  secret,
	}).Error; err != nil {
		t.Fatalf("failed to enable totp: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("backup-code"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash backup code: %v", err)
	}
	if err := svc.DB.Create(&model.UserBackupCode{UserID: user.ID, CodeHash: string(hash)}).Error; err != nil {
		t.Fatalf("failed to create backup code: %v", err)
	}

	if err := svc.Disable(user.ID); err != nil {
		t.Fatalf("Disable() error = %v, want nil", err)
	}

	var updated model.User
	if err := svc.DB.Select("id", "totp_enabled", "totp_secret", "totp_temp_secret").First(&updated, user.ID).Error; err != nil {
		t.Fatalf("failed to load user after disable: %v", err)
	}
	if updated.TotpEnabled || updated.TotpSecret != nil || updated.TotpTempSecret != nil {
		t.Fatalf("user TOTP state = enabled:%t secret:%v temp:%v, want disabled and cleared", updated.TotpEnabled, updated.TotpSecret, updated.TotpTempSecret)
	}

	var backupCount int64
	if err := svc.DB.Model(&model.UserBackupCode{}).Where("user_id = ?", user.ID).Count(&backupCount).Error; err != nil {
		t.Fatalf("failed to count backup codes: %v", err)
	}
	if backupCount != 0 {
		t.Fatalf("backup code count = %d, want 0", backupCount)
	}
	if _, err := svc.getSetupSession(setup.SessionID); !errors.Is(err, ErrTOTPSetupExpired) {
		t.Fatalf("getSetupSession() error = %v, want ErrTOTPSetupExpired", err)
	}
}

func TestTOTPBeginSetupPropagatesInternalDBErrors(t *testing.T) {
	svc, user := newTOTPTestService(t)

	sqlDB, err := svc.DB.DB()
	if err != nil {
		t.Fatalf("open sql DB handle error = %v, want nil", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql DB handle error = %v, want nil", err)
	}

	_, err = svc.BeginSetup(user.ID)
	if err == nil {
		t.Fatal("BeginSetup() error = nil, want database failure")
	}
	if errors.Is(err, ErrUserNotFound) {
		t.Fatalf("BeginSetup() error = %v, should not collapse internal db failure into ErrUserNotFound", err)
	}
}
