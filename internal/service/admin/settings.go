package admin

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/kasuha07/subdux/internal/model"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	iconproxy "github.com/kasuha07/subdux/internal/service/iconproxy"
	serviceoutbound "github.com/kasuha07/subdux/internal/service/outbound"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
	servicesmtp "github.com/kasuha07/subdux/internal/service/smtp"
	"gorm.io/gorm"
)

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
		IconProxyDomainWhitelist:             iconproxy.DefaultDomainWhitelist,
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

func (s *Service) GetSettings() (*SystemSettings, error) {
	settings := defaultAdminSystemSettings()

	var items []model.SystemSetting
	s.DB.Find(&items)

	for _, item := range items {
		settingValue := item.Value
		decryptedValue, decryptErr := systemsettings.DecryptValueIfNeeded(item.Key, item.Value)
		if decryptErr == nil {
			settingValue = decryptedValue
		}

		switch item.Key {
		case "registration_enabled":
			settings.RegistrationEnabled = settingValue == "true"
		case "registration_email_verification_enabled":
			settings.RegistrationEmailVerificationEnabled = settingValue == "true"
		case "email_domain_whitelist":
			settings.EmailDomainWhitelist = settingValue
		case "site_name":
			settings.SiteName = settingValue
		case "site_url":
			settings.SiteURL = settingValue
		case "currencyapi_key":
			settings.CurrencyAPIKeySet = strings.TrimSpace(settingValue) != ""
		case "exchange_rate_source":
			settings.ExchangeRateSource = settingValue
		case "allow_image_upload":
			settings.AllowImageUpload = settingValue == "true"
		case "max_icon_file_size":
			if v, err := strconv.ParseInt(settingValue, 10, 64); err == nil {
				settings.MaxIconFileSize = v
			}
		case "icon_proxy_enabled":
			settings.IconProxyEnabled = settingValue == "true"
		case "icon_proxy_domain_whitelist":
			settings.IconProxyDomainWhitelist = settingValue
		case "mcp_enabled":
			settings.MCPEnabled = settingValue == "true"
		case "audit_enabled":
			settings.AuditEnabled = settingValue == "true"
		case "system_proxy_enabled":
			settings.SystemProxyEnabled = settingValue == "true"
		case "system_proxy_type":
			if normalizedType, err := serviceoutbound.NormalizeSystemProxyType(settingValue); err == nil {
				settings.SystemProxyType = normalizedType
			}
		case "system_proxy_url":
			settings.SystemProxyURLSet = strings.TrimSpace(settingValue) != ""
		case serviceoutbound.ProtectionEnabledKey:
			settings.SSRFProtectionEnabled = settingValue == "true"
		case serviceoutbound.AllowPrivateIPKey:
			settings.SSRFAllowPrivateIP = settingValue == "true"
		case serviceoutbound.DomainFilterModeKey:
			if mode, err := serviceoutbound.NormalizeFilterMode(settingValue); err == nil {
				settings.SSRFDomainFilterMode = mode
			}
		case serviceoutbound.DomainFilterListKey:
			if normalized, err := serviceoutbound.NormalizeDomainFilterList(settingValue); err == nil {
				settings.SSRFDomainFilterList = normalized
			}
		case serviceoutbound.IPFilterModeKey:
			if mode, err := serviceoutbound.NormalizeFilterMode(settingValue); err == nil {
				settings.SSRFIPFilterMode = mode
			}
		case serviceoutbound.IPFilterListKey:
			if normalized, err := serviceoutbound.NormalizeIPFilterList(settingValue); err == nil {
				settings.SSRFIPFilterList = normalized
			}
		case serviceoutbound.FilterResolvedIPsKey:
			settings.SSRFFilterResolvedIPs = settingValue == "true"
		case "smtp_enabled":
			settings.SMTPEnabled = settingValue == "true"
		case "smtp_host":
			settings.SMTPHost = settingValue
		case "smtp_port":
			if v, err := strconv.ParseInt(settingValue, 10, 64); err == nil {
				settings.SMTPPort = v
			}
		case "smtp_username":
			settings.SMTPUsername = settingValue
		case "smtp_password":
			settings.SMTPPasswordSet = strings.TrimSpace(settingValue) != ""
		case "smtp_from_email":
			settings.SMTPFromEmail = settingValue
		case "smtp_from_name":
			settings.SMTPFromName = settingValue
		case "smtp_encryption":
			settings.SMTPEncryption = settingValue
		case "smtp_auth_method":
			settings.SMTPAuthMethod = settingValue
		case "smtp_helo_name":
			settings.SMTPHeloName = settingValue
		case "smtp_timeout_seconds":
			if v, err := strconv.ParseInt(settingValue, 10, 64); err == nil {
				settings.SMTPTimeoutSeconds = v
			}
		case "smtp_rate_limit_seconds":
			if v, err := strconv.ParseInt(settingValue, 10, 64); err == nil && v >= 0 {
				settings.SMTPRateLimitSeconds = v
			}
		case "smtp_skip_tls_verify":
			settings.SMTPSkipTLSVerify = settingValue == "true"
		case "oidc_enabled":
			settings.OIDCEnabled = settingValue == "true"
		case "oidc_provider_name":
			settings.OIDCProviderName = settingValue
		case "oidc_issuer_url":
			settings.OIDCIssuerURL = settingValue
		case "oidc_client_id":
			settings.OIDCClientID = settingValue
		case "oidc_client_secret":
			settings.OIDCClientSecretSet = strings.TrimSpace(settingValue) != ""
		case "oidc_redirect_url":
			settings.OIDCRedirectURL = settingValue
		case "oidc_scopes":
			settings.OIDCScopes = settingValue
		case "oidc_auto_create_user":
			settings.OIDCAutoCreateUser = settingValue == "true"
		case "oidc_authorization_endpoint":
			settings.OIDCAuthorizeURL = settingValue
		case "oidc_token_endpoint":
			settings.OIDCTokenURL = settingValue
		case "oidc_userinfo_endpoint":
			settings.OIDCUserinfoURL = settingValue
		case "oidc_audience":
			settings.OIDCAudience = settingValue
		case "oidc_resource":
			settings.OIDCResource = settingValue
		case "oidc_extra_auth_params":
			settings.OIDCExtraAuthParams = settingValue
		case "oidc_reauth_acr_mfa":
			settings.OIDCReauthACRMFA = settingValue
		case "oidc_reauth_acr_phishing_resistant":
			settings.OIDCReauthACRPhishingResistant = settingValue
		case servicebackup.KeyScheduleEnabled:
			settings.BackupScheduleEnabled = settingValue == "true"
		case servicebackup.KeyTimeOfDay:
			settings.BackupTimeOfDay = settingValue
		case servicebackup.KeyIncludeAssets:
			settings.BackupIncludeAssets = settingValue == "true"
		case servicebackup.KeyEncryptEnabled:
			settings.BackupEncryptEnabled = settingValue == "true"
		case servicebackup.KeyEncryptionPassword:
			settings.BackupEncryptionPasswordSet = strings.TrimSpace(settingValue) != ""
		case servicebackup.KeyLocalDir:
			settings.BackupLocalDir = settingValue
		case servicebackup.KeyRetentionCount:
			if v, err := strconv.ParseInt(settingValue, 10, 64); err == nil {
				settings.BackupRetentionCount = v
			}
		case servicebackup.KeyLastRunAt:
			settings.BackupLastRunAt = settingValue
		case servicebackup.KeyLastStatus:
			settings.BackupLastStatus = settingValue
		case servicebackup.KeyLastError:
			settings.BackupLastError = settingValue
		}
	}

	return settings, nil
}

func (s *Service) UpdateSettings(input UpdateSettingsInput) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if input.RegistrationEnabled != nil {
			if err := saveBoolSystemSetting(tx, "registration_enabled", *input.RegistrationEnabled); err != nil {
				return err
			}
		}

		if input.RegistrationEmailVerificationEnabled != nil {
			if err := saveBoolSystemSetting(tx, "registration_email_verification_enabled", *input.RegistrationEmailVerificationEnabled); err != nil {
				return err
			}
		}

		if input.EmailDomainWhitelist != nil {
			normalized, err := serviceauth.NormalizeEmailDomainWhitelist(*input.EmailDomainWhitelist)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, "email_domain_whitelist", normalized); err != nil {
				return err
			}
		}

		if input.SiteName != nil {
			if err := saveStringSystemSetting(tx, "site_name", *input.SiteName); err != nil {
				return err
			}
		}

		if input.SiteURL != nil {
			if err := saveStringSystemSetting(tx, "site_url", *input.SiteURL); err != nil {
				return err
			}
		}

		if input.CurrencyAPIKey != nil {
			if err := saveEncryptedSystemSetting(tx, "currencyapi_key", *input.CurrencyAPIKey); err != nil {
				return err
			}
		}

		if input.ExchangeRateSource != nil {
			if err := saveStringSystemSetting(tx, "exchange_rate_source", *input.ExchangeRateSource); err != nil {
				return err
			}
		}

		if input.AllowImageUpload != nil {
			if err := saveBoolSystemSetting(tx, "allow_image_upload", *input.AllowImageUpload); err != nil {
				return err
			}
		}

		if input.MaxIconFileSize != nil {
			if err := saveStringSystemSetting(tx, "max_icon_file_size", strconv.FormatInt(*input.MaxIconFileSize, 10)); err != nil {
				return err
			}
		}

		if input.IconProxyEnabled != nil {
			if err := saveBoolSystemSetting(tx, "icon_proxy_enabled", *input.IconProxyEnabled); err != nil {
				return err
			}
		}

		if input.IconProxyDomainWhitelist != nil {
			normalized, err := iconproxy.NormalizeDomainWhitelist(*input.IconProxyDomainWhitelist)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, "icon_proxy_domain_whitelist", normalized); err != nil {
				return err
			}
		}

		if input.MCPEnabled != nil {
			if err := saveBoolSystemSetting(tx, "mcp_enabled", *input.MCPEnabled); err != nil {
				return err
			}
		}

		if input.AuditEnabled != nil {
			if err := saveBoolSystemSetting(tx, "audit_enabled", *input.AuditEnabled); err != nil {
				return err
			}
		}

		if err := serviceoutbound.ValidateIncomingSystemProxySettings(tx, input.SystemProxyEnabled, input.SystemProxyType, input.SystemProxyURL); err != nil {
			return err
		}

		if input.SystemProxyEnabled != nil {
			if err := saveBoolSystemSetting(tx, "system_proxy_enabled", *input.SystemProxyEnabled); err != nil {
				return err
			}
		}

		if input.SystemProxyType != nil {
			normalizedType, err := serviceoutbound.NormalizeSystemProxyType(*input.SystemProxyType)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, "system_proxy_type", normalizedType); err != nil {
				return err
			}
		}

		if input.SystemProxyURL != nil {
			normalizedType := serviceoutbound.SystemProxyTypeHTTP
			if input.SystemProxyType != nil {
				var err error
				normalizedType, err = serviceoutbound.NormalizeSystemProxyType(*input.SystemProxyType)
				if err != nil {
					return err
				}
			} else if existingCfg, err := serviceoutbound.LoadSystemProxyConfig(tx); err == nil {
				normalizedType = existingCfg.Type
			}

			trimmedURL := strings.TrimSpace(*input.SystemProxyURL)
			value := ""
			if trimmedURL != "" {
				normalizedURL, err := serviceoutbound.NormalizeSystemProxyURL(normalizedType, trimmedURL)
				if err != nil {
					return err
				}
				value = normalizedURL.String()
			}
			// Write-only like the other secrets: an empty value keeps the
			// existing proxy URL rather than clearing it.
			if err := saveEncryptedSystemSetting(tx, "system_proxy_url", value); err != nil {
				return err
			}
		}

		if err := serviceoutbound.ValidatePolicyUpdate(
			input.SSRFDomainFilterMode,
			input.SSRFIPFilterMode,
			input.SSRFDomainFilterList,
			input.SSRFIPFilterList,
		); err != nil {
			return err
		}

		if input.SSRFProtectionEnabled != nil {
			if err := saveBoolSystemSetting(tx, serviceoutbound.ProtectionEnabledKey, *input.SSRFProtectionEnabled); err != nil {
				return err
			}
		}

		if input.SSRFAllowPrivateIP != nil {
			if err := saveBoolSystemSetting(tx, serviceoutbound.AllowPrivateIPKey, *input.SSRFAllowPrivateIP); err != nil {
				return err
			}
		}

		if input.SSRFDomainFilterMode != nil {
			normalizedMode, err := serviceoutbound.NormalizeFilterMode(*input.SSRFDomainFilterMode)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, serviceoutbound.DomainFilterModeKey, normalizedMode); err != nil {
				return err
			}
		}

		if input.SSRFDomainFilterList != nil {
			normalizedList, err := serviceoutbound.NormalizeDomainFilterList(*input.SSRFDomainFilterList)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, serviceoutbound.DomainFilterListKey, normalizedList); err != nil {
				return err
			}
		}

		if input.SSRFIPFilterMode != nil {
			normalizedMode, err := serviceoutbound.NormalizeFilterMode(*input.SSRFIPFilterMode)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, serviceoutbound.IPFilterModeKey, normalizedMode); err != nil {
				return err
			}
		}

		if input.SSRFIPFilterList != nil {
			normalizedList, err := serviceoutbound.NormalizeIPFilterList(*input.SSRFIPFilterList)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, serviceoutbound.IPFilterListKey, normalizedList); err != nil {
				return err
			}
		}

		if input.SSRFFilterResolvedIPs != nil {
			if err := saveBoolSystemSetting(tx, serviceoutbound.FilterResolvedIPsKey, *input.SSRFFilterResolvedIPs); err != nil {
				return err
			}
		}

		if input.SMTPEnabled != nil {
			if err := saveBoolSystemSetting(tx, "smtp_enabled", *input.SMTPEnabled); err != nil {
				return err
			}
		}

		if input.SMTPHost != nil {
			if err := saveStringSystemSetting(tx, "smtp_host", *input.SMTPHost); err != nil {
				return err
			}
		}

		if input.SMTPPort != nil {
			if err := saveStringSystemSetting(tx, "smtp_port", strconv.FormatInt(*input.SMTPPort, 10)); err != nil {
				return err
			}
		}

		if input.SMTPUsername != nil {
			if err := saveStringSystemSetting(tx, "smtp_username", *input.SMTPUsername); err != nil {
				return err
			}
		}

		if input.SMTPPassword != nil {
			if err := saveEncryptedSystemSetting(tx, "smtp_password", *input.SMTPPassword); err != nil {
				return err
			}
		}

		if input.SMTPFromEmail != nil {
			if err := saveStringSystemSetting(tx, "smtp_from_email", *input.SMTPFromEmail); err != nil {
				return err
			}
		}

		if input.SMTPFromName != nil {
			if err := saveStringSystemSetting(tx, "smtp_from_name", *input.SMTPFromName); err != nil {
				return err
			}
		}

		if input.SMTPEncryption != nil {
			if err := saveStringSystemSetting(tx, "smtp_encryption", *input.SMTPEncryption); err != nil {
				return err
			}
		}

		if input.SMTPAuthMethod != nil {
			if err := saveStringSystemSetting(tx, "smtp_auth_method", *input.SMTPAuthMethod); err != nil {
				return err
			}
		}

		if input.SMTPHeloName != nil {
			if err := saveStringSystemSetting(tx, "smtp_helo_name", *input.SMTPHeloName); err != nil {
				return err
			}
		}

		if input.SMTPTimeoutSeconds != nil {
			if err := saveStringSystemSetting(tx, "smtp_timeout_seconds", strconv.FormatInt(*input.SMTPTimeoutSeconds, 10)); err != nil {
				return err
			}
		}

		if input.SMTPRateLimitSeconds != nil {
			rateLimitSeconds, err := servicesmtp.NormalizeRateLimitSeconds(*input.SMTPRateLimitSeconds)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, "smtp_rate_limit_seconds", strconv.FormatInt(rateLimitSeconds, 10)); err != nil {
				return err
			}
		}

		if input.SMTPSkipTLSVerify != nil {
			if err := saveBoolSystemSetting(tx, "smtp_skip_tls_verify", *input.SMTPSkipTLSVerify); err != nil {
				return err
			}
		}

		if input.OIDCEnabled != nil {
			if err := saveBoolSystemSetting(tx, "oidc_enabled", *input.OIDCEnabled); err != nil {
				return err
			}
		}

		if input.OIDCProviderName != nil {
			if err := saveStringSystemSetting(tx, "oidc_provider_name", *input.OIDCProviderName); err != nil {
				return err
			}
		}

		if input.OIDCIssuerURL != nil {
			if err := saveStringSystemSetting(tx, "oidc_issuer_url", *input.OIDCIssuerURL); err != nil {
				return err
			}
		}

		if input.OIDCClientID != nil {
			if err := saveStringSystemSetting(tx, "oidc_client_id", *input.OIDCClientID); err != nil {
				return err
			}
		}

		if input.OIDCClientSecret != nil {
			if err := saveEncryptedSystemSetting(tx, "oidc_client_secret", *input.OIDCClientSecret); err != nil {
				return err
			}
		}

		if input.OIDCRedirectURL != nil {
			if err := saveStringSystemSetting(tx, "oidc_redirect_url", *input.OIDCRedirectURL); err != nil {
				return err
			}
		}

		if input.OIDCScopes != nil {
			if err := saveStringSystemSetting(tx, "oidc_scopes", *input.OIDCScopes); err != nil {
				return err
			}
		}

		if input.OIDCAutoCreateUser != nil {
			if err := saveBoolSystemSetting(tx, "oidc_auto_create_user", *input.OIDCAutoCreateUser); err != nil {
				return err
			}
		}

		if input.OIDCAuthorizeURL != nil {
			if err := saveStringSystemSetting(tx, "oidc_authorization_endpoint", *input.OIDCAuthorizeURL); err != nil {
				return err
			}
		}

		if input.OIDCTokenURL != nil {
			if err := saveStringSystemSetting(tx, "oidc_token_endpoint", *input.OIDCTokenURL); err != nil {
				return err
			}
		}

		if input.OIDCUserinfoURL != nil {
			if err := saveStringSystemSetting(tx, "oidc_userinfo_endpoint", *input.OIDCUserinfoURL); err != nil {
				return err
			}
		}

		if input.OIDCAudience != nil {
			if err := saveStringSystemSetting(tx, "oidc_audience", *input.OIDCAudience); err != nil {
				return err
			}
		}

		if input.OIDCResource != nil {
			if err := saveStringSystemSetting(tx, "oidc_resource", *input.OIDCResource); err != nil {
				return err
			}
		}

		if input.OIDCExtraAuthParams != nil {
			if err := saveStringSystemSetting(tx, "oidc_extra_auth_params", *input.OIDCExtraAuthParams); err != nil {
				return err
			}
		}

		if input.OIDCReauthACRMFA != nil {
			if err := saveStringSystemSetting(tx, "oidc_reauth_acr_mfa", *input.OIDCReauthACRMFA); err != nil {
				return err
			}
		}

		if input.OIDCReauthACRPhishingResistant != nil {
			if err := saveStringSystemSetting(tx, "oidc_reauth_acr_phishing_resistant", *input.OIDCReauthACRPhishingResistant); err != nil {
				return err
			}
		}

		if err := servicebackup.ApplySettings(tx, servicebackup.UpdateSettingsInput{
			ScheduleEnabled:    input.BackupScheduleEnabled,
			TimeOfDay:          input.BackupTimeOfDay,
			IncludeAssets:      input.BackupIncludeAssets,
			EncryptEnabled:     input.BackupEncryptEnabled,
			EncryptionPassword: input.BackupEncryptionPassword,
			LocalDir:           input.BackupLocalDir,
			RetentionCount:     input.BackupRetentionCount,
		}); err != nil {
			return err
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
			if _, err := servicesmtp.LoadRuntimeConfig(tx); err != nil {
				return errors.New("smtp settings must be valid when registration email verification is enabled")
			}
		}

		return nil
	})
}

func isSystemSettingEnabled(tx *gorm.DB, key string, defaultValue bool) (bool, error) {
	return systemsettings.GetBool(context.Background(), tx, key, defaultValue)
}

func saveBoolSystemSetting(tx *gorm.DB, key string, enabled bool) error {
	return systemsettings.SaveBool(tx, key, enabled)
}

func saveStringSystemSetting(tx *gorm.DB, key string, value string) error {
	return systemsettings.SaveString(tx, key, value)
}

func saveEncryptedSystemSetting(tx *gorm.DB, key string, value string) error {
	return systemsettings.SaveEncrypted(tx, key, value)
}
