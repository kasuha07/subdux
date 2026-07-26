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

// TestParseScheduleConfigDefaultsTimeOfDay covers the "a destination is always a
// plan" invariant: an absent or blank firing time means the default, never
// "this destination never runs".
func TestParseScheduleConfigDefaultsTimeOfDay(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]any
	}{
		{name: "absent", config: map[string]any{"dir": "backups"}},
		{name: "empty string", config: map[string]any{"time_of_day": ""}},
		{name: "whitespace only", config: map[string]any{"time_of_day": "   "}},
		{name: "nil config", config: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schedule, err := parseScheduleConfig(tc.config)
			if err != nil {
				t.Fatalf("parseScheduleConfig() error = %v", err)
			}
			if schedule.TimeOfDay != defaultBackupTimeOfDay {
				t.Fatalf("time of day = %q, want default %q", schedule.TimeOfDay, defaultBackupTimeOfDay)
			}
		})
	}
}

func TestParseScheduleConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]any
		want   error
	}{
		{
			name:   "hour out of range",
			config: map[string]any{"time_of_day": "24:00"},
			want:   ErrInvalidBackupTimeOfDay,
		},
		{
			// The pattern demands a zero-padded hour so the stored value sorts and
			// renders identically everywhere it is displayed.
			name:   "hour not zero padded",
			config: map[string]any{"time_of_day": "7:00"},
			want:   ErrInvalidBackupTimeOfDay,
		},
		{
			name:   "not a time at all",
			config: map[string]any{"time_of_day": "abc"},
			want:   ErrInvalidBackupTimeOfDay,
		},
		{
			name:   "encryption without a password",
			config: map[string]any{"encrypt_enabled": true},
			want:   ErrBackupEncryptionPasswordRequired,
		},
		{
			name:   "encryption with a blank password",
			config: map[string]any{"encrypt_enabled": true, "encryption_password": ""},
			want:   ErrBackupEncryptionPasswordRequired,
		},
		{
			// Whitespace is not a password: accepting it would produce an archive
			// nobody could plausibly re-enter the key for.
			name:   "encryption with a whitespace password",
			config: map[string]any{"encrypt_enabled": true, "encryption_password": "   "},
			want:   ErrBackupEncryptionPasswordRequired,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseScheduleConfig(tc.config); !errors.Is(err, tc.want) {
				t.Fatalf("parseScheduleConfig() error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestParseScheduleConfigAcceptsEncryptionWithPassword(t *testing.T) {
	const password = "  correct horse  "

	schedule, err := parseScheduleConfig(map[string]any{
		"time_of_day":         "23:45",
		"include_assets":      true,
		"encrypt_enabled":     true,
		"encryption_password": password,
	})
	if err != nil {
		t.Fatalf("parseScheduleConfig() error = %v", err)
	}
	if schedule.TimeOfDay != "23:45" || !schedule.IncludeAssets || !schedule.EncryptEnabled {
		t.Fatalf("schedule = %+v, want 23:45 with assets and encryption", schedule)
	}
	// Only the blank check trims. The password itself is kept verbatim, because
	// the archive must be openable with exactly what the admin typed.
	if schedule.EncryptPassword != password {
		t.Fatalf("password = %q, want the value verbatim %q", schedule.EncryptPassword, password)
	}
	if spec := schedule.archiveSpec(); !spec.IncludeAssets || spec.Password != password {
		t.Fatalf("archiveSpec() = %+v, want the encrypting spec", spec)
	}
}

// TestArchiveSpecIgnoresPasswordWhenEncryptionIsOff matters because the spec is
// the scheduler's grouping key: a leftover password on a destination that no
// longer encrypts must not split it onto an archive of its own.
func TestArchiveSpecIgnoresPasswordWhenEncryptionIsOff(t *testing.T) {
	schedule := destinationSchedule{
		TimeOfDay:       "03:00",
		EncryptEnabled:  false,
		EncryptPassword: "left-over",
	}

	if spec := schedule.archiveSpec(); spec.Password != "" {
		t.Fatalf("archiveSpec() = %+v, want no password when encryption is off", spec)
	}
	if schedule.archiveSpec() != (destinationSchedule{TimeOfDay: "23:00"}).archiveSpec() {
		t.Fatal("two plain plans that differ only in firing time must share an archive spec")
	}
}

// TestDecodeDestinationConfigStrictWithScheduleFields proves the schedule half is
// validated but withheld from the transport decode: the target struct never
// declares those fields, so a leak would trip DisallowUnknownFields, while a
// typo or a wrong type must still be rejected.
func TestDecodeDestinationConfigStrictWithScheduleFields(t *testing.T) {
	dir := t.TempDir()
	withSchedule := func(extra map[string]any) map[string]any {
		config := map[string]any{
			"dir":                 dir,
			"time_of_day":         "04:30",
			"include_assets":      true,
			"encrypt_enabled":     true,
			"encryption_password": "archive-secret",
		}
		for key, value := range extra {
			config[key] = value
		}
		return config
	}

	t.Run("schedule fields are stripped before the transport decode", func(t *testing.T) {
		var parsed localDestinationConfig
		if err := decodeDestinationConfigStrict(withSchedule(nil), "local", &parsed); err != nil {
			t.Fatalf("decodeDestinationConfigStrict() error = %v", err)
		}
		if parsed.Dir != dir {
			t.Fatalf("decoded dir = %q, want %q", parsed.Dir, dir)
		}
		if parsed.RetentionCount != 0 {
			t.Fatalf("decoded retention_count = %d, want 0 for an absent field", parsed.RetentionCount)
		}
	})

	rejected := []struct {
		name   string
		config map[string]any
	}{
		{name: "unknown field", config: withSchedule(map[string]any{"directory": dir})},
		{name: "typo on a schedule field", config: withSchedule(map[string]any{"time_of_days": "04:30"})},
		{name: "wrong type on a schedule field", config: withSchedule(map[string]any{"include_assets": "yes"})},
		{name: "fractional retention", config: withSchedule(map[string]any{"retention_count": 5.5})},
		{name: "integral float retention", config: withSchedule(map[string]any{"retention_count": float64(5)})},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			var parsed localDestinationConfig
			if err := decodeDestinationConfigStrict(tc.config, "local", &parsed); !errors.Is(err, ErrInvalidBackupDestinationConfig) {
				t.Fatalf("decodeDestinationConfigStrict() error = %v, want ErrInvalidBackupDestinationConfig", err)
			}
		})
	}
}

// TestDestinationConfigSchemasShareScheduleFields guards the merge in
// buildDestinationConfigSchemas: a plan field added for one transport but
// forgotten on another would be silently rejected there instead of honored.
func TestDestinationConfigSchemasShareScheduleFields(t *testing.T) {
	for destinationType, schema := range destinationConfigSchemas {
		for field, want := range scheduleConfigFields {
			got, ok := schema[field]
			if !ok || got != want {
				t.Fatalf("%s schema field %q = %v (present=%t), want %v", destinationType, field, got, ok, want)
			}
		}
		if _, ok := schema["retention_count"]; !ok {
			t.Fatalf("%s schema is missing retention_count", destinationType)
		}
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
