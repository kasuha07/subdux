package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	totpSetupSessionTTL   = 5 * time.Minute
	maxTOTPSetupSessions  = 128
	totpSetupSessionBytes = 24
)

var (
	ErrTOTPAlreadyEnabled  = errors.New("two-factor authentication is already enabled")
	ErrTOTPSetupExpired    = errors.New("two-factor setup expired, start again")
	ErrTOTPInvalidCode     = errors.New("invalid verification code")
	ErrTOTPInvalidPassword = errors.New("invalid password")
	ErrTOTPInvalidAuthCode = errors.New("invalid authentication code")
)

type totpSetupSession struct {
	userID     uint
	secret     string
	otpauthURI string
	expiresAt  time.Time
	createdAt  time.Time
}

type TOTPService struct {
	DB *gorm.DB

	mu            *sync.Mutex
	setupSessions map[string]totpSetupSession
}

func NewTOTPService(db *gorm.DB) *TOTPService {
	return &TOTPService{
		DB:            db,
		mu:            &sync.Mutex{},
		setupSessions: make(map[string]totpSetupSession),
	}
}

type TotpSetupResult struct {
	SessionID  string `json:"session_id"`
	OtpauthURI string `json:"otpauth_uri"`
	Secret     string `json:"secret"`
}

func (s *TOTPService) WithContext(ctx context.Context) *TOTPService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *TOTPService) BeginSetup(userID uint) (*TotpSetupResult, error) {
	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if user.TotpEnabled {
		return nil, ErrTOTPAlreadyEnabled
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Subdux",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, err
	}

	secret := key.Secret()
	sessionID, err := s.storeSetupSession(totpSetupSession{
		userID:     userID,
		secret:     secret,
		otpauthURI: key.URL(),
		expiresAt:  pkg.NowUTC().Add(totpSetupSessionTTL),
	})
	if err != nil {
		return nil, err
	}

	return &TotpSetupResult{
		SessionID:  sessionID,
		OtpauthURI: key.URL(),
		Secret:     secret,
	}, nil
}

func (s *TOTPService) ConfirmSetup(userID uint, sessionID string, code string) ([]string, error) {
	session, err := s.getSetupSession(sessionID)
	if err != nil {
		return nil, err
	}
	if session.userID != userID {
		return nil, ErrTOTPSetupExpired
	}

	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if user.TotpEnabled {
		s.deleteSetupSession(sessionID)
		return nil, ErrTOTPAlreadyEnabled
	}

	valid := totp.Validate(code, session.secret)
	if !valid {
		return nil, ErrTOTPInvalidCode
	}

	plainCodes, backupCodes, err := generateBackupCodes()
	if err != nil {
		return nil, err
	}

	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Updates(map[string]interface{}{
			"totp_secret":      session.secret,
			"totp_enabled":     true,
			"totp_temp_secret": nil,
		}).Error; err != nil {
			return err
		}

		return replaceBackupCodes(tx, userID, backupCodes)
	}); err != nil {
		return nil, err
	}

	s.deleteSetupSession(sessionID)

	return plainCodes, nil
}

func generateBackupCodes() ([]string, []model.UserBackupCode, error) {
	plainCodes := make([]string, 8)
	backupCodes := make([]model.UserBackupCode, 8)
	for i := range plainCodes {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, err
		}
		plainCodes[i] = hex.EncodeToString(b)

		hash, err := bcrypt.GenerateFromPassword([]byte(plainCodes[i]), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, err
		}

		backupCodes[i] = model.UserBackupCode{CodeHash: string(hash)}
	}

	return plainCodes, backupCodes, nil
}

func replaceBackupCodes(tx *gorm.DB, userID uint, backupCodes []model.UserBackupCode) error {
	if err := tx.Where("user_id = ?", userID).Delete(&model.UserBackupCode{}).Error; err != nil {
		return err
	}

	for _, backupCode := range backupCodes {
		backupCode.UserID = userID
		if err := tx.Create(&backupCode).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *TOTPService) Disable(userID uint) error {
	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	s.clearUserSetupSessions(userID)

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

func (s *TOTPService) storeSetupSession(session totpSetupSession) (string, error) {
	sessionID, err := generateSecureToken(totpSetupSessionBytes)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupSetupSessionsLocked()
	s.clearUserSetupSessionsLocked(session.userID)
	if session.createdAt.IsZero() {
		session.createdAt = pkg.NowUTC()
	}
	s.enforceSetupSessionLimitLocked()
	s.setupSessions[sessionID] = session
	return sessionID, nil
}

func (s *TOTPService) getSetupSession(sessionID string) (totpSetupSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupSetupSessionsLocked()

	session, ok := s.setupSessions[sessionID]
	if !ok {
		return totpSetupSession{}, ErrTOTPSetupExpired
	}
	if pkg.NowUTC().After(session.expiresAt) {
		delete(s.setupSessions, sessionID)
		return totpSetupSession{}, ErrTOTPSetupExpired
	}
	return session, nil
}

func (s *TOTPService) deleteSetupSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.setupSessions, sessionID)
}

func (s *TOTPService) clearUserSetupSessions(userID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearUserSetupSessionsLocked(userID)
}

func (s *TOTPService) clearUserSetupSessionsLocked(userID uint) {
	for sessionID, session := range s.setupSessions {
		if session.userID == userID {
			delete(s.setupSessions, sessionID)
		}
	}
}

func (s *TOTPService) cleanupSetupSessionsLocked() {
	now := pkg.NowUTC()
	for sessionID, session := range s.setupSessions {
		if now.After(session.expiresAt) {
			delete(s.setupSessions, sessionID)
		}
	}
}

func (s *TOTPService) enforceSetupSessionLimitLocked() {
	overflow := len(s.setupSessions) - maxTOTPSetupSessions + 1
	if overflow <= 0 {
		return
	}

	for i := 0; i < overflow; i++ {
		oldestID := ""
		var oldestTime time.Time
		for sessionID, session := range s.setupSessions {
			if oldestID == "" || session.createdAt.Before(oldestTime) {
				oldestID = sessionID
				oldestTime = session.createdAt
			}
		}
		if oldestID == "" {
			return
		}
		delete(s.setupSessions, oldestID)
	}
}
