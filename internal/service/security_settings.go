package service

import (
	"strings"

	"github.com/shiroha/subdux/internal/pkg"
)

const bcryptMaxPasswordBytes = 72

var encryptedSystemSettingKeys = map[string]struct{}{
	"smtp_password":      {},
	"oidc_client_secret": {},
}

func isEncryptedSystemSettingKey(key string) bool {
	_, exists := encryptedSystemSettingKeys[key]
	return exists
}

func encryptSystemSettingValueIfNeeded(key string, value string) (string, error) {
	if !isEncryptedSystemSettingKey(key) {
		return value, nil
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}

	return pkg.EncryptSystemSettingValue(value)
}

func decryptSystemSettingValueIfNeeded(key string, value string) (string, error) {
	if !isEncryptedSystemSettingKey(key) {
		return value, nil
	}

	return pkg.DecryptSystemSettingValue(value)
}

func validateBcryptPasswordLength(password string) error {
	if len([]byte(password)) > bcryptMaxPasswordBytes {
		return ErrPasswordTooLong
	}
	return nil
}
