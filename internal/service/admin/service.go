package admin

import (
	"context"

	serviceoutbound "github.com/kasuha07/subdux/internal/service/outbound"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
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

type ChangeRoleInput struct {
	Role string `json:"role"`
}

type ChangeStatusInput struct {
	Status string `json:"status"`
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
	IconProxyEnabled                     bool   `json:"icon_proxy_enabled"`
	IconProxyDomainWhitelist             string `json:"icon_proxy_domain_whitelist"`
	MCPEnabled                           bool   `json:"mcp_enabled"`
	AuditEnabled                         bool   `json:"audit_enabled"`
	SystemProxyEnabled                   bool   `json:"system_proxy_enabled"`
	SystemProxyType                      string `json:"system_proxy_type"`
	SystemProxyURLSet                    bool   `json:"system_proxy_url_configured"`
	SSRFProtectionEnabled                bool   `json:"ssrf_protection_enabled"`
	SSRFAllowPrivateIP                   bool   `json:"ssrf_allow_private_ip"`
	SSRFDomainFilterMode                 string `json:"ssrf_domain_filter_mode"`
	SSRFDomainFilterList                 string `json:"ssrf_domain_filter_list"`
	SSRFIPFilterMode                     string `json:"ssrf_ip_filter_mode"`
	SSRFIPFilterList                     string `json:"ssrf_ip_filter_list"`
	SSRFFilterResolvedIPs                bool   `json:"ssrf_filter_resolved_ips"`
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
	SMTPRateLimitSeconds                 int64  `json:"smtp_rate_limit_seconds"`
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
	OIDCReauthACRMFA                     string `json:"oidc_reauth_acr_mfa"`
	OIDCReauthACRPhishingResistant       string `json:"oidc_reauth_acr_phishing_resistant"`
	BackupScheduleEnabled                bool   `json:"backup_schedule_enabled"`
	BackupTimeOfDay                      string `json:"backup_time_of_day"`
	BackupIncludeAssets                  bool   `json:"backup_include_assets"`
	BackupEncryptEnabled                 bool   `json:"backup_encrypt_enabled"`
	BackupEncryptionPasswordSet          bool   `json:"backup_encryption_password_configured"`
	BackupLastRunAt                      string `json:"backup_last_run_at"`
	BackupLastStatus                     string `json:"backup_last_status"`
	BackupLastError                      string `json:"backup_last_error"`
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
	IconProxyEnabled                     *bool   `json:"icon_proxy_enabled"`
	IconProxyDomainWhitelist             *string `json:"icon_proxy_domain_whitelist"`
	MCPEnabled                           *bool   `json:"mcp_enabled"`
	AuditEnabled                         *bool   `json:"audit_enabled"`
	SystemProxyEnabled                   *bool   `json:"system_proxy_enabled"`
	SystemProxyType                      *string `json:"system_proxy_type"`
	SystemProxyURL                       *string `json:"system_proxy_url"`
	SSRFProtectionEnabled                *bool   `json:"ssrf_protection_enabled"`
	SSRFAllowPrivateIP                   *bool   `json:"ssrf_allow_private_ip"`
	SSRFDomainFilterMode                 *string `json:"ssrf_domain_filter_mode"`
	SSRFDomainFilterList                 *string `json:"ssrf_domain_filter_list"`
	SSRFIPFilterMode                     *string `json:"ssrf_ip_filter_mode"`
	SSRFIPFilterList                     *string `json:"ssrf_ip_filter_list"`
	SSRFFilterResolvedIPs                *bool   `json:"ssrf_filter_resolved_ips"`
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
	SMTPRateLimitSeconds                 *int64  `json:"smtp_rate_limit_seconds"`
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
	OIDCReauthACRMFA                     *string `json:"oidc_reauth_acr_mfa"`
	OIDCReauthACRPhishingResistant       *string `json:"oidc_reauth_acr_phishing_resistant"`
	BackupScheduleEnabled                *bool   `json:"backup_schedule_enabled"`
	BackupTimeOfDay                      *string `json:"backup_time_of_day"`
	BackupIncludeAssets                  *bool   `json:"backup_include_assets"`
	BackupEncryptEnabled                 *bool   `json:"backup_encrypt_enabled"`
	BackupEncryptionPassword             *string `json:"backup_encryption_password"`
}

type SSRFTestInput struct {
	Target string `json:"target"`
}

type SSRFTestResult struct {
	Target                  string   `json:"target"`
	Host                    string   `json:"host"`
	Allowed                 bool     `json:"allowed"`
	Reason                  string   `json:"reason"`
	ResolvedIPs             []string `json:"resolved_ips"`
	ProtectionEnabled       bool     `json:"protection_enabled"`
	AllowPrivateIP          bool     `json:"allow_private_ip"`
	DomainFilterMode        string   `json:"domain_filter_mode"`
	IPFilterMode            string   `json:"ip_filter_mode"`
	FilterResolvedIPs       bool     `json:"filter_resolved_ips"`
	ProxyMediated           bool     `json:"proxy_mediated"`
	ResolvedIPFilterApplied bool     `json:"resolved_ip_filter_applied"`
}

type CreateUserInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (s *Service) TestSSRF(input SSRFTestInput) (*SSRFTestResult, error) {
	requestContext := context.Background()
	if s != nil && s.DB != nil && s.DB.Statement != nil && s.DB.Statement.Context != nil {
		requestContext = s.DB.Statement.Context
	}

	result, err := serviceoutbound.TestSSRF(requestContext, s.DB, serviceoutbound.SSRFTestInput{
		Target: input.Target,
	})
	if err != nil {
		return nil, err
	}

	return &SSRFTestResult{
		Target:                  result.Target,
		Host:                    result.Host,
		Allowed:                 result.Allowed,
		Reason:                  result.Reason,
		ResolvedIPs:             result.ResolvedIPs,
		ProtectionEnabled:       result.ProtectionEnabled,
		AllowPrivateIP:          result.AllowPrivateIP,
		DomainFilterMode:        result.DomainFilterMode,
		IPFilterMode:            result.IPFilterMode,
		FilterResolvedIPs:       result.FilterResolvedIPs,
		ProxyMediated:           result.ProxyMediated,
		ResolvedIPFilterApplied: result.ResolvedIPFilterApplied,
	}, nil
}
