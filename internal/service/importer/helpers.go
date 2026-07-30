package importer

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/kasuha07/subdux/internal/service/money"
)

var (
	errImportedAmountInvalid   = errors.New("invalid amount")
	errImportedAmountNonFinite = errors.New("amount must be finite")
	errImportedAmountNegative  = errors.New("amount must not be negative")
	errImportedAmountTooLarge  = errors.New("amount is too large")
)

// importAmountKey renders an amount for an in-batch dedup key. Both importers
// use it so "same amount" means the same thing in each: the exact shortest
// round-trip form, with no six-decimal truncation and no exponent notation.
func importAmountKey(amount float64) string {
	return strconv.FormatFloat(amount, 'f', -1, 64)
}

// normalizeImportedAmount validates an imported amount without repairing an
// invalid value into zero. Import files bypass the HTTP boundary, so this
// check must run before preview, deduplication, and persistence.
func normalizeImportedAmount(amount float64) (float64, error) {
	switch money.ValidateAmount(amount) {
	case money.AmountValid:
		return amount, nil
	case money.AmountNegative:
		return 0, errImportedAmountNegative
	case money.AmountAboveMaximum:
		return 0, errImportedAmountTooLarge
	default:
		return 0, errImportedAmountNonFinite
	}
}

func invalidImportedAmountError(price string) error {
	return fmt.Errorf("%w: %q", errImportedAmountInvalid, price)
}

func recordInvalidImportedAmount(
	subscriptions *[]PreviewSubscriptionChange,
	result *ImportResult,
	confirm bool,
	change PreviewSubscriptionChange,
	amountErr error,
) {
	change.Skipped = true
	change.SkipReason = "invalid_amount"
	*subscriptions = append(*subscriptions, change)
	if !confirm {
		return
	}

	result.Errors = append(result.Errors, fmt.Sprintf(
		"skipped subscription %q with invalid amount: %v",
		change.Name,
		amountErr,
	))
	result.Skipped++
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
