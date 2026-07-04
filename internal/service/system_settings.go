package service

import (
	"context"

	serviceoutbound "github.com/kasuha07/subdux/internal/service/outbound"
	"github.com/kasuha07/subdux/internal/service/settings"
	"gorm.io/gorm"
)

type SiteInfo = settings.SiteInfo

type SystemSettingsService struct {
	DB *gorm.DB
}

func NewSystemSettingsService(db *gorm.DB) *SystemSettingsService {
	return &SystemSettingsService{DB: db}
}

func (s *SystemSettingsService) SeedDefaults() error {
	return settings.NewService(s.DB).SeedDefaults()
}

func (s *SystemSettingsService) GetSiteInfo() (*SiteInfo, error) {
	return settings.NewService(s.DB).GetSiteInfo()
}

func (s *SystemSettingsService) IsMCPEnabled() (bool, error) {
	return settings.NewService(s.DB).IsMCPEnabled()
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
		SystemProxyType:                      serviceoutbound.SystemProxyTypeHTTP,
		SystemProxyURLSet:                    false,
		SSRFProtectionEnabled:                true,
		SSRFAllowPrivateIP:                   false,
		SSRFDomainFilterMode:                 serviceoutbound.FilterModeBlacklist,
		SSRFDomainFilterList:                 "",
		SSRFIPFilterMode:                     serviceoutbound.FilterModeBlacklist,
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

func getSystemSettingString(ctx context.Context, db *gorm.DB, key string, defaultValue string) (string, error) {
	return settings.GetString(ctx, db, key, defaultValue)
}

func getRawSystemSettingString(ctx context.Context, db *gorm.DB, key string, defaultValue string) (value string, storedValue string, found bool, err error) {
	return settings.GetRawString(ctx, db, key, defaultValue)
}

func getSystemSettingBool(ctx context.Context, db *gorm.DB, key string, defaultValue bool) (bool, error) {
	return settings.GetBool(ctx, db, key, defaultValue)
}

func getSystemSettingInt(ctx context.Context, db *gorm.DB, key string, defaultValue int) (int, error) {
	return settings.GetInt(ctx, db, key, defaultValue)
}

func loadRawSystemSettingStrings(ctx context.Context, db *gorm.DB, defaults map[string]string) (map[string]string, error) {
	return settings.LoadRawStrings(ctx, db, defaults)
}

func boolSystemSettingValue(value bool) string {
	return settings.BoolValue(value)
}
