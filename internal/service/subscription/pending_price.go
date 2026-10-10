package subscription

import (
	"time"

	"github.com/kasuha07/subdux/internal/model"
)

// A pending price is a single scheduled price change: every charge dated on or
// after PendingFrom is billed at PendingAmount instead of Amount. It models
// introductory pricing (Amount is the discounted price) and free trials (Amount
// is zero) without a separate lifecycle state: once PendingFrom is reached the
// scheduled price becomes the subscription's Amount and the pair is cleared.

func subscriptionHasPendingPrice(sub model.Subscription) bool {
	return sub.PendingAmount != nil && sub.PendingFrom != nil
}

// pendingPriceAppliesOn reports whether a charge dated date is billed at the
// scheduled pending price rather than the current amount.
func pendingPriceAppliesOn(sub model.Subscription, date time.Time) bool {
	return subscriptionHasPendingPrice(sub) &&
		!normalizeDateUTC(date).Before(normalizeDateUTC(*sub.PendingFrom))
}

// subscriptionChargeAmountOn returns the amount, in the subscription currency,
// charged on date.
func subscriptionChargeAmountOn(sub model.Subscription, date time.Time) float64 {
	if pendingPriceAppliesOn(sub, date) {
		return *sub.PendingAmount
	}
	return sub.Amount
}

// applyDuePendingPrice switches a subscription to its scheduled price once
// referenceDate reaches PendingFrom. Only subscriptions that still bill reach
// the scheduled price: a canceling or ended subscription keeps its pending pair
// untouched so reactivating it restores the schedule. It mutates sub in memory
// and reports whether anything changed.
func applyDuePendingPrice(sub *model.Subscription, referenceDate time.Time) bool {
	if sub == nil || !subscriptionHasPendingPrice(*sub) || !subscriptionHasFutureCharge(*sub) {
		return false
	}
	if normalizeDateUTC(referenceDate).Before(normalizeDateUTC(*sub.PendingFrom)) {
		return false
	}
	sub.Amount = *sub.PendingAmount
	sub.PendingAmount = nil
	sub.PendingFrom = nil
	return true
}

// normalizePendingPrice validates a scheduled price change. Both values must be
// set together, the amount follows the regular amount rules, and the change
// must take effect after today: a change effective today or earlier is simply
// the current amount.
func normalizePendingPrice(amount *float64, from *time.Time, currency string, schedule billingDraft, now time.Time) (*float64, *time.Time, error) {
	if amount == nil && from == nil {
		return nil, nil, nil
	}
	if amount == nil || from == nil {
		return nil, nil, ErrPendingPriceIncomplete
	}
	if err := ValidateBillingAmount(*amount, currency, schedule); err != nil {
		return nil, nil, err
	}
	effective := normalizeDateUTC(*from)
	if !effective.After(normalizeDateUTC(now)) {
		return nil, nil, ErrPendingFromMustBeFuture
	}
	pendingAmount := *amount
	return &pendingAmount, &effective, nil
}

// convertedChargeAmounts holds a subscription's current and scheduled prices
// converted once into a target currency, so per-charge sums do not repeat the
// conversion for every occurrence.
type convertedChargeAmounts struct {
	sub     model.Subscription
	current float64
	pending float64
}

func convertSubscriptionChargeAmounts(sub model.Subscription, targetCurrency string, converter CurrencyConverter) (convertedChargeAmounts, error) {
	current, err := convertSubscriptionAmount(sub, targetCurrency, converter)
	if err != nil {
		return convertedChargeAmounts{}, err
	}
	amounts := convertedChargeAmounts{sub: sub, current: current}
	if subscriptionHasPendingPrice(sub) {
		pendingSub := sub
		pendingSub.Amount = *sub.PendingAmount
		amounts.pending, err = convertSubscriptionAmount(pendingSub, targetCurrency, converter)
		if err != nil {
			return convertedChargeAmounts{}, err
		}
	}
	return amounts, nil
}

func (a convertedChargeAmounts) on(date time.Time) float64 {
	if pendingPriceAppliesOn(a.sub, date) {
		return a.pending
	}
	return a.current
}

// total sums the converted charge for each date.
func (a convertedChargeAmounts) total(dates []time.Time, targetCurrency string) (float64, error) {
	currentCount, pendingCount := int64(0), int64(0)
	for _, date := range dates {
		if pendingPriceAppliesOn(a.sub, date) {
			pendingCount++
		} else {
			currentCount++
		}
	}
	total := 0.0
	for _, part := range []struct {
		amount float64
		count  int64
	}{{a.current, currentCount}, {a.pending, pendingCount}} {
		if part.count == 0 {
			continue
		}
		due, err := multiplyAggregateAmount(part.amount, part.count, targetCurrency)
		if err != nil {
			return 0, err
		}
		total, err = addAggregateAmounts(total, due, targetCurrency)
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}

// SubscriptionChargeAmountOn returns the amount charged on date, honoring a
// scheduled price change.
func SubscriptionChargeAmountOn(sub model.Subscription, date time.Time) float64 {
	return subscriptionChargeAmountOn(sub, date)
}
