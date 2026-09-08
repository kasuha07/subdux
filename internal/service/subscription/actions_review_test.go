package subscription

import (
	"errors"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"gorm.io/gorm"
)

func createActionReviewFixture(t *testing.T, mode string) (*Service, model.User, *model.Subscription) {
	t.Helper()
	t.Cleanup(pkg.SetNowForTest(mustDate(t, "2026-03-01")))
	db := newTestDB(t)
	user := createTestUser(t, db)
	svc := NewService(db)
	monthly := 1
	sub, err := svc.Create(user.ID, CreateSubscriptionInput{
		Name: "Review fixture", Amount: 10, Currency: "USD", Status: subscriptionStatusActive,
		RenewalMode: mode, BillingType: billingTypeRecurring, RecurrenceType: recurrenceTypeInterval,
		IntervalCount: &monthly, IntervalUnit: intervalUnitMonth, NextBillingDate: "2026-03-05",
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, user, sub
}

func assertActionReviewTypes(t *testing.T, svc *Service, userID uint, want ...string) {
	t.Helper()
	// Construct a fresh service to prove that decisions survive page/service reloads.
	center, err := NewService(svc.DB).GetActionCenter(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(center.Items) != len(want) || center.Counts.Total != len(want) || center.Counts.Snoozed != 0 {
		t.Fatalf("center = %+v, want types %v with no temporary snoozes", center, want)
	}
	for _, typ := range want {
		found := false
		for _, item := range center.Items {
			found = found || item.Type == typ
		}
		if !found {
			t.Fatalf("missing %s in %+v", typ, center.Items)
		}
	}
}

func TestActionReviewCompletesCancelAndKeepDecisions(t *testing.T) {
	for _, tc := range []struct{ before, after, original string }{
		{renewalModeAutoRenew, renewalModeCancelAtPeriodEnd, actionTypeUpcomingRenewal},
		{renewalModeManualRenew, renewalModeCancelAtPeriodEnd, actionTypeManualRenewalDue},
		{renewalModeCancelAtPeriodEnd, renewalModeAutoRenew, actionTypeEndingSoon},
	} {
		t.Run(tc.before+" to "+tc.after, func(t *testing.T) {
			svc, user, sub := createActionReviewFixture(t, tc.before)
			assertActionReviewTypes(t, svc, user.ID, tc.original)
			if _, err := svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{RenewalMode: &tc.after}); err != nil {
				t.Fatal(err)
			}
			assertActionReviewTypes(t, svc, user.ID)
			name := "Renamed"
			if _, err := svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{Name: &name}); err != nil {
				t.Fatal(err)
			}
			assertActionReviewTypes(t, svc, user.ID)
		})
	}
}

func TestActionReviewPreservesIndependentIssuesAndNewSameDayPriceIncrease(t *testing.T) {
	svc, user, sub := createActionReviewFixture(t, renewalModeAutoRenew)
	price := 20.0
	if _, err := svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{Amount: &price}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DB.Create(&model.NotificationLog{
		UserID: user.ID, SubscriptionID: sub.ID, ChannelType: "webhook",
		NotifyDate: mustDate(t, "2026-03-04"), Status: notificationLogStatusFailed,
		Error: "timeout", SentAt: pkg.NowUTC(),
	}).Error; err != nil {
		t.Fatal(err)
	}
	mode := renewalModeCancelAtPeriodEnd
	if _, err := svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{RenewalMode: &mode}); err != nil {
		t.Fatal(err)
	}
	assertActionReviewTypes(t, svc, user.ID, actionTypeNotificationFailed)
	price = 30
	if _, err := svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{Amount: &price}); err != nil {
		t.Fatal(err)
	}
	assertActionReviewTypes(t, svc, user.ID, actionTypeNotificationFailed, actionTypePriceIncrease)
}

func TestActionReviewInvalidatesOnScheduleChanges(t *testing.T) {
	for _, field := range []string{"next_billing_date", "ends_at", "renewal_mode"} {
		t.Run(field, func(t *testing.T) {
			svc, user, sub := createActionReviewFixture(t, renewalModeAutoRenew)
			mode := renewalModeCancelAtPeriodEnd
			if _, err := svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{RenewalMode: &mode}); err != nil {
				t.Fatal(err)
			}
			date := "2026-03-10"
			input := UpdateSubscriptionInput{}
			want := actionTypeEndingSoon
			switch field {
			case "next_billing_date":
				input.NextBillingDate = &date
			case "ends_at":
				// The service normalizes this back to the unchanged billing date.
				input.EndsAt = &date
				want = ""
			case "renewal_mode":
				mode = renewalModeManualRenew
				input.RenewalMode = &mode
				want = actionTypeManualRenewalDue
			}
			if _, err := svc.Update(user.ID, sub.ID, input); err != nil {
				t.Fatal(err)
			}
			if want == "" {
				assertActionReviewTypes(t, svc, user.ID)
			} else {
				assertActionReviewTypes(t, svc, user.ID, want)
			}
		})
	}
}

func TestActionReviewDoesNotHideNextBillingPeriod(t *testing.T) {
	svc, user, sub := createActionReviewFixture(t, renewalModeCancelAtPeriodEnd)
	mode := renewalModeAutoRenew
	if _, err := svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{RenewalMode: &mode}); err != nil {
		t.Fatal(err)
	}
	assertActionReviewTypes(t, svc, user.ID)
	restore := pkg.SetNowForTest(mustDate(t, "2026-03-06"))
	defer restore()
	assertActionReviewTypes(t, svc, user.ID, actionTypeUpcomingRenewal)
}

func TestActionReviewFailureRollsBackSubscriptionDecision(t *testing.T) {
	svc, user, sub := createActionReviewFixture(t, renewalModeAutoRenew)
	failure := errors.New("review storage unavailable")
	if err := svc.DB.Callback().Create().Before("gorm:create").Register("test:reject_review", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "subscription_action_snoozes" {
			tx.AddError(failure)
		}
	}); err != nil {
		t.Fatal(err)
	}
	mode := renewalModeCancelAtPeriodEnd
	if _, err := svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{RenewalMode: &mode}); !errors.Is(err, failure) {
		t.Fatalf("Update error = %v, want review storage failure", err)
	}
	var stored model.Subscription
	if err := svc.DB.First(&stored, sub.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.RenewalMode != sub.RenewalMode || stored.Revision != sub.Revision {
		t.Fatalf("failed decision persisted: %+v", stored)
	}
	assertActionReviewTypes(t, svc, user.ID, actionTypeUpcomingRenewal)
}

func TestActionReviewDoesNotReapplyAfterDateIsChangedBack(t *testing.T) {
	svc, user, sub := createActionReviewFixture(t, renewalModeCancelAtPeriodEnd)
	mode := renewalModeAutoRenew
	if _, err := svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{RenewalMode: &mode}); err != nil {
		t.Fatal(err)
	}
	for _, date := range []string{"2026-03-10", "2026-03-05"} {
		if _, err := svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{NextBillingDate: &date}); err != nil {
			t.Fatal(err)
		}
		assertActionReviewTypes(t, svc, user.ID, actionTypeUpcomingRenewal)
	}
}

func TestActionReviewRejectsStaleAndForeignDecisions(t *testing.T) {
	svc, user, sub := createActionReviewFixture(t, renewalModeAutoRenew)
	name := "New revision"
	if _, err := svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{Name: &name}); err != nil {
		t.Fatal(err)
	}
	mode := renewalModeCancelAtPeriodEnd
	for _, input := range []struct {
		userID   uint
		revision uint64
	}{
		{user.ID, sub.Revision}, {user.ID + 1, 0},
	} {
		if _, err := svc.Update(input.userID, sub.ID, UpdateSubscriptionInput{Revision: input.revision, RenewalMode: &mode}); err == nil {
			t.Fatal("invalid decision succeeded")
		}
	}
	assertActionReviewTypes(t, svc, user.ID, actionTypeUpcomingRenewal)
	var count int64
	if err := svc.DB.Model(&model.SubscriptionActionSnooze{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("invalid decisions wrote %d review markers: %v", count, err)
	}
}
