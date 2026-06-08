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
	"log"
	"os"
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
	dataPath := GetDataPath()
	if err := os.MkdirAll(dataPath, 0o750); err != nil {
		return "", fmt.Errorf("failed to create settings key directory: %w", err)
	}

	root, err := os.OpenRoot(dataPath)
	if err != nil {
		return "", fmt.Errorf("failed to open settings key directory: %w", err)
	}
	defer func() {
		_ = root.Close()
	}()

	if existing, err := root.ReadFile(settingsKeyFileName); err == nil {
		if value := strings.TrimSpace(string(existing)); value != "" {
			return value, nil
		}
		// Self-healing: existing file is empty or whitespace-only (e.g. from a
		// previous crash during write). Treat as non-existent and regenerate.
		log.Printf("WARNING: local settings key file %q is empty, regenerating", settingsKeyFileName)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read local settings key: %w", err)
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate local settings key: %w", err)
	}
	newKey := hex.EncodeToString(randomBytes)

	// Atomic write: write to a temporary file, sync to disk, then rename.
	// This ensures the key file is never left in a partially-written state.
	tmpName := settingsKeyFileName + ".tmp"
	tmpFile, err := root.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary settings key file: %w", err)
	}

	if _, err := tmpFile.WriteString(newKey); err != nil {
		cleanupLocalSettingsKeyTemp(root, tmpFile, tmpName)
		return "", fmt.Errorf("failed to write temporary settings key file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		cleanupLocalSettingsKeyTemp(root, tmpFile, tmpName)
		return "", fmt.Errorf("failed to sync temporary settings key file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = root.Remove(tmpName)
		return "", fmt.Errorf("failed to close temporary settings key file: %w", err)
	}

	if err := root.Rename(tmpName, settingsKeyFileName); err != nil {
		_ = root.Remove(tmpName)
		return "", fmt.Errorf("failed to finalize settings key file: %w", err)
	}

	// After rename, verify the final file as a safety check against race
	// conditions with concurrent process starts.
	verifyContent, err := root.ReadFile(settingsKeyFileName)
	if err != nil {
		return "", fmt.Errorf("failed to verify settings key file after write: %w", err)
	}
	verified := strings.TrimSpace(string(verifyContent))
	if verified == "" {
		return "", fmt.Errorf("settings key file is empty after write")
	}

	return verified, nil
}

func cleanupLocalSettingsKeyTemp(root *os.Root, tmpFile *os.File, tmpName string) {
	_ = tmpFile.Close()
	_ = root.Remove(tmpName)
}
