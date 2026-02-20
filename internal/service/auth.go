package service

import (
	"errors"
	"sync"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	DB *gorm.DB

	passkeyMu       sync.Mutex
	passkeySessions map[string]passkeySession

	oidcMu             sync.Mutex
	oidcStateSessions  map[string]oidcStateSession
	oidcResultSessions map[string]oidcResultSession
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{
		DB:                 db,
		passkeySessions:    make(map[string]passkeySession),
		oidcStateSessions:  make(map[string]oidcStateSession),
		oidcResultSessions: make(map[string]oidcResultSession),
	}
}

type RegisterInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginInput struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type AuthResponse struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}

type LoginResponse struct {
	RequiresTotp bool        `json:"requires_totp"`
	TotpToken    string      `json:"totp_token,omitempty"`
	Token        string      `json:"token,omitempty"`
	User         *model.User `json:"user,omitempty"`
}

func (s *AuthService) Register(input RegisterInput) (*AuthResponse, error) {
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
	if err := s.DB.Where("username = ?", input.Username).First(&existing).Error; err == nil {
		return nil, errors.New("username already taken")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	role := "user"
	if userCount == 0 {
		role = "admin"
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

	token, err := pkg.GenerateToken(user.ID, user.Username, user.Email, user.Role)
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

func (s *AuthService) Login(input LoginInput) (*LoginResponse, error) {
	var user model.User
	if err := s.DB.Where("email = ? OR username = ?", input.Identifier, input.Identifier).First(&user).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}

	if user.Status == "disabled" {
		return nil, errors.New("account is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	if user.TotpEnabled {
		pendingToken, err := pkg.GenerateTOTPPendingToken(user.ID)
		if err != nil {
			return nil, err
		}
		return &LoginResponse{RequiresTotp: true, TotpToken: pendingToken}, nil
	}

	token, err := pkg.GenerateToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{Token: token, User: &user}, nil
}
