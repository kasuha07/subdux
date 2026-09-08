package serviceutil

import (
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

var ErrRevisionConflict = serviceerr.New(serviceerr.KindConflict, "revision_conflict", "record changed; reload before trying again")
var ErrRevisionRequired = serviceerr.New(serviceerr.KindInvalid, "revision_required", "a positive revision is required")

func DeleteRevision(query *gorm.DB, model interface{}, revision uint64) error {
	result := query.Where("revision = ?", revision).Delete(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrRevisionConflict
	}
	return nil
}

// CheckRevision permits zero only for internal callers operating on a freshly
// loaded snapshot. REST and MCP require a client revision at their boundaries.
// Every write still compares the snapshot used to compute its changes.
func CheckRevision(expected, current uint64) error {
	if expected != 0 && expected != current {
		return ErrRevisionConflict
	}
	return nil
}

// UpdateRevision must be given an ownership-scoped query and the revision of
// the snapshot used to validate and compute updates. Never use Save here: its
// fallback insert could resurrect a concurrently deleted record.
func UpdateRevision(query *gorm.DB, revision uint64, updates map[string]interface{}) error {
	updates["revision"] = gorm.Expr("revision + 1")
	result := query.Where("revision = ?", revision).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrRevisionConflict
	}
	return nil
}
