package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
)

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
		return "", errors.New("webhook method must be one of: GET, POST, PUT")
	}
}

func normalizeWebhookHeaders(headers map[string]string) (map[string]string, error) {
	normalized := make(map[string]string, len(headers))

	for key, value := range headers {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return nil, errors.New("webhook headers cannot contain empty key")
		}
		if trimmedKey != key {
			return nil, errors.New("webhook header name must not contain leading or trailing spaces")
		}
		if !isValidHTTPHeaderName(trimmedKey) {
			return nil, errors.New("webhook header name contains invalid characters")
		}
		if strings.ContainsAny(trimmedKey, "\r\n") {
			return nil, errors.New("webhook header name contains invalid newline characters")
		}
		if strings.ContainsAny(value, "\r\n") {
			return nil, errors.New("webhook header value contains invalid newline characters")
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

func validateChannelConfig(channelType, config string) error {
	if strings.TrimSpace(config) == "" {
		config = "{}"
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(config), &raw); err != nil {
		return errors.New("config must be valid JSON")
	}

	switch channelType {
	case "smtp":
		var cfg struct {
			Host      string `json:"host"`
			Port      int64  `json:"port"`
			FromEmail string `json:"from_email"`
			ToEmail   string `json:"to_email"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid smtp config format")
		}
		if cfg.Host == "" {
			return errors.New("smtp channel requires host")
		}
		if cfg.FromEmail == "" {
			return errors.New("smtp channel requires from_email")
		}
		if _, err := mail.ParseAddress(cfg.FromEmail); err != nil {
			return errors.New("invalid from_email address")
		}
		if cfg.ToEmail == "" {
			return errors.New("smtp channel requires to_email")
		}
		if _, err := mail.ParseAddress(cfg.ToEmail); err != nil {
			return errors.New("invalid to_email address")
		}
		if cfg.Port < 0 || cfg.Port > 65535 {
			return errors.New("smtp port must be between 0 and 65535")
		}
		return nil
	case "resend":
		var cfg struct {
			APIKey    string `json:"api_key"`
			FromEmail string `json:"from_email"`
			ToEmail   string `json:"to_email"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid resend config format")
		}
		if cfg.APIKey == "" {
			return errors.New("resend channel requires api_key")
		}
		if cfg.FromEmail == "" {
			return errors.New("resend channel requires from_email")
		}
		if cfg.ToEmail == "" {
			return errors.New("resend channel requires to_email")
		}
		if _, err := mail.ParseAddress(cfg.FromEmail); err != nil {
			return errors.New("invalid from_email address")
		}
		if _, err := mail.ParseAddress(cfg.ToEmail); err != nil {
			return errors.New("invalid to_email address")
		}
		return nil
	case "telegram":
		var cfg struct {
			BotToken string `json:"bot_token"`
			ChatID   string `json:"chat_id"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid telegram config format")
		}
		if cfg.BotToken == "" {
			return errors.New("telegram channel requires bot_token")
		}
		if cfg.ChatID == "" {
			return errors.New("telegram channel requires chat_id")
		}
		return nil
	case "webhook":
		var cfg struct {
			URL     string            `json:"url"`
			Secret  string            `json:"secret"`
			Method  string            `json:"method"`
			Headers map[string]string `json:"headers"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid webhook config format")
		}
		if cfg.URL == "" {
			return errors.New("webhook channel requires url")
		}
		if err := validateOutboundChannelURL(cfg.URL, "webhook url", false); err != nil {
			return err
		}
		method, err := normalizeWebhookMethod(cfg.Method)
		if err != nil {
			return err
		}
		if method == http.MethodGet && strings.TrimSpace(cfg.Secret) != "" {
			return errors.New("webhook secret is not supported when method is GET")
		}
		if _, err := normalizeWebhookHeaders(cfg.Headers); err != nil {
			return err
		}
		return nil
	case "gotify":
		var cfg struct {
			URL   string `json:"url"`
			Token string `json:"token"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid gotify config format")
		}
		if cfg.URL == "" {
			return errors.New("gotify channel requires url")
		}
		if err := validateOutboundChannelURL(cfg.URL, "gotify url", false); err != nil {
			return err
		}
		if cfg.Token == "" {
			return errors.New("gotify channel requires token")
		}
		return nil
	case "ntfy":
		var cfg struct {
			Topic string `json:"topic"`
			URL   string `json:"url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid ntfy config format")
		}
		if cfg.Topic == "" {
			return errors.New("ntfy channel requires topic")
		}
		if cfg.URL != "" {
			if err := validateOutboundChannelURL(cfg.URL, "ntfy url", false); err != nil {
				return err
			}
		}
		return nil
	case "bark":
		var cfg struct {
			DeviceKey string `json:"device_key"`
			URL       string `json:"url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid bark config format")
		}
		if cfg.DeviceKey == "" {
			return errors.New("bark channel requires device_key")
		}
		if cfg.URL != "" {
			if err := validateOutboundChannelURL(cfg.URL, "bark url", false); err != nil {
				return err
			}
		}
		return nil
	case "serverchan":
		var cfg struct {
			SendKey string `json:"send_key"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid serverchan config format")
		}
		if cfg.SendKey == "" {
			return errors.New("serverchan channel requires send_key")
		}
		return nil
	case "feishu":
		var cfg struct {
			WebhookURL string `json:"webhook_url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid feishu config format")
		}
		if cfg.WebhookURL == "" {
			return errors.New("feishu channel requires webhook_url")
		}
		if err := validateOutboundChannelURL(cfg.WebhookURL, "feishu webhook_url", true); err != nil {
			return err
		}
		return nil
	case "wecom":
		var cfg struct {
			WebhookURL string `json:"webhook_url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid wecom config format")
		}
		if cfg.WebhookURL == "" {
			return errors.New("wecom channel requires webhook_url")
		}
		if err := validateOutboundChannelURL(cfg.WebhookURL, "wecom webhook_url", true); err != nil {
			return err
		}
		return nil
	case "dingtalk":
		var cfg struct {
			WebhookURL string `json:"webhook_url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid dingtalk config format")
		}
		if cfg.WebhookURL == "" {
			return errors.New("dingtalk channel requires webhook_url")
		}
		if err := validateOutboundChannelURL(cfg.WebhookURL, "dingtalk webhook_url", true); err != nil {
			return err
		}
		return nil
	case "pushdeer":
		var cfg struct {
			PushKey   string `json:"push_key"`
			ServerURL string `json:"server_url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid pushdeer config format")
		}
		if strings.TrimSpace(cfg.PushKey) == "" {
			return errors.New("pushdeer channel requires push_key")
		}
		serverURL := strings.TrimSpace(cfg.ServerURL)
		if serverURL != "" {
			if err := validateOutboundChannelURL(serverURL, "pushdeer server_url", false); err != nil {
				return err
			}
		}
		return nil
	case "pushplus":
		var cfg struct {
			Token    string `json:"token"`
			Endpoint string `json:"endpoint"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid pushplus config format")
		}
		if strings.TrimSpace(cfg.Token) == "" {
			return errors.New("pushplus channel requires token")
		}
		endpoint := strings.TrimSpace(cfg.Endpoint)
		if endpoint != "" {
			if err := validateOutboundChannelURL(endpoint, "pushplus endpoint", false); err != nil {
				return err
			}
		}
		return nil
	case "pushover":
		var cfg struct {
			Token    string `json:"token"`
			User     string `json:"user"`
			Endpoint string `json:"endpoint"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid pushover config format")
		}
		if strings.TrimSpace(cfg.Token) == "" {
			return errors.New("pushover channel requires token")
		}
		if strings.TrimSpace(cfg.User) == "" {
			return errors.New("pushover channel requires user")
		}
		endpoint := strings.TrimSpace(cfg.Endpoint)
		if endpoint != "" {
			if err := validateOutboundChannelURL(endpoint, "pushover endpoint", false); err != nil {
				return err
			}
		}
		return nil
	case "napcat":
		var cfg struct {
			URL         string `json:"url"`
			MessageType string `json:"message_type"`
			UserID      string `json:"user_id"`
			GroupID     string `json:"group_id"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid napcat config format")
		}
		if strings.TrimSpace(cfg.URL) == "" {
			return errors.New("napcat channel requires url")
		}
		napcatURL := strings.TrimSpace(cfg.URL)
		if err := validateOutboundChannelURL(napcatURL, "napcat url", false); err != nil {
			return err
		}
		msgType := strings.ToLower(strings.TrimSpace(cfg.MessageType))
		if msgType == "" {
			msgType = "private"
		}
		if msgType != "private" && msgType != "group" {
			return errors.New("napcat message_type must be private or group")
		}
		if msgType == "private" && strings.TrimSpace(cfg.UserID) == "" {
			return errors.New("napcat channel requires user_id for private messages")
		}
		if msgType == "group" && strings.TrimSpace(cfg.GroupID) == "" {
			return errors.New("napcat channel requires group_id for group messages")
		}
		return nil
	default:
		return errors.New("unsupported channel type")
	}
}
