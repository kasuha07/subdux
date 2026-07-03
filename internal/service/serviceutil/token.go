package serviceutil

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateSecureToken(byteLen int) (string, error) {
	if byteLen <= 0 {
		byteLen = 16
	}

	buffer := make([]byte, byteLen)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
