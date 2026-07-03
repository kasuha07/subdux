package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"gorm.io/gorm"
)

type SystemSettingsService struct {
	DB *gorm.DB
}

func NewSystemSettingsService(db *gorm.DB) *SystemSettingsService {
	return &SystemSettingsService{DB: db}
}

type SiteInfo struct {
	SiteName   string `json:"site_name"`
	MCPEnabled bool   `json:"mcp_enabled"`
}

func defaultAdminSystemSettings() *SystemSettings {
	return &SystemSettings{
		RegistrationEnabled:                  false,
		RegistrationEmailVerificationEnabled: false,
		EmailDomainWhitelist:                 "",
		SiteName:                             "Subdux",
		SiteURL:                              "",
		CurrencyAPIKeySet:                    false,
		ExchangeRateSource:                   "auto",
		AllowImageUpload:                     true,
		MaxIconFileSize:                      65536,
		IconProxyEnabled:                     true,
		IconProxyDomainWhitelist:             defaultIconProxyDomainWhitelist,
		MCPEnabled:                           false,
		AuditEnabled:                         true,
		SystemProxyEnabled:                   false,
		SystemProxyType:                      systemProxyTypeHTTP,
		SystemProxyURLSet:                    false,
		SSRFProtectionEnabled:                true,
		SSRFAllowPrivateIP:                   false,
		SSRFDomainFilterMode:                 ssrfFilterModeBlacklist,
		SSRFDomainFilterList:                 "",
		SSRFIPFilterMode:                     ssrfFilterModeBlacklist,
		SSRFIPFilterList:                     "",
		SSRFFilterResolvedIPs:                true,
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
		SMTPRateLimitSeconds:                 0,
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
		OIDCReauthACRMFA:                     "",
		OIDCReauthACRPhishingResistant:       "",
		BackupScheduleEnabled:                false,
		BackupTimeOfDay:                      "03:00",
		BackupIncludeAssets:                  false,
		BackupEncryptEnabled:                 false,
		BackupEncryptionPasswordSet:          false,
		BackupLocalDir:                       "",
		BackupRetentionCount:                 7,
		BackupLastRunAt:                      "",
		BackupLastStatus:                     "",
		BackupLastError:                      "",
	}
}

func (s *SystemSettingsService) SeedDefaults() error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		for _, setting := range defaultSystemSettings {
			if err := tx.Where("key = ?", setting.Key).
				Attrs(model.SystemSetting{Value: setting.Value}).
				FirstOrCreate(&model.SystemSetting{Key: setting.Key}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SystemSettingsService) GetSiteInfo() (*SiteInfo, error) {
	siteName, err := getSystemSettingString(context.Background(), s.DB, "site_name", "Subdux")
	if err != nil {
		return nil, err
	}
	if siteName == "" {
		siteName = "Subdux"
	}

	mcpEnabled, err := getSystemSettingBool(context.Background(), s.DB, "mcp_enabled", false)
	if err != nil {
		return nil, err
	}

	return &SiteInfo{
		SiteName:   siteName,
		MCPEnabled: mcpEnabled,
	}, nil
}

func (s *SystemSettingsService) IsMCPEnabled() (bool, error) {
	return getSystemSettingBool(context.Background(), s.DB, "mcp_enabled", false)
}

var defaultSystemSettings = []model.SystemSetting{
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
	{Key: "icon_proxy_domain_whitelist", Value: defaultIconProxyDomainWhitelist},
	{Key: "mcp_enabled", Value: "false"},
	{Key: "audit_enabled", Value: "true"},
	{Key: "system_proxy_enabled", Value: "false"},
	{Key: "system_proxy_type", Value: systemProxyTypeHTTP},
	{Key: "system_proxy_url", Value: ""},
	{Key: ssrfProtectionEnabledKey, Value: "true"},
	{Key: ssrfAllowPrivateIPKey, Value: "false"},
	{Key: ssrfDomainFilterModeKey, Value: ssrfFilterModeBlacklist},
	{Key: ssrfDomainFilterListKey, Value: ""},
	{Key: ssrfIPFilterModeKey, Value: ssrfFilterModeBlacklist},
	{Key: ssrfIPFilterListKey, Value: ""},
	{Key: ssrfFilterResolvedIPsKey, Value: "true"},
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
	{Key: backupScheduleEnabledKey, Value: "false"},
	{Key: backupTimeOfDayKey, Value: "03:00"},
	{Key: backupIncludeAssetsKey, Value: "false"},
	{Key: backupEncryptEnabledKey, Value: "false"},
	{Key: backupEncryptionPasswordKey, Value: ""},
	{Key: backupLocalDirKey, Value: ""},
	{Key: backupRetentionCountKey, Value: "7"},
	{Key: backupLastRunAtKey, Value: ""},
	{Key: backupLastStatusKey, Value: ""},
	{Key: backupLastErrorKey, Value: ""},
}

func getSystemSettingString(ctx context.Context, db *gorm.DB, key string, defaultValue string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	value, storedValue, found, err := getRawSystemSettingString(ctx, db, key, defaultValue)
	if err != nil || !found {
		return value, err
	}

	value, err = decryptSystemSettingValueIfNeeded(key, storedValue)
	if err != nil {
		return defaultValue, err
	}

	if !pkg.IsSystemSettingEncrypted(storedValue) && value != "" && isEncryptedSystemSettingKey(key) {
		if encryptedValue, encryptErr := encryptSystemSettingValueIfNeeded(key, value); encryptErr == nil {
			_ = db.WithContext(ctx).Model(&model.SystemSetting{}).
				Where("key = ?", key).
				Update("value", encryptedValue).Error
		}
	}

	return value, nil
}

func getRawSystemSettingString(ctx context.Context, db *gorm.DB, key string, defaultValue string) (value string, storedValue string, found bool, err error) {
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

func getSystemSettingBool(ctx context.Context, db *gorm.DB, key string, defaultValue bool) (bool, error) {
	value, err := getSystemSettingString(ctx, db, key, boolSystemSettingValue(defaultValue))
	if err != nil {
		return defaultValue, err
	}
	return value == "true", nil
}

func getSystemSettingInt(ctx context.Context, db *gorm.DB, key string, defaultValue int) (int, error) {
	value, err := getSystemSettingString(ctx, db, key, strconv.Itoa(defaultValue))
	if err != nil {
		return defaultValue, err
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return defaultValue, nil
	}
	return parsed, nil
}

func loadRawSystemSettingStrings(ctx context.Context, db *gorm.DB, defaults map[string]string) (map[string]string, error) {
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

func boolSystemSettingValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
