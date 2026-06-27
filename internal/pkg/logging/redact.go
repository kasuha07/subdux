package logging

import (
	"log/slog"
	"net/url"
	"strings"
)

// redactedPlaceholder replaces the value of any field deemed sensitive.
const redactedPlaceholder = "[REDACTED]"

// sensitiveKeys lists exact field names whose values must never be logged, in
// any structured log attribute, anywhere in the codebase. Matching is
// case-insensitive. These names denote secret material unambiguously, so
// redacting them globally cannot hide legitimate diagnostics.
var sensitiveKeys = map[string]struct{}{
	"access_token":  {},
	"api_key":       {},
	"apikey":        {},
	"authorization": {},
	"id_token":      {},
	"jwt":           {},
	"otp":           {},
	"password":      {},
	"refresh_token": {},
	"secret":        {},
	"token":         {},
	"totp_token":    {},
}

// queryOnlySensitiveKeys lists short, ambiguous parameter names that reliably
// denote secrets only in the context of a URL query string (an OAuth
// authorization "code", a generic "key" parameter). They are redacted by
// SanitizeQuery but deliberately excluded from the global attribute redaction
// so structured fields legitimately named "code" (a country/status code) or
// "key" (a cache or map key) are not silently masked.
var queryOnlySensitiveKeys = map[string]struct{}{
	"code": {},
	"key":  {},
}

// sensitiveSuffixes flags families of secret-bearing field names without
// enumerating every variant (e.g. "client_secret", "session_token").
var sensitiveSuffixes = []string{
	"_token",
	"_secret",
	"_password",
	"_key",
}

// IsSensitiveKey reports whether a structured log attribute with this name
// should have its value redacted. This is the global predicate applied to
// every log record; it intentionally excludes the query-only ambiguous names.
func IsSensitiveKey(key string) bool {
	normalized := normalizeKey(key)
	if normalized == "" {
		return false
	}

	if _, ok := sensitiveKeys[normalized]; ok {
		return true
	}

	for _, suffix := range sensitiveSuffixes {
		if strings.HasSuffix(normalized, suffix) {
			return true
		}
	}

	return false
}

// isSensitiveQueryParam reports whether a URL query parameter should have its
// value masked. It is broader than IsSensitiveKey because short parameter names
// are unambiguous in a query context.
func isSensitiveQueryParam(key string) bool {
	if IsSensitiveKey(key) {
		return true
	}
	if _, ok := queryOnlySensitiveKeys[normalizeKey(key)]; ok {
		return true
	}
	return false
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

// redactAttr is installed as the slog ReplaceAttr hook. It masks the value of
// any attribute whose key is sensitive, regardless of where in the codebase
// the record originated, so secrets cannot leak through a careless log call.
func redactAttr(_ []string, attr slog.Attr) slog.Attr {
	if IsSensitiveKey(attr.Key) {
		attr.Value = slog.StringValue(redactedPlaceholder)
	}
	return attr
}

// SanitizeURI returns the request path with its query string rewritten so that
// sensitive parameters are masked. The path itself is preserved; only
// parameter values are touched. An empty path is normalized to "/".
func SanitizeURI(path string, query url.Values) string {
	if path == "" {
		path = "/"
	}

	sanitized := SanitizeQuery(query)
	if sanitized == "" {
		return path
	}
	return path + "?" + sanitized
}

// SanitizeQuery re-encodes query parameters, masking the values of sensitive
// keys while preserving order-independent equality with url.Values.Encode.
func SanitizeQuery(query url.Values) string {
	if len(query) == 0 {
		return ""
	}

	sanitized := make(url.Values, len(query))
	for key, values := range query {
		if isSensitiveQueryParam(key) {
			sanitized[key] = []string{redactedPlaceholder}
			continue
		}

		cloned := make([]string, len(values))
		copy(cloned, values)
		sanitized[key] = cloned
	}

	return sanitized.Encode()
}
