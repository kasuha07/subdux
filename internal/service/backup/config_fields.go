package backup

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

type destinationConfigFieldType uint8

const (
	destinationConfigString destinationConfigFieldType = iota + 1
	destinationConfigBool
	destinationConfigInteger
)

// ErrInvalidBackupDestinationConfig is returned when a destination config does
// not match the schema for its destination type. Keeping this separate from
// the target-specific semantic errors lets callers distinguish malformed
// config fields from, for example, an invalid endpoint URL.
var ErrInvalidBackupDestinationConfig = serviceerr.New(
	serviceerr.KindInvalid,
	"invalid_backup_destination_config",
	"backup destination config does not match its destination type schema",
)

// scheduleConfigFields are the plan fields every destination carries regardless
// of where it stores archives: when it runs, what the archive contains, and how
// it is protected. They live in the same encrypted config blob as the transport
// settings so the archive password reuses the destination secret machinery
// (masking on read, preserve-on-empty and explicit clearing on update) instead
// of needing a second encrypted-secret path.
var scheduleConfigFields = map[string]destinationConfigFieldType{
	"time_of_day":         destinationConfigString,
	"include_assets":      destinationConfigBool,
	"encrypt_enabled":     destinationConfigBool,
	"encryption_password": destinationConfigString,
}

// destinationConfigSchemas is deliberately explicit. The target constructors
// receive map[string]any because configs are encrypted JSON, but they must not
// silently coerce an arbitrary value into a target-specific field. JSON
// numbers are decoded as json.Number by parseDestinationConfigMap; direct
// callers may provide Go integer types, but floating-point values are rejected.
//
// Each per-type schema below lists only its transport fields; buildDestination
// ConfigSchemas merges the shared retention and schedule fields into all of
// them so a new plan field cannot be added to one type and forgotten on another.
var destinationConfigSchemas = buildDestinationConfigSchemas(map[string]map[string]destinationConfigFieldType{
	"local": {
		"dir": destinationConfigString,
	},
	"s3": {
		"endpoint":          destinationConfigString,
		"use_ssl":           destinationConfigBool,
		"region":            destinationConfigString,
		"bucket":            destinationConfigString,
		"prefix":            destinationConfigString,
		"access_key_id":     destinationConfigString,
		"secret_access_key": destinationConfigString,
		"use_path_style":    destinationConfigBool,
		"skip_tls_verify":   destinationConfigBool,
	},
	"webdav": {
		"url":             destinationConfigString,
		"path":            destinationConfigString,
		"username":        destinationConfigString,
		"password":        destinationConfigString,
		"skip_tls_verify": destinationConfigBool,
	},
})

func buildDestinationConfigSchemas(
	transport map[string]map[string]destinationConfigFieldType,
) map[string]map[string]destinationConfigFieldType {
	schemas := make(map[string]map[string]destinationConfigFieldType, len(transport))
	for destinationType, fields := range transport {
		schema := make(map[string]destinationConfigFieldType, len(fields)+len(scheduleConfigFields)+1)
		for field, fieldType := range fields {
			schema[field] = fieldType
		}
		for field, fieldType := range scheduleConfigFields {
			schema[field] = fieldType
		}
		schema["retention_count"] = destinationConfigInteger
		schemas[destinationType] = schema
	}
	return schemas
}

// decodeDestinationConfigStrict validates a map against the destination's
// field schema, then decodes it into the target-specific struct. The second
// decode check rejects trailing JSON values, while DisallowUnknownFields keeps
// the per-type schema closed rather than silently ignoring typos.
//
// Schedule fields are validated here but withheld from the transport decode:
// they describe when and what to archive, not how to reach the storage target,
// so a target struct must not have to declare them merely to satisfy
// DisallowUnknownFields. parseScheduleConfig owns their interpretation.
func decodeDestinationConfigStrict(config map[string]any, destinationType string, target any) error {
	schema, ok := destinationConfigSchemas[destinationType]
	if !ok {
		return ErrInvalidBackupDestinationConfig
	}
	if config == nil {
		config = map[string]any{}
	}

	transport := make(map[string]any, len(config))
	for key, raw := range config {
		fieldType, ok := schema[key]
		if !ok || !isDestinationConfigFieldType(raw, fieldType) {
			return ErrInvalidBackupDestinationConfig
		}
		if _, isSchedule := scheduleConfigFields[key]; isSchedule {
			continue
		}
		transport[key] = raw
	}

	encoded, err := json.Marshal(transport)
	if err != nil {
		return ErrInvalidBackupDestinationConfig
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return ErrInvalidBackupDestinationConfig
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return ErrInvalidBackupDestinationConfig
	}
	return nil
}

func isDestinationConfigFieldType(raw any, fieldType destinationConfigFieldType) bool {
	switch fieldType {
	case destinationConfigString:
		_, ok := raw.(string)
		return ok
	case destinationConfigBool:
		_, ok := raw.(bool)
		return ok
	case destinationConfigInteger:
		switch raw.(type) {
		case json.Number, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

// destinationSchedule is the plan half of a destination: when its archive is
// taken and what that archive contains. Two destinations whose schedules agree
// on everything but TimeOfDay can still share one archive, which is why the
// grouping key in the scheduler is built from the content fields alone.
type destinationSchedule struct {
	TimeOfDay       string
	IncludeAssets   bool
	EncryptEnabled  bool
	EncryptPassword string
}

// archiveSpec is the subset of a schedule that determines the archive bytes.
// It is comparable so the scheduler can use it directly as a map key when
// grouping due destinations onto a shared archive.
type archiveSpec struct {
	IncludeAssets bool
	Password      string
}

func (s destinationSchedule) archiveSpec() archiveSpec {
	spec := archiveSpec{IncludeAssets: s.IncludeAssets}
	if s.EncryptEnabled {
		spec.Password = s.EncryptPassword
	}
	return spec
}

// parseScheduleConfig reads the schedule fields out of an already type-checked
// config map. An absent time_of_day means the default rather than "never run":
// a destination is a backup plan, so it always has a firing time.
func parseScheduleConfig(config map[string]any) (destinationSchedule, error) {
	schedule := destinationSchedule{TimeOfDay: defaultBackupTimeOfDay}

	if raw, ok := config["time_of_day"].(string); ok {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			schedule.TimeOfDay = trimmed
		}
	}
	if !backupTimeOfDayPattern.MatchString(schedule.TimeOfDay) {
		return destinationSchedule{}, ErrInvalidBackupTimeOfDay
	}

	schedule.IncludeAssets, _ = config["include_assets"].(bool)
	schedule.EncryptEnabled, _ = config["encrypt_enabled"].(bool)
	if raw, ok := config["encryption_password"].(string); ok {
		schedule.EncryptPassword = raw
	}
	if schedule.EncryptEnabled && strings.TrimSpace(schedule.EncryptPassword) == "" {
		return destinationSchedule{}, ErrBackupEncryptionPasswordRequired
	}

	return schedule, nil
}

// destinationScheduleFromStored decrypts a persisted destination config and
// returns its schedule. Runtime callers go through this rather than through the
// masked DestinationView, whose encryption password is deliberately blanked.
func destinationScheduleFromStored(config string) (destinationSchedule, error) {
	plain, err := decryptDestinationConfig(config)
	if err != nil {
		return destinationSchedule{}, ErrInvalidBackupDestinationConfig
	}
	parsed, err := parseDestinationConfigMap(plain)
	if err != nil {
		return destinationSchedule{}, err
	}
	return parseScheduleConfig(parsed)
}

// retentionCountFromConfig applies the shared retention default and range
// invariant after the destination-specific strict decode has proved that the
// field is an integer. A present but invalid value is an error; it is never
// converted to or replaced with the default.
func retentionCountFromConfig(config map[string]any, parsed int) (int, error) {
	if _, present := config["retention_count"]; !present {
		return defaultRetentionCount, nil
	}
	if parsed < minBackupRetention || parsed > maxBackupRetention {
		return 0, ErrInvalidBackupRetentionCount
	}
	return parsed, nil
}

// parseDestinationConfigMap parses one JSON object while preserving the JSON
// number representation. That distinction is required to reject 5.5 (and
// 5.0) when a schema says that a field is an integer.
func parseDestinationConfigMap(raw string) (map[string]any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "{}"
	}

	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	var parsed map[string]any
	if err := decoder.Decode(&parsed); err != nil {
		return nil, serviceerr.New(serviceerr.KindInvalid, "backup_destination_config_must_be_valid_json", "backup destination config must be valid JSON")
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, serviceerr.New(serviceerr.KindInvalid, "backup_destination_config_must_be_valid_json", "backup destination config must be valid JSON")
	}
	if parsed == nil {
		return nil, ErrInvalidBackupDestinationConfig
	}
	return parsed, nil
}
