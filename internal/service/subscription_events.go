package service

import (
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

const (
	subscriptionEventCreated       = "created"
	subscriptionEventUpdated       = "updated"
	subscriptionEventManualRenewed = "manual_renewed"
	subscriptionEventDeleted       = "deleted"
	subscriptionEventSystemChange  = "system_change"
)

type subscriptionEventSnapshot struct {
	Amount            float64
	Currency          string
	MonthlyAmount     float64
	NextBillingDate   *time.Time
	Status            string
	RenewalMode       string
	CategoryID        *uint
	CategoryName      string
	PaymentMethodID   *uint
	PaymentMethodName string
}

func (s *SubscriptionService) recordSubscriptionCreated(userID uint, sub model.Subscription) error {
	snapshot, err := s.buildSubscriptionEventSnapshot(userID, sub)
	if err != nil {
		return err
	}

	subscriptionID := sub.ID
	actorUserID := userID
	return s.DB.Create(&model.SubscriptionEvent{
		UserID:               userID,
		ActorUserID:          &actorUserID,
		SubscriptionID:       &subscriptionID,
		SubscriptionName:     sub.Name,
		Type:                 subscriptionEventCreated,
		ChangedFields:        encodeSubscriptionEventFields([]string{"created"}),
		NewAmount:            float64Ptr(snapshot.Amount),
		NewMonthlyAmount:     float64Ptr(snapshot.MonthlyAmount),
		NewCurrency:          snapshot.Currency,
		NewNextBillingDate:   copyTimePointer(snapshot.NextBillingDate),
		NewStatus:            snapshot.Status,
		NewRenewalMode:       snapshot.RenewalMode,
		NewCategoryID:        copyUintPointer(snapshot.CategoryID),
		NewCategoryName:      snapshot.CategoryName,
		NewPaymentMethodID:   copyUintPointer(snapshot.PaymentMethodID),
		NewPaymentMethodName: snapshot.PaymentMethodName,
		CreatedAt:            pkg.NowUTC(),
	}).Error
}

func (s *SubscriptionService) recordSubscriptionDeleted(userID uint, sub model.Subscription) error {
	snapshot, err := s.buildSubscriptionEventSnapshot(userID, sub)
	if err != nil {
		return err
	}

	subscriptionID := sub.ID
	actorUserID := userID
	return s.DB.Create(&model.SubscriptionEvent{
		UserID:                    userID,
		ActorUserID:               &actorUserID,
		SubscriptionID:            &subscriptionID,
		SubscriptionName:          sub.Name,
		Type:                      subscriptionEventDeleted,
		ChangedFields:             encodeSubscriptionEventFields([]string{"deleted"}),
		PreviousAmount:            float64Ptr(snapshot.Amount),
		PreviousMonthlyAmount:     float64Ptr(snapshot.MonthlyAmount),
		PreviousCurrency:          snapshot.Currency,
		PreviousNextBillingDate:   copyTimePointer(snapshot.NextBillingDate),
		PreviousStatus:            snapshot.Status,
		PreviousRenewalMode:       snapshot.RenewalMode,
		PreviousCategoryID:        copyUintPointer(snapshot.CategoryID),
		PreviousCategoryName:      snapshot.CategoryName,
		PreviousPaymentMethodID:   copyUintPointer(snapshot.PaymentMethodID),
		PreviousPaymentMethodName: snapshot.PaymentMethodName,
		CreatedAt:                 pkg.NowUTC(),
	}).Error
}

func (s *SubscriptionService) recordSubscriptionChanged(userID uint, before, after model.Subscription, eventType string) error {
	beforeSnapshot, err := s.buildSubscriptionEventSnapshot(userID, before)
	if err != nil {
		return err
	}
	afterSnapshot, err := s.buildSubscriptionEventSnapshot(userID, after)
	if err != nil {
		return err
	}

	changedFields := subscriptionChangedFields(beforeSnapshot, afterSnapshot)
	if len(changedFields) == 0 {
		return nil
	}
	if eventType == "" {
		eventType = subscriptionEventUpdated
	}

	subscriptionID := after.ID
	actorUserID := userID
	return s.DB.Create(&model.SubscriptionEvent{
		UserID:                    userID,
		ActorUserID:               &actorUserID,
		SubscriptionID:            &subscriptionID,
		SubscriptionName:          after.Name,
		Type:                      eventType,
		ChangedFields:             encodeSubscriptionEventFields(changedFields),
		PreviousAmount:            float64Ptr(beforeSnapshot.Amount),
		NewAmount:                 float64Ptr(afterSnapshot.Amount),
		PreviousMonthlyAmount:     float64Ptr(beforeSnapshot.MonthlyAmount),
		NewMonthlyAmount:          float64Ptr(afterSnapshot.MonthlyAmount),
		PreviousCurrency:          beforeSnapshot.Currency,
		NewCurrency:               afterSnapshot.Currency,
		PreviousNextBillingDate:   copyTimePointer(beforeSnapshot.NextBillingDate),
		NewNextBillingDate:        copyTimePointer(afterSnapshot.NextBillingDate),
		PreviousStatus:            beforeSnapshot.Status,
		NewStatus:                 afterSnapshot.Status,
		PreviousRenewalMode:       beforeSnapshot.RenewalMode,
		NewRenewalMode:            afterSnapshot.RenewalMode,
		PreviousCategoryID:        copyUintPointer(beforeSnapshot.CategoryID),
		NewCategoryID:             copyUintPointer(afterSnapshot.CategoryID),
		PreviousCategoryName:      beforeSnapshot.CategoryName,
		NewCategoryName:           afterSnapshot.CategoryName,
		PreviousPaymentMethodID:   copyUintPointer(beforeSnapshot.PaymentMethodID),
		NewPaymentMethodID:        copyUintPointer(afterSnapshot.PaymentMethodID),
		PreviousPaymentMethodName: beforeSnapshot.PaymentMethodName,
		NewPaymentMethodName:      afterSnapshot.PaymentMethodName,
		CreatedAt:                 pkg.NowUTC(),
	}).Error
}

func (s *SubscriptionService) buildSubscriptionEventSnapshot(userID uint, sub model.Subscription) (subscriptionEventSnapshot, error) {
	categoryName := strings.TrimSpace(sub.Category)
	if sub.CategoryID != nil {
		var category model.Category
		if err := s.DB.Select("name").Where("id = ? AND user_id = ?", *sub.CategoryID, userID).First(&category).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return subscriptionEventSnapshot{}, err
			}
		} else {
			categoryName = category.Name
		}
	}

	paymentMethodName := ""
	if sub.PaymentMethodID != nil {
		var method model.PaymentMethod
		if err := s.DB.Select("name").Where("id = ? AND user_id = ?", *sub.PaymentMethodID, userID).First(&method).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return subscriptionEventSnapshot{}, err
			}
		} else {
			paymentMethodName = method.Name
		}
	}

	return subscriptionEventSnapshot{
		Amount:            sub.Amount,
		Currency:          strings.ToUpper(strings.TrimSpace(sub.Currency)),
		MonthlyAmount:     sub.Amount * subscriptionMonthlyFactor(sub),
		NextBillingDate:   copyTimePointer(sub.NextBillingDate),
		Status:            normalizeStatus(sub.Status),
		RenewalMode:       normalizeRenewalMode(sub.RenewalMode),
		CategoryID:        copyUintPointer(sub.CategoryID),
		CategoryName:      categoryName,
		PaymentMethodID:   copyUintPointer(sub.PaymentMethodID),
		PaymentMethodName: paymentMethodName,
	}, nil
}

func subscriptionChangedFields(before, after subscriptionEventSnapshot) []string {
	changed := make([]string, 0, 8)
	if !floatEqual(before.Amount, after.Amount) {
		changed = append(changed, "amount")
	}
	if before.Currency != after.Currency {
		changed = append(changed, "currency")
	}
	if !datePtrEqual(before.NextBillingDate, after.NextBillingDate) {
		changed = append(changed, "next_billing_date")
	}
	if before.Status != after.Status {
		changed = append(changed, "status")
	}
	if before.RenewalMode != after.RenewalMode {
		changed = append(changed, "renewal_mode")
	}
	if !uintPtrEqual(before.CategoryID, after.CategoryID) || before.CategoryName != after.CategoryName {
		changed = append(changed, "category")
	}
	if !uintPtrEqual(before.PaymentMethodID, after.PaymentMethodID) || before.PaymentMethodName != after.PaymentMethodName {
		changed = append(changed, "payment_method")
	}
	if !floatEqual(before.MonthlyAmount, after.MonthlyAmount) && !containsString(changed, "amount") {
		changed = append(changed, "monthly_amount")
	}

	sort.Strings(changed)
	return changed
}

func encodeSubscriptionEventFields(fields []string) string {
	if fields == nil {
		fields = []string{}
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func decodeSubscriptionEventFields(value string) []string {
	var fields []string
	if err := json.Unmarshal([]byte(value), &fields); err != nil {
		return []string{}
	}
	if fields == nil {
		return []string{}
	}
	return fields
}

func floatPtr(value float64) *float64 {
	return &value
}

func float64Ptr(value float64) *float64 {
	return floatPtr(value)
}

func copyUintPointer(value *uint) *uint {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func copyFloatPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.000001
}

func datePtrEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return normalizeDateUTC(*a).Equal(normalizeDateUTC(*b))
}

func uintPtrEqual(a, b *uint) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
