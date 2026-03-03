package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncryptDecryptSystemSettingValue(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "unit-test-settings-key")
	t.Setenv("JWT_SECRET", "")

	encrypted, err := EncryptSystemSettingValue("smtp-secret")
	if err != nil {
		t.Fatalf("EncryptSystemSettingValue() error = %v", err)
	}
	if !strings.HasPrefix(encrypted, settingsEncryptedPrefix) {
		t.Fatalf("encrypted value prefix mismatch: %q", encrypted)
	}

	decrypted, err := DecryptSystemSettingValue(encrypted)
	if err != nil {
		t.Fatalf("DecryptSystemSettingValue() error = %v", err)
	}
	if decrypted != "smtp-secret" {
		t.Fatalf("decrypted value = %q, want %q", decrypted, "smtp-secret")
	}
}

func TestDecryptSystemSettingValueLegacyPlaintext(t *testing.T) {
	plain, err := DecryptSystemSettingValue("legacy-plain")
	if err != nil {
		t.Fatalf("DecryptSystemSettingValue() error = %v", err)
	}
	if plain != "legacy-plain" {
		t.Fatalf("value = %q, want %q", plain, "legacy-plain")
	}
}

func TestEncryptSystemSettingValueUsesLocalKeyFileFallback(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DATA_PATH", t.TempDir())

	encrypted, err := EncryptSystemSettingValue("secret")
	if err != nil {
		t.Fatalf("EncryptSystemSettingValue() error = %v", err)
	}
	decrypted, err := DecryptSystemSettingValue(encrypted)
	if err != nil {
		t.Fatalf("DecryptSystemSettingValue() error = %v", err)
	}
	if decrypted != "secret" {
		t.Fatalf("decrypted value = %q, want %q", decrypted, "secret")
	}

	keyPath := filepath.Join(GetDataPath(), settingsKeyFileName)
	content, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("os.ReadFile(local settings key) error = %v", err)
	}
	if strings.TrimSpace(string(content)) == "" {
		t.Fatal("local settings key file should not be empty")
	}
}

func TestLoadOrCreateLocalSettingsKeyAtomicRecovery(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DATA_PATH", t.TempDir())

	// Pre-create an empty key file to simulate a crash during previous write
	keyPath := filepath.Join(GetDataPath(), settingsKeyFileName)
	if err := os.WriteFile(keyPath, []byte(""), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	// loadOrCreateLocalSettingsKey should self-heal: detect the empty file and regenerate
	key, err := loadOrCreateLocalSettingsKey()
	if err != nil {
		t.Fatalf("loadOrCreateLocalSettingsKey() error = %v", err)
	}
	if strings.TrimSpace(key) == "" {
		t.Fatal("loadOrCreateLocalSettingsKey() returned empty key after recovery")
	}

	// Verify the file on disk is now valid
	content, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if strings.TrimSpace(string(content)) == "" {
		t.Fatal("key file should not be empty after recovery")
	}
}

func TestLoadOrCreateLocalSettingsKeyPreservesExisting(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DATA_PATH", t.TempDir())

	// Pre-create a valid key file
	keyPath := filepath.Join(GetDataPath(), settingsKeyFileName)
	expectedKey := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	if err := os.WriteFile(keyPath, []byte(expectedKey), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	key, err := loadOrCreateLocalSettingsKey()
	if err != nil {
		t.Fatalf("loadOrCreateLocalSettingsKey() error = %v", err)
	}
	if key != expectedKey {
		t.Fatalf("loadOrCreateLocalSettingsKey() = %q, want %q", key, expectedKey)
	}
}
