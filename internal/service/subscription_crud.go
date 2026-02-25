package service

import (
	"errors"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

func (s *SubscriptionService) List(userID uint) ([]model.Subscription, error) {
	now := time.Now().In(pkg.GetSystemTimezone())
	if err := autoAdvanceRecurringNextBillingDatesForUser(s.DB, userID, now); err != nil {
		return nil, err
	}

	var subs []model.Subscription
	err := s.DB.Where("user_id = ?", userID).
		Order("next_billing_date IS NULL ASC").
		Order("next_billing_date ASC").
		Order("id ASC").
		Find(&subs).Error
	if err != nil {
		return nil, err
	}

	for i := range subs {
		normalizeSubscriptionForResponse(&subs[i])
	}
	return subs, err
}

func (s *SubscriptionService) GetByID(userID, id uint) (*model.Subscription, error) {
	var sub model.Subscription
	err := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&sub).Error
	if err == nil {
		normalizeSubscriptionForResponse(&sub)
	}
	return &sub, err
}

func (s *SubscriptionService) Create(userID uint, input CreateSubscriptionInput) (*model.Subscription, error) {
	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "USD"
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	nextBillingDate, err := parseOptionalDateString(input.NextBillingDate)
	if err != nil {
		return nil, err
	}

	draft := billingDraft{
		BillingType:     input.BillingType,
		RecurrenceType:  input.RecurrenceType,
		IntervalCount:   copyIntPointer(input.IntervalCount),
		IntervalUnit:    input.IntervalUnit,
		NextBillingDate: nextBillingDate,
		MonthlyDay:      copyIntPointer(input.MonthlyDay),
		YearlyMonth:     copyIntPointer(input.YearlyMonth),
		YearlyDay:       copyIntPointer(input.YearlyDay),
	}

	normalizedDraft, nextBillingDate, err := normalizeBillingDraft(draft)
	if err != nil {
		return nil, err
	}

	var paymentMethodID *uint
	if input.PaymentMethodID != nil && *input.PaymentMethodID != 0 {
		if err := s.validatePaymentMethod(userID, *input.PaymentMethodID); err != nil {
			return nil, err
		}
		paymentMethodID = input.PaymentMethodID
	}
	if input.NotifyDaysBefore != nil {
		if err := validateNotifyDaysBefore(*input.NotifyDaysBefore); err != nil {
			return nil, err
		}
	}

	sub := model.Subscription{
		UserID:           userID,
		Name:             input.Name,
		Amount:           input.Amount,
		Currency:         currency,
		Enabled:          enabled,
		BillingType:      normalizedDraft.BillingType,
		RecurrenceType:   normalizedDraft.RecurrenceType,
		IntervalCount:    copyIntPointer(normalizedDraft.IntervalCount),
		IntervalUnit:     normalizedDraft.IntervalUnit,
		MonthlyDay:       copyIntPointer(normalizedDraft.MonthlyDay),
		YearlyMonth:      copyIntPointer(normalizedDraft.YearlyMonth),
		YearlyDay:        copyIntPointer(normalizedDraft.YearlyDay),
		NextBillingDate:  copyTimePointer(nextBillingDate),
		Category:         input.Category,
		CategoryID:       input.CategoryID,
		PaymentMethodID:  paymentMethodID,
		NotifyEnabled:    input.NotifyEnabled,
		NotifyDaysBefore: input.NotifyDaysBefore,
		Icon:             input.Icon,
		URL:              input.URL,
		Notes:            input.Notes,
	}

	if err := s.DB.Create(&sub).Error; err != nil {
		return nil, err
	}

	return &sub, nil
}

func (s *SubscriptionService) Update(userID, id uint, input UpdateSubscriptionInput) (*model.Subscription, error) {
	sub, err := s.GetByID(userID, id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Amount != nil {
		updates["amount"] = *input.Amount
	}
	if input.Currency != nil {
		updates["currency"] = strings.TrimSpace(*input.Currency)
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}
	if input.Category != nil {
		updates["category"] = *input.Category
	}
	if input.CategoryID != nil {
		updates["category_id"] = *input.CategoryID
	}
	if input.PaymentMethodID != nil {
		if *input.PaymentMethodID == 0 {
			updates["payment_method_id"] = nil
		} else {
			if err := s.validatePaymentMethod(userID, *input.PaymentMethodID); err != nil {
				return nil, err
			}
			updates["payment_method_id"] = *input.PaymentMethodID
		}
	}
	if input.Icon != nil {
		updates["icon"] = *input.Icon
	}
	if input.URL != nil {
		updates["url"] = *input.URL
	}
	if input.Notes != nil {
		updates["notes"] = *input.Notes
	}
	if input.NotifyEnabledSet || input.NotifyEnabled != nil {
		if input.NotifyEnabled == nil {
			updates["notify_enabled"] = nil
		} else {
			updates["notify_enabled"] = *input.NotifyEnabled
		}
	}
	if input.NotifyDaysBeforeSet || input.NotifyDaysBefore != nil {
		if input.NotifyDaysBefore == nil {
			updates["notify_days_before"] = nil
		} else {
			if err := validateNotifyDaysBefore(*input.NotifyDaysBefore); err != nil {
				return nil, err
			}
			updates["notify_days_before"] = *input.NotifyDaysBefore
		}
	}

	hasScheduleUpdate := input.BillingType != nil ||
		input.RecurrenceType != nil ||
		input.IntervalCount != nil ||
		input.IntervalUnit != nil ||
		input.NextBillingDate != nil ||
		input.MonthlyDay != nil ||
		input.YearlyMonth != nil ||
		input.YearlyDay != nil

	if hasScheduleUpdate {
		draft := billingDraft{
			BillingType:     sub.BillingType,
			RecurrenceType:  sub.RecurrenceType,
			IntervalCount:   copyIntPointer(sub.IntervalCount),
			IntervalUnit:    sub.IntervalUnit,
			NextBillingDate: copyTimePointer(sub.NextBillingDate),
			MonthlyDay:      copyIntPointer(sub.MonthlyDay),
			YearlyMonth:     copyIntPointer(sub.YearlyMonth),
			YearlyDay:       copyIntPointer(sub.YearlyDay),
		}

		if input.BillingType != nil {
			draft.BillingType = *input.BillingType
		}
		if input.RecurrenceType != nil {
			draft.RecurrenceType = *input.RecurrenceType
		}
		if input.IntervalCount != nil {
			draft.IntervalCount = copyIntPointer(input.IntervalCount)
		}
		if input.IntervalUnit != nil {
			draft.IntervalUnit = *input.IntervalUnit
		}
		if input.NextBillingDate != nil {
			parsed, err := parseOptionalDateString(*input.NextBillingDate)
			if err != nil {
				return nil, err
			}
			draft.NextBillingDate = parsed
		}
		if input.MonthlyDay != nil {
			draft.MonthlyDay = copyIntPointer(input.MonthlyDay)
		}
		if input.YearlyMonth != nil {
			draft.YearlyMonth = copyIntPointer(input.YearlyMonth)
		}
		if input.YearlyDay != nil {
			draft.YearlyDay = copyIntPointer(input.YearlyDay)
		}
		normalizedDraft, nextBillingDate, err := normalizeBillingDraft(draft)
		if err != nil {
			return nil, err
		}

		updates["billing_type"] = normalizedDraft.BillingType
		updates["recurrence_type"] = normalizedDraft.RecurrenceType
		updates["interval_count"] = copyIntPointer(normalizedDraft.IntervalCount)
		updates["interval_unit"] = normalizedDraft.IntervalUnit
		updates["monthly_day"] = copyIntPointer(normalizedDraft.MonthlyDay)
		updates["yearly_month"] = copyIntPointer(normalizedDraft.YearlyMonth)
		updates["yearly_day"] = copyIntPointer(normalizedDraft.YearlyDay)
		updates["next_billing_date"] = copyTimePointer(nextBillingDate)
	}

	if err := s.DB.Model(sub).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.GetByID(userID, id)
}

func (s *SubscriptionService) validatePaymentMethod(userID, paymentMethodID uint) error {
	var method model.PaymentMethod
	if err := s.DB.Where("id = ? AND user_id = ?", paymentMethodID, userID).First(&method).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("payment method not found")
		}
		return err
	}
	return nil
}

func (s *SubscriptionService) Delete(userID, id uint) error {
	sub, err := s.GetByID(userID, id)
	if err != nil {
		return err
	}

	if err := s.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Subscription{}).Error; err != nil {
		return err
	}

	s.removeManagedIconFile(sub.Icon)

	return nil
}

func copyIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func copyTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := normalizeDateUTC(*value)
	return &copied
}

func normalizeSubscriptionForResponse(sub *model.Subscription) {
	if sub == nil {
		return
	}
	billingType := strings.ToLower(strings.TrimSpace(sub.BillingType))
	if billingType == billingTypeLifetime || billingType == "payg" || billingType == billingTypeOneTime {
		sub.BillingType = billingTypeOneTime
	}
}
