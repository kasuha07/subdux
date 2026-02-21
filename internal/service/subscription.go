package service

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

const (
	billingTypeRecurring = "recurring"
	billingTypeOneTime   = "one_time"
	billingTypeLifetime  = "lifetime"

	recurrenceTypeInterval    = "interval"
	recurrenceTypeMonthlyDate = "monthly_date"
	recurrenceTypeYearlyDate  = "yearly_date"

	intervalUnitDay   = "day"
	intervalUnitWeek  = "week"
	intervalUnitMonth = "month"
	intervalUnitYear  = "year"
)

type CurrencyConverter interface {
	Convert(amount float64, from, to string) float64
}

type SubscriptionService struct {
	DB *gorm.DB
}

func NewSubscriptionService(db *gorm.DB) *SubscriptionService {
	return &SubscriptionService{DB: db}
}

type CreateSubscriptionInput struct {
	Name              string  `json:"name"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	Enabled           *bool   `json:"enabled"`
	BillingType       string  `json:"billing_type"`
	RecurrenceType    string  `json:"recurrence_type"`
	IntervalCount     *int    `json:"interval_count"`
	IntervalUnit      string  `json:"interval_unit"`
	BillingAnchorDate string  `json:"billing_anchor_date"`
	MonthlyDay        *int    `json:"monthly_day"`
	YearlyMonth       *int    `json:"yearly_month"`
	YearlyDay         *int    `json:"yearly_day"`
	TrialEnabled      bool    `json:"trial_enabled"`
	TrialStartDate    string  `json:"trial_start_date"`
	TrialEndDate      string  `json:"trial_end_date"`
	Category          string  `json:"category"`
	CategoryID        *uint   `json:"category_id"`
	PaymentMethodID   *uint   `json:"payment_method_id"`
	NotifyEnabled     *bool   `json:"notify_enabled"`
	NotifyDaysBefore  *int    `json:"notify_days_before"`
	Icon              string  `json:"icon"`
	URL               string  `json:"url"`
	Notes             string  `json:"notes"`
}

type UpdateSubscriptionInput struct {
	Name              *string  `json:"name"`
	Amount            *float64 `json:"amount"`
	Currency          *string  `json:"currency"`
	Enabled           *bool    `json:"enabled"`
	BillingType       *string  `json:"billing_type"`
	RecurrenceType    *string  `json:"recurrence_type"`
	IntervalCount     *int     `json:"interval_count"`
	IntervalUnit      *string  `json:"interval_unit"`
	BillingAnchorDate *string  `json:"billing_anchor_date"`
	MonthlyDay        *int     `json:"monthly_day"`
	YearlyMonth       *int     `json:"yearly_month"`
	YearlyDay         *int     `json:"yearly_day"`
	TrialEnabled      *bool    `json:"trial_enabled"`
	TrialStartDate    *string  `json:"trial_start_date"`
	TrialEndDate      *string  `json:"trial_end_date"`
	Category          *string  `json:"category"`
	CategoryID        *uint    `json:"category_id"`
	PaymentMethodID   *uint    `json:"payment_method_id"`
	NotifyEnabled     *bool    `json:"notify_enabled"`
	NotifyDaysBefore  *int     `json:"notify_days_before"`
	Icon              *string  `json:"icon"`
	URL               *string  `json:"url"`
	Notes             *string  `json:"notes"`
}

type DashboardSummary struct {
	TotalMonthly         float64 `json:"total_monthly"`
	TotalYearly          float64 `json:"total_yearly"`
	EnabledCount         int64   `json:"enabled_count"`
	UpcomingRenewalCount int64   `json:"upcoming_renewal_count"`
	Currency             string  `json:"currency"`
}

type billingDraft struct {
	BillingType       string
	RecurrenceType    string
	IntervalCount     *int
	IntervalUnit      string
	BillingAnchorDate *time.Time
	MonthlyDay        *int
	YearlyMonth       *int
	YearlyDay         *int
	TrialEnabled      bool
	TrialStartDate    *time.Time
	TrialEndDate      *time.Time
}

func (s *SubscriptionService) List(userID uint) ([]model.Subscription, error) {
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

	anchorDate, err := parseOptionalDateString(input.BillingAnchorDate)
	if err != nil {
		return nil, err
	}
	trialStartDate, err := parseOptionalDateString(input.TrialStartDate)
	if err != nil {
		return nil, err
	}
	trialEndDate, err := parseOptionalDateString(input.TrialEndDate)
	if err != nil {
		return nil, err
	}

	draft := billingDraft{
		BillingType:       input.BillingType,
		RecurrenceType:    input.RecurrenceType,
		IntervalCount:     copyIntPointer(input.IntervalCount),
		IntervalUnit:      input.IntervalUnit,
		BillingAnchorDate: anchorDate,
		MonthlyDay:        copyIntPointer(input.MonthlyDay),
		YearlyMonth:       copyIntPointer(input.YearlyMonth),
		YearlyDay:         copyIntPointer(input.YearlyDay),
		TrialEnabled:      input.TrialEnabled,
		TrialStartDate:    trialStartDate,
		TrialEndDate:      trialEndDate,
	}

	normalizedDraft, nextBillingDate, err := normalizeBillingDraft(draft, time.Now().UTC())
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

	sub := model.Subscription{
		UserID:            userID,
		Name:              input.Name,
		Amount:            input.Amount,
		Currency:          currency,
		Enabled:           enabled,
		BillingType:       normalizedDraft.BillingType,
		RecurrenceType:    normalizedDraft.RecurrenceType,
		IntervalCount:     copyIntPointer(normalizedDraft.IntervalCount),
		IntervalUnit:      normalizedDraft.IntervalUnit,
		BillingAnchorDate: copyTimePointer(normalizedDraft.BillingAnchorDate),
		MonthlyDay:        copyIntPointer(normalizedDraft.MonthlyDay),
		YearlyMonth:       copyIntPointer(normalizedDraft.YearlyMonth),
		YearlyDay:         copyIntPointer(normalizedDraft.YearlyDay),
		TrialEnabled:      normalizedDraft.TrialEnabled,
		TrialStartDate:    copyTimePointer(normalizedDraft.TrialStartDate),
		TrialEndDate:      copyTimePointer(normalizedDraft.TrialEndDate),
		NextBillingDate:   copyTimePointer(nextBillingDate),
		Category:          input.Category,
		CategoryID:        input.CategoryID,
		PaymentMethodID:   paymentMethodID,
		NotifyEnabled:     input.NotifyEnabled,
		NotifyDaysBefore:  input.NotifyDaysBefore,
		Icon:              input.Icon,
		URL:               input.URL,
		Notes:             input.Notes,
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
	if input.NotifyEnabled != nil {
		updates["notify_enabled"] = *input.NotifyEnabled
	}
	if input.NotifyDaysBefore != nil {
		updates["notify_days_before"] = *input.NotifyDaysBefore
	}

	hasScheduleUpdate := input.BillingType != nil ||
		input.RecurrenceType != nil ||
		input.IntervalCount != nil ||
		input.IntervalUnit != nil ||
		input.BillingAnchorDate != nil ||
		input.MonthlyDay != nil ||
		input.YearlyMonth != nil ||
		input.YearlyDay != nil ||
		input.TrialEnabled != nil ||
		input.TrialStartDate != nil ||
		input.TrialEndDate != nil

	if hasScheduleUpdate {
		draft := billingDraft{
			BillingType:       sub.BillingType,
			RecurrenceType:    sub.RecurrenceType,
			IntervalCount:     copyIntPointer(sub.IntervalCount),
			IntervalUnit:      sub.IntervalUnit,
			BillingAnchorDate: copyTimePointer(sub.BillingAnchorDate),
			MonthlyDay:        copyIntPointer(sub.MonthlyDay),
			YearlyMonth:       copyIntPointer(sub.YearlyMonth),
			YearlyDay:         copyIntPointer(sub.YearlyDay),
			TrialEnabled:      sub.TrialEnabled,
			TrialStartDate:    copyTimePointer(sub.TrialStartDate),
			TrialEndDate:      copyTimePointer(sub.TrialEndDate),
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
		if input.BillingAnchorDate != nil {
			parsed, err := parseOptionalDateString(*input.BillingAnchorDate)
			if err != nil {
				return nil, err
			}
			draft.BillingAnchorDate = parsed
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
		if input.TrialEnabled != nil {
			draft.TrialEnabled = *input.TrialEnabled
		}
		if input.TrialStartDate != nil {
			parsed, err := parseOptionalDateString(*input.TrialStartDate)
			if err != nil {
				return nil, err
			}
			draft.TrialStartDate = parsed
		}
		if input.TrialEndDate != nil {
			parsed, err := parseOptionalDateString(*input.TrialEndDate)
			if err != nil {
				return nil, err
			}
			draft.TrialEndDate = parsed
		}

		normalizedDraft, nextBillingDate, err := normalizeBillingDraft(draft, time.Now().UTC())
		if err != nil {
			return nil, err
		}

		updates["billing_type"] = normalizedDraft.BillingType
		updates["recurrence_type"] = normalizedDraft.RecurrenceType
		updates["interval_count"] = copyIntPointer(normalizedDraft.IntervalCount)
		updates["interval_unit"] = normalizedDraft.IntervalUnit
		updates["billing_anchor_date"] = copyTimePointer(normalizedDraft.BillingAnchorDate)
		updates["monthly_day"] = copyIntPointer(normalizedDraft.MonthlyDay)
		updates["yearly_month"] = copyIntPointer(normalizedDraft.YearlyMonth)
		updates["yearly_day"] = copyIntPointer(normalizedDraft.YearlyDay)
		updates["trial_enabled"] = normalizedDraft.TrialEnabled
		updates["trial_start_date"] = copyTimePointer(normalizedDraft.TrialStartDate)
		updates["trial_end_date"] = copyTimePointer(normalizedDraft.TrialEndDate)
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

func (s *SubscriptionService) GetMaxIconFileSize() int64 {
	var setting model.SystemSetting
	if err := s.DB.Where("key = ?", "max_icon_file_size").First(&setting).Error; err == nil {
		if v, err := strconv.ParseInt(setting.Value, 10, 64); err == nil {
			return v
		}
	}
	return 65536
}

func (s *SubscriptionService) UploadSubscriptionIcon(userID, subID uint, file io.Reader, filename string, maxSize int64) (string, error) {
	sub, err := s.GetByID(userID, subID)
	if err != nil {
		return "", errors.New("subscription not found")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return "", errors.New("only PNG and JPG images are supported")
	}

	buf, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		return "", errors.New("failed to read file")
	}
	if int64(len(buf)) > maxSize {
		return "", errors.New("file size exceeds limit")
	}

	contentType := http.DetectContentType(buf)
	if contentType != "image/png" && contentType != "image/jpeg" {
		return "", errors.New("only PNG and JPG images are supported")
	}

	if ext == ".jpeg" {
		ext = ".jpg"
	}

	iconDir := filepath.Join(pkg.GetDataPath(), "assets", "icons")
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		return "", errors.New("failed to create icon directory")
	}

	newFilename := fmt.Sprintf("%d_%d_%d%s", userID, subID, time.Now().UnixNano(), ext)
	destPath := filepath.Join(iconDir, newFilename)

	if err := os.WriteFile(destPath, buf, 0644); err != nil {
		return "", errors.New("failed to save icon file")
	}

	s.removeManagedIconFile(sub.Icon)

	iconValue := "assets/icons/" + newFilename
	if err := s.DB.Model(&model.Subscription{}).Where("id = ? AND user_id = ?", subID, userID).Update("icon", iconValue).Error; err != nil {
		os.Remove(destPath)
		return "", err
	}

	return iconValue, nil
}

func (s *SubscriptionService) removeManagedIconFile(icon string) {
	if path, ok := managedIconFilePath(icon); ok {
		_ = os.Remove(path)
	}
}

func managedIconFilePath(icon string) (string, bool) {
	const iconPrefix = "assets/icons/"
	if !strings.HasPrefix(icon, iconPrefix) {
		return "", false
	}

	filename := strings.TrimPrefix(icon, iconPrefix)
	if filename == "" {
		return "", false
	}
	if strings.Contains(filename, "/") || strings.Contains(filename, `\`) {
		return "", false
	}
	if filepath.Base(filename) != filename {
		return "", false
	}

	return filepath.Join(pkg.GetDataPath(), "assets", "icons", filename), true
}

func (s *SubscriptionService) GetDashboardSummary(userID uint, targetCurrency string, converter CurrencyConverter) (*DashboardSummary, error) {
	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND enabled = ?", userID, true).Find(&subs).Error; err != nil {
		return nil, err
	}

	if targetCurrency == "" {
		targetCurrency = "USD"
	}

	var totalMonthly float64
	for _, sub := range subs {
		factor := subscriptionMonthlyFactor(sub)
		if factor <= 0 {
			continue
		}

		amount := sub.Amount
		if converter != nil && sub.Currency != targetCurrency {
			amount = converter.Convert(amount, sub.Currency, targetCurrency)
		}
		totalMonthly += amount * factor
	}

	today := normalizeDateUTC(time.Now().UTC())
	sevenDays := today.AddDate(0, 0, 7)
	var upcomingRenewalCount int64
	if err := s.DB.Model(&model.Subscription{}).Where(
		"user_id = ? AND enabled = ? AND billing_type = ? AND next_billing_date IS NOT NULL AND next_billing_date >= ? AND next_billing_date <= ?",
		userID,
		true,
		billingTypeRecurring,
		today,
		sevenDays,
	).Count(&upcomingRenewalCount).Error; err != nil {
		return nil, err
	}

	return &DashboardSummary{
		TotalMonthly:         totalMonthly,
		TotalYearly:          totalMonthly * 12,
		EnabledCount:         int64(len(subs)),
		UpcomingRenewalCount: upcomingRenewalCount,
		Currency:             targetCurrency,
	}, nil
}

func normalizeBillingDraft(draft billingDraft, now time.Time) (billingDraft, *time.Time, error) {
	draft.BillingType = normalizeBillingType(draft.BillingType)
	if draft.BillingType == "" {
		draft.BillingType = billingTypeRecurring
	}

	referenceDate := normalizeDateUTC(now)

	switch draft.BillingType {
	case billingTypeRecurring:
		draft.RecurrenceType = normalizeRecurrenceType(draft.RecurrenceType)
		if draft.RecurrenceType == "" {
			draft.RecurrenceType = recurrenceTypeInterval
		}

		if draft.BillingAnchorDate == nil {
			return draft, nil, errors.New("billing_anchor_date is required for recurring subscriptions")
		}
		anchorDate := normalizeDateUTC(*draft.BillingAnchorDate)
		draft.BillingAnchorDate = &anchorDate

		if referenceDate.Before(anchorDate) {
			referenceDate = anchorDate
		}

		if draft.TrialEnabled {
			if draft.TrialStartDate == nil || draft.TrialEndDate == nil {
				return draft, nil, errors.New("trial_start_date and trial_end_date are required when trial is enabled")
			}
			trialStartDate := normalizeDateUTC(*draft.TrialStartDate)
			trialEndDate := normalizeDateUTC(*draft.TrialEndDate)
			if trialEndDate.Before(trialStartDate) {
				return draft, nil, errors.New("trial_end_date must be on or after trial_start_date")
			}

			draft.TrialStartDate = &trialStartDate
			draft.TrialEndDate = &trialEndDate
			if referenceDate.Before(trialEndDate) {
				referenceDate = trialEndDate
			}
		} else {
			draft.TrialStartDate = nil
			draft.TrialEndDate = nil
		}

		switch draft.RecurrenceType {
		case recurrenceTypeInterval:
			if draft.IntervalCount == nil || *draft.IntervalCount < 1 {
				return draft, nil, errors.New("interval_count must be at least 1 for interval recurrence")
			}
			intervalCount := *draft.IntervalCount
			draft.IntervalCount = &intervalCount

			draft.IntervalUnit = normalizeIntervalUnit(draft.IntervalUnit)
			if !isValidIntervalUnit(draft.IntervalUnit) {
				return draft, nil, errors.New("interval_unit must be one of: day, week, month, year")
			}

			draft.MonthlyDay = nil
			draft.YearlyMonth = nil
			draft.YearlyDay = nil

			next := nextIntervalOccurrence(anchorDate, referenceDate, intervalCount, draft.IntervalUnit)
			return draft, &next, nil
		case recurrenceTypeMonthlyDate:
			if draft.MonthlyDay == nil || *draft.MonthlyDay < 1 || *draft.MonthlyDay > 31 {
				return draft, nil, errors.New("monthly_day must be between 1 and 31 for monthly date recurrence")
			}
			monthlyDay := *draft.MonthlyDay
			draft.MonthlyDay = &monthlyDay
			draft.IntervalCount = nil
			draft.IntervalUnit = ""
			draft.YearlyMonth = nil
			draft.YearlyDay = nil

			next := nextMonthlyDayOccurrence(referenceDate, monthlyDay)
			return draft, &next, nil
		case recurrenceTypeYearlyDate:
			if draft.YearlyMonth == nil || *draft.YearlyMonth < 1 || *draft.YearlyMonth > 12 {
				return draft, nil, errors.New("yearly_month must be between 1 and 12 for yearly date recurrence")
			}
			if draft.YearlyDay == nil || *draft.YearlyDay < 1 || *draft.YearlyDay > 31 {
				return draft, nil, errors.New("yearly_day must be between 1 and 31 for yearly date recurrence")
			}

			yearlyMonth := *draft.YearlyMonth
			yearlyDay := *draft.YearlyDay
			draft.YearlyMonth = &yearlyMonth
			draft.YearlyDay = &yearlyDay
			draft.IntervalCount = nil
			draft.IntervalUnit = ""
			draft.MonthlyDay = nil

			next := nextYearlyDateOccurrence(referenceDate, yearlyMonth, yearlyDay)
			return draft, &next, nil
		default:
			return draft, nil, errors.New("recurrence_type must be one of: interval, monthly_date, yearly_date")
		}
	case billingTypeOneTime:
		if draft.BillingAnchorDate == nil {
			return draft, nil, errors.New("billing_anchor_date is required for one-time subscriptions")
		}
		purchaseDate := normalizeDateUTC(*draft.BillingAnchorDate)
		draft.BillingAnchorDate = &purchaseDate
		draft.RecurrenceType = ""
		draft.IntervalCount = nil
		draft.IntervalUnit = ""
		draft.MonthlyDay = nil
		draft.YearlyMonth = nil
		draft.YearlyDay = nil
		draft.TrialEnabled = false
		draft.TrialStartDate = nil
		draft.TrialEndDate = nil
		return draft, nil, nil
	default:
		return draft, nil, errors.New("billing_type must be one of: recurring, one_time")
	}
}

func normalizeBillingType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == billingTypeLifetime {
		return billingTypeOneTime
	}
	return normalized
}

func normalizeRecurrenceType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeIntervalUnit(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isValidIntervalUnit(unit string) bool {
	switch unit {
	case intervalUnitDay, intervalUnitWeek, intervalUnitMonth, intervalUnitYear:
		return true
	default:
		return false
	}
}

func parseOptionalDateString(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, errors.New("invalid date format, expected YYYY-MM-DD")
	}

	normalized := normalizeDateUTC(parsed)
	return &normalized, nil
}

func normalizeDateUTC(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
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

func subscriptionMonthlyFactor(sub model.Subscription) float64 {
	if sub.BillingType != billingTypeRecurring {
		return 0
	}

	switch sub.RecurrenceType {
	case recurrenceTypeInterval:
		if sub.IntervalCount == nil || *sub.IntervalCount <= 0 {
			return 0
		}
		count := float64(*sub.IntervalCount)
		switch sub.IntervalUnit {
		case intervalUnitDay:
			return 30.436875 / count
		case intervalUnitWeek:
			return 4.348125 / count
		case intervalUnitMonth:
			return 1 / count
		case intervalUnitYear:
			return 1 / (12 * count)
		default:
			return 0
		}
	case recurrenceTypeMonthlyDate:
		return 1
	case recurrenceTypeYearlyDate:
		return 1.0 / 12.0
	default:
		return 0
	}
}

func nextIntervalOccurrence(anchor, from time.Time, intervalCount int, intervalUnit string) time.Time {
	anchor = normalizeDateUTC(anchor)
	from = normalizeDateUTC(from)
	if !from.After(anchor) {
		return anchor
	}

	current := anchor
	switch intervalUnit {
	case intervalUnitDay:
		for current.Before(from) {
			current = current.AddDate(0, 0, intervalCount)
		}
	case intervalUnitWeek:
		for current.Before(from) {
			current = current.AddDate(0, 0, intervalCount*7)
		}
	case intervalUnitMonth:
		preferredDay := anchor.Day()
		for current.Before(from) {
			current = addMonthsPreservePreferredDay(current, intervalCount, preferredDay)
		}
	case intervalUnitYear:
		preferredDay := anchor.Day()
		preferredMonth := anchor.Month()
		for current.Before(from) {
			current = addYearsPreservePreferredDate(current, intervalCount, preferredMonth, preferredDay)
		}
	}

	return current
}

func nextMonthlyDayOccurrence(from time.Time, day int) time.Time {
	from = normalizeDateUTC(from)
	year, month, _ := from.Date()

	candidate := buildDate(year, month, day)
	if candidate.Before(from) {
		candidate = addMonthsPreservePreferredDay(candidate, 1, day)
	}

	return candidate
}

func nextYearlyDateOccurrence(from time.Time, month int, day int) time.Time {
	from = normalizeDateUTC(from)
	year := from.Year()

	candidate := buildDate(year, time.Month(month), day)
	if candidate.Before(from) {
		candidate = buildDate(year+1, time.Month(month), day)
	}

	return candidate
}

func addMonthsPreservePreferredDay(base time.Time, months int, preferredDay int) time.Time {
	base = normalizeDateUTC(base)
	year, month, _ := base.Date()
	targetMonthIndex := int(month) - 1 + months
	targetYear := year + targetMonthIndex/12
	targetMonth := targetMonthIndex % 12
	if targetMonth < 0 {
		targetMonth += 12
		targetYear--
	}

	return buildDate(targetYear, time.Month(targetMonth+1), preferredDay)
}

func addYearsPreservePreferredDate(base time.Time, years int, preferredMonth time.Month, preferredDay int) time.Time {
	base = normalizeDateUTC(base)
	targetYear := base.Year() + years
	return buildDate(targetYear, preferredMonth, preferredDay)
}

func buildDate(year int, month time.Month, preferredDay int) time.Time {
	day := clampDay(year, month, preferredDay)
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func clampDay(year int, month time.Month, day int) int {
	if day < 1 {
		return 1
	}
	maxDay := daysInMonth(year, month)
	if day > maxDay {
		return maxDay
	}
	return day
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
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
