package service

import (
	"errors"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

const subscriptionDetailUpcomingChargeCount = 12

type SubscriptionDetail struct {
	Subscription     model.Subscription                   `json:"-"`
	Timeline         []SubscriptionDetailEvent            `json:"timeline"`
	PriceHistory     []SubscriptionDetailPriceHistoryItem `json:"price_history"`
	NotificationLogs []SubscriptionDetailNotificationLog  `json:"notification_logs"`
	UpcomingCharges  []SubscriptionDetailUpcomingCharge   `json:"upcoming_charges"`
	Calendar         SubscriptionDetailCalendar           `json:"calendar"`
}

type SubscriptionDetailEvent struct {
	ID                        uint     `json:"id"`
	Type                      string   `json:"type"`
	ChangedFields             []string `json:"changed_fields"`
	PreviousAmount            *float64 `json:"previous_amount"`
	NewAmount                 *float64 `json:"new_amount"`
	PreviousMonthlyAmount     *float64 `json:"previous_monthly_amount"`
	NewMonthlyAmount          *float64 `json:"new_monthly_amount"`
	PreviousCurrency          string   `json:"previous_currency"`
	NewCurrency               string   `json:"new_currency"`
	PreviousNextBillingDate   *string  `json:"previous_next_billing_date"`
	NewNextBillingDate        *string  `json:"new_next_billing_date"`
	PreviousStatus            string   `json:"previous_status"`
	NewStatus                 string   `json:"new_status"`
	PreviousRenewalMode       string   `json:"previous_renewal_mode"`
	NewRenewalMode            string   `json:"new_renewal_mode"`
	PreviousCategoryName      string   `json:"previous_category_name"`
	NewCategoryName           string   `json:"new_category_name"`
	PreviousPaymentMethodName string   `json:"previous_payment_method_name"`
	NewPaymentMethodName      string   `json:"new_payment_method_name"`
	ChangedAt                 string   `json:"changed_at"`
}

type SubscriptionDetailPriceHistoryItem struct {
	EventID               uint     `json:"event_id"`
	Type                  string   `json:"type"`
	Amount                float64  `json:"amount"`
	Currency              string   `json:"currency"`
	MonthlyAmount         *float64 `json:"monthly_amount"`
	PreviousAmount        *float64 `json:"previous_amount"`
	PreviousCurrency      string   `json:"previous_currency"`
	PreviousMonthlyAmount *float64 `json:"previous_monthly_amount"`
	ChangedAt             string   `json:"changed_at"`
}

type SubscriptionDetailNotificationLog struct {
	ID          uint   `json:"id"`
	ChannelType string `json:"channel_type"`
	NotifyDate  string `json:"notify_date"`
	Status      string `json:"status"`
	Error       string `json:"error"`
	SentAt      string `json:"sent_at"`
}

type SubscriptionDetailUpcomingCharge struct {
	Date        string  `json:"date"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	RenewalMode string  `json:"renewal_mode"`
}

type SubscriptionDetailCalendar struct {
	Path             string  `json:"path"`
	FeedPath         string  `json:"feed_path"`
	HasUpcomingEvent bool    `json:"has_upcoming_event"`
	NextEventDate    *string `json:"next_event_date"`
}

func (s *SubscriptionService) GetDetail(userID, id uint) (*SubscriptionDetail, error) {
	sub, err := s.GetByID(userID, id)
	if err != nil {
		return nil, err
	}

	timeline, err := s.subscriptionDetailTimeline(userID, id)
	if err != nil {
		return nil, err
	}
	priceHistory, err := s.subscriptionDetailPriceHistory(userID, id)
	if err != nil {
		return nil, err
	}

	logs, err := s.subscriptionDetailNotificationLogs(userID, id)
	if err != nil {
		return nil, err
	}

	upcomingCharges := subscriptionDetailUpcomingCharges(*sub, subscriptionDetailUpcomingChargeCount, pkg.NowInSystemTimezone())

	return &SubscriptionDetail{
		Subscription:     *sub,
		Timeline:         timeline,
		PriceHistory:     priceHistory,
		NotificationLogs: logs,
		UpcomingCharges:  upcomingCharges,
		Calendar:         subscriptionDetailCalendar(upcomingCharges),
	}, nil
}

func (s *SubscriptionService) subscriptionDetailTimeline(userID, subscriptionID uint) ([]SubscriptionDetailEvent, error) {
	var events []model.SubscriptionEvent
	if err := s.DB.Where("user_id = ? AND subscription_id = ?", userID, subscriptionID).
		Order("created_at DESC").
		Limit(50).
		Find(&events).Error; err != nil {
		return nil, err
	}

	items := make([]SubscriptionDetailEvent, 0, len(events))
	for _, event := range events {
		items = append(items, mapSubscriptionDetailEvent(event))
	}
	return items, nil
}

func mapSubscriptionDetailEvent(event model.SubscriptionEvent) SubscriptionDetailEvent {
	return SubscriptionDetailEvent{
		ID:                        event.ID,
		Type:                      event.Type,
		ChangedFields:             decodeSubscriptionEventFields(event.ChangedFields),
		PreviousAmount:            copyFloatPointer(event.PreviousAmount),
		NewAmount:                 copyFloatPointer(event.NewAmount),
		PreviousMonthlyAmount:     copyFloatPointer(event.PreviousMonthlyAmount),
		NewMonthlyAmount:          copyFloatPointer(event.NewMonthlyAmount),
		PreviousCurrency:          strings.ToUpper(strings.TrimSpace(event.PreviousCurrency)),
		NewCurrency:               strings.ToUpper(strings.TrimSpace(event.NewCurrency)),
		PreviousNextBillingDate:   formatDetailDateOnly(event.PreviousNextBillingDate),
		NewNextBillingDate:        formatDetailDateOnly(event.NewNextBillingDate),
		PreviousStatus:            event.PreviousStatus,
		NewStatus:                 event.NewStatus,
		PreviousRenewalMode:       event.PreviousRenewalMode,
		NewRenewalMode:            event.NewRenewalMode,
		PreviousCategoryName:      event.PreviousCategoryName,
		NewCategoryName:           event.NewCategoryName,
		PreviousPaymentMethodName: event.PreviousPaymentMethodName,
		NewPaymentMethodName:      event.NewPaymentMethodName,
		ChangedAt:                 event.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *SubscriptionService) subscriptionDetailPriceHistory(userID, subscriptionID uint) ([]SubscriptionDetailPriceHistoryItem, error) {
	var events []model.SubscriptionEvent
	if err := s.DB.Where("user_id = ? AND subscription_id = ?", userID, subscriptionID).
		Where("type = ? OR changed_fields LIKE ? OR changed_fields LIKE ? OR changed_fields LIKE ?",
			subscriptionEventCreated,
			`%"amount"%`,
			`%"currency"%`,
			`%"monthly_amount"%`,
		).
		Order("created_at ASC").
		Order("id ASC").
		Find(&events).Error; err != nil {
		return nil, err
	}

	items := make([]SubscriptionDetailPriceHistoryItem, 0, len(events))
	for _, event := range events {
		if !subscriptionEventHasPriceState(event) {
			continue
		}

		amount := 0.0
		if event.NewAmount != nil {
			amount = *event.NewAmount
		}
		currency := strings.ToUpper(strings.TrimSpace(event.NewCurrency))
		if currency == "" {
			currency = strings.ToUpper(strings.TrimSpace(event.PreviousCurrency))
		}

		items = append(items, SubscriptionDetailPriceHistoryItem{
			EventID:               event.ID,
			Type:                  event.Type,
			Amount:                amount,
			Currency:              currency,
			MonthlyAmount:         copyFloatPointer(event.NewMonthlyAmount),
			PreviousAmount:        copyFloatPointer(event.PreviousAmount),
			PreviousCurrency:      strings.ToUpper(strings.TrimSpace(event.PreviousCurrency)),
			PreviousMonthlyAmount: copyFloatPointer(event.PreviousMonthlyAmount),
			ChangedAt:             event.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return items, nil
}

func subscriptionEventHasPriceState(event model.SubscriptionEvent) bool {
	if event.Type == subscriptionEventCreated && event.NewAmount != nil {
		return true
	}
	for _, field := range decodeSubscriptionEventFields(event.ChangedFields) {
		switch field {
		case "amount", "currency", "monthly_amount":
			return event.NewAmount != nil
		}
	}
	return false
}

func (s *SubscriptionService) subscriptionDetailNotificationLogs(userID, subscriptionID uint) ([]SubscriptionDetailNotificationLog, error) {
	var logs []model.NotificationLog
	err := s.DB.Where("user_id = ? AND subscription_id = ?", userID, subscriptionID).
		Order("sent_at DESC").
		Limit(10).
		Find(&logs).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []SubscriptionDetailNotificationLog{}, nil
		}
		return nil, err
	}

	items := make([]SubscriptionDetailNotificationLog, 0, len(logs))
	for _, logEntry := range logs {
		items = append(items, SubscriptionDetailNotificationLog{
			ID:          logEntry.ID,
			ChannelType: logEntry.ChannelType,
			NotifyDate:  normalizeDateUTC(logEntry.NotifyDate).Format("2006-01-02"),
			Status:      logEntry.Status,
			Error:       logEntry.Error,
			SentAt:      logEntry.SentAt.UTC().Format(time.RFC3339),
		})
	}
	return items, nil
}

func subscriptionDetailUpcomingCharges(sub model.Subscription, limit int, now time.Time) []SubscriptionDetailUpcomingCharge {
	if limit <= 0 || sub.NextBillingDate == nil || normalizeStatus(sub.Status) != subscriptionStatusActive {
		return []SubscriptionDetailUpcomingCharge{}
	}

	current := normalizeDateUTC(*sub.NextBillingDate)
	today := normalizeDateUTC(now)

	if normalizeBillingType(sub.BillingType) != billingTypeRecurring || !isRecurringScheduleValid(sub) {
		return subscriptionDetailSingleUpcomingCharge(sub, current, today)
	}

	renewalMode := normalizeRenewalMode(sub.RenewalMode)
	if renewalMode != renewalModeAutoRenew {
		return subscriptionDetailSingleUpcomingCharge(sub, current, today)
	}

	if current.Before(today) {
		next, ok := nextRecurringOccurrenceOnOrAfter(sub, normalizeDateUTC(*sub.NextBillingDate), today)
		if !ok {
			return []SubscriptionDetailUpcomingCharge{}
		}
		current = next
	}

	items := make([]SubscriptionDetailUpcomingCharge, 0, limit)
	for len(items) < limit {
		if current.Before(today) {
			break
		}
		items = append(items, mapSubscriptionDetailUpcomingCharge(sub, current))

		next, ok := nextRecurringOccurrenceAfter(sub, current)
		if !ok || !next.After(current) {
			break
		}
		current = next
	}

	return items
}

func subscriptionDetailSingleUpcomingCharge(sub model.Subscription, current, today time.Time) []SubscriptionDetailUpcomingCharge {
	if current.Before(today) {
		return []SubscriptionDetailUpcomingCharge{}
	}
	return []SubscriptionDetailUpcomingCharge{mapSubscriptionDetailUpcomingCharge(sub, current)}
}

func mapSubscriptionDetailUpcomingCharge(sub model.Subscription, date time.Time) SubscriptionDetailUpcomingCharge {
	return SubscriptionDetailUpcomingCharge{
		Date:        normalizeDateUTC(date).Format("2006-01-02"),
		Amount:      sub.Amount,
		Currency:    strings.ToUpper(strings.TrimSpace(sub.Currency)),
		RenewalMode: normalizeRenewalMode(sub.RenewalMode),
	}
}

func subscriptionDetailCalendar(upcomingCharges []SubscriptionDetailUpcomingCharge) SubscriptionDetailCalendar {
	calendar := SubscriptionDetailCalendar{
		Path:     "/calendar",
		FeedPath: "/api/calendar/feed",
	}
	if len(upcomingCharges) > 0 {
		calendar.HasUpcomingEvent = true
		calendar.NextEventDate = detailStringPtr(upcomingCharges[0].Date)
	}
	return calendar
}

func formatDetailDateOnly(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := normalizeDateUTC(*value).Format("2006-01-02")
	return &formatted
}

func detailStringPtr(value string) *string {
	return &value
}
