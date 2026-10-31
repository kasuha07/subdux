package notification

import (
	"encoding/json"
	"slices"
	"sort"
	"strings"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
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

func DecryptNotificationChannelConfig(config string) (string, error) {
	return decryptNotificationChannelConfig(config)
}

func encryptNotificationChannelConfig(config string) (string, error) {
	return pkg.EncryptNotificationChannelConfig(config)
}

func EncryptNotificationChannelConfig(config string) (string, error) {
	return encryptNotificationChannelConfig(config)
}

func getNotificationChannelSecretFields(channelType string) map[string]struct{} {
	fields, ok := notificationChannelSecretFields[strings.ToLower(strings.TrimSpace(channelType))]
	if !ok {
		return nil
	}
	return fields
}

func GetNotificationChannelSecretFields(channelType string) map[string]struct{} {
	return getNotificationChannelSecretFields(channelType)
}

func parseNotificationConfigMap(raw string) (map[string]interface{}, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "{}"
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, serviceerr.New(serviceerr.KindInvalid, "config_must_be_valid_json", "config must be valid JSON")
	}
	if parsed == nil {
		parsed = map[string]interface{}{}
	}
	return parsed, nil
}

func ParseNotificationConfigMap(raw string) (map[string]interface{}, error) {
	return parseNotificationConfigMap(raw)
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

func SanitizeNotificationConfig(channelType, config string) (string, []string, []string) {
	return sanitizeNotificationConfig(channelType, config)
}

// mergeNotificationConfigWithExistingSecrets rebuilds a channel config from an
// update payload, carrying forward stored secrets (and webhook header values)
// that the client left blank because responses only ever expose them masked.
//
// Carrying a secret forward is only safe while it keeps flowing to the same
// destination it was entered for. When the update changes where the channel
// delivers (see notificationChannelDestinationFingerprint), every stored secret
// that would otherwise be retained must be re-entered or explicitly cleared;
// otherwise the update is rejected so a caller cannot rebind a write-only
// credential to an endpoint it controls.
func mergeNotificationConfigWithExistingSecrets(channelType, existingConfig, incomingConfig string, clearedFields []string, clearedHeaderKeys []string) (string, error) {
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

	destinationChanged := notificationChannelDestinationChanged(channelType, existingParsed, incomingParsed)
	unresolved := make([]string, 0)

	for field := range secretFields {
		if sliceContains(clearedFields, field) {
			incomingParsed[field] = ""
			continue
		}

		existingValue, hasExisting := nonBlankStringValue(existingParsed, field)

		rawIncoming, hasIncoming := incomingParsed[field]
		if !hasIncoming {
			if !hasExisting {
				if rawExisting, ok := existingParsed[field]; ok {
					incomingParsed[field] = rawExisting
				}
				continue
			}
			if destinationChanged {
				unresolved = append(unresolved, field)
				continue
			}
			incomingParsed[field] = existingValue
			continue
		}

		incomingValue, ok := rawIncoming.(string)
		if !ok {
			continue
		}
		if strings.TrimSpace(incomingValue) != "" {
			continue
		}
		if !hasExisting {
			continue
		}
		if destinationChanged {
			unresolved = append(unresolved, field)
			continue
		}
		incomingParsed[field] = existingValue
	}
	if strings.EqualFold(strings.TrimSpace(channelType), "webhook") {
		unresolved = append(unresolved, mergeWebhookHeadersWithExisting(existingParsed, incomingParsed, clearedHeaderKeys, destinationChanged)...)
	}
	if len(unresolved) > 0 {
		sort.Strings(unresolved)
		return "", errNotificationSecretsRequireReentry(unresolved)
	}

	encoded, err := json.Marshal(incomingParsed)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func nonBlankStringValue(config map[string]interface{}, field string) (string, bool) {
	raw, ok := config[field]
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}
	return value, true
}

func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
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

// mergeWebhookHeadersWithExisting carries forward masked webhook header values
// the client left blank. When destinationChanged is true nothing is carried
// forward; instead the keys of stored header values that would have been
// retained are returned (prefixed with "headers.") so the caller can reject the
// update until they are re-entered or explicitly cleared.
func mergeWebhookHeadersWithExisting(existingConfig map[string]interface{}, incomingConfig map[string]interface{}, clearedHeaderKeys []string, destinationChanged bool) []string {
	existingRawHeaders, hasExistingHeaders := existingConfig["headers"]
	if !hasExistingHeaders {
		return nil
	}
	existingHeaders, ok := existingRawHeaders.(map[string]interface{})
	if !ok {
		return nil
	}

	incomingRawHeaders, hasIncomingHeaders := incomingConfig["headers"]
	if !hasIncomingHeaders {
		if !destinationChanged {
			incomingConfig["headers"] = existingHeaders
			return nil
		}
		unresolved := make([]string, 0)
		for key := range existingHeaders {
			if sliceContains(clearedHeaderKeys, key) {
				continue
			}
			if _, ok := nonBlankStringValue(existingHeaders, key); ok {
				unresolved = append(unresolved, "headers."+key)
			}
		}
		return unresolved
	}

	incomingHeaders, ok := incomingRawHeaders.(map[string]interface{})
	if !ok {
		return nil
	}
	if len(incomingHeaders) == 0 {
		return nil
	}

	unresolved := make([]string, 0)
	for key, rawIncoming := range incomingHeaders {
		if sliceContains(clearedHeaderKeys, key) {
			incomingHeaders[key] = ""
			continue
		}
		incomingValue, ok := rawIncoming.(string)
		if !ok {
			continue
		}
		if strings.TrimSpace(incomingValue) != "" {
			continue
		}
		existingValue, ok := nonBlankStringValue(existingHeaders, key)
		if !ok {
			continue
		}
		if destinationChanged {
			unresolved = append(unresolved, "headers."+key)
			continue
		}
		incomingHeaders[key] = existingValue
	}
	return unresolved
}

func (s *Service) SanitizeChannelForResponse(channel model.NotificationChannel) (model.NotificationChannel, []string, []string) {
	sanitizedConfig, configuredSecretFields, configuredWebhookHeaderKeys := sanitizeNotificationConfig(channel.Type, channel.Config)
	channel.Config = sanitizedConfig
	return channel, configuredSecretFields, configuredWebhookHeaderKeys
}
