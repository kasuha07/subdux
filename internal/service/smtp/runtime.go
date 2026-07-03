package smtp

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/outbound"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
	"gorm.io/gorm"
)

type RuntimeConfig struct {
	Host             string
	Port             int64
	Username         string
	Password         string
	FromEmail        string
	FromName         string
	Encryption       string
	AuthMethod       string
	HeloName         string
	TimeoutSeconds   int64
	RateLimitSeconds int64
	RateLimitDB      *gorm.DB
	SkipTLSVerify    bool
	DialContext      func(context.Context, string, string) (net.Conn, error)
}

func LoadRuntimeConfig(db *gorm.DB) (*RuntimeConfig, error) {
	if db == nil {
		return nil, errors.New("failed to load smtp settings")
	}

	values, err := loadSMTPRuntimeSettingValues(context.Background(), db)
	if err != nil {
		return nil, err
	}

	if values["smtp_enabled"] != "true" {
		return nil, errors.New("smtp is disabled")
	}

	host := strings.TrimSpace(values["smtp_host"])
	if host == "" {
		return nil, errors.New("smtp host is required")
	}

	port, err := strconv.ParseInt(strings.TrimSpace(values["smtp_port"]), 10, 64)
	if err != nil || port < 1 || port > 65535 {
		return nil, errors.New("smtp port must be between 1 and 65535")
	}

	fromEmail := strings.TrimSpace(values["smtp_from_email"])
	if fromEmail == "" {
		return nil, errors.New("smtp from email is required")
	}

	encryption := strings.ToLower(strings.TrimSpace(values["smtp_encryption"]))
	if encryption == "" {
		encryption = "starttls"
	}
	switch encryption {
	case "starttls", "ssl_tls", "none":
	default:
		return nil, errors.New("unsupported smtp encryption mode")
	}

	authMethod := strings.ToLower(strings.TrimSpace(values["smtp_auth_method"]))
	if authMethod == "" {
		authMethod = "auto"
	}
	switch authMethod {
	case "auto", "plain", "login", "cram_md5", "none":
	default:
		return nil, errors.New("unsupported smtp auth method")
	}

	timeoutSeconds, err := strconv.ParseInt(strings.TrimSpace(values["smtp_timeout_seconds"]), 10, 64)
	if err != nil || timeoutSeconds <= 0 {
		timeoutSeconds = 10
	}

	rateLimitSeconds := parseSMTPRateLimitSeconds(values["smtp_rate_limit_seconds"])
	username := strings.TrimSpace(values["smtp_username"])
	password := values["smtp_password"]

	if authMethod != "auto" && authMethod != "none" && (username == "" || strings.TrimSpace(password) == "") {
		return nil, errors.New("smtp username and password are required for selected auth method")
	}

	return &RuntimeConfig{
		Host:             host,
		Port:             port,
		Username:         username,
		Password:         password,
		FromEmail:        fromEmail,
		FromName:         strings.TrimSpace(values["smtp_from_name"]),
		Encryption:       encryption,
		AuthMethod:       authMethod,
		HeloName:         strings.TrimSpace(values["smtp_helo_name"]),
		TimeoutSeconds:   timeoutSeconds,
		RateLimitSeconds: rateLimitSeconds,
		RateLimitDB:      db,
		SkipTLSVerify:    values["smtp_skip_tls_verify"] == "true",
		DialContext:      outbound.NewOutboundDialContext(db, time.Duration(timeoutSeconds)*time.Second),
	}, nil
}

func loadSMTPRuntimeSettingValues(ctx context.Context, db *gorm.DB) (map[string]string, error) {
	defaults := map[string]string{
		"smtp_enabled":            "false",
		"smtp_host":               "",
		"smtp_port":               "587",
		"smtp_username":           "",
		"smtp_password":           "",
		"smtp_from_email":         "",
		"smtp_from_name":          "",
		"smtp_encryption":         "starttls",
		"smtp_auth_method":        "auto",
		"smtp_helo_name":          "",
		"smtp_timeout_seconds":    "10",
		"smtp_rate_limit_seconds": "0",
		"smtp_skip_tls_verify":    "false",
	}

	values, err := systemsettings.LoadRawStrings(ctx, db, defaults)
	if err != nil {
		return nil, errors.New("failed to load smtp settings")
	}

	for key, storedValue := range values {
		value, decryptErr := systemsettings.DecryptValueIfNeeded(key, storedValue)
		if decryptErr != nil {
			return nil, errors.New("failed to decrypt smtp settings")
		}
		if !pkg.IsSystemSettingEncrypted(storedValue) && value != "" && systemsettings.IsEncryptedKey(key) {
			if encryptedValue, encryptErr := systemsettings.EncryptValueIfNeeded(key, value); encryptErr == nil {
				_ = db.WithContext(ctx).Model(&model.SystemSetting{}).Where("key = ?", key).Update("value", encryptedValue).Error
			}
		}
		values[key] = value
	}

	return values, nil
}
