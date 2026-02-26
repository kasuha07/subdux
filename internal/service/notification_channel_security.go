package service

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
)

var notificationChannelSecretFields = map[string]map[string]struct{}{
	"smtp":       {"password": {}},
	"resend":     {"api_key": {}},
	"telegram":   {"bot_token": {}},
	"webhook":    {"secret": {}},
	"gotify":     {"token": {}},
	"ntfy":       {"token": {}, "password": {}},
	"bark":       {"device_key": {}},
	"serverchan": {"send_key": {}},
	"feishu":     {"webhook_url": {}, "secret": {}},
	"wecom":      {"webhook_url": {}},
	"dingtalk":   {"webhook_url": {}, "secret": {}},
	"pushdeer":   {"push_key": {}},
	"pushplus":   {"token": {}},
	"pushover":   {"token": {}, "user": {}},
	"napcat":     {"access_token": {}},
}

func decryptNotificationChannelConfig(config string) (string, error) {
	return pkg.DecryptNotificationChannelConfig(config)
}

func encryptNotificationChannelConfig(config string) (string, error) {
	return pkg.EncryptNotificationChannelConfig(config)
}

func getNotificationChannelSecretFields(channelType string) map[string]struct{} {
	fields, ok := notificationChannelSecretFields[strings.ToLower(strings.TrimSpace(channelType))]
	if !ok {
		return nil
	}
	return fields
}

func parseNotificationConfigMap(raw string) (map[string]interface{}, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "{}"
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, fmt.Errorf("config must be valid JSON")
	}
	if parsed == nil {
		parsed = map[string]interface{}{}
	}
	return parsed, nil
}

func sanitizeNotificationConfig(channelType, config string) (string, []string, []string) {
	plain, err := decryptNotificationChannelConfig(config)
	if err != nil {
		return "{}", nil, nil
	}

	parsed, err := parseNotificationConfigMap(plain)
	if err != nil {
		return "{}", nil, nil
	}

	configuredSecretFields := make([]string, 0)
	secretFields := getNotificationChannelSecretFields(channelType)
	if len(secretFields) > 0 {
		for field := range secretFields {
			if raw, ok := parsed[field]; ok {
				if value, ok := raw.(string); ok && strings.TrimSpace(value) != "" {
					parsed[field] = ""
					if !slices.Contains(configuredSecretFields, field) {
						configuredSecretFields = append(configuredSecretFields, field)
					}
				}
			}
		}
	}
	configuredWebhookHeaderKeys := make([]string, 0)
	if strings.EqualFold(strings.TrimSpace(channelType), "webhook") {
		configuredWebhookHeaderKeys = maskWebhookHeaders(parsed)
	}
	sort.Strings(configuredSecretFields)
	sort.Strings(configuredWebhookHeaderKeys)

	encoded, err := json.Marshal(parsed)
	if err != nil {
		return "{}", configuredSecretFields, configuredWebhookHeaderKeys
	}
	return string(encoded), configuredSecretFields, configuredWebhookHeaderKeys
}

func mergeNotificationConfigWithExistingSecrets(channelType, existingConfig, incomingConfig string) (string, error) {
	incomingParsed, err := parseNotificationConfigMap(incomingConfig)
	if err != nil {
		return "", err
	}

	secretFields := getNotificationChannelSecretFields(channelType)
	if len(secretFields) == 0 {
		encoded, err := json.Marshal(incomingParsed)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}

	existingPlain, err := decryptNotificationChannelConfig(existingConfig)
	if err != nil {
		return "", err
	}
	existingParsed, err := parseNotificationConfigMap(existingPlain)
	if err != nil {
		return "", err
	}

	for field := range secretFields {
		rawIncoming, hasIncoming := incomingParsed[field]
		if !hasIncoming {
			if rawExisting, ok := existingParsed[field]; ok {
				incomingParsed[field] = rawExisting
			}
			continue
		}

		incomingValue, ok := rawIncoming.(string)
		if !ok {
			continue
		}

		trimmedIncoming := strings.TrimSpace(incomingValue)
		if trimmedIncoming != "" {
			continue
		}

		rawExisting, ok := existingParsed[field]
		if !ok {
			continue
		}
		if existingValue, ok := rawExisting.(string); ok && strings.TrimSpace(existingValue) != "" {
			incomingParsed[field] = existingValue
		}
	}
	if strings.EqualFold(strings.TrimSpace(channelType), "webhook") {
		mergeWebhookHeadersWithExisting(existingParsed, incomingParsed)
	}

	encoded, err := json.Marshal(incomingParsed)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func maskWebhookHeaders(config map[string]interface{}) []string {
	rawHeaders, ok := config["headers"]
	if !ok {
		return nil
	}

	headers, ok := rawHeaders.(map[string]interface{})
	if !ok {
		return nil
	}

	maskedHeaderKeys := make([]string, 0)
	for key, rawValue := range headers {
		value, ok := rawValue.(string)
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}
		headers[key] = ""
		maskedHeaderKeys = append(maskedHeaderKeys, key)
	}
	return maskedHeaderKeys
}

func mergeWebhookHeadersWithExisting(existingConfig map[string]interface{}, incomingConfig map[string]interface{}) {
	existingRawHeaders, hasExistingHeaders := existingConfig["headers"]
	if !hasExistingHeaders {
		return
	}
	existingHeaders, ok := existingRawHeaders.(map[string]interface{})
	if !ok {
		return
	}

	incomingRawHeaders, hasIncomingHeaders := incomingConfig["headers"]
	if !hasIncomingHeaders {
		incomingConfig["headers"] = existingHeaders
		return
	}

	incomingHeaders, ok := incomingRawHeaders.(map[string]interface{})
	if !ok {
		return
	}
	if len(incomingHeaders) == 0 {
		return
	}

	for key, rawIncoming := range incomingHeaders {
		incomingValue, ok := rawIncoming.(string)
		if !ok {
			continue
		}
		trimmedIncoming := strings.TrimSpace(incomingValue)
		if trimmedIncoming != "" {
			continue
		}
		rawExisting, ok := existingHeaders[key]
		if !ok {
			continue
		}
		existingValue, ok := rawExisting.(string)
		if !ok || strings.TrimSpace(existingValue) == "" {
			continue
		}
		incomingHeaders[key] = existingValue
	}
}

func (s *NotificationService) SanitizeChannelForResponse(channel model.NotificationChannel) (model.NotificationChannel, []string, []string) {
	sanitizedConfig, configuredSecretFields, configuredWebhookHeaderKeys := sanitizeNotificationConfig(channel.Type, channel.Config)
	channel.Config = sanitizedConfig
	return channel, configuredSecretFields, configuredWebhookHeaderKeys
}
