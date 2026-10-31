package notification

import (
	"net/url"
	"strconv"
	"strings"
)

// notificationChannelDestinationFields lists, per channel type, the config
// fields that decide where a channel's stored credentials are transmitted. A
// change to any of them is a change of trust authority: stored secrets entered
// for the old destination must not silently follow the channel to the new one.
//
// Channel types not listed here deliver to a fixed provider endpoint, or their
// destination is itself a secret field (feishu/wecom/dingtalk webhook_url), so
// changing it already requires re-entering that secret.
var notificationChannelDestinationFields = map[string][]string{
	"smtp":     {"host", "port", "encryption", "skip_tls_verify"},
	"webhook":  {"url"},
	"gotify":   {"url"},
	"ntfy":     {"url"},
	"bark":     {"url"},
	"pushdeer": {"server_url"},
	"pushplus": {"endpoint"},
	"pushover": {"endpoint"},
	"napcat":   {"url"},
}

// notificationChannelDestinationChanged reports whether the incoming config
// delivers somewhere other than the existing config, after normalizing
// representational differences (surrounding whitespace, host/scheme case,
// trailing slashes, SMTP defaults) that do not change the receiving party.
func notificationChannelDestinationChanged(channelType string, existing, incoming map[string]interface{}) bool {
	normalizedType := strings.ToLower(strings.TrimSpace(channelType))
	fields, ok := notificationChannelDestinationFields[normalizedType]
	if !ok {
		return false
	}
	for _, field := range fields {
		if normalizeNotificationDestinationValue(normalizedType, field, existing[field]) !=
			normalizeNotificationDestinationValue(normalizedType, field, incoming[field]) {
			return true
		}
	}
	return false
}

func normalizeNotificationDestinationValue(channelType, field string, raw interface{}) string {
	if channelType == "smtp" {
		switch field {
		case "host":
			return strings.ToLower(strings.TrimSpace(destinationScalarString(raw)))
		case "port":
			// Mirrors sendSMTP: out-of-range or missing ports fall back to 587.
			port, err := strconv.ParseInt(strings.TrimSpace(destinationScalarString(raw)), 10, 64)
			if err != nil || port <= 0 || port > 65535 {
				port = 587
			}
			return strconv.FormatInt(port, 10)
		case "encryption":
			// Mirrors sendSMTP: blank encryption means STARTTLS.
			encryption := strings.ToLower(strings.TrimSpace(destinationScalarString(raw)))
			if encryption == "" {
				encryption = "starttls"
			}
			return encryption
		case "skip_tls_verify":
			if value, ok := raw.(bool); ok && value {
				return "true"
			}
			return "false"
		}
	}
	return normalizeNotificationDestinationURL(destinationScalarString(raw))
}

func destinationScalarString(raw interface{}) string {
	switch value := raw.(type) {
	case nil:
		return ""
	case string:
		return value
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(value)
	default:
		return ""
	}
}

// normalizeNotificationDestinationURL canonicalizes the parts of a URL that do
// not change the receiving party. Path and query are kept (minus a trailing
// slash) because shared relay hosts route to different owners by path.
func normalizeNotificationDestinationURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return trimmed
	}
	normalized := strings.ToLower(parsed.Scheme) + "://"
	if parsed.User != nil {
		normalized += parsed.User.String() + "@"
	}
	normalized += strings.ToLower(parsed.Host) + strings.TrimRight(parsed.EscapedPath(), "/")
	if parsed.RawQuery != "" {
		normalized += "?" + parsed.RawQuery
	}
	return normalized
}
