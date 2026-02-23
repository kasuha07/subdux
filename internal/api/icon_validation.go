package api

import (
	"net/url"
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

	if isIconGoIcon(icon) {
		return true
	}

	for _, r := range icon {
		if !isEmojiRune(r) {
			return false
		}
	}
	return true
}

func validateSubscriptionIcon(icon string) bool {
	if validateIcon(icon) {
		return true
	}
	return isExternalImageIconURL(icon)
}

func isManagedAssetIcon(icon string) bool {
	const iconPrefix = "file:"
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
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".ico" {
		return false
	}

	return true
}

func isExternalImageIconURL(icon string) bool {
	parsed, err := url.ParseRequestURI(icon)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}

func isIconGoIcon(icon string) bool {
	prefix, slug, found := strings.Cut(icon, ":")
	if !found || prefix == "" || slug == "" {
		return false
	}

	if prefix == "si" || prefix == "file" || len(prefix) < 2 || len(prefix) > 16 {
		return false
	}

	for _, r := range prefix {
		if r < 'a' || r > 'z' {
			return false
		}
	}

	for _, r := range slug {
		isLowerAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isLowerAlpha || isDigit || r == '-' || r == '_' {
			continue
		}
		return false
	}

	return true
}
