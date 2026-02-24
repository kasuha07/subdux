package pkg

import (
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

func TestEncryptSystemSettingValueRequiresKey(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "")
	t.Setenv("JWT_SECRET", "")

	if _, err := EncryptSystemSettingValue("secret"); err == nil {
		t.Fatal("expected missing-key error, got nil")
	}
}
