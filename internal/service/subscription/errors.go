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
	ErrSubscriptionNotFound = serviceerr.New(serviceerr.KindNotFound, "subscription_not_found", "subscription not found")

	ErrCategoryNotFound      = serviceerr.New(serviceerr.KindInvalid, "category_not_found", "category not found")
	ErrPaymentMethodNotFound = serviceerr.New(serviceerr.KindInvalid, "payment_method_not_found", "payment method not found")

	ErrInvalidSubscriptionURL = serviceerr.New(serviceerr.KindInvalid, "subscription_url_must_be_a_valid_http_or_https_url", "subscription url must be a valid http or https URL")

	// Amount and conversion validation.
	ErrAmountMustBeFinite      = serviceerr.New(serviceerr.KindInvalid, "amount_must_be_finite", "amount must be finite")
	ErrAmountMustNotBeNegative = serviceerr.New(serviceerr.KindInvalid, "amount_must_not_be_negative", "amount must not be negative")
	ErrAmountTooLarge          = serviceerr.New(serviceerr.KindInvalid, "amount_too_large", "amount is too large")
	ErrExchangeRateUnavailable = serviceerr.New(serviceerr.KindUnavailable, "exchange_rate_unavailable", "exchange rate is unavailable")

	// Billing / recurrence validation.
	ErrNextBillingDateRequiredRecurring = serviceerr.New(serviceerr.KindInvalid, "next_billing_date_is_required_for_recurring_subscriptions", "next_billing_date is required for recurring subscriptions")
	ErrIntervalCountTooLow              = serviceerr.New(serviceerr.KindInvalid, "interval_count_must_be_at_least_1_for_interval_recurrence", "interval_count must be at least 1 for interval recurrence")
	ErrIntervalUnitInvalid              = serviceerr.New(serviceerr.KindInvalid, "interval_unit_must_be_one_of_day_week_month_year", "interval_unit must be one of: day, week, month, year")
	ErrMonthlyDayInvalid                = serviceerr.New(serviceerr.KindInvalid, "monthly_day_must_be_between_1_and_31_for_monthly_date_recurrence", "monthly_day must be between 1 and 31 for monthly date recurrence")
	ErrYearlyMonthInvalid               = serviceerr.New(serviceerr.KindInvalid, "yearly_month_must_be_between_1_and_12_for_yearly_date_recurrence", "yearly_month must be between 1 and 12 for yearly date recurrence")
	ErrYearlyDayInvalid                 = serviceerr.New(serviceerr.KindInvalid, "yearly_day_must_be_between_1_and_31_for_yearly_date_recurrence", "yearly_day must be between 1 and 31 for yearly date recurrence")
	ErrRecurrenceTypeInvalid            = serviceerr.New(serviceerr.KindInvalid, "recurrence_type_must_be_one_of_interval_monthly_date_yearly_date", "recurrence_type must be one of: interval, monthly_date, yearly_date")
	ErrBillingTypeMustBeRecurring       = serviceerr.New(serviceerr.KindInvalid, "billing_type_must_be_recurring", "billing_type must be recurring")
	ErrInvalidDateFormat                = serviceerr.New(serviceerr.KindInvalid, "invalid_date_format_expected_yyyy_mm_dd", "invalid date format, expected YYYY-MM-DD")

	// Lifecycle validation.
	ErrStatusInvalid                      = serviceerr.New(serviceerr.KindInvalid, "status_must_be_one_of_active_ended", "status must be one of: active, ended")
	ErrRenewalModeInvalid                 = serviceerr.New(serviceerr.KindInvalid, "renewal_mode_must_be_one_of_auto_renew_manual_renew_cancel_at_period_end", "renewal_mode must be one of: auto_renew, manual_renew, cancel_at_period_end")
	ErrNextBillingDateRequiredCancelAtEnd = serviceerr.New(serviceerr.KindInvalid, "next_billing_date_is_required_for_cancel_at_period_end_subscriptions", "next_billing_date is required for cancel_at_period_end subscriptions")

	// Manual renewal validation.
	ErrOnlyActiveCanRenew          = serviceerr.New(serviceerr.KindInvalid, "only_active_subscriptions_can_be_marked_as_renewed", "only active subscriptions can be marked as renewed")
	ErrOnlyManualRenewCanRenew     = serviceerr.New(serviceerr.KindInvalid, "only_manual_renew_subscriptions_can_be_marked_as_renewed", "only manual_renew subscriptions can be marked as renewed")
	ErrOnlyRecurringCanRenew       = serviceerr.New(serviceerr.KindInvalid, "only_recurring_subscriptions_can_be_marked_as_renewed", "only recurring subscriptions can be marked as renewed")
	ErrNextBillingDateRequiredMark = serviceerr.New(serviceerr.KindInvalid, "next_billing_date_is_required_to_mark_subscription_as_renewed", "next_billing_date is required to mark subscription as renewed")
	ErrRecurrenceSettingsInvalid   = serviceerr.New(serviceerr.KindInvalid, "subscription_recurrence_settings_are_invalid", "subscription recurrence settings are invalid")
	ErrNextBillingDateCalcFailed   = serviceerr.New(serviceerr.KindInvalid, "failed_to_calculate_next_billing_date", "failed to calculate next billing date")

	// Batch operations.
	ErrBatchInternal = serviceerr.New(serviceerr.KindInternal, "batch_internal_error", "internal error while processing a subscription in the batch")

	// Action-center / snooze validation.
	ErrInvalidActionKey       = serviceerr.New(serviceerr.KindInvalid, "invalid_action_key", "invalid action key")
	ErrSnoozeDateRequired     = serviceerr.New(serviceerr.KindInvalid, "snooze_date_is_required", "snooze date is required")
	ErrSnoozeDateMustBeFuture = serviceerr.New(serviceerr.KindInvalid, "snooze_date_must_be_in_the_future", "snooze date must be in the future")
)
