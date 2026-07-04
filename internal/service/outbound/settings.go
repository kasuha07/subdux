package outbound

import (
	"net"
	"net/url"
	"strings"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
	"gorm.io/gorm"
)

const (
	SystemProxyTypeHTTP   = "http"
	SystemProxyTypeSOCKS5 = "socks5"
)

var (
	ErrInvalidSystemProxyType = serviceerr.New(serviceerr.KindInvalid, "system proxy type must be http or socks5")
	ErrInvalidSystemProxyURL  = serviceerr.New(serviceerr.KindInvalid, "system proxy url must include a host")
)

type SystemProxyConfig struct {
	Enabled  bool
	Type     string
	URL      string
	HasValue bool
}

func LoadSystemProxyConfig(db *gorm.DB) (SystemProxyConfig, error) {
	cfg := SystemProxyConfig{
		Enabled: false,
		Type:    SystemProxyTypeHTTP,
		URL:     "",
	}
	if db == nil {
		return cfg, nil
	}

	var items []model.SystemSetting
	if err := db.Where("key IN ?", []string{
		"system_proxy_enabled",
		"system_proxy_type",
		"system_proxy_url",
	}).Find(&items).Error; err != nil {
		return cfg, err
	}

	for _, item := range items {
		settingValue := item.Value
		decryptedValue, decryptErr := systemsettings.DecryptValueIfNeeded(item.Key, item.Value)
		if decryptErr == nil {
			settingValue = decryptedValue
		}

		switch item.Key {
		case "system_proxy_enabled":
			cfg.Enabled = settingValue == "true"
		case "system_proxy_type":
			cfg.Type = strings.TrimSpace(strings.ToLower(settingValue))
		case "system_proxy_url":
			cfg.URL = settingValue
			cfg.HasValue = strings.TrimSpace(settingValue) != ""
		}
	}

	if strings.TrimSpace(cfg.Type) == "" {
		cfg.Type = SystemProxyTypeHTTP
	}

	return cfg, nil
}

func ValidateIncomingSystemProxySettings(tx *gorm.DB, enabledInput *bool, typeInput *string, urlInput *string) error {
	cfg, err := LoadSystemProxyConfig(tx)
	if err != nil {
		return err
	}

	proxyType := cfg.Type
	if typeInput != nil {
		proxyType, err = NormalizeSystemProxyType(*typeInput)
		if err != nil {
			return err
		}
	}

	proxyURL := cfg.URL
	if urlInput != nil {
		proxyURL = *urlInput
	}

	enabled := cfg.Enabled
	if enabledInput != nil {
		enabled = *enabledInput
	}

	if urlInput == nil && !enabled {
		return nil
	}

	return ValidateSystemProxySettings(proxyType, proxyURL, enabled)
}

func ValidateSystemProxySettings(proxyType string, proxyURL string, enabled bool) error {
	normalizedType, err := NormalizeSystemProxyType(proxyType)
	if err != nil {
		return err
	}

	trimmedURL := strings.TrimSpace(proxyURL)
	if trimmedURL == "" {
		if enabled {
			return ErrInvalidSystemProxyURL
		}
		return nil
	}

	_, err = NormalizeSystemProxyURL(normalizedType, trimmedURL)
	return err
}

func NormalizeSystemProxyType(proxyType string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(proxyType))
	if normalized == "" {
		return SystemProxyTypeHTTP, nil
	}
	switch normalized {
	case SystemProxyTypeHTTP, SystemProxyTypeSOCKS5:
		return normalized, nil
	default:
		return "", ErrInvalidSystemProxyType
	}
}

func NormalizeSystemProxyURL(proxyType string, rawURL string) (*url.URL, error) {
	normalizedType, err := NormalizeSystemProxyType(proxyType)
	if err != nil {
		return nil, err
	}

	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return nil, ErrInvalidSystemProxyURL
	}
	if !strings.Contains(trimmedURL, "://") {
		trimmedURL = normalizedType + "://" + trimmedURL
	}

	parsed, err := url.Parse(trimmedURL)
	if err != nil || parsed.Hostname() == "" {
		return nil, ErrInvalidSystemProxyURL
	}

	switch normalizedType {
	case SystemProxyTypeHTTP:
		if parsed.Scheme != "http" {
			return nil, serviceerr.Wrap(serviceerr.KindInvalid, "system proxy url must start with http://", ErrInvalidSystemProxyURL)
		}
	case SystemProxyTypeSOCKS5:
		if parsed.Scheme != "socks5" && parsed.Scheme != "socks5h" {
			return nil, serviceerr.Wrap(serviceerr.KindInvalid, "system proxy url must start with socks5://", ErrInvalidSystemProxyURL)
		}
		parsed.Scheme = "socks5"
	}

	if parsed.Port() != "" {
		port, err := net.LookupPort("tcp", parsed.Port())
		if err != nil || port < 1 || port > 65535 {
			return nil, serviceerr.Wrap(serviceerr.KindInvalid, "system proxy url port is invalid", ErrInvalidSystemProxyURL)
		}
	}

	return parsed, nil
}
