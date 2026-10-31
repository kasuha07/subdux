package subscription

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

func TestCreateEnforcesPerUserSubscriptionLimit(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	other := model.User{Username: "limit-other", Email: "limit-other@example.com", Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}

	next := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]model.Subscription, MaxSubscriptionsPerUser-1)
	for i := range rows {
		rows[i] = model.Subscription{UserID: user.ID, Name: fmt.Sprintf("bulk-%d", i), Amount: 1, Currency: "USD", Status: "active", RenewalMode: "auto_renew", BillingType: "recurring", NextBillingDate: &next}
	}
	if err := db.CreateInBatches(rows, 500).Error; err != nil {
		t.Fatal(err)
	}
	service := NewService(db)

	if _, err := service.Create(user.ID, validSubscriptionInput(1)); err != nil {
		t.Fatalf("creating the last allowed subscription failed: %v", err)
	}
	if _, err := service.Create(user.ID, validSubscriptionInput(1)); !errors.Is(err, ErrSubscriptionLimitReached) {
		t.Fatalf("err = %v, want ErrSubscriptionLimitReached", err)
	}
	var count int64
	if err := db.Model(&model.Subscription{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != MaxSubscriptionsPerUser {
		t.Fatalf("subscriptions = %d, want %d", count, MaxSubscriptionsPerUser)
	}
	// The limit is per user.
	if _, err := service.Create(other.ID, validSubscriptionInput(1)); err != nil {
		t.Fatalf("another user's create failed: %v", err)
	}
}

func TestSubscriptionTextFieldsAreBounded(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	assertTooLong := func(t *testing.T, err error, field string) {
		t.Helper()
		var typed *serviceerr.Error
		if !errors.As(err, &typed) || typed.Code != "subscription_field_too_long" || typed.Params["field"] != field {
			t.Fatalf("err = %v, want subscription_field_too_long for %s", err, field)
		}
	}

	for field, mutate := range map[string]func(*CreateSubscriptionInput){
		"name":     func(in *CreateSubscriptionInput) { in.Name = strings.Repeat("n", MaxSubscriptionNameRunes+1) },
		"category": func(in *CreateSubscriptionInput) { in.Category = strings.Repeat("c", MaxSubscriptionCategoryRunes+1) },
		"notes":    func(in *CreateSubscriptionInput) { in.Notes = strings.Repeat("字", MaxSubscriptionNotesRunes+1) },
		"url": func(in *CreateSubscriptionInput) {
			in.URL = "https://example.com/" + strings.Repeat("u", MaxSubscriptionURLBytes)
		},
	} {
		input := validSubscriptionInput(1)
		mutate(&input)
		_, err := service.Create(user.ID, input)
		assertTooLong(t, err, field)
	}

	atLimit := validSubscriptionInput(1)
	atLimit.Name = strings.Repeat("名", MaxSubscriptionNameRunes)
	atLimit.Notes = strings.Repeat("字", MaxSubscriptionNotesRunes)
	created, err := service.Create(user.ID, atLimit)
	if err != nil {
		t.Fatalf("values at the limit were rejected: %v", err)
	}

	tooLongNotes := strings.Repeat("x", MaxSubscriptionNotesRunes+1)
	_, err = service.Update(user.ID, created.ID, UpdateSubscriptionInput{Revision: created.Revision, Notes: &tooLongNotes})
	assertTooLong(t, err, "notes")
	var saved model.Subscription
	if err := db.First(&saved, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Revision != created.Revision {
		t.Fatal("rejected update changed the subscription")
	}
}
