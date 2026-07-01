package service

import (
	"errors"

	"github.com/shiroha/subdux/internal/model"
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
	siteName, err := getSystemSettingValue(s.DB, "site_name", "Subdux")
	if err != nil {
		return nil, err
	}
	if siteName == "" {
		siteName = "Subdux"
	}

	mcpEnabled, err := getBoolSystemSettingValue(s.DB, "mcp_enabled", false)
	if err != nil {
		return nil, err
	}

	return &SiteInfo{
		SiteName:   siteName,
		MCPEnabled: mcpEnabled,
	}, nil
}

func (s *SystemSettingsService) IsMCPEnabled() (bool, error) {
	return getBoolSystemSettingValue(s.DB, "mcp_enabled", false)
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

func getSystemSettingValue(db *gorm.DB, key string, defaultValue string) (string, error) {
	var setting model.SystemSetting
	if err := db.Where("key = ?", key).First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultValue, nil
		}
		return defaultValue, err
	}
	return setting.Value, nil
}

func getBoolSystemSettingValue(db *gorm.DB, key string, defaultValue bool) (bool, error) {
	value, err := getSystemSettingValue(db, key, boolSystemSettingValue(defaultValue))
	if err != nil {
		return defaultValue, err
	}
	return value == "true", nil
}

func boolSystemSettingValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
