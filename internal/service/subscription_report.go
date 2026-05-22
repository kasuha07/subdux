package service

import (
	"sort"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
)

const (
	reportNoCategoryKey      = "__none__"
	reportNoPaymentMethodKey = "__none__"
)

type AnalyticsReport struct {
	Currency               string                    `json:"currency"`
	GeneratedAt            time.Time                 `json:"generated_at"`
	KPIs                   AnalyticsReportKPIs       `json:"kpis"`
	MonthlyForecast        []MonthlyForecastItem     `json:"monthly_forecast"`
	CategoryBreakdown      []ReportBreakdownItem     `json:"category_breakdown"`
	PaymentMethodBreakdown []ReportBreakdownItem     `json:"payment_method_breakdown"`
	RenewalModeBreakdown   []ReportBreakdownItem     `json:"renewal_mode_breakdown"`
	TopSubscriptions       []ReportSubscriptionSpend `json:"top_subscriptions"`
	UpcomingRenewals       []ReportUpcomingRenewal   `json:"upcoming_renewals"`
}

type AnalyticsReportKPIs struct {
	ActiveCount          int64   `json:"active_count"`
	AutoRenewCount       int64   `json:"auto_renew_count"`
	ManualRenewCount     int64   `json:"manual_renew_count"`
	CancelingCount       int64   `json:"canceling_count"`
	TotalMonthly         float64 `json:"total_monthly"`
	TotalYearly          float64 `json:"total_yearly"`
	CommittedMonthly     float64 `json:"committed_monthly"`
	CommittedYearly      float64 `json:"committed_yearly"`
	DueThisMonth         float64 `json:"due_this_month"`
	DueNext30Days        float64 `json:"due_next_30_days"`
	UpcomingRenewalCount int64   `json:"upcoming_renewal_count"`
}

type MonthlyForecastItem struct {
	Month           string  `json:"month"`
	AmountDue       float64 `json:"amount_due"`
	OccurrenceCount int     `json:"occurrence_count"`
}

type ReportBreakdownItem struct {
	Key           string  `json:"key"`
	Label         string  `json:"label"`
	Count         int64   `json:"count"`
	MonthlyAmount float64 `json:"monthly_amount"`
	YearlyAmount  float64 `json:"yearly_amount"`
	Percentage    float64 `json:"percentage"`
}

type ReportSubscriptionSpend struct {
	ID               uint    `json:"id"`
	Name             string  `json:"name"`
	Icon             string  `json:"icon"`
	Category         string  `json:"category"`
	PaymentMethod    string  `json:"payment_method"`
	RenewalMode      string  `json:"renewal_mode"`
	NextBillingDate  string  `json:"next_billing_date"`
	MonthlyAmount    float64 `json:"monthly_amount"`
	YearlyAmount     float64 `json:"yearly_amount"`
	OriginalAmount   float64 `json:"original_amount"`
	OriginalCurrency string  `json:"original_currency"`
}

type ReportUpcomingRenewal struct {
	ID            uint    `json:"id"`
	Name          string  `json:"name"`
	Icon          string  `json:"icon"`
	BillingDate   string  `json:"billing_date"`
	DaysUntil     int     `json:"days_until"`
	Amount        float64 `json:"amount"`
	Category      string  `json:"category"`
	PaymentMethod string  `json:"payment_method"`
	RenewalMode   string  `json:"renewal_mode"`
}

type reportBreakdownAccumulator struct {
	key           string
	label         string
	count         int64
	monthlyAmount float64
}

func (s *SubscriptionService) GetAnalyticsReport(userID uint, targetCurrency string, converter CurrencyConverter) (*AnalyticsReport, error) {
	now := pkg.NowInSystemTimezone()
	if err := reconcileSubscriptionLifecycleForUser(s.DB, userID, now); err != nil {
		return nil, err
	}

	if strings.TrimSpace(targetCurrency) == "" {
		targetCurrency = "USD"
	}
	targetCurrency = strings.ToUpper(strings.TrimSpace(targetCurrency))

	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND status = ?", userID, subscriptionStatusActive).Find(&subs).Error; err != nil {
		return nil, err
	}

	categoryLabels, err := s.reportCategoryLabels(userID)
	if err != nil {
		return nil, err
	}
	paymentMethodLabels, err := s.reportPaymentMethodLabels(userID)
	if err != nil {
		return nil, err
	}

	today := normalizeDateUTC(now)
	startOfThisMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	startOfNextMonth := startOfThisMonth.AddDate(0, 1, 0)
	next30DaysExclusive := today.AddDate(0, 0, 31)

	report := &AnalyticsReport{
		Currency:    targetCurrency,
		GeneratedAt: now,
		KPIs: AnalyticsReportKPIs{
			ActiveCount: int64(len(subs)),
		},
		MonthlyForecast:        make([]MonthlyForecastItem, 0, 12),
		CategoryBreakdown:      []ReportBreakdownItem{},
		PaymentMethodBreakdown: []ReportBreakdownItem{},
		RenewalModeBreakdown:   []ReportBreakdownItem{},
		TopSubscriptions:       []ReportSubscriptionSpend{},
		UpcomingRenewals:       []ReportUpcomingRenewal{},
	}

	categoryBreakdowns := map[string]*reportBreakdownAccumulator{}
	paymentMethodBreakdowns := map[string]*reportBreakdownAccumulator{}
	renewalModeBreakdowns := map[string]*reportBreakdownAccumulator{}

	for i := 0; i < 12; i++ {
		periodStart := startOfThisMonth.AddDate(0, i, 0)
		if i == 0 {
			periodStart = today
		}
		report.MonthlyForecast = append(report.MonthlyForecast, MonthlyForecastItem{
			Month: periodStart.Format("2006-01"),
		})
	}

	for _, sub := range subs {
		amount := convertSubscriptionAmount(sub, targetCurrency, converter)
		factor := subscriptionMonthlyFactor(sub)
		monthlyAmount := amount * factor

		renewalMode := normalizeRenewalMode(sub.RenewalMode)
		switch renewalMode {
		case renewalModeAutoRenew:
			report.KPIs.AutoRenewCount++
			report.KPIs.CommittedMonthly += monthlyAmount
		case renewalModeManualRenew:
			report.KPIs.ManualRenewCount++
		case renewalModeCancelAtPeriodEnd:
			report.KPIs.CancelingCount++
		}

		report.KPIs.TotalMonthly += monthlyAmount

		thisMonthRenewalDates := subscriptionReportOccurrenceDatesInRange(sub, today, startOfNextMonth)
		report.KPIs.DueThisMonth += amount * float64(len(thisMonthRenewalDates))

		renewalDates := subscriptionReportOccurrenceDatesInRange(sub, today, next30DaysExclusive)
		report.KPIs.UpcomingRenewalCount += int64(len(renewalDates))
		report.KPIs.DueNext30Days += amount * float64(len(renewalDates))
		for _, renewalDate := range renewalDates {
			report.UpcomingRenewals = append(report.UpcomingRenewals, ReportUpcomingRenewal{
				ID:            sub.ID,
				Name:          sub.Name,
				Icon:          sub.Icon,
				BillingDate:   renewalDate.Format("2006-01-02"),
				DaysUntil:     int(renewalDate.Sub(today).Hours() / 24),
				Amount:        amount,
				Category:      reportSubscriptionCategory(sub, categoryLabels),
				PaymentMethod: reportSubscriptionPaymentMethod(sub, paymentMethodLabels),
				RenewalMode:   renewalMode,
			})
		}

		for i := range report.MonthlyForecast {
			periodStart := startOfThisMonth.AddDate(0, i, 0)
			if i == 0 {
				periodStart = today
			}
			periodEnd := startOfThisMonth.AddDate(0, i+1, 0)
			occurrences := subscriptionReportOccurrenceDatesInRange(sub, periodStart, periodEnd)
			if len(occurrences) > 0 {
				report.MonthlyForecast[i].OccurrenceCount += len(occurrences)
				report.MonthlyForecast[i].AmountDue += amount * float64(len(occurrences))
			}
		}

		categoryKey, categoryLabel := reportCategoryKeyAndLabel(sub, categoryLabels)
		addReportBreakdown(categoryBreakdowns, categoryKey, categoryLabel, monthlyAmount)

		paymentKey, paymentLabel := reportPaymentMethodKeyAndLabel(sub, paymentMethodLabels)
		addReportBreakdown(paymentMethodBreakdowns, paymentKey, paymentLabel, monthlyAmount)

		addReportBreakdown(renewalModeBreakdowns, renewalMode, renewalMode, monthlyAmount)

		if monthlyAmount > 0 {
			nextBillingDate := ""
			if sub.NextBillingDate != nil {
				nextBillingDate = normalizeDateUTC(*sub.NextBillingDate).Format("2006-01-02")
			}
			report.TopSubscriptions = append(report.TopSubscriptions, ReportSubscriptionSpend{
				ID:               sub.ID,
				Name:             sub.Name,
				Icon:             sub.Icon,
				Category:         reportSubscriptionCategory(sub, categoryLabels),
				PaymentMethod:    reportSubscriptionPaymentMethod(sub, paymentMethodLabels),
				RenewalMode:      renewalMode,
				NextBillingDate:  nextBillingDate,
				MonthlyAmount:    monthlyAmount,
				YearlyAmount:     monthlyAmount * 12,
				OriginalAmount:   sub.Amount,
				OriginalCurrency: strings.ToUpper(sub.Currency),
			})
		}
	}

	report.KPIs.TotalYearly = report.KPIs.TotalMonthly * 12
	report.KPIs.CommittedYearly = report.KPIs.CommittedMonthly * 12
	report.CategoryBreakdown = buildReportBreakdown(categoryBreakdowns, report.KPIs.TotalMonthly)
	report.PaymentMethodBreakdown = buildReportBreakdown(paymentMethodBreakdowns, report.KPIs.TotalMonthly)
	report.RenewalModeBreakdown = buildReportBreakdown(renewalModeBreakdowns, report.KPIs.TotalMonthly)

	sort.Slice(report.TopSubscriptions, func(i, j int) bool {
		if report.TopSubscriptions[i].MonthlyAmount == report.TopSubscriptions[j].MonthlyAmount {
			return report.TopSubscriptions[i].Name < report.TopSubscriptions[j].Name
		}
		return report.TopSubscriptions[i].MonthlyAmount > report.TopSubscriptions[j].MonthlyAmount
	})
	if len(report.TopSubscriptions) > 8 {
		report.TopSubscriptions = report.TopSubscriptions[:8]
	}

	sort.Slice(report.UpcomingRenewals, func(i, j int) bool {
		if report.UpcomingRenewals[i].BillingDate == report.UpcomingRenewals[j].BillingDate {
			return report.UpcomingRenewals[i].Name < report.UpcomingRenewals[j].Name
		}
		return report.UpcomingRenewals[i].BillingDate < report.UpcomingRenewals[j].BillingDate
	})
	if len(report.UpcomingRenewals) > 12 {
		report.UpcomingRenewals = report.UpcomingRenewals[:12]
	}

	return report, nil
}

func convertSubscriptionAmount(sub model.Subscription, targetCurrency string, converter CurrencyConverter) float64 {
	if converter == nil || strings.EqualFold(sub.Currency, targetCurrency) {
		return sub.Amount
	}
	return converter.Convert(sub.Amount, sub.Currency, targetCurrency)
}

func (s *SubscriptionService) reportCategoryLabels(userID uint) (map[uint]string, error) {
	var categories []model.Category
	if err := s.DB.Where("user_id = ?", userID).Find(&categories).Error; err != nil {
		return nil, err
	}

	labels := make(map[uint]string, len(categories))
	for _, category := range categories {
		labels[category.ID] = category.Name
	}
	return labels, nil
}

func (s *SubscriptionService) reportPaymentMethodLabels(userID uint) (map[uint]string, error) {
	var paymentMethods []model.PaymentMethod
	if err := s.DB.Where("user_id = ?", userID).Find(&paymentMethods).Error; err != nil {
		return nil, err
	}

	labels := make(map[uint]string, len(paymentMethods))
	for _, method := range paymentMethods {
		labels[method.ID] = method.Name
	}
	return labels, nil
}

func reportCategoryKeyAndLabel(sub model.Subscription, labels map[uint]string) (string, string) {
	if sub.CategoryID != nil {
		if label := strings.TrimSpace(labels[*sub.CategoryID]); label != "" {
			return "category:" + label, label
		}
	}
	if label := strings.TrimSpace(sub.Category); label != "" {
		return "category:" + label, label
	}
	return reportNoCategoryKey, ""
}

func reportPaymentMethodKeyAndLabel(sub model.Subscription, labels map[uint]string) (string, string) {
	if sub.PaymentMethodID != nil {
		if label := strings.TrimSpace(labels[*sub.PaymentMethodID]); label != "" {
			return "payment:" + label, label
		}
	}
	return reportNoPaymentMethodKey, ""
}

func reportSubscriptionCategory(sub model.Subscription, labels map[uint]string) string {
	_, label := reportCategoryKeyAndLabel(sub, labels)
	return label
}

func reportSubscriptionPaymentMethod(sub model.Subscription, labels map[uint]string) string {
	_, label := reportPaymentMethodKeyAndLabel(sub, labels)
	return label
}

func addReportBreakdown(items map[string]*reportBreakdownAccumulator, key, label string, monthlyAmount float64) {
	item, ok := items[key]
	if !ok {
		item = &reportBreakdownAccumulator{
			key:   key,
			label: label,
		}
		items[key] = item
	}
	item.count++
	item.monthlyAmount += monthlyAmount
}

func buildReportBreakdown(items map[string]*reportBreakdownAccumulator, totalMonthly float64) []ReportBreakdownItem {
	result := make([]ReportBreakdownItem, 0, len(items))
	for _, item := range items {
		percentage := 0.0
		if totalMonthly > 0 {
			percentage = item.monthlyAmount / totalMonthly * 100
		}
		result = append(result, ReportBreakdownItem{
			Key:           item.key,
			Label:         item.label,
			Count:         item.count,
			MonthlyAmount: item.monthlyAmount,
			YearlyAmount:  item.monthlyAmount * 12,
			Percentage:    percentage,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].MonthlyAmount == result[j].MonthlyAmount {
			return result[i].Label < result[j].Label
		}
		return result[i].MonthlyAmount > result[j].MonthlyAmount
	})
	return result
}

func subscriptionOccurrenceDatesInRange(sub model.Subscription, startInclusive, endExclusive time.Time) []time.Time {
	startInclusive = normalizeDateUTC(startInclusive)
	endExclusive = normalizeDateUTC(endExclusive)
	if !startInclusive.Before(endExclusive) || sub.NextBillingDate == nil {
		return nil
	}

	current := normalizeDateUTC(*sub.NextBillingDate)
	if sub.BillingType != billingTypeRecurring {
		if current.Before(startInclusive) || !current.Before(endExclusive) {
			return nil
		}
		return []time.Time{current}
	}
	if !isRecurringScheduleValid(sub) {
		return nil
	}

	if current.Before(startInclusive) {
		next, ok := nextRecurringOccurrenceOnOrAfter(sub, current, startInclusive)
		if !ok {
			return nil
		}
		current = next
	}

	var dates []time.Time
	for current.Before(endExclusive) {
		if !current.Before(startInclusive) {
			dates = append(dates, current)
		}

		next, ok := nextRecurringOccurrenceAfter(sub, current)
		if !ok || !next.After(current) {
			break
		}
		current = next
	}
	return dates
}

func subscriptionReportOccurrenceDatesInRange(sub model.Subscription, startInclusive, endExclusive time.Time) []time.Time {
	if normalizeRenewalMode(sub.RenewalMode) == renewalModeAutoRenew {
		return subscriptionOccurrenceDatesInRange(sub, startInclusive, endExclusive)
	}

	startInclusive = normalizeDateUTC(startInclusive)
	endExclusive = normalizeDateUTC(endExclusive)
	if !startInclusive.Before(endExclusive) || sub.NextBillingDate == nil {
		return nil
	}

	nextBillingDate := normalizeDateUTC(*sub.NextBillingDate)
	if nextBillingDate.Before(startInclusive) || !nextBillingDate.Before(endExclusive) {
		return nil
	}
	return []time.Time{nextBillingDate}
}
