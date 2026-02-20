package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/pquerna/otp/totp"
	"github.com/shiroha/subdux/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type TOTPService struct {
	DB *gorm.DB
}

func NewTOTPService(db *gorm.DB) *TOTPService {
	return &TOTPService{DB: db}
}

type TotpSetupResult struct {
	OtpauthURI string `json:"otpauth_uri"`
	Secret     string `json:"secret"`
}

func (s *TOTPService) GenerateSetup(userID uint) (*TotpSetupResult, error) {
	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Subdux",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, err
	}

	secret := key.Secret()
	if err := s.DB.Model(&user).Update("totp_temp_secret", secret).Error; err != nil {
		return nil, err
	}

	return &TotpSetupResult{
		OtpauthURI: key.URL(),
		Secret:     secret,
	}, nil
}

func (s *TOTPService) ConfirmSetup(userID uint, code string) ([]string, error) {
	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	if user.TotpTempSecret == nil {
		return nil, errors.New("no pending 2FA setup")
	}

	valid := totp.Validate(code, *user.TotpTempSecret)
	if !valid {
		return nil, errors.New("invalid verification code")
	}

	secret := *user.TotpTempSecret
	if err := s.DB.Model(&user).Updates(map[string]interface{}{
		"totp_secret":      secret,
		"totp_enabled":     true,
		"totp_temp_secret": nil,
	}).Error; err != nil {
		return nil, err
	}

	plainCodes, err := s.generateBackupCodes(userID)
	if err != nil {
		return nil, err
	}

	return plainCodes, nil
}

func (s *TOTPService) generateBackupCodes(userID uint) ([]string, error) {
	s.DB.Where("user_id = ?", userID).Delete(&model.UserBackupCode{})

	plainCodes := make([]string, 8)
	for i := range plainCodes {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		plainCodes[i] = hex.EncodeToString(b)

		hash, err := bcrypt.GenerateFromPassword([]byte(plainCodes[i]), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		if err := s.DB.Create(&model.UserBackupCode{
			UserID:   userID,
			CodeHash: string(hash),
		}).Error; err != nil {
			return nil, err
		}
	}

	return plainCodes, nil
}

func (s *TOTPService) Disable(userID uint, password string, code string) error {
	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return errors.New("invalid password")
	}

	if !s.VerifyLogin(userID, code) && !s.VerifyBackupCode(userID, code) {
		return errors.New("invalid authentication code")
	}

	if err := s.DB.Model(&user).Updates(map[string]interface{}{
		"totp_secret":      nil,
		"totp_enabled":     false,
		"totp_temp_secret": nil,
	}).Error; err != nil {
		return err
	}

	s.DB.Where("user_id = ?", userID).Delete(&model.UserBackupCode{})
	return nil
}

func (s *TOTPService) VerifyLogin(userID uint, code string) bool {
	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return false
	}
	if user.TotpSecret == nil {
		return false
	}
	return totp.Validate(code, *user.TotpSecret)
}

func (s *TOTPService) VerifyBackupCode(userID uint, code string) bool {
	var backupCodes []model.UserBackupCode
	if err := s.DB.Where("user_id = ?", userID).Find(&backupCodes).Error; err != nil {
		return false
	}

	for _, bc := range backupCodes {
		if bcrypt.CompareHashAndPassword([]byte(bc.CodeHash), []byte(code)) == nil {
			s.DB.Delete(&bc)
			return true
		}
	}
	return false
}
