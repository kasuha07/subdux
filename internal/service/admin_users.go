package service

import (
	"errors"
	"os"

	"github.com/shiroha/subdux/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func (s *AdminService) ListUsers() ([]model.User, error) {
	var users []model.User
	err := s.DB.Select("id, email, role, status, created_at").Order("id ASC").Find(&users).Error
	return users, err
}

func (s *AdminService) ChangeUserRole(userID uint, role string) error {
	if role != "admin" && role != "user" {
		return errors.New("invalid role")
	}
	// Prevent demoting the first user (ID=1) to regular user
	if userID == 1 && role == "user" {
		return errors.New("cannot change the first user's role to regular user")
	}
	return s.DB.Model(&model.User{}).Where("id = ?", userID).Update("role", role).Error
}

func (s *AdminService) ChangeUserStatus(userID uint, status string) error {
	if status != "active" && status != "disabled" {
		return errors.New("invalid status")
	}
	// Prevent disabling the first user (ID=1)
	if userID == 1 && status == "disabled" {
		return errors.New("cannot disable the first user")
	}
	return s.DB.Model(&model.User{}).Where("id = ?", userID).Update("status", status).Error
}

func (s *AdminService) DeleteUser(userID uint) error {
	// Prevent deleting the first user (ID=1)
	if userID == 1 {
		return errors.New("cannot delete the first user")
	}

	var subscriptionIcons []string
	if err := s.DB.Model(&model.Subscription{}).Where("user_id = ?", userID).Pluck("icon", &subscriptionIcons).Error; err != nil {
		return err
	}
	var paymentMethodIcons []string
	if err := s.DB.Model(&model.PaymentMethod{}).Where("user_id = ?", userID).Pluck("icon", &paymentMethodIcons).Error; err != nil {
		return err
	}

	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.Subscription{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.PaymentMethod{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserCurrency{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.Category{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserPreference{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserBackupCode{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.PasskeyCredential{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.OIDCConnection{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.EmailVerificationCode{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.User{}, userID).Error
	}); err != nil {
		return err
	}

	for _, icon := range subscriptionIcons {
		if path, ok := managedIconFilePath(icon); ok {
			_ = os.Remove(path)
		}
	}
	for _, icon := range paymentMethodIcons {
		if path, ok := managedIconFilePath(icon); ok {
			_ = os.Remove(path)
		}
	}

	return nil
}

func (s *AdminService) CreateUser(input CreateUserInput) (*model.User, error) {
	if input.Username == "" || input.Email == "" || input.Password == "" {
		return nil, errors.New("username, email and password are required")
	}

	if len(input.Password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}
	if err := validateBcryptPasswordLength(input.Password); err != nil {
		return nil, err
	}

	role := input.Role
	if role != "admin" && role != "user" {
		role = "user"
	}

	var existing model.User
	if err := s.DB.Where("email = ?", input.Email).First(&existing).Error; err == nil {
		return nil, errors.New("email already registered")
	}
	if err := s.DB.Where("username = ?", input.Username).First(&existing).Error; err == nil {
		return nil, errors.New("username already taken")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := model.User{
		Username: input.Username,
		Email:    input.Email,
		Password: string(hash),
		Role:     role,
		Status:   "active",
	}

	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return SeedUserDefaults(tx, user.ID)
	}); err != nil {
		return nil, err
	}

	return &user, nil
}
