package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func (s *AdminService) GetSettings() (*SystemSettings, error) {
	settings := &SystemSettings{
		RegistrationEnabled:                  true,
		RegistrationEmailVerificationEnabled: false,
		EmailDomainWhitelist:                 "",
		SiteName:                             "Subdux",
		SiteURL:                              "",
		CurrencyAPIKeySet:                    false,
		ExchangeRateSource:                   "auto",
		AllowImageUpload:                     true,
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

		if input.EmailDomainWhitelist != nil {
			normalized, err := normalizeEmailDomainWhitelist(*input.EmailDomainWhitelist)
			if err != nil {
				return err
			}
			if err := tx.Where("key = ?", "email_domain_whitelist").
				Assign(model.SystemSetting{Value: normalized}).
				FirstOrCreate(&model.SystemSetting{Key: "email_domain_whitelist"}).Error; err != nil {
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

		if input.AllowImageUpload != nil {
			value := "false"
			if *input.AllowImageUpload {
				value = "true"
			}
			if err := tx.Where("key = ?", "allow_image_upload").
				Assign(model.SystemSetting{Value: value}).
				FirstOrCreate(&model.SystemSetting{Key: "allow_image_upload"}).Error; err != nil {
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
			encryptedSMTPPassword, err := encryptSystemSettingValueIfNeeded("smtp_password", *input.SMTPPassword)
			if err != nil {
				return err
			}
			if err := tx.Where("key = ?", "smtp_password").
				Assign(model.SystemSetting{Value: encryptedSMTPPassword}).
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
			encryptedOIDCClientSecret, err := encryptSystemSettingValueIfNeeded("oidc_client_secret", *input.OIDCClientSecret)
			if err != nil {
				return err
			}
			if err := tx.Where("key = ?", "oidc_client_secret").
				Assign(model.SystemSetting{Value: encryptedOIDCClientSecret}).
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
