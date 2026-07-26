package migrations

import (
	"errors"
	"testing"
)

// testSecretCodec returns an identity codec. Migration tests seed and assert on
// plaintext fixtures, so round-tripping through identity exercises the same
// rewrite paths without coupling the tests to the application's key material.
func testSecretCodec() SecretCodec {
	return SecretCodec{
		Encrypt: func(value string) (string, error) { return value, nil },
		Decrypt: func(value string) (string, error) { return value, nil },
	}
}

// A migration that rewrites a secret must fail loudly when no codec is wired.
// Falling back to a pass-through would write the secret back in plaintext.
func TestSecretCodecRequiresFuncs(t *testing.T) {
	var empty SecretCodec

	if _, err := empty.decrypt("value"); !errors.Is(err, errSecretCodecUnavailable) {
		t.Fatalf("decrypt without codec error = %v, want errSecretCodecUnavailable", err)
	}
	if _, err := empty.encrypt("value"); !errors.Is(err, errSecretCodecUnavailable) {
		t.Fatalf("encrypt without codec error = %v, want errSecretCodecUnavailable", err)
	}
}
