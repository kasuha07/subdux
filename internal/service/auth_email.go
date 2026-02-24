package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	verificationPurposeRegister      = "register"
	verificationPurposePasswordReset = "password_reset"
	verificationPurposeChangeEmail   = "change_email"

	verificationCodeLength       = 6
	verificationCodeTTL          = 10 * time.Minute
	verificationCodeRequestDelay = 60 * time.Second
	verificationCodeMaxFailures  = 5
)

var (
	ErrRegistrationDisabled                  = errors.New("registration is disabled")
	ErrRegistrationEmailVerificationDisabled = errors.New("registration email verification is disabled")
	ErrVerificationCodeRequired              = errors.New("verification code is required")
	ErrVerificationCodeInvalid               = errors.New("invalid or expired verification code")
	ErrVerificationCodeTooManyAttempts       = errors.New("verification code has too many failed attempts, request a new code")
	ErrVerificationCodeTooFrequent           = errors.New("please wait before requesting another verification code")
	ErrInvalidEmail                          = errors.New("invalid email")
	ErrSMTPUnavailable                       = errors.New("email service is unavailable")
	ErrEmailAlreadyRegistered                = errors.New("email already registered")
	ErrUsernameAlreadyTaken                  = errors.New("username already taken")
	ErrUserNotFound                          = errors.New("user not found")
	ErrCurrentPasswordIncorrect              = errors.New("current password is incorrect")
	ErrNewEmailSameAsCurrent                 = errors.New("new email must be different from current email")
)

type RegistrationConfig struct {
	RegistrationEnabled      bool `json:"registration_enabled"`
	EmailVerificationEnabled bool `json:"email_verification_enabled"`
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *AuthService) GetRegistrationConfig() (*RegistrationConfig, error) {
	var userCount int64
	if err := s.DB.Model(&model.User{}).Count(&userCount).Error; err != nil {
		return nil, err
	}

	return &RegistrationConfig{
		RegistrationEnabled:      s.isRegistrationEnabled(userCount),
		EmailVerificationEnabled: s.isRegistrationEmailVerificationEnabled(),
	}, nil
}

func (s *AuthService) SendRegistrationVerificationCode(email string) error {
	normalizedEmail, err := sanitizeAndValidateEmail(email)
	if err != nil {
		return err
	}

	var userCount int64
	if err := s.DB.Model(&model.User{}).Count(&userCount).Error; err != nil {
		return err
	}
	if !s.isRegistrationEnabled(userCount) {
		return ErrRegistrationDisabled
	}
	if !s.isRegistrationEmailVerificationEnabled() {
		return ErrRegistrationEmailVerificationDisabled
	}

	if exists, err := s.emailExists(normalizedEmail, 0); err != nil {
		return err
	} else if exists {
		return ErrEmailAlreadyRegistered
	}

	return s.issueVerificationCode(nil, normalizedEmail, verificationPurposeRegister)
}

func (s *AuthService) RequestPasswordReset(email string) error {
	normalizedEmail, err := sanitizeAndValidateEmail(email)
	if err != nil {
		return err
	}

	if _, err := loadSMTPRuntimeConfig(s.DB); err != nil {
		return ErrSMTPUnavailable
	}

	var user model.User
	if err := s.DB.Where("LOWER(email) = ?", normalizedEmail).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	if user.Status == "disabled" {
		return nil
	}

	return s.issueVerificationCode(&user.ID, normalizedEmail, verificationPurposePasswordReset)
}

func (s *AuthService) ResetPassword(email string, verificationCode string, newPassword string) error {
	normalizedEmail, err := sanitizeAndValidateEmail(email)
	if err != nil {
		return err
	}

	var user model.User
	if err := s.DB.Where("LOWER(email) = ?", normalizedEmail).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVerificationCodeInvalid
		}
		return err
	}

	if err := s.consumeVerificationCode(&user.ID, normalizedEmail, verificationPurposePasswordReset, verificationCode); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id = ?", user.ID).Update("password", string(hash)).Error; err != nil {
			return err
		}
		if err := revokeAllRefreshTokens(tx, user.ID); err != nil {
			return err
		}
		return tx.Model(&model.EmailVerificationCode{}).
			Where("user_id = ? AND email = ? AND purpose = ? AND consumed_at IS NULL", user.ID, normalizedEmail, verificationPurposePasswordReset).
			Update("consumed_at", &now).Error
	}); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) SendEmailChangeVerificationCode(userID uint, newEmail string, currentPassword string) error {
	normalizedEmail, err := sanitizeAndValidateEmail(newEmail)
	if err != nil {
		return err
	}

	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)) != nil {
		return ErrCurrentPasswordIncorrect
	}

	if normalizeEmail(user.Email) == normalizedEmail {
		return ErrNewEmailSameAsCurrent
	}

	if exists, err := s.emailExists(normalizedEmail, userID); err != nil {
		return err
	} else if exists {
		return ErrEmailAlreadyRegistered
	}

	return s.issueVerificationCode(&user.ID, normalizedEmail, verificationPurposeChangeEmail)
}

func (s *AuthService) ConfirmEmailChange(userID uint, newEmail string, verificationCode string) (*AuthResponse, error) {
	normalizedEmail, err := sanitizeAndValidateEmail(newEmail)
	if err != nil {
		return nil, err
	}

	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if normalizeEmail(user.Email) == normalizedEmail {
		return nil, ErrNewEmailSameAsCurrent
	}

	if exists, err := s.emailExists(normalizedEmail, userID); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrEmailAlreadyRegistered
	}

	if err := s.consumeVerificationCode(&user.ID, normalizedEmail, verificationPurposeChangeEmail, verificationCode); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("email", normalizedEmail).Error; err != nil {
			return err
		}
		if err := revokeAllRefreshTokens(tx, user.ID); err != nil {
			return err
		}
		return tx.Model(&model.EmailVerificationCode{}).
			Where("user_id = ? AND email = ? AND purpose = ? AND consumed_at IS NULL", userID, normalizedEmail, verificationPurposeChangeEmail).
			Update("consumed_at", &now).Error
	}); err != nil {
		return nil, err
	}

	user.Email = normalizedEmail
	return s.issueAuthResponse(user)
}

func (s *AuthService) isRegistrationEnabled(userCount int64) bool {
	if userCount == 0 {
		return true
	}
	return s.getBoolSystemSetting("registration_enabled", true)
}

func (s *AuthService) isRegistrationEmailVerificationEnabled() bool {
	return s.getBoolSystemSetting("registration_email_verification_enabled", false)
}

func (s *AuthService) getBoolSystemSetting(key string, defaultValue bool) bool {
	var setting model.SystemSetting
	if err := s.DB.Where("key = ?", key).First(&setting).Error; err != nil {
		return defaultValue
	}
	return setting.Value == "true"
}

func sanitizeAndValidateEmail(email string) (string, error) {
	normalized := normalizeEmail(email)
	if normalized == "" {
		return "", ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

func generateVerificationCode() (string, error) {
	max := big.NewInt(1000000)
	randomNumber, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", randomNumber.Int64()), nil
}

func (s *AuthService) issueVerificationCode(userID *uint, email string, purpose string) error {
	if err := s.ensureVerificationCodeCooldown(userID, email, purpose); err != nil {
		return err
	}

	code, err := generateVerificationCode()
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	expiresAt := time.Now().UTC().Add(verificationCodeTTL)
	verification := model.EmailVerificationCode{
		UserID:    userID,
		Email:     email,
		Purpose:   purpose,
		CodeHash:  string(hash),
		ExpiresAt: expiresAt,
	}

	if err := s.DB.Create(&verification).Error; err != nil {
		return err
	}

	if err := s.sendVerificationCodeEmail(email, purpose, code); err != nil {
		_ = s.DB.Delete(&verification).Error
		return err
	}

	s.cleanupVerificationCodes(email, purpose)
	return nil
}

func (s *AuthService) ensureVerificationCodeCooldown(userID *uint, email string, purpose string) error {
	query := s.DB.Where("email = ? AND purpose = ?", email, purpose)
	if userID == nil {
		query = query.Where("user_id IS NULL")
	} else {
		query = query.Where("user_id = ?", *userID)
	}

	var latest model.EmailVerificationCode
	if err := query.Order("created_at DESC").First(&latest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	if latest.CreatedAt.After(time.Now().UTC().Add(-verificationCodeRequestDelay)) {
		return ErrVerificationCodeTooFrequent
	}
	return nil
}

func (s *AuthService) consumeVerificationCode(userID *uint, email string, purpose string, code string) error {
	trimmedCode := strings.TrimSpace(code)
	if len(trimmedCode) != verificationCodeLength {
		return ErrVerificationCodeInvalid
	}
	for _, ch := range trimmedCode {
		if ch < '0' || ch > '9' {
			return ErrVerificationCodeInvalid
		}
	}

	now := time.Now().UTC()
	query := s.DB.Where("email = ? AND purpose = ? AND consumed_at IS NULL AND expires_at > ?", email, purpose, now)
	if userID == nil {
		query = query.Where("user_id IS NULL")
	} else {
		query = query.Where("user_id = ?", *userID)
	}

	var verification model.EmailVerificationCode
	if err := query.Order("created_at DESC").First(&verification).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVerificationCodeInvalid
		}
		return err
	}

	if verification.FailedAttempts >= verificationCodeMaxFailures {
		_ = s.DB.Model(&verification).Update("consumed_at", &now).Error
		return ErrVerificationCodeTooManyAttempts
	}

	if err := bcrypt.CompareHashAndPassword([]byte(verification.CodeHash), []byte(trimmedCode)); err != nil {
		nextAttempts := verification.FailedAttempts + 1
		updates := map[string]interface{}{"failed_attempts": nextAttempts}
		if nextAttempts >= verificationCodeMaxFailures {
			updates["consumed_at"] = &now
		}
		_ = s.DB.Model(&verification).Updates(updates).Error
		if nextAttempts >= verificationCodeMaxFailures {
			return ErrVerificationCodeTooManyAttempts
		}
		return ErrVerificationCodeInvalid
	}

	if err := s.DB.Model(&verification).Update("consumed_at", &now).Error; err != nil {
		return err
	}

	return nil
}

func (s *AuthService) sendVerificationCodeEmail(recipient string, purpose string, code string) error {
	cfg, err := loadSMTPRuntimeConfig(s.DB)
	if err != nil {
		return ErrSMTPUnavailable
	}

	expiresMinutes := int(verificationCodeTTL / time.Minute)
	subject := "Subdux verification code"
	body := fmt.Sprintf("Your verification code is: %s\r\nThis code expires in %d minutes.", code, expiresMinutes)

	switch purpose {
	case verificationPurposeRegister:
		subject = "Subdux registration verification code"
		body = fmt.Sprintf("Use this code to complete your registration: %s\r\nThis code expires in %d minutes.", code, expiresMinutes)
	case verificationPurposePasswordReset:
		subject = "Subdux password reset code"
		body = fmt.Sprintf("Use this code to reset your password: %s\r\nThis code expires in %d minutes.\r\nIf you did not request this, you can ignore this email.", code, expiresMinutes)
	case verificationPurposeChangeEmail:
		subject = "Subdux email change code"
		body = fmt.Sprintf("Use this code to confirm your new email address: %s\r\nThis code expires in %d minutes.\r\nIf you did not request this, you can ignore this email.", code, expiresMinutes)
	}

	message := buildSMTPMessage(cfg.FromEmail, cfg.FromName, recipient, subject, body)
	if err := sendSMTPMessage(*cfg, recipient, message); err != nil {
		return ErrSMTPUnavailable
	}
	return nil
}

func (s *AuthService) cleanupVerificationCodes(email string, purpose string) {
	threshold := time.Now().UTC().Add(-24 * time.Hour)
	_ = s.DB.Where("email = ? AND purpose = ? AND (consumed_at IS NOT NULL OR expires_at < ?)", email, purpose, threshold).
		Delete(&model.EmailVerificationCode{}).Error
}

func (s *AuthService) emailExists(email string, excludeUserID uint) (bool, error) {
	query := s.DB.Model(&model.User{}).Where("LOWER(email) = ?", email)
	if excludeUserID > 0 {
		query = query.Where("id <> ?", excludeUserID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
