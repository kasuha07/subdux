package pkg

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

const settingsEncryptedPrefix = "enc:v1:"

func EncryptSystemSettingValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	key, err := getSystemSettingsKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := aead.Seal(nil, nonce, []byte(value), nil)
	payload := append(nonce, ciphertext...)
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return settingsEncryptedPrefix + encoded, nil
}

func DecryptSystemSettingValue(value string) (string, error) {
	if !strings.HasPrefix(value, settingsEncryptedPrefix) {
		return value, nil
	}

	key, err := getSystemSettingsKey()
	if err != nil {
		return "", err
	}

	encoded := strings.TrimPrefix(value, settingsEncryptedPrefix)
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New("invalid encrypted system setting")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aead.NonceSize()
	if len(payload) < nonceSize {
		return "", errors.New("invalid encrypted system setting")
	}

	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]

	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("failed to decrypt system setting")
	}

	return string(plain), nil
}

func IsSystemSettingEncrypted(value string) bool {
	return strings.HasPrefix(value, settingsEncryptedPrefix)
}

func getSystemSettingsKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("SETTINGS_ENCRYPTION_KEY"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	}
	if raw == "" {
		return nil, fmt.Errorf("system settings encryption key is not configured (set SETTINGS_ENCRYPTION_KEY or JWT_SECRET environment variable)")
	}

	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}
