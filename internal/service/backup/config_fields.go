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

// destinationConfigSchemas is deliberately explicit. The target constructors
// receive map[string]any because configs are encrypted JSON, but they must not
// silently coerce an arbitrary value into a target-specific field. JSON
// numbers are decoded as json.Number by parseDestinationConfigMap; direct
// callers may provide Go integer types, but floating-point values are rejected.
var destinationConfigSchemas = map[string]map[string]destinationConfigFieldType{
	"local": {
		"dir":             destinationConfigString,
		"retention_count": destinationConfigInteger,
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
		"retention_count":   destinationConfigInteger,
	},
	"webdav": {
		"url":             destinationConfigString,
		"path":            destinationConfigString,
		"username":        destinationConfigString,
		"password":        destinationConfigString,
		"retention_count": destinationConfigInteger,
	},
}

// decodeDestinationConfigStrict validates a map against the destination's
// field schema, then decodes it into the target-specific struct. The second
// decode check rejects trailing JSON values, while DisallowUnknownFields keeps
// the per-type schema closed rather than silently ignoring typos.
func decodeDestinationConfigStrict(config map[string]any, destinationType string, target any) error {
	schema, ok := destinationConfigSchemas[destinationType]
	if !ok {
		return ErrInvalidBackupDestinationConfig
	}
	if config == nil {
		config = map[string]any{}
	}

	for key, raw := range config {
		fieldType, ok := schema[key]
		if !ok || !isDestinationConfigFieldType(raw, fieldType) {
			return ErrInvalidBackupDestinationConfig
		}
	}

	encoded, err := json.Marshal(config)
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
