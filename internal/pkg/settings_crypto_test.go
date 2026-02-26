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
