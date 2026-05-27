package service

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
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
	PriceIncreases         []ReportPriceIncrease     `json:"price_increases"`
	RecentChanges          []ReportSubscriptionEvent `json:"recent_changes"`
	AnnualGrowth           []ReportAnnualGrowthItem  `json:"annual_growth"`
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

type ReportPriceIncrease struct {
	SubscriptionID        uint    `json:"subscription_id"`
	Name                  string  `json:"name"`
	PreviousMonthlyAmount float64 `json:"previous_monthly_amount"`
	NewMonthlyAmount      float64 `json:"new_monthly_amount"`
	DeltaMonthlyAmount    float64 `json:"delta_monthly_amount"`
	DeltaPercentage       float64 `json:"delta_percentage"`
	Currency              string  `json:"currency"`
	ChangedAt             string  `json:"changed_at"`
}

type ReportSubscriptionEvent struct {
	ID               uint     `json:"id"`
	SubscriptionID   *uint    `json:"subscription_id"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	ChangedFields    []string `json:"changed_fields"`
	PreviousAmount   *float64 `json:"previous_amount"`
	NewAmount        *float64 `json:"new_amount"`
	PreviousCurrency string   `json:"previous_currency"`
	NewCurrency      string   `json:"new_currency"`
	ChangedAt        string   `json:"changed_at"`
}

type ReportAnnualGrowthItem struct {
	SubscriptionID        uint    `json:"subscription_id"`
	Name                  string  `json:"name"`
	BaselineMonthlyAmount float64 `json:"baseline_monthly_amount"`
	CurrentMonthlyAmount  float64 `json:"current_monthly_amount"`
	DeltaMonthlyAmount    float64 `json:"delta_monthly_amount"`
	DeltaPercentage       float64 `json:"delta_percentage"`
	Currency              string  `json:"currency"`
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
		PriceIncreases:         []ReportPriceIncrease{},
		RecentChanges:          []ReportSubscriptionEvent{},
		AnnualGrowth:           []ReportAnnualGrowthItem{},
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
		monthlyAmount := 0.0
		if subscriptionContributesToOngoingSpend(sub) {
			monthlyAmount = amount * factor
		}

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

		thisMonthRenewalDates := subscriptionChargeDatesInRange(sub, today, startOfNextMonth)
		report.KPIs.DueThisMonth += amount * float64(len(thisMonthRenewalDates))

		renewalDates := subscriptionChargeDatesInRange(sub, today, next30DaysExclusive)
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
			occurrences := subscriptionChargeDatesInRange(sub, periodStart, periodEnd)
			if len(occurrences) > 0 {
				report.MonthlyForecast[i].OccurrenceCount += len(occurrences)
				report.MonthlyForecast[i].AmountDue += amount * float64(len(occurrences))
			}
		}

		if monthlyAmount > 0 {
			categoryKey, categoryLabel := reportCategoryKeyAndLabel(sub, categoryLabels)
			addReportBreakdown(categoryBreakdowns, categoryKey, categoryLabel, monthlyAmount)

			paymentKey, paymentLabel := reportPaymentMethodKeyAndLabel(sub, paymentMethodLabels)
			addReportBreakdown(paymentMethodBreakdowns, paymentKey, paymentLabel, monthlyAmount)

			addReportBreakdown(renewalModeBreakdowns, renewalMode, renewalMode, monthlyAmount)

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

	if err := s.addSubscriptionHistoryInsights(report, userID, targetCurrency, converter, today); err != nil {
		return nil, err
	}

	return report, nil
}

func (s *SubscriptionService) addSubscriptionHistoryInsights(
	report *AnalyticsReport,
	userID uint,
	targetCurrency string,
	converter CurrencyConverter,
	today time.Time,
) error {
	priceIncreases, err := s.reportPriceIncreases(userID, targetCurrency, converter)
	if err != nil {
		return err
	}
	report.PriceIncreases = priceIncreases

	recentChanges, err := s.reportRecentSubscriptionChanges(userID, targetCurrency, converter, today)
	if err != nil {
		return err
	}
	report.RecentChanges = recentChanges

	annualGrowth, err := s.reportAnnualGrowth(userID, targetCurrency, converter)
	if err != nil {
		return err
	}
	report.AnnualGrowth = annualGrowth

	return nil
}

func (s *SubscriptionService) reportPriceIncreases(userID uint, targetCurrency string, converter CurrencyConverter) ([]ReportPriceIncrease, error) {
	var events []model.SubscriptionEvent
	if err := s.DB.Where(
		"user_id = ? AND previous_monthly_amount IS NOT NULL AND new_monthly_amount IS NOT NULL",
		userID,
	).Order("created_at DESC").Limit(100).Find(&events).Error; err != nil {
		return nil, err
	}

	items := make([]ReportPriceIncrease, 0, len(events))
	for _, event := range events {
		if event.SubscriptionID == nil || event.PreviousMonthlyAmount == nil || event.NewMonthlyAmount == nil {
			continue
		}
		previousAmount := convertHistoricalAmount(*event.PreviousMonthlyAmount, event.PreviousCurrency, targetCurrency, converter)
		newAmount := convertHistoricalAmount(*event.NewMonthlyAmount, event.NewCurrency, targetCurrency, converter)
		delta := newAmount - previousAmount
		if delta <= 0 {
			continue
		}
		items = append(items, ReportPriceIncrease{
			SubscriptionID:        *event.SubscriptionID,
			Name:                  event.SubscriptionName,
			PreviousMonthlyAmount: previousAmount,
			NewMonthlyAmount:      newAmount,
			DeltaMonthlyAmount:    delta,
			DeltaPercentage:       percentageDelta(previousAmount, newAmount),
			Currency:              targetCurrency,
			ChangedAt:             event.CreatedAt.Format("2006-01-02"),
		})
		if len(items) == 12 {
			break
		}
	}
	return items, nil
}

func (s *SubscriptionService) reportRecentSubscriptionChanges(userID uint, targetCurrency string, converter CurrencyConverter, today time.Time) ([]ReportSubscriptionEvent, error) {
	since := normalizeDateUTC(today).AddDate(0, 0, -90)
	var events []model.SubscriptionEvent
	if err := s.DB.Where("user_id = ? AND created_at >= ?", userID, since).
		Order("created_at DESC").
		Limit(20).
		Find(&events).Error; err != nil {
		return nil, err
	}

	items := make([]ReportSubscriptionEvent, 0, len(events))
	for _, event := range events {
		previousAmount := copyFloatPointer(event.PreviousAmount)
		if previousAmount != nil {
			converted := convertHistoricalAmount(*previousAmount, event.PreviousCurrency, targetCurrency, converter)
			previousAmount = &converted
		}
		newAmount := copyFloatPointer(event.NewAmount)
		if newAmount != nil {
			converted := convertHistoricalAmount(*newAmount, event.NewCurrency, targetCurrency, converter)
			newAmount = &converted
		}
		items = append(items, ReportSubscriptionEvent{
			ID:               event.ID,
			SubscriptionID:   copyUintPointer(event.SubscriptionID),
			Name:             event.SubscriptionName,
			Type:             event.Type,
			ChangedFields:    decodeSubscriptionEventFields(event.ChangedFields),
			PreviousAmount:   previousAmount,
			NewAmount:        newAmount,
			PreviousCurrency: targetCurrency,
			NewCurrency:      targetCurrency,
			ChangedAt:        event.CreatedAt.Format("2006-01-02"),
		})
	}
	return items, nil
}

func (s *SubscriptionService) reportAnnualGrowth(userID uint, targetCurrency string, converter CurrencyConverter) ([]ReportAnnualGrowthItem, error) {
	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND status = ?", userID, subscriptionStatusActive).Find(&subs).Error; err != nil {
		return nil, err
	}

	items := make([]ReportAnnualGrowthItem, 0, len(subs))
	for _, sub := range subs {
		if !subscriptionContributesToOngoingSpend(sub) {
			continue
		}
		currentMonthly := convertHistoricalAmount(sub.Amount*subscriptionMonthlyFactor(sub), sub.Currency, targetCurrency, converter)
		if currentMonthly <= 0 {
			continue
		}

		baselineMonthly, ok, err := s.subscriptionAnnualGrowthBaselineMonthlyAmount(userID, sub.ID, targetCurrency, converter, pkg.NowInSystemTimezone())
		if err != nil {
			return nil, err
		}
		if !ok || baselineMonthly <= 0 {
			continue
		}

		delta := currentMonthly - baselineMonthly
		if delta <= 0 {
			continue
		}
		items = append(items, ReportAnnualGrowthItem{
			SubscriptionID:        sub.ID,
			Name:                  sub.Name,
			BaselineMonthlyAmount: baselineMonthly,
			CurrentMonthlyAmount:  currentMonthly,
			DeltaMonthlyAmount:    delta,
			DeltaPercentage:       percentageDelta(baselineMonthly, currentMonthly),
			Currency:              targetCurrency,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].DeltaMonthlyAmount == items[j].DeltaMonthlyAmount {
			return items[i].Name < items[j].Name
		}
		return items[i].DeltaMonthlyAmount > items[j].DeltaMonthlyAmount
	})
	if len(items) > 8 {
		items = items[:8]
	}
	return items, nil
}

func (s *SubscriptionService) subscriptionAnnualGrowthBaselineMonthlyAmount(
	userID uint,
	subscriptionID uint,
	targetCurrency string,
	converter CurrencyConverter,
	now time.Time,
) (float64, bool, error) {
	oneYearAgo := normalizeDateUTC(now).AddDate(-1, 0, 0)
	var event model.SubscriptionEvent
	err := s.DB.Where(
		"user_id = ? AND subscription_id = ? AND type != ? AND previous_monthly_amount IS NOT NULL AND new_monthly_amount IS NOT NULL AND created_at >= ?",
		userID,
		subscriptionID,
		subscriptionEventCreated,
		oneYearAgo,
	).Order("created_at ASC").First(&event).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if event.PreviousMonthlyAmount == nil {
		return 0, false, nil
	}
	return convertHistoricalAmount(*event.PreviousMonthlyAmount, event.PreviousCurrency, targetCurrency, converter), true, nil
}

func convertSubscriptionAmount(sub model.Subscription, targetCurrency string, converter CurrencyConverter) float64 {
	if converter == nil || strings.EqualFold(sub.Currency, targetCurrency) {
		return sub.Amount
	}
	return converter.Convert(sub.Amount, sub.Currency, targetCurrency)
}

func convertHistoricalAmount(amount float64, currency, targetCurrency string, converter CurrencyConverter) float64 {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	targetCurrency = strings.ToUpper(strings.TrimSpace(targetCurrency))
	if targetCurrency == "" {
		targetCurrency = "USD"
	}
	if currency == "" || converter == nil || strings.EqualFold(currency, targetCurrency) {
		return amount
	}
	return converter.Convert(amount, currency, targetCurrency)
}

func percentageDelta(previousAmount, newAmount float64) float64 {
	if previousAmount <= 0 {
		return 0
	}
	return (newAmount - previousAmount) / previousAmount * 100
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
