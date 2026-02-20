package api

import (
	"path/filepath"
	"strings"
	"unicode"
)

func isEmojiRune(r rune) bool {
	if r == '\u200D' || r == '\uFE0F' || r == '\uFE0E' {
		return true
	}
	if r >= 0x1F1E0 && r <= 0x1F1FF {
		return true
	}
	if r < 0x00A0 {
		return false
	}
	return unicode.IsGraphic(r) && !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsPunct(r) && !unicode.IsSpace(r)
}

func validateIcon(icon string) bool {
	if icon == "" {
		return true
	}

	if isManagedAssetIcon(icon) {
		return true
	}

	if strings.HasPrefix(icon, "si:") ||
		strings.HasPrefix(icon, "http://") ||
		strings.HasPrefix(icon, "https://") {
		return true
	}
	for _, r := range icon {
		if !isEmojiRune(r) {
			return false
		}
	}
	return true
}

func isManagedAssetIcon(icon string) bool {
	const iconPrefix = "assets/icons/"
	if !strings.HasPrefix(icon, iconPrefix) {
		return false
	}

	filename := strings.TrimPrefix(icon, iconPrefix)
	if filename == "" {
		return false
	}
	if strings.Contains(filename, "/") || strings.Contains(filename, `\`) {
		return false
	}
	if filepath.Base(filename) != filename {
		return false
	}

	return true
}
