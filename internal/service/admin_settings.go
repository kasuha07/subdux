package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func (s *AdminService) GetSettings() (*SystemSettings, error) {
	settings := defaultAdminSystemSettings()

	var items []model.SystemSetting
	s.DB.Find(&items)

	for _, item := range items {
		settingValue := item.Value
		decryptedValue, decryptErr := decryptSystemSettingValueIfNeeded(item.Key, item.Value)
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
			if normalizedType, err := normalizeSystemProxyType(settingValue); err == nil {
				settings.SystemProxyType = normalizedType
			}
		case "system_proxy_url":
			settings.SystemProxyURLSet = strings.TrimSpace(settingValue) != ""
		case ssrfProtectionEnabledKey:
			settings.SSRFProtectionEnabled = settingValue == "true"
		case ssrfAllowPrivateIPKey:
			settings.SSRFAllowPrivateIP = settingValue == "true"
		case ssrfDomainFilterModeKey:
			if mode, err := normalizeSSRFFilterMode(settingValue); err == nil {
				settings.SSRFDomainFilterMode = mode
			}
		case ssrfDomainFilterListKey:
			if normalized, err := normalizeSSRFDomainFilterList(settingValue); err == nil {
				settings.SSRFDomainFilterList = normalized
			}
		case ssrfIPFilterModeKey:
			if mode, err := normalizeSSRFFilterMode(settingValue); err == nil {
				settings.SSRFIPFilterMode = mode
			}
		case ssrfIPFilterListKey:
			if normalized, err := normalizeSSRFIPFilterList(settingValue); err == nil {
				settings.SSRFIPFilterList = normalized
			}
		case ssrfFilterResolvedIPsKey:
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
		case backupScheduleEnabledKey:
			settings.BackupScheduleEnabled = settingValue == "true"
		case backupTimeOfDayKey:
			settings.BackupTimeOfDay = settingValue
		case backupIncludeAssetsKey:
			settings.BackupIncludeAssets = settingValue == "true"
		case backupEncryptEnabledKey:
			settings.BackupEncryptEnabled = settingValue == "true"
		case backupEncryptionPasswordKey:
			settings.BackupEncryptionPasswordSet = strings.TrimSpace(settingValue) != ""
		case backupLocalDirKey:
			settings.BackupLocalDir = settingValue
		case backupRetentionCountKey:
			if v, err := strconv.ParseInt(settingValue, 10, 64); err == nil {
				settings.BackupRetentionCount = v
			}
		case backupLastRunAtKey:
			settings.BackupLastRunAt = settingValue
		case backupLastStatusKey:
			settings.BackupLastStatus = settingValue
		case backupLastErrorKey:
			settings.BackupLastError = settingValue
		}
	}

	return settings, nil
}

func (s *AdminService) UpdateSettings(input UpdateSettingsInput) error {
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
			normalized, err := normalizeEmailDomainWhitelist(*input.EmailDomainWhitelist)
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
			normalized, err := normalizeIconProxyDomainWhitelist(*input.IconProxyDomainWhitelist)
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

		if err := validateIncomingSystemProxySettings(tx, input); err != nil {
			return err
		}

		if input.SystemProxyEnabled != nil {
			if err := saveBoolSystemSetting(tx, "system_proxy_enabled", *input.SystemProxyEnabled); err != nil {
				return err
			}
		}

		if input.SystemProxyType != nil {
			normalizedType, err := normalizeSystemProxyType(*input.SystemProxyType)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, "system_proxy_type", normalizedType); err != nil {
				return err
			}
		}

		if input.SystemProxyURL != nil {
			normalizedType := systemProxyTypeHTTP
			if input.SystemProxyType != nil {
				var err error
				normalizedType, err = normalizeSystemProxyType(*input.SystemProxyType)
				if err != nil {
					return err
				}
			} else if existingCfg, err := loadSystemProxyConfig(tx); err == nil {
				normalizedType = existingCfg.Type
			}

			trimmedURL := strings.TrimSpace(*input.SystemProxyURL)
			value := ""
			if trimmedURL != "" {
				normalizedURL, err := normalizeSystemProxyURL(normalizedType, trimmedURL)
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

		if err := validateSSRFProtectionSettings(input); err != nil {
			return err
		}

		if input.SSRFProtectionEnabled != nil {
			if err := saveBoolSystemSetting(tx, ssrfProtectionEnabledKey, *input.SSRFProtectionEnabled); err != nil {
				return err
			}
		}

		if input.SSRFAllowPrivateIP != nil {
			if err := saveBoolSystemSetting(tx, ssrfAllowPrivateIPKey, *input.SSRFAllowPrivateIP); err != nil {
				return err
			}
		}

		if input.SSRFDomainFilterMode != nil {
			normalizedMode, err := normalizeSSRFFilterMode(*input.SSRFDomainFilterMode)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, ssrfDomainFilterModeKey, normalizedMode); err != nil {
				return err
			}
		}

		if input.SSRFDomainFilterList != nil {
			normalizedList, err := normalizeSSRFDomainFilterList(*input.SSRFDomainFilterList)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, ssrfDomainFilterListKey, normalizedList); err != nil {
				return err
			}
		}

		if input.SSRFIPFilterMode != nil {
			normalizedMode, err := normalizeSSRFFilterMode(*input.SSRFIPFilterMode)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, ssrfIPFilterModeKey, normalizedMode); err != nil {
				return err
			}
		}

		if input.SSRFIPFilterList != nil {
			normalizedList, err := normalizeSSRFIPFilterList(*input.SSRFIPFilterList)
			if err != nil {
				return err
			}
			if err := saveStringSystemSetting(tx, ssrfIPFilterListKey, normalizedList); err != nil {
				return err
			}
		}

		if input.SSRFFilterResolvedIPs != nil {
			if err := saveBoolSystemSetting(tx, ssrfFilterResolvedIPsKey, *input.SSRFFilterResolvedIPs); err != nil {
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
			rateLimitSeconds, err := normalizeSMTPRateLimitSeconds(*input.SMTPRateLimitSeconds)
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

		if err := applyBackupSettings(tx, input); err != nil {
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
			if _, err := loadSMTPRuntimeConfig(tx); err != nil {
				return errors.New("smtp settings must be valid when registration email verification is enabled")
			}
		}

		return nil
	})
}

func isSystemSettingEnabled(tx *gorm.DB, key string, defaultValue bool) (bool, error) {
	return getBoolSystemSettingValue(tx, key, defaultValue)
}

func saveBoolSystemSetting(tx *gorm.DB, key string, enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	return saveStringSystemSetting(tx, key, value)
}

func saveStringSystemSetting(tx *gorm.DB, key string, value string) error {
	// Use a map rather than a struct for Assign so that empty-string values are
	// persisted. GORM omits zero-value struct fields from updates, which would
	// otherwise make it impossible to clear a setting back to an empty value.
	return tx.Where("key = ?", key).
		Assign(map[string]interface{}{"value": value}).
		FirstOrCreate(&model.SystemSetting{Key: key}).Error
}

func saveEncryptedSystemSetting(tx *gorm.DB, key string, value string) error {
	// Encrypted keys are write-only: their values are never returned to clients
	// (only a "<key>Set" flag is). By convention an empty incoming value means
	// "keep the existing secret unchanged" rather than "clear it", because a
	// blank field cannot be distinguished from "leave as-is" when the current
	// value is never shown back. Skip the write so the stored secret survives.
	if strings.TrimSpace(value) == "" {
		return nil
	}
	encrypted, err := encryptSystemSettingValueIfNeeded(key, value)
	if err != nil {
		return err
	}
	return saveStringSystemSetting(tx, key, encrypted)
}

func validateIncomingSystemProxySettings(tx *gorm.DB, input UpdateSettingsInput) error {
	cfg, err := loadSystemProxyConfig(tx)
	if err != nil {
		return err
	}

	proxyType := cfg.Type
	if input.SystemProxyType != nil {
		proxyType, err = normalizeSystemProxyType(*input.SystemProxyType)
		if err != nil {
			return err
		}
	}

	proxyURL := cfg.URL
	if input.SystemProxyURL != nil {
		proxyURL = *input.SystemProxyURL
	}

	enabled := cfg.Enabled
	if input.SystemProxyEnabled != nil {
		enabled = *input.SystemProxyEnabled
	}

	if input.SystemProxyURL == nil && !enabled {
		return nil
	}

	return validateSystemProxySettings(proxyType, proxyURL, enabled)
}
