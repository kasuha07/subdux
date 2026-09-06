package subscription

import (
	"github.com/kasuha07/subdux/internal/model"
	"testing"
	"time"
)

func TestIntervalBoundsOnWritesAndLegacyReads(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	svc := NewService(db)
	count := 144115188075855872
	input := CreateSubscriptionInput{Name: "bad interval", Amount: 1, IntervalCount: &count, IntervalUnit: "day", NextBillingDate: "2020-01-01"}
	if _, err := svc.Create(user.ID, input); err != ErrIntervalCountTooHigh {
		t.Fatalf("create: %v", err)
	}
	count = 1
	sub, err := svc.Create(user.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	count = 144115188075855872
	if _, err := svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{IntervalCount: &count}); err != ErrIntervalCountTooHigh {
		t.Fatalf("update: %v", err)
	}
	if err := db.Model(&model.Subscription{}).Where("id = ?", sub.ID).Update("interval_count", count).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.List(user.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReconcileUserLifecycle(user.ID); err != nil {
		t.Fatal(err)
	}
}

func TestIntervalOverflowRejectedAndLegacyScheduleTerminates(t *testing.T) {
	anchor := time.Date(2020, 1, 31, 0, 0, 0, 0, time.UTC)
	from := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	for _, count := range []int{10001, 144115188075855872} {
		for _, unit := range []string{"day", "week", "month", "year"} {
			_, _, err := normalizeBillingDraft(billingDraft{NextBillingDate: &anchor, IntervalCount: &count, IntervalUnit: unit})
			if err != ErrIntervalCountTooHigh {
				t.Fatalf("%d %s: got %v", count, unit, err)
			}
			if got := nextIntervalOccurrence(anchor, from, count, unit); !got.IsZero() {
				t.Fatalf("unsafe interval returned %v", got)
			}
		}
	}
}

func TestIntervalJumpPreservesPreferredDate(t *testing.T) {
	for _, tc := range []struct {
		anchor, from, want, unit string
		count                    int
	}{
		{"2020-01-31", "2020-03-01", "2020-03-31", "month", 1},
		{"2020-02-29", "2024-02-28", "2024-02-29", "year", 1},
		{"0001-01-01", "9999-12-31", "9999-12-31", "day", 1},
		{"2020-01-01", "2020-01-09", "2020-01-15", "week", 1},
	} {
		anchor, _ := time.Parse("2006-01-02", tc.anchor)
		from, _ := time.Parse("2006-01-02", tc.from)
		if got := nextIntervalOccurrence(anchor, from, tc.count, tc.unit).Format("2006-01-02"); got != tc.want {
			t.Fatalf("%+v: got %s", tc, got)
		}
	}
}
