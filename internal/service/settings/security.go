package settings

import (
	"strings"

	"github.com/kasuha07/subdux/internal/pkg"
)

var encryptedKeys = map[string]struct{}{
	"smtp_password":              {},
	"oidc_client_secret":         {},
	"currencyapi_key":            {},
	"system_proxy_url":           {},
	"backup_encryption_password": {},
}

func IsEncryptedKey(key string) bool {
	_, exists := encryptedKeys[key]
	return exists
}

func EncryptValueIfNeeded(key string, value string) (string, error) {
	if !IsEncryptedKey(key) {
		return value, nil
	}

	if strings.TrimSpace(value) == "" {
		return "", nil
	}

	return pkg.EncryptSystemSettingValue(value)
}

func DecryptValueIfNeeded(key string, value string) (string, error) {
	if !IsEncryptedKey(key) {
		return value, nil
	}

	return pkg.DecryptSystemSettingValue(value)
}
