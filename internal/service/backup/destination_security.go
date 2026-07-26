package backup

import (
	"encoding/json"
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/kasuha07/subdux/internal/pkg"
)

// destinationSecretFields lists, per destination type, the config fields that
// hold credentials. These are masked on read (never returned to the client in
// plaintext) and preserved on update when the client submits an empty value,
// mirroring the notification-channel secret handling. "local" has no secrets.
var destinationSecretFields = map[string]map[string]struct{}{
	"local":  {},
	"s3":     {"secret_access_key": {}},
	"webdav": {"password": {}},
}

func destinationSecretFieldSet(destinationType string) map[string]struct{} {
	fields, ok := destinationSecretFields[strings.ToLower(strings.TrimSpace(destinationType))]
	if !ok {
		return nil
	}
	return fields
}

// encryptDestinationConfig encrypts a config JSON blob using the same envelope
// as notification channel configs so destination secrets never rest in
// plaintext. Empty input yields empty output.
func encryptDestinationConfig(config string) (string, error) {
	return pkg.EncryptNotificationChannelConfig(config)
}

// decryptDestinationConfig reverses encryptDestinationConfig. Values without the
// encryption envelope prefix (e.g. the plaintext local config written by the
// backfill migration) are returned unchanged.
func decryptDestinationConfig(config string) (string, error) {
	return pkg.DecryptNotificationChannelConfig(config)
}

// sanitizeDestinationConfig decrypts stored config and rebuilds it from the
// destination type's explicit schema. It blanks every secret field and hides
// unsafe legacy WebDAV URLs and S3 endpoints with embedded userinfo, returning
// the sorted list of secret fields that were actually configured. The client
// uses the field list to render "configured / re-enter" affordances without
// ever receiving the secret. An unreadable or malformed stored config returns
// ErrInvalidBackupDestinationConfig instead of being represented as an empty
// object.
func sanitizeDestinationConfig(destinationType, config string) (string, []string, error) {
	plain, err := decryptDestinationConfig(config)
	if err != nil {
		return "", nil, ErrInvalidBackupDestinationConfig
	}
	parsed, err := parseDestinationConfigMap(plain)
	if err != nil {
		return "", nil, ErrInvalidBackupDestinationConfig
	}

	destinationType = strings.ToLower(strings.TrimSpace(destinationType))
	allowedFields, ok := destinationConfigSchemas[destinationType]
	if !ok {
		return "", nil, ErrInvalidBackupDestinationConfig
	}

	secretFields := destinationSecretFieldSet(destinationType)
	sanitized := make(map[string]any, len(allowedFields))
	configured := make([]string, 0)
	for field := range allowedFields {
		raw, ok := parsed[field]
		if !ok {
			continue
		}
		if !isDestinationConfigFieldType(raw, allowedFields[field]) {
			return "", nil, ErrInvalidBackupDestinationConfig
		}

		if _, isSecret := secretFields[field]; isSecret {
			if strings.TrimSpace(raw.(string)) != "" {
				configured = append(configured, field)
			}
			sanitized[field] = ""
			continue
		}

		if (destinationType == "webdav" && field == "url") ||
			(destinationType == "s3" && field == "endpoint") {
			value := raw.(string)
			parsedURL, parseErr := url.Parse(strings.TrimSpace(value))
			if parseErr != nil || parsedURL == nil || parsedURL.User != nil ||
				(destinationType == "s3" && strings.Contains(value, "@")) {
				// New writes reject URL userinfo. Keep old or externally-mutated
				// rows from reflecting credentials through the DTO as well. S3's
				// bare host form is also checked for '@' because url.Parse does
				// not treat userinfo without a scheme as URL userinfo.
				sanitized[field] = ""
				continue
			}
		}

		sanitized[field] = raw
	}
	sort.Strings(configured)

	encoded, err := json.Marshal(sanitized)
	if err != nil {
		return "", nil, ErrInvalidBackupDestinationConfig
	}
	return string(encoded), configured, nil
}

// mergeDestinationConfigWithExistingSecrets carries forward previously-stored
// secret values when the client submits an empty (masked) field, and clears a
// secret when the field name is listed in clearedFields. Non-secret fields are
// taken from the incoming config verbatim.
func mergeDestinationConfigWithExistingSecrets(destinationType, existingConfig, incomingConfig string, clearedFields []string) (string, error) {
	incomingParsed, err := parseDestinationConfigMap(incomingConfig)
	if err != nil {
		return "", err
	}

	secretFields := destinationSecretFieldSet(destinationType)
	if len(secretFields) == 0 {
		encoded, err := json.Marshal(incomingParsed)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}

	existingPlain, err := decryptDestinationConfig(existingConfig)
	if err != nil {
		return "", err
	}
	existingParsed, err := parseDestinationConfigMap(existingPlain)
	if err != nil {
		return "", err
	}

	for field := range secretFields {
		if slices.Contains(clearedFields, field) {
			incomingParsed[field] = ""
			continue
		}

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
		if strings.TrimSpace(incomingValue) != "" {
			continue
		}

		if rawExisting, ok := existingParsed[field]; ok {
			if existingValue, ok := rawExisting.(string); ok && strings.TrimSpace(existingValue) != "" {
				incomingParsed[field] = existingValue
			}
		}
	}

	encoded, err := json.Marshal(incomingParsed)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
