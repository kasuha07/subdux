package service

import (
	"errors"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

func (s *AuthService) CreateSession(userID uint) (*AuthResponse, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}
	if user.Status != "active" {
		return nil, errors.New("account is disabled")
	}
	return s.issueAuthResponse(*user)
}

func (s *AuthService) issueAuthResponse(user model.User) (*AuthResponse, error) {
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

func (s *AuthService) RefreshSession(rawRefreshToken string) (*AuthResponse, error) {
	rawRefreshToken = strings.TrimSpace(rawRefreshToken)
	if rawRefreshToken == "" {
		return nil, ErrInvalidRefreshToken
	}

	tokenHash := pkg.HashRefreshToken(rawRefreshToken)
	now := time.Now().UTC()

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

func revokeAllRefreshTokens(tx *gorm.DB, userID uint) error {
	now := time.Now().UTC()
	return tx.Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error
}
