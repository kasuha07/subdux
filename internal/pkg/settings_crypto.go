package pkg

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const settingsEncryptedPrefix = "enc:v1:"
const settingsKeyFileName = ".subdux_settings_key"

func EncryptNotificationChannelConfig(value string) (string, error) {
	return encryptWithDerivedKey(value)
}

func DecryptNotificationChannelConfig(value string) (string, error) {
	return decryptWithDerivedKey(value)
}

func EncryptSystemSettingValue(value string) (string, error) {
	return encryptWithDerivedKey(value)
}

func encryptWithDerivedKey(value string) (string, error) {
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
	return decryptWithDerivedKey(value)
}

func decryptWithDerivedKey(value string) (string, error) {
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
		var err error
		raw, err = loadOrCreateLocalSettingsKey()
		if err != nil {
			return nil, err
		}
	}
	if raw == "" {
		return nil, fmt.Errorf("system settings encryption key is not configured")
	}

	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}

func loadOrCreateLocalSettingsKey() (string, error) {
	keyPath := filepath.Join(GetDataPath(), settingsKeyFileName)

	if existing, err := os.ReadFile(keyPath); err == nil {
		if value := strings.TrimSpace(string(existing)); value != "" {
			return value, nil
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read local settings key: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0o755); err != nil {
		return "", fmt.Errorf("failed to create settings key directory: %w", err)
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate local settings key: %w", err)
	}
	newKey := hex.EncodeToString(randomBytes)

	file, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			existing, readErr := os.ReadFile(keyPath)
			if readErr != nil {
				return "", fmt.Errorf("failed to read existing local settings key: %w", readErr)
			}
			if value := strings.TrimSpace(string(existing)); value != "" {
				return value, nil
			}
			return "", fmt.Errorf("existing local settings key file is empty")
		}
		return "", fmt.Errorf("failed to create local settings key file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(newKey); err != nil {
		return "", fmt.Errorf("failed to write local settings key file: %w", err)
	}

	return newKey, nil
}
