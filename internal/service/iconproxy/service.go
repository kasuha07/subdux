package iconproxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	serviceoutbound "github.com/kasuha07/subdux/internal/service/outbound"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

const (
	iconProxyProviderGoogle    = "google"
	iconProxyProviderIconHorse = "icon-horse"
)

var (
	ErrInvalidIconProxyProvider     = serviceerr.New(serviceerr.KindInvalid, "invalid_icon_proxy_provider", "invalid icon proxy provider")
	ErrInvalidIconProxyTargetDomain = serviceerr.New(serviceerr.KindInvalid, "invalid_icon_proxy_target_domain", "invalid icon proxy target domain")
	ErrIconProxyDomainNotAllowed    = serviceerr.New(serviceerr.KindForbidden, "icon_proxy_domain_is_not_allowed", "icon proxy domain is not allowed")
)

type Service struct {
	DB         *gorm.DB
	httpClient *http.Client
}

type Resolution struct {
	Proxy        bool
	Provider     string
	TargetDomain string
	UpstreamHost string
	UpstreamURL  string
	AllowedHosts string
}

func NewService(db *gorm.DB) *Service {
	client, err := serviceoutbound.BuildHTTPClientWithTimeout(context.Background(), db, serviceoutbound.PurposeIconProxy, 10*time.Second)
	if err != nil {
		client = serviceoutbound.NewOutboundHTTPClient(db, 10*time.Second)
	}
	return &Service{
		DB:         db,
		httpClient: client,
	}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	clone := *s
	if s.DB != nil {
		clone.DB = s.DB.WithContext(ctx)
	}
	return &clone
}

func (s *Service) Resolve(provider string, rawDomain string) (*Resolution, error) {
	spec, err := getIconProxyProviderSpec(provider)
	if err != nil {
		return nil, err
	}

	targetDomain, err := normalizeIconProxyTargetDomain(rawDomain)
	if err != nil {
		return nil, err
	}

	enabled := s.getBoolSetting("icon_proxy_enabled", true)
	allowedHosts := s.getStringSetting("icon_proxy_domain_whitelist", DefaultDomainWhitelist)

	resolution := &Resolution{
		Proxy:        enabled,
		Provider:     spec.Provider,
		TargetDomain: targetDomain,
		UpstreamHost: spec.UpstreamHost,
		UpstreamURL:  spec.BuildURL(targetDomain),
		AllowedHosts: allowedHosts,
	}

	if enabled && !IsDomainAllowed(spec.UpstreamHost, allowedHosts) {
		return nil, ErrIconProxyDomainNotAllowed
	}

	return resolution, nil
}

func (s *Service) Fetch(ctx context.Context, resolution *Resolution) (*http.Response, error) {
	if resolution == nil || resolution.UpstreamURL == "" {
		return nil, errors.New("invalid icon proxy request")
	}

	parsed, err := url.Parse(resolution.UpstreamURL)
	if err != nil || parsed.Hostname() == "" {
		return nil, errors.New("invalid icon proxy request")
	}

	client := s.outboundHTTPClient()
	if !IsDomainAllowed(parsed.Hostname(), resolution.AllowedHosts) {
		return nil, ErrIconProxyDomainNotAllowed
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolution.UpstreamURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "SubduxIconProxy/1.0")

	checkedClient := *client
	originalCheckRedirect := client.CheckRedirect
	checkedClient.CheckRedirect = func(redirectReq *http.Request, via []*http.Request) error {
		if redirectReq == nil || redirectReq.URL == nil {
			return errors.New("invalid outbound request")
		}
		if !IsDomainAllowed(redirectReq.URL.Hostname(), resolution.AllowedHosts) {
			return ErrIconProxyDomainNotAllowed
		}
		if originalCheckRedirect != nil {
			return originalCheckRedirect(redirectReq, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}

	return checkedClient.Do(req)
}

func (s *Service) outboundHTTPClient() *http.Client {
	if s.httpClient != nil {
		return s.httpClient
	}
	client, err := serviceoutbound.BuildHTTPClientWithTimeout(context.Background(), s.DB, serviceoutbound.PurposeIconProxy, 10*time.Second)
	if err != nil {
		return serviceoutbound.NewOutboundHTTPClient(s.DB, 10*time.Second)
	}
	return client
}

func (s *Service) getBoolSetting(key string, defaultValue bool) bool {
	var setting model.SystemSetting
	if err := s.DB.Where("key = ?", key).First(&setting).Error; err != nil {
		return defaultValue
	}
	return setting.Value == "true"
}

func (s *Service) getStringSetting(key string, defaultValue string) string {
	var setting model.SystemSetting
	if err := s.DB.Where("key = ?", key).First(&setting).Error; err != nil {
		return defaultValue
	}
	if strings.TrimSpace(setting.Value) == "" {
		return defaultValue
	}
	return setting.Value
}

type iconProxyProviderSpec struct {
	Provider     string
	UpstreamHost string
	BuildURL     func(domain string) string
}

func getIconProxyProviderSpec(provider string) (iconProxyProviderSpec, error) {
	switch strings.TrimSpace(strings.ToLower(provider)) {
	case iconProxyProviderGoogle:
		return iconProxyProviderSpec{
			Provider:     iconProxyProviderGoogle,
			UpstreamHost: "www.google.com",
			BuildURL: func(domain string) string {
				return fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=64", url.QueryEscape(domain))
			},
		}, nil
	case iconProxyProviderIconHorse:
		return iconProxyProviderSpec{
			Provider:     iconProxyProviderIconHorse,
			UpstreamHost: "icon.horse",
			BuildURL: func(domain string) string {
				return fmt.Sprintf("https://icon.horse/icon/%s", url.PathEscape(domain))
			},
		}, nil
	default:
		return iconProxyProviderSpec{}, ErrInvalidIconProxyProvider
	}
}

func normalizeIconProxyTargetDomain(raw string) (string, error) {
	domain := strings.ToLower(strings.TrimSpace(raw))
	domain = strings.TrimRight(domain, ".")
	if domain == "" || !isValidDomainName(domain) {
		return "", ErrInvalidIconProxyTargetDomain
	}
	return domain, nil
}
