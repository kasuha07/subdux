package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/url"
	"strings"
	"testing"
)

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		// Exact matches (case-insensitive, trimmed).
		{key: "password", want: true},
		{key: "Password", want: true},
		{key: "  token  ", want: true},
		{key: "access_token", want: true},
		{key: "refresh_token", want: true},
		{key: "api_key", want: true},
		{key: "apikey", want: true},
		{key: "secret", want: true},
		{key: "id_token", want: true},
		{key: "otp", want: true},
		{key: "totp_token", want: true},
		{key: "authorization", want: true},
		{key: "jwt", want: true},
		// Suffix families.
		{key: "client_secret", want: true},
		{key: "session_token", want: true},
		{key: "user_password", want: true},
		{key: "encryption_key", want: true},
		// Ambiguous short names are NOT globally redacted (only in query
		// strings) so legitimate structured fields are not silently masked.
		{key: "code", want: false},
		{key: "key", want: false},
		// Non-sensitive.
		{key: "username", want: false},
		{key: "email", want: false},
		{key: "page", want: false},
		{key: "", want: false},
		{key: "keyboard", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := IsSensitiveKey(tt.key); got != tt.want {
				t.Fatalf("IsSensitiveKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestIsSensitiveQueryParamIncludesAmbiguousNames(t *testing.T) {
	// In a query-string context, the short ambiguous names ARE sensitive (an
	// OAuth authorization "code", a generic "key" parameter).
	tests := []struct {
		key  string
		want bool
	}{
		{key: "code", want: true},
		{key: "key", want: true},
		{key: "password", want: true}, // also covered by the global set
		{key: "client_secret", want: true},
		{key: "page", want: false},
		{key: "state", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := isSensitiveQueryParam(tt.key); got != tt.want {
				t.Fatalf("isSensitiveQueryParam(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestRedactAttrLeavesAmbiguousNamesIntact(t *testing.T) {
	var buf bytes.Buffer
	logger := Configure(Options{Level: slog.LevelInfo, Format: FormatJSON, Output: &buf})

	logger.Info("operational detail",
		slog.String("code", "US"),
		slog.String("key", "cache:user:42"),
	)

	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &record); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if record["code"] != "US" {
		t.Fatalf("code = %v, want US (must not be redacted as a structured field)", record["code"])
	}
	if record["key"] != "cache:user:42" {
		t.Fatalf("key = %v, want cache:user:42 (must not be redacted as a structured field)", record["key"])
	}
}

func TestSanitizeURI(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		query url.Values
		want  string
	}{
		{
			name: "no query",
			path: "/api/subscriptions",
			want: "/api/subscriptions",
		},
		{
			name:  "empty path normalizes to slash",
			path:  "",
			query: nil,
			want:  "/",
		},
		{
			name:  "sensitive value redacted",
			path:  "/api/auth/oidc/callback",
			query: url.Values{"code": {"super-secret"}, "state": {"xyz"}},
			want:  "/api/auth/oidc/callback?code=%5BREDACTED%5D&state=xyz",
		},
		{
			name:  "non-sensitive preserved",
			path:  "/api/items",
			query: url.Values{"page": {"2"}, "limit": {"50"}},
			want:  "/api/items?limit=50&page=2",
		},
		{
			name:  "suffix family redacted",
			path:  "/cb",
			query: url.Values{"client_secret": {"hunter2"}},
			want:  "/cb?client_secret=%5BREDACTED%5D",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeURI(tt.path, tt.query); got != tt.want {
				t.Fatalf("SanitizeURI(%q, %v) = %q, want %q", tt.path, tt.query, got, tt.want)
			}
		})
	}
}

func TestRedactAttrMasksSensitiveFieldsInHandler(t *testing.T) {
	var buf bytes.Buffer
	logger := Configure(Options{Level: slog.LevelInfo, Format: FormatJSON, Output: &buf})

	logger.Info("login attempt",
		slog.String("username", "alice"),
		slog.String("password", "should-not-appear"),
		slog.String("refresh_token", "also-secret"),
	)

	out := buf.String()
	if strings.Contains(out, "should-not-appear") {
		t.Fatalf("password value leaked into log output: %s", out)
	}
	if strings.Contains(out, "also-secret") {
		t.Fatalf("refresh_token value leaked into log output: %s", out)
	}

	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &record); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if record["username"] != "alice" {
		t.Fatalf("non-sensitive field altered: username = %v", record["username"])
	}
	if record["password"] != redactedPlaceholder {
		t.Fatalf("password = %v, want %q", record["password"], redactedPlaceholder)
	}
	if record["refresh_token"] != redactedPlaceholder {
		t.Fatalf("refresh_token = %v, want %q", record["refresh_token"], redactedPlaceholder)
	}
}
