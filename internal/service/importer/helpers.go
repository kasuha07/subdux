package importer

import "time"

func cloneImportedInt(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func normalizeImportedDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	return &normalized
}
