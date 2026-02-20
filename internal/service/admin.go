package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminService struct {
	DB *gorm.DB
}

func NewAdminService(db *gorm.DB) *AdminService {
	return &AdminService{DB: db}
}

type ChangeRoleInput struct {
	Role string `json:"role"`
}

type ChangeStatusInput struct {
	Status string `json:"status"`
}

type AdminStats struct {
	TotalUsers         int64   `json:"total_users"`
	TotalSubscriptions int64   `json:"total_subscriptions"`
	TotalMonthlySpend  float64 `json:"total_monthly_spend"`
}

type SystemSettings struct {
	RegistrationEnabled bool   `json:"registration_enabled"`
	SiteName            string `json:"site_name"`
	SiteURL             string `json:"site_url"`
	CurrencyAPIKey      string `json:"currencyapi_key"`
	ExchangeRateSource  string `json:"exchange_rate_source"`
	MaxIconFileSize     int64  `json:"max_icon_file_size"`
}

type UpdateSettingsInput struct {
	RegistrationEnabled *bool   `json:"registration_enabled"`
	SiteName            *string `json:"site_name"`
	SiteURL             *string `json:"site_url"`
	CurrencyAPIKey      *string `json:"currencyapi_key"`
	ExchangeRateSource  *string `json:"exchange_rate_source"`
	MaxIconFileSize     *int64  `json:"max_icon_file_size"`
}

type CreateUserInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (s *AdminService) ListUsers() ([]model.User, error) {
	var users []model.User
	err := s.DB.Select("id, email, role, status, created_at, updated_at").Order("id ASC").Find(&users).Error
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

func (s *AdminService) GetStats() (*AdminStats, error) {
	var stats AdminStats

	s.DB.Model(&model.User{}).Count(&stats.TotalUsers)
	s.DB.Model(&model.Subscription{}).Count(&stats.TotalSubscriptions)

	var subs []model.Subscription
	if err := s.DB.Where("enabled = ?", true).Find(&subs).Error; err != nil {
		return nil, err
	}

	for _, sub := range subs {
		factor := subscriptionMonthlyFactor(sub)
		if factor > 0 {
			stats.TotalMonthlySpend += sub.Amount * factor
		}
	}

	return &stats, nil
}

func (s *AdminService) GetSettings() (*SystemSettings, error) {
	settings := &SystemSettings{
		RegistrationEnabled: true,
		SiteName:            "Subdux",
		SiteURL:             "",
		CurrencyAPIKey:      "",
		ExchangeRateSource:  "auto",
		MaxIconFileSize:     65536,
	}

	var items []model.SystemSetting
	s.DB.Find(&items)

	for _, item := range items {
		switch item.Key {
		case "registration_enabled":
			settings.RegistrationEnabled = item.Value == "true"
		case "site_name":
			settings.SiteName = item.Value
		case "site_url":
			settings.SiteURL = item.Value
		case "currencyapi_key":
			settings.CurrencyAPIKey = item.Value
		case "exchange_rate_source":
			settings.ExchangeRateSource = item.Value
		case "max_icon_file_size":
			if v, err := strconv.ParseInt(item.Value, 10, 64); err == nil {
				settings.MaxIconFileSize = v
			}
		}
	}

	return settings, nil
}

func (s *AdminService) UpdateSettings(input UpdateSettingsInput) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if input.RegistrationEnabled != nil {
			value := "false"
			if *input.RegistrationEnabled {
				value = "true"
			}
			if err := tx.Where("key = ?", "registration_enabled").
				Assign(model.SystemSetting{Value: value}).
				FirstOrCreate(&model.SystemSetting{Key: "registration_enabled"}).Error; err != nil {
				return err
			}
		}

		if input.SiteName != nil {
			if err := tx.Where("key = ?", "site_name").
				Assign(model.SystemSetting{Value: *input.SiteName}).
				FirstOrCreate(&model.SystemSetting{Key: "site_name"}).Error; err != nil {
				return err
			}
		}

		if input.SiteURL != nil {
			if err := tx.Where("key = ?", "site_url").
				Assign(model.SystemSetting{Value: *input.SiteURL}).
				FirstOrCreate(&model.SystemSetting{Key: "site_url"}).Error; err != nil {
				return err
			}
		}

		if input.CurrencyAPIKey != nil {
			if err := tx.Where("key = ?", "currencyapi_key").
				Assign(model.SystemSetting{Value: *input.CurrencyAPIKey}).
				FirstOrCreate(&model.SystemSetting{Key: "currencyapi_key"}).Error; err != nil {
				return err
			}
		}

		if input.ExchangeRateSource != nil {
			if err := tx.Where("key = ?", "exchange_rate_source").
				Assign(model.SystemSetting{Value: *input.ExchangeRateSource}).
				FirstOrCreate(&model.SystemSetting{Key: "exchange_rate_source"}).Error; err != nil {
				return err
			}
		}

		if input.MaxIconFileSize != nil {
			value := strconv.FormatInt(*input.MaxIconFileSize, 10)
			if err := tx.Where("key = ?", "max_icon_file_size").
				Assign(model.SystemSetting{Value: value}).
				FirstOrCreate(&model.SystemSetting{Key: "max_icon_file_size"}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *AdminService) CreateUser(input CreateUserInput) (*model.User, error) {
	if input.Username == "" || input.Email == "" || input.Password == "" {
		return nil, errors.New("username, email and password are required")
	}

	if len(input.Password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
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

func (s *AdminService) BackupDB() (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-backup-%s.db", timestamp))

	query := fmt.Sprintf(`VACUUM INTO '%s'`, backupPath)
	if err := s.DB.Exec(query).Error; err != nil {
		return "", err
	}

	return backupPath, nil
}
