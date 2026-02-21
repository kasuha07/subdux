package service

import (
	"archive/zip"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
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
	RegistrationEnabled                  bool   `json:"registration_enabled"`
	RegistrationEmailVerificationEnabled bool   `json:"registration_email_verification_enabled"`
	SiteName                             string `json:"site_name"`
	SiteURL                              string `json:"site_url"`
	CurrencyAPIKeySet                    bool   `json:"currencyapi_key_configured"`
	ExchangeRateSource                   string `json:"exchange_rate_source"`
	MaxIconFileSize                      int64  `json:"max_icon_file_size"`
	SMTPEnabled                          bool   `json:"smtp_enabled"`
	SMTPHost                             string `json:"smtp_host"`
	SMTPPort                             int64  `json:"smtp_port"`
	SMTPUsername                         string `json:"smtp_username"`
	SMTPPasswordSet                      bool   `json:"smtp_password_configured"`
	SMTPFromEmail                        string `json:"smtp_from_email"`
	SMTPFromName                         string `json:"smtp_from_name"`
	SMTPEncryption                       string `json:"smtp_encryption"`
	SMTPAuthMethod                       string `json:"smtp_auth_method"`
	SMTPHeloName                         string `json:"smtp_helo_name"`
	SMTPTimeoutSeconds                   int64  `json:"smtp_timeout_seconds"`
	SMTPSkipTLSVerify                    bool   `json:"smtp_skip_tls_verify"`
	OIDCEnabled                          bool   `json:"oidc_enabled"`
	OIDCProviderName                     string `json:"oidc_provider_name"`
	OIDCIssuerURL                        string `json:"oidc_issuer_url"`
	OIDCClientID                         string `json:"oidc_client_id"`
	OIDCClientSecretSet                  bool   `json:"oidc_client_secret_configured"`
	OIDCRedirectURL                      string `json:"oidc_redirect_url"`
	OIDCScopes                           string `json:"oidc_scopes"`
	OIDCAutoCreateUser                   bool   `json:"oidc_auto_create_user"`
	OIDCAuthorizeURL                     string `json:"oidc_authorization_endpoint"`
	OIDCTokenURL                         string `json:"oidc_token_endpoint"`
	OIDCUserinfoURL                      string `json:"oidc_userinfo_endpoint"`
	OIDCAudience                         string `json:"oidc_audience"`
	OIDCResource                         string `json:"oidc_resource"`
	OIDCExtraAuthParams                  string `json:"oidc_extra_auth_params"`
}

type UpdateSettingsInput struct {
	RegistrationEnabled                  *bool   `json:"registration_enabled"`
	RegistrationEmailVerificationEnabled *bool   `json:"registration_email_verification_enabled"`
	SiteName                             *string `json:"site_name"`
	SiteURL                              *string `json:"site_url"`
	CurrencyAPIKey                       *string `json:"currencyapi_key"`
	ExchangeRateSource                   *string `json:"exchange_rate_source"`
	MaxIconFileSize                      *int64  `json:"max_icon_file_size"`
	SMTPEnabled                          *bool   `json:"smtp_enabled"`
	SMTPHost                             *string `json:"smtp_host"`
	SMTPPort                             *int64  `json:"smtp_port"`
	SMTPUsername                         *string `json:"smtp_username"`
	SMTPPassword                         *string `json:"smtp_password"`
	SMTPFromEmail                        *string `json:"smtp_from_email"`
	SMTPFromName                         *string `json:"smtp_from_name"`
	SMTPEncryption                       *string `json:"smtp_encryption"`
	SMTPAuthMethod                       *string `json:"smtp_auth_method"`
	SMTPHeloName                         *string `json:"smtp_helo_name"`
	SMTPTimeoutSeconds                   *int64  `json:"smtp_timeout_seconds"`
	SMTPSkipTLSVerify                    *bool   `json:"smtp_skip_tls_verify"`
	OIDCEnabled                          *bool   `json:"oidc_enabled"`
	OIDCProviderName                     *string `json:"oidc_provider_name"`
	OIDCIssuerURL                        *string `json:"oidc_issuer_url"`
	OIDCClientID                         *string `json:"oidc_client_id"`
	OIDCClientSecret                     *string `json:"oidc_client_secret"`
	OIDCRedirectURL                      *string `json:"oidc_redirect_url"`
	OIDCScopes                           *string `json:"oidc_scopes"`
	OIDCAutoCreateUser                   *bool   `json:"oidc_auto_create_user"`
	OIDCAuthorizeURL                     *string `json:"oidc_authorization_endpoint"`
	OIDCTokenURL                         *string `json:"oidc_token_endpoint"`
	OIDCUserinfoURL                      *string `json:"oidc_userinfo_endpoint"`
	OIDCAudience                         *string `json:"oidc_audience"`
	OIDCResource                         *string `json:"oidc_resource"`
	OIDCExtraAuthParams                  *string `json:"oidc_extra_auth_params"`
}

type CreateUserInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

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
		RegistrationEnabled:                  true,
		RegistrationEmailVerificationEnabled: false,
		SiteName:                             "Subdux",
		SiteURL:                              "",
		CurrencyAPIKeySet:                    false,
		ExchangeRateSource:                   "auto",
		MaxIconFileSize:                      65536,
		SMTPEnabled:                          false,
		SMTPHost:                             "",
		SMTPPort:                             587,
		SMTPUsername:                         "",
		SMTPPasswordSet:                      false,
		SMTPFromEmail:                        "",
		SMTPFromName:                         "",
		SMTPEncryption:                       "starttls",
		SMTPAuthMethod:                       "auto",
		SMTPHeloName:                         "",
		SMTPTimeoutSeconds:                   10,
		SMTPSkipTLSVerify:                    false,
		OIDCEnabled:                          false,
		OIDCProviderName:                     "OIDC",
		OIDCIssuerURL:                        "",
		OIDCClientID:                         "",
		OIDCClientSecretSet:                  false,
		OIDCRedirectURL:                      "",
		OIDCScopes:                           "openid profile email",
		OIDCAutoCreateUser:                   false,
		OIDCAuthorizeURL:                     "",
		OIDCTokenURL:                         "",
		OIDCUserinfoURL:                      "",
		OIDCAudience:                         "",
		OIDCResource:                         "",
		OIDCExtraAuthParams:                  "",
	}

	var items []model.SystemSetting
	s.DB.Find(&items)

	for _, item := range items {
		switch item.Key {
		case "registration_enabled":
			settings.RegistrationEnabled = item.Value == "true"
		case "registration_email_verification_enabled":
			settings.RegistrationEmailVerificationEnabled = item.Value == "true"
		case "site_name":
			settings.SiteName = item.Value
		case "site_url":
			settings.SiteURL = item.Value
		case "currencyapi_key":
			settings.CurrencyAPIKeySet = strings.TrimSpace(item.Value) != ""
		case "exchange_rate_source":
			settings.ExchangeRateSource = item.Value
		case "max_icon_file_size":
			if v, err := strconv.ParseInt(item.Value, 10, 64); err == nil {
				settings.MaxIconFileSize = v
			}
		case "smtp_enabled":
			settings.SMTPEnabled = item.Value == "true"
		case "smtp_host":
			settings.SMTPHost = item.Value
		case "smtp_port":
			if v, err := strconv.ParseInt(item.Value, 10, 64); err == nil {
				settings.SMTPPort = v
			}
		case "smtp_username":
			settings.SMTPUsername = item.Value
		case "smtp_password":
			settings.SMTPPasswordSet = strings.TrimSpace(item.Value) != ""
		case "smtp_from_email":
			settings.SMTPFromEmail = item.Value
		case "smtp_from_name":
			settings.SMTPFromName = item.Value
		case "smtp_encryption":
			settings.SMTPEncryption = item.Value
		case "smtp_auth_method":
			settings.SMTPAuthMethod = item.Value
		case "smtp_helo_name":
			settings.SMTPHeloName = item.Value
		case "smtp_timeout_seconds":
			if v, err := strconv.ParseInt(item.Value, 10, 64); err == nil {
				settings.SMTPTimeoutSeconds = v
			}
		case "smtp_skip_tls_verify":
			settings.SMTPSkipTLSVerify = item.Value == "true"
		case "oidc_enabled":
			settings.OIDCEnabled = item.Value == "true"
		case "oidc_provider_name":
			settings.OIDCProviderName = item.Value
		case "oidc_issuer_url":
			settings.OIDCIssuerURL = item.Value
		case "oidc_client_id":
			settings.OIDCClientID = item.Value
		case "oidc_client_secret":
			settings.OIDCClientSecretSet = strings.TrimSpace(item.Value) != ""
		case "oidc_redirect_url":
			settings.OIDCRedirectURL = item.Value
		case "oidc_scopes":
			settings.OIDCScopes = item.Value
		case "oidc_auto_create_user":
			settings.OIDCAutoCreateUser = item.Value == "true"
		case "oidc_authorization_endpoint":
			settings.OIDCAuthorizeURL = item.Value
		case "oidc_token_endpoint":
			settings.OIDCTokenURL = item.Value
		case "oidc_userinfo_endpoint":
			settings.OIDCUserinfoURL = item.Value
		case "oidc_audience":
			settings.OIDCAudience = item.Value
		case "oidc_resource":
			settings.OIDCResource = item.Value
		case "oidc_extra_auth_params":
			settings.OIDCExtraAuthParams = item.Value
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

		if input.RegistrationEmailVerificationEnabled != nil {
			value := "false"
			if *input.RegistrationEmailVerificationEnabled {
				value = "true"
			}
			if err := tx.Where("key = ?", "registration_email_verification_enabled").
				Assign(model.SystemSetting{Value: value}).
				FirstOrCreate(&model.SystemSetting{Key: "registration_email_verification_enabled"}).Error; err != nil {
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

		if input.SMTPEnabled != nil {
			value := "false"
			if *input.SMTPEnabled {
				value = "true"
			}
			if err := tx.Where("key = ?", "smtp_enabled").
				Assign(model.SystemSetting{Value: value}).
				FirstOrCreate(&model.SystemSetting{Key: "smtp_enabled"}).Error; err != nil {
				return err
			}
		}

		if input.SMTPHost != nil {
			if err := tx.Where("key = ?", "smtp_host").
				Assign(model.SystemSetting{Value: *input.SMTPHost}).
				FirstOrCreate(&model.SystemSetting{Key: "smtp_host"}).Error; err != nil {
				return err
			}
		}

		if input.SMTPPort != nil {
			value := strconv.FormatInt(*input.SMTPPort, 10)
			if err := tx.Where("key = ?", "smtp_port").
				Assign(model.SystemSetting{Value: value}).
				FirstOrCreate(&model.SystemSetting{Key: "smtp_port"}).Error; err != nil {
				return err
			}
		}

		if input.SMTPUsername != nil {
			if err := tx.Where("key = ?", "smtp_username").
				Assign(model.SystemSetting{Value: *input.SMTPUsername}).
				FirstOrCreate(&model.SystemSetting{Key: "smtp_username"}).Error; err != nil {
				return err
			}
		}

		if input.SMTPPassword != nil {
			if err := tx.Where("key = ?", "smtp_password").
				Assign(model.SystemSetting{Value: *input.SMTPPassword}).
				FirstOrCreate(&model.SystemSetting{Key: "smtp_password"}).Error; err != nil {
				return err
			}
		}

		if input.SMTPFromEmail != nil {
			if err := tx.Where("key = ?", "smtp_from_email").
				Assign(model.SystemSetting{Value: *input.SMTPFromEmail}).
				FirstOrCreate(&model.SystemSetting{Key: "smtp_from_email"}).Error; err != nil {
				return err
			}
		}

		if input.SMTPFromName != nil {
			if err := tx.Where("key = ?", "smtp_from_name").
				Assign(model.SystemSetting{Value: *input.SMTPFromName}).
				FirstOrCreate(&model.SystemSetting{Key: "smtp_from_name"}).Error; err != nil {
				return err
			}
		}

		if input.SMTPEncryption != nil {
			if err := tx.Where("key = ?", "smtp_encryption").
				Assign(model.SystemSetting{Value: *input.SMTPEncryption}).
				FirstOrCreate(&model.SystemSetting{Key: "smtp_encryption"}).Error; err != nil {
				return err
			}
		}

		if input.SMTPAuthMethod != nil {
			if err := tx.Where("key = ?", "smtp_auth_method").
				Assign(model.SystemSetting{Value: *input.SMTPAuthMethod}).
				FirstOrCreate(&model.SystemSetting{Key: "smtp_auth_method"}).Error; err != nil {
				return err
			}
		}

		if input.SMTPHeloName != nil {
			if err := tx.Where("key = ?", "smtp_helo_name").
				Assign(model.SystemSetting{Value: *input.SMTPHeloName}).
				FirstOrCreate(&model.SystemSetting{Key: "smtp_helo_name"}).Error; err != nil {
				return err
			}
		}

		if input.SMTPTimeoutSeconds != nil {
			value := strconv.FormatInt(*input.SMTPTimeoutSeconds, 10)
			if err := tx.Where("key = ?", "smtp_timeout_seconds").
				Assign(model.SystemSetting{Value: value}).
				FirstOrCreate(&model.SystemSetting{Key: "smtp_timeout_seconds"}).Error; err != nil {
				return err
			}
		}

		if input.SMTPSkipTLSVerify != nil {
			value := "false"
			if *input.SMTPSkipTLSVerify {
				value = "true"
			}
			if err := tx.Where("key = ?", "smtp_skip_tls_verify").
				Assign(model.SystemSetting{Value: value}).
				FirstOrCreate(&model.SystemSetting{Key: "smtp_skip_tls_verify"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCEnabled != nil {
			value := "false"
			if *input.OIDCEnabled {
				value = "true"
			}
			if err := tx.Where("key = ?", "oidc_enabled").
				Assign(model.SystemSetting{Value: value}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_enabled"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCProviderName != nil {
			if err := tx.Where("key = ?", "oidc_provider_name").
				Assign(model.SystemSetting{Value: *input.OIDCProviderName}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_provider_name"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCIssuerURL != nil {
			if err := tx.Where("key = ?", "oidc_issuer_url").
				Assign(model.SystemSetting{Value: *input.OIDCIssuerURL}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_issuer_url"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCClientID != nil {
			if err := tx.Where("key = ?", "oidc_client_id").
				Assign(model.SystemSetting{Value: *input.OIDCClientID}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_client_id"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCClientSecret != nil {
			if err := tx.Where("key = ?", "oidc_client_secret").
				Assign(model.SystemSetting{Value: *input.OIDCClientSecret}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_client_secret"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCRedirectURL != nil {
			if err := tx.Where("key = ?", "oidc_redirect_url").
				Assign(model.SystemSetting{Value: *input.OIDCRedirectURL}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_redirect_url"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCScopes != nil {
			if err := tx.Where("key = ?", "oidc_scopes").
				Assign(model.SystemSetting{Value: *input.OIDCScopes}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_scopes"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCAutoCreateUser != nil {
			value := "false"
			if *input.OIDCAutoCreateUser {
				value = "true"
			}
			if err := tx.Where("key = ?", "oidc_auto_create_user").
				Assign(model.SystemSetting{Value: value}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_auto_create_user"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCAuthorizeURL != nil {
			if err := tx.Where("key = ?", "oidc_authorization_endpoint").
				Assign(model.SystemSetting{Value: *input.OIDCAuthorizeURL}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_authorization_endpoint"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCTokenURL != nil {
			if err := tx.Where("key = ?", "oidc_token_endpoint").
				Assign(model.SystemSetting{Value: *input.OIDCTokenURL}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_token_endpoint"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCUserinfoURL != nil {
			if err := tx.Where("key = ?", "oidc_userinfo_endpoint").
				Assign(model.SystemSetting{Value: *input.OIDCUserinfoURL}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_userinfo_endpoint"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCAudience != nil {
			if err := tx.Where("key = ?", "oidc_audience").
				Assign(model.SystemSetting{Value: *input.OIDCAudience}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_audience"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCResource != nil {
			if err := tx.Where("key = ?", "oidc_resource").
				Assign(model.SystemSetting{Value: *input.OIDCResource}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_resource"}).Error; err != nil {
				return err
			}
		}

		if input.OIDCExtraAuthParams != nil {
			if err := tx.Where("key = ?", "oidc_extra_auth_params").
				Assign(model.SystemSetting{Value: *input.OIDCExtraAuthParams}).
				FirstOrCreate(&model.SystemSetting{Key: "oidc_extra_auth_params"}).Error; err != nil {
				return err
			}
		}

		registrationEmailVerificationEnabled, err := isSystemSettingEnabled(
			tx,
			"registration_email_verification_enabled",
			false,
		)
		if err != nil {
			return err
		}
		if registrationEmailVerificationEnabled {
			if _, err := loadSMTPRuntimeConfig(tx); err != nil {
				return errors.New("smtp settings must be valid when registration email verification is enabled")
			}
		}

		return nil
	})
}

func isSystemSettingEnabled(tx *gorm.DB, key string, defaultValue bool) (bool, error) {
	var setting model.SystemSetting
	if err := tx.Where("key = ?", key).First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultValue, nil
		}
		return defaultValue, err
	}
	return setting.Value == "true", nil
}

type smtpRuntimeConfig struct {
	Host           string
	Port           int64
	Username       string
	Password       string
	FromEmail      string
	FromName       string
	Encryption     string
	AuthMethod     string
	HeloName       string
	TimeoutSeconds int64
	SkipTLSVerify  bool
}

type smtpLoginAuth struct {
	username string
	password string
}

func (a *smtpLoginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *smtpLoginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}

	prompt := strings.ToLower(strings.TrimSpace(string(fromServer)))
	switch {
	case strings.Contains(prompt, "username"):
		return []byte(a.username), nil
	case strings.Contains(prompt, "password"):
		return []byte(a.password), nil
	default:
		return nil, errors.New("unsupported smtp login challenge")
	}
}

func (s *AdminService) SendSMTPTestEmail(userID uint, recipientOverride string) error {
	cfg, err := loadSMTPRuntimeConfig(s.DB)
	if err != nil {
		return err
	}

	recipient := strings.TrimSpace(recipientOverride)
	if recipient == "" {
		var user model.User
		if err := s.DB.Select("email").First(&user, userID).Error; err != nil {
			return errors.New("failed to load current user email")
		}
		recipient = strings.TrimSpace(user.Email)
	}

	if recipient == "" {
		return errors.New("recipient email is required for smtp test")
	}

	if _, err := mail.ParseAddress(recipient); err != nil {
		return errors.New("invalid recipient email")
	}

	subject := "Subdux SMTP Test"
	body := fmt.Sprintf("This is a test email from Subdux.\r\nSent at: %s", time.Now().Format(time.RFC3339))
	message := buildSMTPMessage(cfg.FromEmail, cfg.FromName, recipient, subject, body)

	if err := sendSMTPMessage(*cfg, recipient, message); err != nil {
		return err
	}

	return nil
}

func (s *AdminService) loadSMTPRuntimeConfig() (*smtpRuntimeConfig, error) {
	return loadSMTPRuntimeConfig(s.DB)
}

func loadSMTPRuntimeConfig(db *gorm.DB) (*smtpRuntimeConfig, error) {
	if db == nil {
		return nil, errors.New("failed to load smtp settings")
	}

	defaults := map[string]string{
		"smtp_enabled":         "false",
		"smtp_host":            "",
		"smtp_port":            "587",
		"smtp_username":        "",
		"smtp_password":        "",
		"smtp_from_email":      "",
		"smtp_from_name":       "",
		"smtp_encryption":      "starttls",
		"smtp_auth_method":     "auto",
		"smtp_helo_name":       "",
		"smtp_timeout_seconds": "10",
		"smtp_skip_tls_verify": "false",
	}

	keys := make([]string, 0, len(defaults))
	values := make(map[string]string, len(defaults))
	for key, value := range defaults {
		keys = append(keys, key)
		values[key] = value
	}

	var items []model.SystemSetting
	if err := db.Where("key IN ?", keys).Find(&items).Error; err != nil {
		return nil, errors.New("failed to load smtp settings")
	}
	for _, item := range items {
		values[item.Key] = item.Value
	}

	if values["smtp_enabled"] != "true" {
		return nil, errors.New("smtp is disabled")
	}

	host := strings.TrimSpace(values["smtp_host"])
	if host == "" {
		return nil, errors.New("smtp host is required")
	}

	port, err := strconv.ParseInt(strings.TrimSpace(values["smtp_port"]), 10, 64)
	if err != nil || port < 1 || port > 65535 {
		return nil, errors.New("smtp port must be between 1 and 65535")
	}

	fromEmail := strings.TrimSpace(values["smtp_from_email"])
	if fromEmail == "" {
		return nil, errors.New("smtp from email is required")
	}

	encryption := strings.ToLower(strings.TrimSpace(values["smtp_encryption"]))
	if encryption == "" {
		encryption = "starttls"
	}
	switch encryption {
	case "starttls", "ssl_tls", "none":
	default:
		return nil, errors.New("unsupported smtp encryption mode")
	}

	authMethod := strings.ToLower(strings.TrimSpace(values["smtp_auth_method"]))
	if authMethod == "" {
		authMethod = "auto"
	}
	switch authMethod {
	case "auto", "plain", "login", "cram_md5", "none":
	default:
		return nil, errors.New("unsupported smtp auth method")
	}

	timeoutSeconds, err := strconv.ParseInt(strings.TrimSpace(values["smtp_timeout_seconds"]), 10, 64)
	if err != nil || timeoutSeconds <= 0 {
		timeoutSeconds = 10
	}

	username := strings.TrimSpace(values["smtp_username"])
	password := values["smtp_password"]

	if authMethod != "auto" && authMethod != "none" && (username == "" || strings.TrimSpace(password) == "") {
		return nil, errors.New("smtp username and password are required for selected auth method")
	}

	return &smtpRuntimeConfig{
		Host:           host,
		Port:           port,
		Username:       username,
		Password:       password,
		FromEmail:      fromEmail,
		FromName:       strings.TrimSpace(values["smtp_from_name"]),
		Encryption:     encryption,
		AuthMethod:     authMethod,
		HeloName:       strings.TrimSpace(values["smtp_helo_name"]),
		TimeoutSeconds: timeoutSeconds,
		SkipTLSVerify:  values["smtp_skip_tls_verify"] == "true",
	}, nil
}

func buildSMTPMessage(fromEmail string, fromName string, toEmail string, subject string, body string) []byte {
	escapedName := strings.ReplaceAll(fromName, "\"", "'")
	fromHeader := fromEmail
	if strings.TrimSpace(escapedName) != "" {
		fromHeader = fmt.Sprintf("\"%s\" <%s>", escapedName, fromEmail)
	}

	headers := []string{
		fmt.Sprintf("From: %s", fromHeader),
		fmt.Sprintf("To: %s", toEmail),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}

	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body + "\r\n")
}

func sendSMTPMessage(cfg smtpRuntimeConfig, recipient string, message []byte) error {
	address := net.JoinHostPort(cfg.Host, strconv.FormatInt(cfg.Port, 10))
	dialer := net.Dialer{
		Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
	}

	var client *smtp.Client
	if cfg.Encryption == "ssl_tls" {
		conn, err := tls.DialWithDialer(&dialer, "tcp", address, &tls.Config{
			ServerName:         cfg.Host,
			InsecureSkipVerify: cfg.SkipTLSVerify,
		})
		if err != nil {
			return fmt.Errorf("failed to connect to smtp server: %w", err)
		}
		client, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("failed to initialize smtp client: %w", err)
		}
	} else {
		conn, err := dialer.Dial("tcp", address)
		if err != nil {
			return fmt.Errorf("failed to connect to smtp server: %w", err)
		}
		client, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("failed to initialize smtp client: %w", err)
		}
	}
	defer client.Close()

	if cfg.HeloName != "" {
		if err := client.Hello(cfg.HeloName); err != nil {
			return fmt.Errorf("smtp HELO/EHLO failed: %w", err)
		}
	}

	if cfg.Encryption == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("smtp server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{
			ServerName:         cfg.Host,
			InsecureSkipVerify: cfg.SkipTLSVerify,
		}); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	auth, err := buildSMTPAuth(cfg)
	if err != nil {
		return err
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp authentication failed: %w", err)
		}
	}

	if err := client.Mail(cfg.FromEmail); err != nil {
		return fmt.Errorf("smtp MAIL FROM failed: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("smtp RCPT TO failed: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA failed: %w", err)
	}

	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("failed to write smtp message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize smtp message: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("failed to close smtp session: %w", err)
	}

	return nil
}

func buildSMTPAuth(cfg smtpRuntimeConfig) (smtp.Auth, error) {
	username := strings.TrimSpace(cfg.Username)
	password := strings.TrimSpace(cfg.Password)

	switch cfg.AuthMethod {
	case "none":
		return nil, nil
	case "auto":
		if username == "" || password == "" {
			return nil, nil
		}
		return smtp.PlainAuth("", username, password, cfg.Host), nil
	case "plain":
		if username == "" || password == "" {
			return nil, errors.New("smtp username and password are required for PLAIN auth")
		}
		return smtp.PlainAuth("", username, password, cfg.Host), nil
	case "login":
		if username == "" || password == "" {
			return nil, errors.New("smtp username and password are required for LOGIN auth")
		}
		return &smtpLoginAuth{username: username, password: password}, nil
	case "cram_md5":
		if username == "" || password == "" {
			return nil, errors.New("smtp username and password are required for CRAM-MD5 auth")
		}
		return smtp.CRAMMD5Auth(username, password), nil
	default:
		return nil, errors.New("unsupported smtp auth method")
	}
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

func (s *AdminService) BackupDB(includeAssets bool) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-backup-%s.db", timestamp))

	query := fmt.Sprintf(`VACUUM INTO '%s'`, backupPath)
	if err := s.DB.Exec(query).Error; err != nil {
		return "", err
	}

	if !includeAssets {
		return backupPath, nil
	}

	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-backup-%s.zip", timestamp))
	if err := createBackupZip(archivePath, backupPath); err != nil {
		_ = os.Remove(backupPath)
		_ = os.Remove(archivePath)
		return "", err
	}

	_ = os.Remove(backupPath)

	return archivePath, nil
}

func createBackupZip(archivePath string, dbPath string) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)

	if err := addFileToBackupZip(zipWriter, dbPath, "subdux.db"); err != nil {
		_ = zipWriter.Close()
		return err
	}

	if err := addAssetsToBackupZip(zipWriter); err != nil {
		_ = zipWriter.Close()
		return err
	}

	return zipWriter.Close()
}

func addFileToBackupZip(zipWriter *zip.Writer, sourcePath string, archivePath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	targetFile, err := zipWriter.Create(archivePath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return err
	}

	return nil
}

func addAssetsToBackupZip(zipWriter *zip.Writer) error {
	assetsRoot := filepath.Join(pkg.GetDataPath(), "assets")
	if err := addDirectoryToBackupZip(zipWriter, "assets/"); err != nil {
		return err
	}

	info, err := os.Stat(assetsRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	return filepath.Walk(assetsRoot, func(path string, fileInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fileInfo.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(assetsRoot, path)
		if err != nil {
			return err
		}

		archivePath := filepath.ToSlash(filepath.Join("assets", relativePath))
		return addFileToBackupZip(zipWriter, path, archivePath)
	})
}

func addDirectoryToBackupZip(zipWriter *zip.Writer, archivePath string) error {
	header := &zip.FileHeader{
		Name: archivePath,
	}
	header.SetMode(os.ModeDir | 0o755)

	_, err := zipWriter.CreateHeader(header)
	return err
}
