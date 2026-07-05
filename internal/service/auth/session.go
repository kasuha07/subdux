package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

var (
	ErrInvalidRefreshToken = serviceerr.New(serviceerr.KindUnauthorized, "invalid_refresh_token", "invalid refresh token")
	// ErrAccountDisabled is returned when a session is requested for a
	// non-active account. It is 401 (KindUnauthorized): the credential is valid
	// but the principal may no longer sign in.
	ErrAccountDisabled = serviceerr.New(serviceerr.KindUnauthorized, "account_is_disabled", "account is disabled")
)

func (s *Service) CreateSession(userID uint) (*AuthResponse, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}
	if user.Status != "active" {
		return nil, ErrAccountDisabled
	}
	return s.issueAuthResponse(*user)
}

func (s *Service) issueAuthResponse(user model.User) (*AuthResponse, error) {
	accessToken, err := pkg.GenerateAccessToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshTokenHash, refreshExpiresAt, err := pkg.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	record := model.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: refreshExpiresAt,
	}

	if err := s.DB.Create(&record).Error; err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *Service) RefreshSession(rawRefreshToken string) (*AuthResponse, error) {
	rawRefreshToken = strings.TrimSpace(rawRefreshToken)
	if rawRefreshToken == "" {
		return nil, ErrInvalidRefreshToken
	}

	tokenHash := pkg.HashRefreshToken(rawRefreshToken)
	now := pkg.NowUTC()

	var (
		user              model.User
		accessToken       string
		newRefreshToken   string
		newRefreshHash    string
		newRefreshExpires time.Time
	)

	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		var stored model.RefreshToken
		if err := tx.Where("token_hash = ?", tokenHash).First(&stored).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidRefreshToken
			}
			return err
		}

		if stored.RevokedAt != nil || now.After(stored.ExpiresAt) {
			return ErrInvalidRefreshToken
		}

		if err := tx.First(&user, stored.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidRefreshToken
			}
			return err
		}
		if user.Status != "active" {
			return ErrInvalidRefreshToken
		}

		var err error
		accessToken, err = pkg.GenerateAccessToken(user.ID, user.Username, user.Email, user.Role)
		if err != nil {
			return err
		}

		newRefreshToken, newRefreshHash, newRefreshExpires, err = pkg.GenerateRefreshToken()
		if err != nil {
			return err
		}

		updateResult := tx.Model(&model.RefreshToken{}).
			Where("id = ? AND revoked_at IS NULL", stored.ID).
			Updates(map[string]interface{}{
				"revoked_at":   &now,
				"last_used_at": &now,
			})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return ErrInvalidRefreshToken
		}

		return tx.Create(&model.RefreshToken{
			UserID:    user.ID,
			TokenHash: newRefreshHash,
			ExpiresAt: newRefreshExpires,
		}).Error
	}); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		User:         user,
	}, nil
}

func (s *Service) Logout(rawRefreshToken string) error {
	rawRefreshToken = strings.TrimSpace(rawRefreshToken)
	if rawRefreshToken == "" {
		return nil
	}

	now := pkg.NowUTC()
	return s.DB.Model(&model.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", pkg.HashRefreshToken(rawRefreshToken)).
		Updates(map[string]interface{}{
			"revoked_at":   &now,
			"last_used_at": &now,
		}).Error
}

func (s *Service) LogoutAll(userID uint) error {
	return revokeAllRefreshTokens(s.DB, userID)
}

func revokeAllRefreshTokens(tx *gorm.DB, userID uint) error {
	now := pkg.NowUTC()
	return tx.Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error
}
