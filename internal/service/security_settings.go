package service

import (
	"github.com/kasuha07/subdux/internal/service/settings"
)

const bcryptMaxPasswordBytes = 72

func isEncryptedSystemSettingKey(key string) bool {
	return settings.IsEncryptedKey(key)
}

func encryptSystemSettingValueIfNeeded(key string, value string) (string, error) {
	return settings.EncryptValueIfNeeded(key, value)
}

func decryptSystemSettingValueIfNeeded(key string, value string) (string, error) {
	return settings.DecryptValueIfNeeded(key, value)
}

func validateBcryptPasswordLength(password string) error {
	if len([]byte(password)) > bcryptMaxPasswordBytes {
		return ErrPasswordTooLong
	}
	return nil
}
