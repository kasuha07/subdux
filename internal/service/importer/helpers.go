package importer

import (
	"strconv"
	"time"

	"github.com/kasuha07/subdux/internal/pkg/money"
)

// importAmountKey renders an amount for an in-batch dedup key. Both importers
// use it so "same amount" means the same thing in each: the exact shortest
// round-trip form, with no six-decimal truncation and no exponent notation.
func importAmountKey(amount float64) string {
	return strconv.FormatFloat(amount, 'f', -1, 64)
}

// normalizeImportedAmount makes an imported amount storable: non-finite,
// negative, and above-money.MaxAmount values all collapse to 0. Non-finite
// and negative collapse because that's what the SQLite integrity-hardening
// migration does for pre-existing rows and what the amount >= 0 check
// constraint requires. The upper bound is the same one the API enforces via
// contract.MaxSubscriptionAmount (now money.MaxAmount) — imported files are
// hand-editable and bypass that API check, so this is where the ceiling gets
// applied for both importers; without it, an amount above ~9e13 would defeat
// money.Round's ability to quantize it at all.
func normalizeImportedAmount(amount float64) float64 {
	if !money.IsFinite(amount) || amount < 0 || amount > money.MaxAmount {
		return 0
	}
	return amount
}

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
