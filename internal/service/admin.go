package service

import "gorm.io/gorm"

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
	EmailDomainWhitelist                 string `json:"email_domain_whitelist"`
	SiteName                             string `json:"site_name"`
	SiteURL                              string `json:"site_url"`
	CurrencyAPIKeySet                    bool   `json:"currencyapi_key_configured"`
	ExchangeRateSource                   string `json:"exchange_rate_source"`
	AllowImageUpload                     bool   `json:"allow_image_upload"`
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
	EmailDomainWhitelist                 *string `json:"email_domain_whitelist"`
	SiteName                             *string `json:"site_name"`
	SiteURL                              *string `json:"site_url"`
	CurrencyAPIKey                       *string `json:"currencyapi_key"`
	ExchangeRateSource                   *string `json:"exchange_rate_source"`
	AllowImageUpload                     *bool   `json:"allow_image_upload"`
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
