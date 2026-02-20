package service

import (
	"errors"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	DB *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{DB: db}
}

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}

func (s *AuthService) Register(input RegisterInput) (*AuthResponse, error) {
	// First registered user becomes admin; registration can be disabled by admin
	var userCount int64
	s.DB.Model(&model.User{}).Count(&userCount)

	if userCount > 0 {
		var setting model.SystemSetting
		if err := s.DB.Where("key = ?", "registration_enabled").First(&setting).Error; err == nil {
			if setting.Value == "false" {
				return nil, errors.New("registration is disabled")
			}
		}
	}

	var existing model.User
	if err := s.DB.Where("email = ?", input.Email).First(&existing).Error; err == nil {
		return nil, errors.New("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// First user becomes admin
	role := "user"
	if userCount == 0 {
		role = "admin"
	}

	user := model.User{
		Email:    input.Email,
		Password: string(hash),
		Role:     role,
		Status:   "active",
	}

	if err := s.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	token, err := pkg.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: user}, nil
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (s *AuthService) GetUser(userID uint) (*model.User, error) {
	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (s *AuthService) ChangePassword(userID uint, input ChangePasswordInput) error {
	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return errors.New("user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.CurrentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.DB.Model(&user).Update("password", string(hash)).Error
}

func (s *AuthService) Login(input LoginInput) (*AuthResponse, error) {
	var user model.User
	if err := s.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}

	if user.Status == "disabled" {
		return nil, errors.New("account is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	token, err := pkg.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: user}, nil
}
