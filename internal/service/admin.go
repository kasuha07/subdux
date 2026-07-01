package service

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

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
}

var ErrInvalidSSRFTestTarget = errors.New("ssrf test target must be a valid hostname or ip address")

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

func (s *AdminService) TestSSRF(input SSRFTestInput) (*SSRFTestResult, error) {
	requestContext := context.Background()
	if s != nil && s.DB != nil && s.DB.Statement != nil && s.DB.Statement.Context != nil {
		requestContext = s.DB.Statement.Context
	}

	host, err := normalizeSSRFTestTarget(input.Target)
	if err != nil {
		return nil, err
	}

	cfg := ssrfProtectionConfigForDB(s.DB)
	proxyMediated := ssrfTestUsesOutboundProxy(s.DB)
	result := &SSRFTestResult{
		Target:                  strings.TrimSpace(input.Target),
		Host:                    host,
		ProtectionEnabled:       cfg.Enabled,
		AllowPrivateIP:          cfg.AllowPrivateIP,
		DomainFilterMode:        cfg.DomainFilterMode,
		IPFilterMode:            cfg.IPFilterMode,
		FilterResolvedIPs:       cfg.FilterResolvedIP,
		ProxyMediated:           proxyMediated,
		ResolvedIPFilterApplied: cfg.Enabled && cfg.FilterResolvedIP && !proxyMediated,
		ResolvedIPs:             []string{},
	}

	if proxyMediated {
		if err := validateOutboundHost(host, "ssrf test target", s.DB); err != nil {
			result.Allowed = false
			result.Reason = err.Error()
			return result, nil
		}
		result.Allowed = true
		result.Reason = ssrfTestAllowedReason(cfg)
		return result, nil
	}

	ctx, cancel := context.WithTimeout(requestContext, 2*time.Second)
	defer cancel()
	ips, err := resolveSafeOutboundHostIPs(ctx, "ip", host, "ssrf test target", s.DB)
	if err != nil {
		result.Allowed = false
		result.Reason = err.Error()
		return result, nil
	}
	result.ResolvedIPs = stringifyIPs(ips)
	result.Allowed = true
	result.Reason = ssrfTestAllowedReason(cfg)
	return result, nil
}

func normalizeSSRFTestTarget(raw string) (string, error) {
	target := strings.TrimSpace(raw)
	if target == "" || len(target) > 253 {
		return "", ErrInvalidSSRFTestTarget
	}

	host := target
	if strings.Contains(target, "://") {
		parsed, err := url.Parse(target)
		if err != nil || parsed.Hostname() == "" {
			return "", ErrInvalidSSRFTestTarget
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			return "", ErrInvalidSSRFTestTarget
		}
		host = parsed.Hostname()
	} else if splitHost, _, err := net.SplitHostPort(target); err == nil {
		host = splitHost
	} else if strings.HasPrefix(target, "[") && strings.HasSuffix(target, "]") {
		host = strings.TrimPrefix(strings.TrimSuffix(target, "]"), "[")
	}

	host, err := normalizeOutboundHostname(host)
	if err != nil {
		return "", ErrInvalidSSRFTestTarget
	}
	if strings.ContainsAny(host, "/\\@?# \t\r\n") {
		return "", ErrInvalidSSRFTestTarget
	}
	if ip := net.ParseIP(host); ip != nil {
		return host, nil
	}
	if !isValidHostnamePattern(host) {
		return "", ErrInvalidSSRFTestTarget
	}
	return host, nil
}

func ssrfTestUsesOutboundProxy(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	cfg, err := loadSystemProxyConfig(db)
	if err != nil || !cfg.Enabled {
		return false
	}
	_, err = normalizeSystemProxyURL(cfg.Type, cfg.URL)
	return err == nil
}

func ssrfTestAllowedReason(cfg ssrfProtectionConfig) string {
	if !cfg.Enabled {
		return "ssrf protection is disabled"
	}
	return "target is allowed by ssrf settings"
}

func stringifyIPs(ips []net.IP) []string {
	values := make([]string, 0, len(ips))
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		values = append(values, ip.String())
	}
	return values
}
