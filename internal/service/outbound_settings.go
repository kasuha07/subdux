package service

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

const (
	systemProxyTypeHTTP   = "http"
	systemProxyTypeSOCKS5 = "socks5"
)

var (
	ErrInvalidSystemProxyType = errors.New("system proxy type must be http or socks5")
	ErrInvalidSystemProxyURL  = errors.New("system proxy url must include a host")
)

type systemProxyConfig struct {
	Enabled  bool
	Type     string
	URL      string
	HasValue bool
}

func loadSystemProxyConfig(db *gorm.DB) (systemProxyConfig, error) {
	cfg := systemProxyConfig{
		Enabled: false,
		Type:    systemProxyTypeHTTP,
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
		decryptedValue, decryptErr := decryptSystemSettingValueIfNeeded(item.Key, item.Value)
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
		cfg.Type = systemProxyTypeHTTP
	}

	return cfg, nil
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

func validateSystemProxySettings(proxyType string, proxyURL string, enabled bool) error {
	normalizedType, err := normalizeSystemProxyType(proxyType)
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

	_, err = normalizeSystemProxyURL(normalizedType, trimmedURL)
	return err
}

func normalizeSystemProxyType(proxyType string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(proxyType))
	if normalized == "" {
		return systemProxyTypeHTTP, nil
	}
	switch normalized {
	case systemProxyTypeHTTP, systemProxyTypeSOCKS5:
		return normalized, nil
	default:
		return "", ErrInvalidSystemProxyType
	}
}

func normalizeSystemProxyURL(proxyType string, rawURL string) (*url.URL, error) {
	normalizedType, err := normalizeSystemProxyType(proxyType)
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
	case systemProxyTypeHTTP:
		if parsed.Scheme != "http" {
			return nil, fmt.Errorf("system proxy url must start with http://")
		}
	case systemProxyTypeSOCKS5:
		if parsed.Scheme != "socks5" && parsed.Scheme != "socks5h" {
			return nil, fmt.Errorf("system proxy url must start with socks5://")
		}
		parsed.Scheme = "socks5"
	}

	if parsed.Port() != "" {
		port, err := net.LookupPort("tcp", parsed.Port())
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("system proxy url port is invalid")
		}
	}

	return parsed, nil
}
