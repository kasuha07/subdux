package settings

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

type SiteInfo struct {
	SiteName   string `json:"site_name"`
	MCPEnabled bool   `json:"mcp_enabled"`
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	clone := *s
	if s.DB != nil {
		clone.DB = s.DB.WithContext(ctx)
	}
	return &clone
}

const DefaultIconProxyDomainWhitelist = "google.com\ngstatic.com\nicon.horse"

var DefaultSystemSettings = []model.SystemSetting{
	{Key: "registration_enabled", Value: "false"},
	{Key: "registration_email_verification_enabled", Value: "false"},
	{Key: "email_domain_whitelist", Value: ""},
	{Key: "site_name", Value: "Subdux"},
	{Key: "site_url", Value: ""},
	{Key: "currencyapi_key", Value: ""},
	{Key: "exchange_rate_source", Value: "auto"},
	{Key: "allow_image_upload", Value: "true"},
	{Key: "max_icon_file_size", Value: "65536"},
	{Key: "icon_proxy_enabled", Value: "true"},
	{Key: "icon_proxy_domain_whitelist", Value: DefaultIconProxyDomainWhitelist},
	{Key: "mcp_enabled", Value: "false"},
	{Key: "audit_enabled", Value: "true"},
	{Key: "system_proxy_enabled", Value: "false"},
	{Key: "system_proxy_type", Value: "http"},
	{Key: "system_proxy_url", Value: ""},
	{Key: "ssrf_protection_enabled", Value: "true"},
	{Key: "ssrf_allow_private_ip", Value: "false"},
	{Key: "ssrf_domain_filter_mode", Value: "blacklist"},
	{Key: "ssrf_domain_filter_list", Value: ""},
	{Key: "ssrf_ip_filter_mode", Value: "blacklist"},
	{Key: "ssrf_ip_filter_list", Value: ""},
	{Key: "ssrf_filter_resolved_ips", Value: "true"},
	{Key: "smtp_enabled", Value: "false"},
	{Key: "smtp_host", Value: ""},
	{Key: "smtp_port", Value: "587"},
	{Key: "smtp_username", Value: ""},
	{Key: "smtp_password", Value: ""},
	{Key: "smtp_from_email", Value: ""},
	{Key: "smtp_from_name", Value: ""},
	{Key: "smtp_encryption", Value: "starttls"},
	{Key: "smtp_auth_method", Value: "auto"},
	{Key: "smtp_helo_name", Value: ""},
	{Key: "smtp_timeout_seconds", Value: "10"},
	{Key: "smtp_rate_limit_seconds", Value: "0"},
	{Key: "smtp_skip_tls_verify", Value: "false"},
	{Key: "oidc_enabled", Value: "false"},
	{Key: "oidc_provider_name", Value: "OIDC"},
	{Key: "oidc_issuer_url", Value: ""},
	{Key: "oidc_client_id", Value: ""},
	{Key: "oidc_client_secret", Value: ""},
	{Key: "oidc_redirect_url", Value: ""},
	{Key: "oidc_scopes", Value: "openid profile email"},
	{Key: "oidc_auto_create_user", Value: "false"},
	{Key: "oidc_authorization_endpoint", Value: ""},
	{Key: "oidc_token_endpoint", Value: ""},
	{Key: "oidc_userinfo_endpoint", Value: ""},
	{Key: "oidc_audience", Value: ""},
	{Key: "oidc_resource", Value: ""},
	{Key: "oidc_extra_auth_params", Value: ""},
	{Key: "backup_schedule_enabled", Value: "false"},
	{Key: "backup_time_of_day", Value: "03:00"},
	{Key: "backup_include_assets", Value: "false"},
	{Key: "backup_encrypt_enabled", Value: "false"},
	{Key: "backup_encryption_password", Value: ""},
	{Key: "backup_last_run_at", Value: ""},
	{Key: "backup_last_status", Value: ""},
	{Key: "backup_last_error", Value: ""},
}

func (s *Service) SeedDefaults() error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		for _, setting := range DefaultSystemSettings {
			if err := tx.Where("key = ?", setting.Key).
				Attrs(model.SystemSetting{Value: setting.Value}).
				FirstOrCreate(&model.SystemSetting{Key: setting.Key}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) GetSiteInfo() (*SiteInfo, error) {
	siteName, err := GetString(context.Background(), s.DB, "site_name", "Subdux")
	if err != nil {
		return nil, err
	}
	if siteName == "" {
		siteName = "Subdux"
	}

	mcpEnabled, err := GetBool(context.Background(), s.DB, "mcp_enabled", false)
	if err != nil {
		return nil, err
	}

	return &SiteInfo{
		SiteName:   siteName,
		MCPEnabled: mcpEnabled,
	}, nil
}

func (s *Service) IsMCPEnabled() (bool, error) {
	return GetBool(context.Background(), s.DB, "mcp_enabled", false)
}

func GetString(ctx context.Context, db *gorm.DB, key string, defaultValue string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	value, storedValue, found, err := GetRawString(ctx, db, key, defaultValue)
	if err != nil || !found {
		return value, err
	}

	value, err = DecryptValueIfNeeded(key, storedValue)
	if err != nil {
		return defaultValue, err
	}

	if !pkg.IsSystemSettingEncrypted(storedValue) && value != "" && IsEncryptedKey(key) {
		if encryptedValue, encryptErr := EncryptValueIfNeeded(key, value); encryptErr == nil {
			_ = db.WithContext(ctx).Model(&model.SystemSetting{}).
				Where("key = ?", key).
				Update("value", encryptedValue).Error
		}
	}

	return value, nil
}

func GetRawString(ctx context.Context, db *gorm.DB, key string, defaultValue string) (value string, storedValue string, found bool, err error) {
	if db == nil {
		return defaultValue, "", false, errors.New("system settings database is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var setting model.SystemSetting
	if err := db.WithContext(ctx).Where("key = ?", key).First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultValue, "", false, nil
		}
		return defaultValue, "", false, err
	}

	return setting.Value, setting.Value, true, nil
}

func GetBool(ctx context.Context, db *gorm.DB, key string, defaultValue bool) (bool, error) {
	value, err := GetString(ctx, db, key, BoolValue(defaultValue))
	if err != nil {
		return defaultValue, err
	}
	return value == "true", nil
}

func GetInt(ctx context.Context, db *gorm.DB, key string, defaultValue int) (int, error) {
	value, err := GetString(ctx, db, key, strconv.Itoa(defaultValue))
	if err != nil {
		return defaultValue, err
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return defaultValue, nil
	}
	return parsed, nil
}

func LoadRawStrings(ctx context.Context, db *gorm.DB, defaults map[string]string) (map[string]string, error) {
	if db == nil {
		return nil, errors.New("system settings database is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	keys := make([]string, 0, len(defaults))
	values := make(map[string]string, len(defaults))
	for key, defaultValue := range defaults {
		keys = append(keys, key)
		values[key] = defaultValue
	}

	var items []model.SystemSetting
	if err := db.WithContext(ctx).Where("key IN ?", keys).Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		values[item.Key] = item.Value
	}

	return values, nil
}

func SaveBool(tx *gorm.DB, key string, enabled bool) error {
	return SaveString(tx, key, BoolValue(enabled))
}

func SaveString(tx *gorm.DB, key string, value string) error {
	return tx.Where("key = ?", key).
		Assign(map[string]interface{}{"value": value}).
		FirstOrCreate(&model.SystemSetting{Key: key}).Error
}

func SaveEncrypted(tx *gorm.DB, key string, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	encrypted, err := EncryptValueIfNeeded(key, value)
	if err != nil {
		return err
	}
	return SaveString(tx, key, encrypted)
}

func BoolValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
