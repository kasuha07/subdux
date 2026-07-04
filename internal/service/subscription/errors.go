package subscription

import "github.com/kasuha07/subdux/internal/service/serviceerr"

// Client-facing subscription errors. Each carries a serviceerr.Kind so the
// transport layer maps it to an HTTP status in one place, replacing the former
// message-substring classification. Messages are preserved verbatim from the
// inline errors they supersede so the external contract does not shift.
//
// Status normalization (per refactor decision): a missing addressed
// subscription is always KindNotFound (404), including on the icon-upload path
// where it previously surfaced as 400. Referenced foreign keys that do not
// exist (category/payment method supplied in a create/update body) remain
// KindInvalid (400): they are input-validation failures, not a missing
// addressed resource.
var (
	ErrSubscriptionNotFound = serviceerr.New(serviceerr.KindNotFound, "subscription not found")

	ErrCategoryNotFound      = serviceerr.New(serviceerr.KindInvalid, "category not found")
	ErrPaymentMethodNotFound = serviceerr.New(serviceerr.KindInvalid, "payment method not found")

	ErrInvalidSubscriptionURL = serviceerr.New(serviceerr.KindInvalid, "subscription url must be a valid http or https URL")

	// Billing / recurrence validation.
	ErrNextBillingDateRequiredRecurring = serviceerr.New(serviceerr.KindInvalid, "next_billing_date is required for recurring subscriptions")
	ErrIntervalCountTooLow              = serviceerr.New(serviceerr.KindInvalid, "interval_count must be at least 1 for interval recurrence")
	ErrIntervalUnitInvalid              = serviceerr.New(serviceerr.KindInvalid, "interval_unit must be one of: day, week, month, year")
	ErrMonthlyDayInvalid                = serviceerr.New(serviceerr.KindInvalid, "monthly_day must be between 1 and 31 for monthly date recurrence")
	ErrYearlyMonthInvalid               = serviceerr.New(serviceerr.KindInvalid, "yearly_month must be between 1 and 12 for yearly date recurrence")
	ErrYearlyDayInvalid                 = serviceerr.New(serviceerr.KindInvalid, "yearly_day must be between 1 and 31 for yearly date recurrence")
	ErrRecurrenceTypeInvalid            = serviceerr.New(serviceerr.KindInvalid, "recurrence_type must be one of: interval, monthly_date, yearly_date")
	ErrBillingTypeMustBeRecurring       = serviceerr.New(serviceerr.KindInvalid, "billing_type must be recurring")
	ErrInvalidDateFormat                = serviceerr.New(serviceerr.KindInvalid, "invalid date format, expected YYYY-MM-DD")

	// Lifecycle validation.
	ErrStatusInvalid                      = serviceerr.New(serviceerr.KindInvalid, "status must be one of: active, ended")
	ErrRenewalModeInvalid                 = serviceerr.New(serviceerr.KindInvalid, "renewal_mode must be one of: auto_renew, manual_renew, cancel_at_period_end")
	ErrNextBillingDateRequiredCancelAtEnd = serviceerr.New(serviceerr.KindInvalid, "next_billing_date is required for cancel_at_period_end subscriptions")

	// Manual renewal validation.
	ErrOnlyActiveCanRenew          = serviceerr.New(serviceerr.KindInvalid, "only active subscriptions can be marked as renewed")
	ErrOnlyManualRenewCanRenew     = serviceerr.New(serviceerr.KindInvalid, "only manual_renew subscriptions can be marked as renewed")
	ErrOnlyRecurringCanRenew       = serviceerr.New(serviceerr.KindInvalid, "only recurring subscriptions can be marked as renewed")
	ErrNextBillingDateRequiredMark = serviceerr.New(serviceerr.KindInvalid, "next_billing_date is required to mark subscription as renewed")
	ErrRecurrenceSettingsInvalid   = serviceerr.New(serviceerr.KindInvalid, "subscription recurrence settings are invalid")
	ErrNextBillingDateCalcFailed   = serviceerr.New(serviceerr.KindInvalid, "failed to calculate next billing date")

	// Action-center / snooze validation.
	ErrInvalidActionKey       = serviceerr.New(serviceerr.KindInvalid, "invalid action key")
	ErrSnoozeDateRequired     = serviceerr.New(serviceerr.KindInvalid, "snooze date is required")
	ErrSnoozeDateMustBeFuture = serviceerr.New(serviceerr.KindInvalid, "snooze date must be in the future")
)
