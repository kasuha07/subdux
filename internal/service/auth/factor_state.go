package auth

import (
	"errors"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type FactorState struct {
	HasPassword bool
	HasTOTP     bool
	HasPasskey  bool
	HasOIDC     bool
}

func (s *Service) FactorState(userID uint) (FactorState, error) {
	var user model.User
	if err := s.DB.Select("id", "password", "totp_enabled").First(&user, userID).Error; err != nil {
		return FactorState{}, err
	}

	hasPasskey, err := s.HasPasskeys(userID)
	if err != nil {
		return FactorState{}, err
	}
	hasOIDC, err := s.CanReauthWithOIDC(userID)
	if err != nil {
		return FactorState{}, err
	}

	return FactorState{
		HasPassword: user.Password != "",
		HasTOTP:     user.TotpEnabled,
		HasPasskey:  hasPasskey,
		HasOIDC:     hasOIDC,
	}, nil
}

func (s *Service) VerifyPasswordForReauth(userID uint, password string, code string, requireTOTP bool) error {
	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return ErrCurrentPasswordIncorrect
	}
	if requireTOTP {
		if code == "" || user.TotpSecret == nil || !totp.Validate(code, *user.TotpSecret) {
			return ErrTOTPInvalidCode
		}
	}
	return nil
}
