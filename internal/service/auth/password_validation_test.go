package auth

import (
	"strings"
	"testing"
)

func TestValidateBcryptPasswordLength(t *testing.T) {
	valid := strings.Repeat("a", bcryptMaxPasswordBytes)
	if err := ValidateBcryptPasswordLength(valid); err != nil {
		t.Fatalf("expected %d-byte password to pass, got %v", bcryptMaxPasswordBytes, err)
	}

	tooLong := strings.Repeat("a", bcryptMaxPasswordBytes+1)
	if err := ValidateBcryptPasswordLength(tooLong); err != ErrPasswordTooLong {
		t.Fatalf("expected ErrPasswordTooLong, got %v", err)
	}
}
