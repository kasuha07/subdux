package service

import (
	"strings"

	"github.com/shiroha/subdux/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func (s *AuthService) EnsureInitialAdmin(input InitialAdminInput) (*InitialAdminResult, error) {
	var userCount int64
	if err := s.DB.Model(&model.User{}).Count(&userCount).Error; err != nil {
		return nil, err
	}
	if userCount > 0 {
		return &InitialAdminResult{Created: false}, nil
	}

	username := strings.TrimSpace(input.Username)
	if username == "" {
		username = "admin"
	}

	email, err := sanitizeAndValidateEmail(input.Email)
	if err != nil {
		return nil, err
	}

	password := input.Password
	if password == "" {
		password, err = generateSecureToken(24)
		if err != nil {
			return nil, err
		}
	}
	if err := validateBcryptPasswordLength(password); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := model.User{
		Username: username,
		Email:    email,
		Password: string(hash),
		Role:     "admin",
		Status:   "active",
	}

	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		var currentCount int64
		if err := tx.Model(&model.User{}).Count(&currentCount).Error; err != nil {
			return err
		}
		if currentCount > 0 {
			return nil
		}

		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return SeedUserDefaults(tx, user.ID)
	}); err != nil {
		return nil, err
	}

	if user.ID == 0 {
		return &InitialAdminResult{Created: false}, nil
	}

	return &InitialAdminResult{
		Created:  true,
		Username: username,
		Email:    email,
		Password: password,
	}, nil
}
