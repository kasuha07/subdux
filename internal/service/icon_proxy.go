package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

const (
	iconProxyProviderGoogle    = "google"
	iconProxyProviderIconHorse = "icon-horse"
)

var (
	ErrInvalidIconProxyProvider     = errors.New("invalid icon proxy provider")
	ErrInvalidIconProxyTargetDomain = errors.New("invalid icon proxy target domain")
	ErrIconProxyDomainNotAllowed    = errors.New("icon proxy domain is not allowed")
)

type IconProxyService struct {
	DB         *gorm.DB
	httpClient *http.Client
}

type IconProxyResolution struct {
	Proxy        bool
	Provider     string
	TargetDomain string
	UpstreamHost string
	UpstreamURL  string
	AllowedHosts string
}

func NewIconProxyService(db *gorm.DB) *IconProxyService {
	return &IconProxyService{
		DB: db,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *IconProxyService) Resolve(provider string, rawDomain string) (*IconProxyResolution, error) {
	spec, err := getIconProxyProviderSpec(provider)
	if err != nil {
		return nil, err
	}

	targetDomain, err := normalizeIconProxyTargetDomain(rawDomain)
	if err != nil {
		return nil, err
	}

	enabled := s.getBoolSetting("icon_proxy_enabled", true)
	allowedHosts := s.getStringSetting("icon_proxy_domain_whitelist", defaultIconProxyDomainWhitelist)

	resolution := &IconProxyResolution{
		Proxy:        enabled,
		Provider:     spec.Provider,
		TargetDomain: targetDomain,
		UpstreamHost: spec.UpstreamHost,
		UpstreamURL:  spec.BuildURL(targetDomain),
		AllowedHosts: allowedHosts,
	}

	if enabled && !isIconProxyDomainAllowed(spec.UpstreamHost, allowedHosts) {
		return nil, ErrIconProxyDomainNotAllowed
	}

	return resolution, nil
}

func (s *IconProxyService) Fetch(ctx context.Context, resolution *IconProxyResolution) (*http.Response, error) {
	if resolution == nil || resolution.UpstreamURL == "" {
		return nil, errors.New("invalid icon proxy request")
	}

	parsed, err := url.Parse(resolution.UpstreamURL)
	if err != nil || parsed.Hostname() == "" {
		return nil, errors.New("invalid icon proxy request")
	}

	if err := validateResolvedOutboundHost(parsed.Hostname()); err != nil {
		return nil, err
	}
	if !isIconProxyDomainAllowed(parsed.Hostname(), resolution.AllowedHosts) {
		return nil, ErrIconProxyDomainNotAllowed
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolution.UpstreamURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "SubduxIconProxy/1.0")

	client := *s.httpClient
	originalCheckRedirect := client.CheckRedirect
	client.CheckRedirect = func(redirectReq *http.Request, via []*http.Request) error {
		if redirectReq == nil || redirectReq.URL == nil {
			return errors.New("invalid outbound request")
		}
		if err := validateResolvedOutboundHost(redirectReq.URL.Hostname()); err != nil {
			return err
		}
		if !isIconProxyDomainAllowed(redirectReq.URL.Hostname(), resolution.AllowedHosts) {
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

	return client.Do(req)
}

func (s *IconProxyService) getBoolSetting(key string, defaultValue bool) bool {
	var setting model.SystemSetting
	if err := s.DB.Where("key = ?", key).First(&setting).Error; err != nil {
		return defaultValue
	}
	return setting.Value == "true"
}

func (s *IconProxyService) getStringSetting(key string, defaultValue string) string {
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
	if domain == "" || !isValidEmailDomain(domain) {
		return "", ErrInvalidIconProxyTargetDomain
	}
	return domain, nil
}
