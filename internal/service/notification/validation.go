package notification

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

// --- Channel config types for strict decoding and validation ---

type smtpChannelConfig struct {
	Host          string `json:"host"`
	Port          int64  `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	FromEmail     string `json:"from_email"`
	FromName      string `json:"from_name"`
	ToEmail       string `json:"to_email"`
	Encryption    string `json:"encryption"`
	SkipTLSVerify bool   `json:"skip_tls_verify"`
}

type resendChannelConfig struct {
	APIKey    string `json:"api_key"`
	FromEmail string `json:"from_email"`
	ToEmail   string `json:"to_email"`
}

type telegramChannelConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type webhookChannelConfig struct {
	URL     string            `json:"url"`
	Secret  string            `json:"secret"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

type gotifyChannelConfig struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type ntfyChannelConfig struct {
	URL      string `json:"url"`
	Topic    string `json:"topic"`
	Token    string `json:"token"`
	Username string `json:"username"`
	Password string `json:"password"`
	Priority string `json:"priority"`
	Tags     string `json:"tags"`
	Click    string `json:"click"`
	Icon     string `json:"icon"`
}

type barkChannelConfig struct {
	DeviceKey string `json:"device_key"`
	URL       string `json:"url"`
}

type serverchanChannelConfig struct {
	SendKey string `json:"send_key"`
}

type feishuChannelConfig struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret"`
}

type wecomChannelConfig struct {
	WebhookURL string `json:"webhook_url"`
}

type dingtalkChannelConfig struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret"`
}

type pushdeerChannelConfig struct {
	PushKey   string `json:"push_key"`
	ServerURL string `json:"server_url"`
}

type pushplusChannelConfig struct {
	Token    string `json:"token"`
	Endpoint string `json:"endpoint"`
	Template string `json:"template"`
	Channel  string `json:"channel"`
	Topic    string `json:"topic"`
}

type pushoverChannelConfig struct {
	Token    string `json:"token"`
	User     string `json:"user"`
	Device   string `json:"device"`
	Priority *int   `json:"priority"`
	Sound    string `json:"sound"`
	Endpoint string `json:"endpoint"`
}

type napcatChannelConfig struct {
	URL         string `json:"url"`
	AccessToken string `json:"access_token"`
	MessageType string `json:"message_type"`
	UserID      string `json:"user_id"`
	GroupID     string `json:"group_id"`
}

// --- Validation helpers ---

func isValidChannelType(t string) bool {
	switch t {
	case "smtp", "resend", "telegram", "webhook", "gotify", "ntfy", "bark", "serverchan", "feishu", "wecom", "dingtalk", "pushdeer", "pushplus", "pushover", "napcat":
		return true
	default:
		return false
	}
}

func normalizeWebhookMethod(method string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(method))
	if normalized == "" {
		return http.MethodPost, nil
	}

	switch normalized {
	case http.MethodGet, http.MethodPost, http.MethodPut:
		return normalized, nil
	default:
		return "", serviceerr.New(serviceerr.KindInvalid, "webhook method must be one of: GET, POST, PUT")
	}
}

func normalizeWebhookHeaders(headers map[string]string) (map[string]string, error) {
	normalized := make(map[string]string, len(headers))

	for key, value := range headers {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return nil, serviceerr.New(serviceerr.KindInvalid, "webhook headers cannot contain empty key")
		}
		if trimmedKey != key {
			return nil, serviceerr.New(serviceerr.KindInvalid, "webhook header name must not contain leading or trailing spaces")
		}
		if !isValidHTTPHeaderName(trimmedKey) {
			return nil, serviceerr.New(serviceerr.KindInvalid, "webhook header name contains invalid characters")
		}
		if strings.ContainsAny(trimmedKey, "\r\n") {
			return nil, serviceerr.New(serviceerr.KindInvalid, "webhook header name contains invalid newline characters")
		}
		if strings.ContainsAny(value, "\r\n") {
			return nil, serviceerr.New(serviceerr.KindInvalid, "webhook header value contains invalid newline characters")
		}

		normalized[trimmedKey] = value
	}

	return normalized, nil
}

func isValidHTTPHeaderName(name string) bool {
	for _, r := range name {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') {
			continue
		}

		switch r {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			continue
		default:
			return false
		}
	}

	return true
}

func IsValidChannelType(t string) bool {
	return isValidChannelType(t)
}

func validateChannelConfig(channelType, config string, db *gorm.DB) error {
	if strings.TrimSpace(config) == "" {
		config = "{}"
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(config), &raw); err != nil {
		return serviceerr.New(serviceerr.KindInvalid, "config must be valid JSON")
	}

	switch channelType {
	case "smtp":
		var cfg smtpChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid smtp config format")
		}
		if cfg.Host == "" {
			return serviceerr.New(serviceerr.KindInvalid, "smtp channel requires host")
		}
		if err := validateOutboundHost(cfg.Host, "smtp host", db); err != nil {
			return err
		}
		if cfg.FromEmail == "" {
			return serviceerr.New(serviceerr.KindInvalid, "smtp channel requires from_email")
		}
		if _, err := mail.ParseAddress(cfg.FromEmail); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid from_email address")
		}
		if cfg.ToEmail == "" {
			return serviceerr.New(serviceerr.KindInvalid, "smtp channel requires to_email")
		}
		if _, err := mail.ParseAddress(cfg.ToEmail); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid to_email address")
		}
		if cfg.Port < 0 || cfg.Port > 65535 {
			return serviceerr.New(serviceerr.KindInvalid, "smtp port must be between 0 and 65535")
		}
		return nil
	case "resend":
		var cfg resendChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid resend config format")
		}
		if cfg.APIKey == "" {
			return serviceerr.New(serviceerr.KindInvalid, "resend channel requires api_key")
		}
		if cfg.FromEmail == "" {
			return serviceerr.New(serviceerr.KindInvalid, "resend channel requires from_email")
		}
		if cfg.ToEmail == "" {
			return serviceerr.New(serviceerr.KindInvalid, "resend channel requires to_email")
		}
		if _, err := mail.ParseAddress(cfg.FromEmail); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid from_email address")
		}
		if _, err := mail.ParseAddress(cfg.ToEmail); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid to_email address")
		}
		return nil
	case "telegram":
		var cfg telegramChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid telegram config format")
		}
		if cfg.BotToken == "" {
			return serviceerr.New(serviceerr.KindInvalid, "telegram channel requires bot_token")
		}
		if cfg.ChatID == "" {
			return serviceerr.New(serviceerr.KindInvalid, "telegram channel requires chat_id")
		}
		return nil
	case "webhook":
		var cfg webhookChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid webhook config format")
		}
		if cfg.URL == "" {
			return serviceerr.New(serviceerr.KindInvalid, "webhook channel requires url")
		}
		if err := validateOutboundChannelURL(cfg.URL, "webhook url", false, db); err != nil {
			return err
		}
		method, err := normalizeWebhookMethod(cfg.Method)
		if err != nil {
			return err
		}
		if method == http.MethodGet && strings.TrimSpace(cfg.Secret) != "" {
			return serviceerr.New(serviceerr.KindInvalid, "webhook secret is not supported when method is GET")
		}
		if _, err := normalizeWebhookHeaders(cfg.Headers); err != nil {
			return err
		}
		return nil
	case "gotify":
		var cfg gotifyChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid gotify config format")
		}
		if cfg.URL == "" {
			return serviceerr.New(serviceerr.KindInvalid, "gotify channel requires url")
		}
		if err := validateOutboundChannelURL(cfg.URL, "gotify url", false, db); err != nil {
			return err
		}
		if cfg.Token == "" {
			return serviceerr.New(serviceerr.KindInvalid, "gotify channel requires token")
		}
		return nil
	case "ntfy":
		var cfg ntfyChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid ntfy config format")
		}
		if cfg.Topic == "" {
			return serviceerr.New(serviceerr.KindInvalid, "ntfy channel requires topic")
		}
		if cfg.URL != "" {
			if err := validateOutboundChannelURL(cfg.URL, "ntfy url", false, db); err != nil {
				return err
			}
		}
		return nil
	case "bark":
		var cfg barkChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid bark config format")
		}
		if cfg.DeviceKey == "" {
			return serviceerr.New(serviceerr.KindInvalid, "bark channel requires device_key")
		}
		if cfg.URL != "" {
			if err := validateOutboundChannelURL(cfg.URL, "bark url", false, db); err != nil {
				return err
			}
		}
		return nil
	case "serverchan":
		var cfg serverchanChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid serverchan config format")
		}
		if cfg.SendKey == "" {
			return serviceerr.New(serviceerr.KindInvalid, "serverchan channel requires send_key")
		}
		return nil
	case "feishu":
		var cfg feishuChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid feishu config format")
		}
		if cfg.WebhookURL == "" {
			return serviceerr.New(serviceerr.KindInvalid, "feishu channel requires webhook_url")
		}
		if err := validateOutboundChannelURL(cfg.WebhookURL, "feishu webhook_url", true, db); err != nil {
			return err
		}
		return nil
	case "wecom":
		var cfg wecomChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid wecom config format")
		}
		if cfg.WebhookURL == "" {
			return serviceerr.New(serviceerr.KindInvalid, "wecom channel requires webhook_url")
		}
		if err := validateOutboundChannelURL(cfg.WebhookURL, "wecom webhook_url", true, db); err != nil {
			return err
		}
		return nil
	case "dingtalk":
		var cfg dingtalkChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid dingtalk config format")
		}
		if cfg.WebhookURL == "" {
			return serviceerr.New(serviceerr.KindInvalid, "dingtalk channel requires webhook_url")
		}
		if err := validateOutboundChannelURL(cfg.WebhookURL, "dingtalk webhook_url", true, db); err != nil {
			return err
		}
		return nil
	case "pushdeer":
		var cfg pushdeerChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid pushdeer config format")
		}
		if strings.TrimSpace(cfg.PushKey) == "" {
			return serviceerr.New(serviceerr.KindInvalid, "pushdeer channel requires push_key")
		}
		serverURL := strings.TrimSpace(cfg.ServerURL)
		if serverURL != "" {
			if err := validateOutboundChannelURL(serverURL, "pushdeer server_url", false, db); err != nil {
				return err
			}
		}
		return nil
	case "pushplus":
		var cfg pushplusChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid pushplus config format")
		}
		if strings.TrimSpace(cfg.Token) == "" {
			return serviceerr.New(serviceerr.KindInvalid, "pushplus channel requires token")
		}
		endpoint := strings.TrimSpace(cfg.Endpoint)
		if endpoint != "" {
			if err := validateOutboundChannelURL(endpoint, "pushplus endpoint", false, db); err != nil {
				return err
			}
		}
		return nil
	case "pushover":
		var cfg pushoverChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid pushover config format")
		}
		if strings.TrimSpace(cfg.Token) == "" {
			return serviceerr.New(serviceerr.KindInvalid, "pushover channel requires token")
		}
		if strings.TrimSpace(cfg.User) == "" {
			return serviceerr.New(serviceerr.KindInvalid, "pushover channel requires user")
		}
		endpoint := strings.TrimSpace(cfg.Endpoint)
		if endpoint != "" {
			if err := validateOutboundChannelURL(endpoint, "pushover endpoint", false, db); err != nil {
				return err
			}
		}
		return nil
	case "napcat":
		var cfg napcatChannelConfig
		if err := decodeChannelConfigStrict(config, &cfg); err != nil {
			return serviceerr.New(serviceerr.KindInvalid, "invalid napcat config format")
		}
		if strings.TrimSpace(cfg.URL) == "" {
			return serviceerr.New(serviceerr.KindInvalid, "napcat channel requires url")
		}
		napcatURL := strings.TrimSpace(cfg.URL)
		if err := validateOutboundChannelURL(napcatURL, "napcat url", false, db); err != nil {
			return err
		}
		msgType := strings.ToLower(strings.TrimSpace(cfg.MessageType))
		if msgType == "" {
			msgType = "private"
		}
		if msgType != "private" && msgType != "group" {
			return serviceerr.New(serviceerr.KindInvalid, "napcat message_type must be private or group")
		}
		if msgType == "private" && strings.TrimSpace(cfg.UserID) == "" {
			return serviceerr.New(serviceerr.KindInvalid, "napcat channel requires user_id for private messages")
		}
		if msgType == "group" && strings.TrimSpace(cfg.GroupID) == "" {
			return serviceerr.New(serviceerr.KindInvalid, "napcat channel requires group_id for group messages")
		}
		return nil
	default:
		return serviceerr.New(serviceerr.KindInvalid, "unsupported channel type")
	}
}

func ValidateChannelConfig(channelType, config string, db *gorm.DB) error {
	return validateChannelConfig(channelType, config, db)
}

func decodeChannelConfigStrict(config string, out interface{}) error {
	decoder := json.NewDecoder(bytes.NewBufferString(config))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("config must contain a single JSON object")
	}
	return nil
}
