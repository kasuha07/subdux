package service

import (
	"testing"
	"time"

	"github.com/shiroha/subdux/internal/model"
)

func TestCountSubscriptionOccurrencesInRange(t *testing.T) {
	intPtr := func(value int) *int {
		return &value
	}
	timePtr := func(value string) *time.Time {
		date := mustDate(t, value)
		return &date
	}

	tests := []struct {
		name  string
		start string
		end   string
		sub   model.Subscription
		want  int
	}{
		{
			name:  "one-time due within range",
			start: "2026-02-01",
			end:   "2026-03-01",
			sub: model.Subscription{
				BillingType:     billingTypeOneTime,
				NextBillingDate: timePtr("2026-02-15"),
			},
			want: 1,
		},
		{
			name:  "one-time due before range",
			start: "2026-02-01",
			end:   "2026-03-01",
			sub: model.Subscription{
				BillingType:     billingTypeOneTime,
				NextBillingDate: timePtr("2026-01-31"),
			},
			want: 0,
		},
		{
			name:  "weekly recurring counts all remaining occurrences",
			start: "2026-02-01",
			end:   "2026-03-01",
			sub: model.Subscription{
				BillingType:     billingTypeRecurring,
				RecurrenceType:  recurrenceTypeInterval,
				IntervalCount:   intPtr(1),
				IntervalUnit:    intervalUnitWeek,
				NextBillingDate: timePtr("2026-02-10"),
			},
			want: 3,
		},
		{
			name:  "monthly-day recurring clamps to month end",
			start: "2026-02-01",
			end:   "2026-03-01",
			sub: model.Subscription{
				BillingType:     billingTypeRecurring,
				RecurrenceType:  recurrenceTypeMonthlyDate,
				MonthlyDay:      intPtr(31),
				NextBillingDate: timePtr("2026-01-31"),
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := mustDate(t, tt.start)
			end := mustDate(t, tt.end)
			if got := countSubscriptionOccurrencesInRange(tt.sub, start, end); got != tt.want {
				t.Fatalf("countSubscriptionOccurrencesInRange() = %d, want %d", got, tt.want)
			}
		})
	}
}
