package backup

import (
	"errors"
	"strings"
	"testing"
)

func TestLocalDestinationConfigStrictSchema(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name   string
		config map[string]any
		want   error
	}{
		{
			name:   "directory has the wrong type",
			config: map[string]any{"dir": 123},
			want:   ErrInvalidBackupDestinationConfig,
		},
		{
			name:   "unknown field is rejected",
			config: map[string]any{"dir": dir, "directory": dir},
			want:   ErrInvalidBackupDestinationConfig,
		},
		{
			name:   "fractional retention is rejected",
			config: map[string]any{"dir": dir, "retention_count": 5.5},
			want:   ErrInvalidBackupDestinationConfig,
		},
		{
			name:   "floating point retention is rejected even when integral",
			config: map[string]any{"dir": dir, "retention_count": float64(5)},
			want:   ErrInvalidBackupDestinationConfig,
		},
		{
			name:   "retention below minimum is rejected",
			config: map[string]any{"dir": dir, "retention_count": 0},
			want:   ErrInvalidBackupRetentionCount,
		},
		{
			name:   "retention above maximum is rejected",
			config: map[string]any{"dir": dir, "retention_count": 1001},
			want:   ErrInvalidBackupRetentionCount,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newLocalTarget(tc.config)
			if !errors.Is(err, tc.want) {
				t.Fatalf("newLocalTarget() error = %v, want %v", err, tc.want)
			}
		})
	}

	target, err := newLocalTarget(map[string]any{"dir": dir, "retention_count": 1000})
	if err != nil {
		t.Fatalf("newLocalTarget() at maximum retention error = %v", err)
	}
	if target.RetentionCount() != 1000 {
		t.Fatalf("RetentionCount() = %d, want 1000", target.RetentionCount())
	}
}

func TestDestinationConfigRejectsJSONFloatingPointRetention(t *testing.T) {
	config, err := parseDestinationConfigMap(`{"dir":"backups","retention_count":5.0}`)
	if err != nil {
		t.Fatalf("parseDestinationConfigMap() error = %v", err)
	}

	if _, err := newLocalTarget(config); !errors.Is(err, ErrInvalidBackupDestinationConfig) {
		t.Fatalf("newLocalTarget() error = %v, want ErrInvalidBackupDestinationConfig", err)
	}
}

func TestS3DestinationConfigRejectsWrongFieldType(t *testing.T) {
	config := map[string]any{
		"endpoint":          "s3.example.com",
		"bucket":            "backups",
		"access_key_id":     "AKIA",
		"secret_access_key": "secret",
		"use_ssl":           "true",
	}

	if _, err := parseS3Config(config); !errors.Is(err, ErrInvalidBackupDestinationConfig) {
		t.Fatalf("parseS3Config() error = %v, want ErrInvalidBackupDestinationConfig", err)
	}
}

func TestSanitizeDestinationConfigRejectsMalformedKnownFields(t *testing.T) {
	tests := []struct {
		name   string
		kind   string
		config string
	}{
		{name: "local directory", kind: "local", config: `{"dir":123}`},
		{name: "s3 boolean", kind: "s3", config: `{"use_ssl":"true"}`},
		{name: "webdav secret", kind: "webdav", config: `{"password":123}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := sanitizeDestinationConfig(tc.kind, tc.config); !errors.Is(err, ErrInvalidBackupDestinationConfig) {
				t.Fatalf("sanitizeDestinationConfig() error = %v, want ErrInvalidBackupDestinationConfig", err)
			}
		})
	}
}

func TestSanitizeDestinationConfigStripsUnknownFields(t *testing.T) {
	config, _, err := sanitizeDestinationConfig("local", `{"dir":"backups","unknown":"not returned"}`)
	if err != nil {
		t.Fatalf("sanitizeDestinationConfig() error = %v", err)
	}
	if strings.Contains(config, "unknown") || !strings.Contains(config, "backups") {
		t.Fatalf("sanitized config = %s, want known field only", config)
	}
}
