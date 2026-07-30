package subscription

import (
	"sort"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/money"
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

func (s *Service) GetAnalyticsReport(userID uint, targetCurrency string, converter CurrencyConverter) (*AnalyticsReport, error) {
	now := pkg.NowInSystemTimezone()

	if strings.TrimSpace(targetCurrency) == "" {
		targetCurrency = "USD"
	}
	targetCurrency = strings.ToUpper(strings.TrimSpace(targetCurrency))

	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND status = ?", userID, subscriptionStatusActive).Find(&subs).Error; err != nil {
		return nil, err
	}
	subs = presentActiveSubscriptions(subs, now)

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
		factor := subscriptionMonthlyFactor(sub)
		contributesOngoingSpend := factor > 0 && subscriptionContributesToOngoingSpend(sub)
		forecastEnd := startOfThisMonth.AddDate(0, len(report.MonthlyForecast), 0)
		hasForecastCharge := len(subscriptionChargeDatesInRange(sub, today, forecastEnd)) > 0
		amount := 0.0
		if contributesOngoingSpend || hasForecastCharge {
			var err error
			amount, err = convertSubscriptionAmount(sub, targetCurrency, converter)
			if err != nil {
				return nil, err
			}
		}
		monthlyAmount := 0.0
		if contributesOngoingSpend {
			var err error
			monthlyAmount, err = roundDerivedAmount(amount*factor, targetCurrency)
			if err != nil {
				return nil, err
			}
		}

		renewalMode := normalizeRenewalMode(sub.RenewalMode)
		switch renewalMode {
		case renewalModeAutoRenew:
			report.KPIs.AutoRenewCount++
			report.KPIs.CommittedMonthly, err = addAggregateAmounts(report.KPIs.CommittedMonthly, monthlyAmount, targetCurrency)
			if err != nil {
				return nil, err
			}
		case renewalModeManualRenew:
			report.KPIs.ManualRenewCount++
		case renewalModeCancelAtPeriodEnd:
			report.KPIs.CancelingCount++
		}

		report.KPIs.TotalMonthly, err = addAggregateAmounts(report.KPIs.TotalMonthly, monthlyAmount, targetCurrency)
		if err != nil {
			return nil, err
		}

		thisMonthRenewalDates := subscriptionChargeDatesInRange(sub, today, startOfNextMonth)
		if len(thisMonthRenewalDates) > 0 {
			due, dueErr := multiplyAggregateAmount(amount, int64(len(thisMonthRenewalDates)), targetCurrency)
			if dueErr != nil {
				return nil, dueErr
			}
			report.KPIs.DueThisMonth, dueErr = addAggregateAmounts(report.KPIs.DueThisMonth, due, targetCurrency)
			if dueErr != nil {
				return nil, dueErr
			}
		}

		renewalDates := subscriptionChargeDatesInRange(sub, today, next30DaysExclusive)
		report.KPIs.UpcomingRenewalCount += int64(len(renewalDates))
		if len(renewalDates) > 0 {
			due, dueErr := multiplyAggregateAmount(amount, int64(len(renewalDates)), targetCurrency)
			if dueErr != nil {
				return nil, dueErr
			}
			report.KPIs.DueNext30Days, dueErr = addAggregateAmounts(report.KPIs.DueNext30Days, due, targetCurrency)
			if dueErr != nil {
				return nil, dueErr
			}
		}
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
				due, dueErr := multiplyAggregateAmount(amount, int64(len(occurrences)), targetCurrency)
				if dueErr != nil {
					return nil, dueErr
				}
				report.MonthlyForecast[i].OccurrenceCount += len(occurrences)
				report.MonthlyForecast[i].AmountDue, dueErr = addAggregateAmounts(report.MonthlyForecast[i].AmountDue, due, targetCurrency)
				if dueErr != nil {
					return nil, dueErr
				}
			}
		}

		if monthlyAmount > 0 {
			categoryKey, categoryLabel := reportCategoryKeyAndLabel(sub, categoryLabels)
			if err := addReportBreakdown(categoryBreakdowns, categoryKey, categoryLabel, monthlyAmount, targetCurrency); err != nil {
				return nil, err
			}

			paymentKey, paymentLabel := reportPaymentMethodKeyAndLabel(sub, paymentMethodLabels)
			if err := addReportBreakdown(paymentMethodBreakdowns, paymentKey, paymentLabel, monthlyAmount, targetCurrency); err != nil {
				return nil, err
			}

			if err := addReportBreakdown(renewalModeBreakdowns, renewalMode, renewalMode, monthlyAmount, targetCurrency); err != nil {
				return nil, err
			}

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
				YearlyAmount:     0,
				OriginalAmount:   sub.Amount,
				OriginalCurrency: strings.ToUpper(sub.Currency),
			})
			report.TopSubscriptions[len(report.TopSubscriptions)-1].YearlyAmount, err = multiplyAggregateAmount(monthlyAmount, 12, targetCurrency)
			if err != nil {
				return nil, err
			}
		}
	}

	// Quantize every accumulated KPI through the aggregate-safe path before it
	// leaves the service. Aggregate values intentionally have a wider safe
	// range than one stored subscription amount.
	report.KPIs.TotalMonthly, err = roundAggregateAmount(report.KPIs.TotalMonthly, targetCurrency)
	if err != nil {
		return nil, err
	}
	report.KPIs.CommittedMonthly, err = roundAggregateAmount(report.KPIs.CommittedMonthly, targetCurrency)
	if err != nil {
		return nil, err
	}
	report.KPIs.DueThisMonth, err = roundAggregateAmount(report.KPIs.DueThisMonth, targetCurrency)
	if err != nil {
		return nil, err
	}
	report.KPIs.DueNext30Days, err = roundAggregateAmount(report.KPIs.DueNext30Days, targetCurrency)
	if err != nil {
		return nil, err
	}
	report.KPIs.TotalYearly, err = multiplyAggregateAmount(report.KPIs.TotalMonthly, 12, targetCurrency)
	if err != nil {
		return nil, err
	}
	report.KPIs.CommittedYearly, err = multiplyAggregateAmount(report.KPIs.CommittedMonthly, 12, targetCurrency)
	if err != nil {
		return nil, err
	}
	for i := range report.MonthlyForecast {
		report.MonthlyForecast[i].AmountDue, err = roundAggregateAmount(report.MonthlyForecast[i].AmountDue, targetCurrency)
		if err != nil {
			return nil, err
		}
	}
	report.CategoryBreakdown, err = buildReportBreakdown(categoryBreakdowns, report.KPIs.TotalMonthly, targetCurrency)
	if err != nil {
		return nil, err
	}
	report.PaymentMethodBreakdown, err = buildReportBreakdown(paymentMethodBreakdowns, report.KPIs.TotalMonthly, targetCurrency)
	if err != nil {
		return nil, err
	}
	report.RenewalModeBreakdown, err = buildReportBreakdown(renewalModeBreakdowns, report.KPIs.TotalMonthly, targetCurrency)
	if err != nil {
		return nil, err
	}

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

func (s *Service) addSubscriptionHistoryInsights(
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

func (s *Service) reportPriceIncreases(userID uint, targetCurrency string, converter CurrencyConverter) ([]ReportPriceIncrease, error) {
	var events []model.SubscriptionEvent
	if err := s.DB.Where(
		"user_id = ? AND previous_monthly_amount IS NOT NULL AND new_monthly_amount IS NOT NULL",
		userID,
	).Order("created_at DESC, id DESC").Limit(100).Find(&events).Error; err != nil {
		return nil, err
	}

	items := make([]ReportPriceIncrease, 0, len(events))
	seen := make(map[uint]struct{}, len(events))
	for _, event := range events {
		if event.SubscriptionID == nil || event.PreviousMonthlyAmount == nil || event.NewMonthlyAmount == nil {
			continue
		}
		subscriptionID := *event.SubscriptionID
		if _, ok := seen[subscriptionID]; ok {
			continue
		}
		// Event amounts are denominated in their respective snapshot
		// currencies. A currency switch is not comparable without an explicit
		// policy for the two snapshots; keep report semantics aligned with the
		// action center and treat a currency switch as a non-price-change event,
		// even when a converter happens to be available.
		if _, sameCurrency := priceChangeEventCurrency(event); !sameCurrency {
			// Consume the subscription just like the action center does: once the
			// newest event switches currencies, an older increase in the abandoned
			// currency must not surface behind it.
			seen[subscriptionID] = struct{}{}
			continue
		}
		// Compare on the minor-unit grid: stored event amounts and converted
		// amounts both carry float drift, and a sub-cent difference is not a
		// price increase.
		previousAmount, err := convertHistoricalAmount(*event.PreviousMonthlyAmount, event.PreviousCurrency, targetCurrency, converter)
		if err != nil {
			return nil, err
		}
		newAmount, err := convertHistoricalAmount(*event.NewMonthlyAmount, event.NewCurrency, targetCurrency, converter)
		if err != nil {
			return nil, err
		}
		direction := money.Cmp(newAmount, previousAmount, targetCurrency)
		seen[subscriptionID] = struct{}{}
		if direction == 0 {
			continue
		}
		if direction < 0 {
			continue
		}
		items = append(items, ReportPriceIncrease{
			SubscriptionID:        subscriptionID,
			Name:                  event.SubscriptionName,
			PreviousMonthlyAmount: previousAmount,
			NewMonthlyAmount:      newAmount,
			DeltaMonthlyAmount:    money.Diff(newAmount, previousAmount, targetCurrency),
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

func (s *Service) reportRecentSubscriptionChanges(userID uint, targetCurrency string, converter CurrencyConverter, today time.Time) ([]ReportSubscriptionEvent, error) {
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
			converted, err := convertHistoricalAmount(*previousAmount, event.PreviousCurrency, targetCurrency, converter)
			if err != nil {
				return nil, err
			}
			previousAmount = &converted
		}
		newAmount := copyFloatPointer(event.NewAmount)
		if newAmount != nil {
			converted, err := convertHistoricalAmount(*newAmount, event.NewCurrency, targetCurrency, converter)
			if err != nil {
				return nil, err
			}
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

func (s *Service) reportAnnualGrowth(userID uint, targetCurrency string, converter CurrencyConverter) ([]ReportAnnualGrowthItem, error) {
	now := pkg.NowInSystemTimezone()
	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND status = ?", userID, subscriptionStatusActive).Find(&subs).Error; err != nil {
		return nil, err
	}
	subs = presentActiveSubscriptions(subs, now)

	eligibleSubs := make([]model.Subscription, 0, len(subs))
	eligibleSubscriptionIDs := make(map[uint]struct{}, len(subs))
	for _, sub := range subs {
		if !subscriptionContributesToOngoingSpend(sub) {
			continue
		}
		eligibleSubs = append(eligibleSubs, sub)
		eligibleSubscriptionIDs[sub.ID] = struct{}{}
	}

	baselines, err := s.annualGrowthBaselineMonthlyAmounts(
		userID,
		eligibleSubscriptionIDs,
		targetCurrency,
		converter,
		now,
	)
	if err != nil {
		return nil, err
	}

	items := make([]ReportAnnualGrowthItem, 0, len(eligibleSubs))
	for _, sub := range eligibleSubs {
		rawCurrentMonthly := sub.Amount * subscriptionMonthlyFactor(sub)
		if err := validateDerivedAmount(rawCurrentMonthly); err != nil {
			return nil, err
		}
		currentMonthly, err := convertHistoricalAmount(rawCurrentMonthly, sub.Currency, targetCurrency, converter)
		if err != nil {
			return nil, err
		}
		if currentMonthly <= 0 {
			continue
		}

		baselineMonthly, ok := baselines[sub.ID]
		if !ok || baselineMonthly <= 0 {
			continue
		}

		// Growth is only real once it shows up on the minor-unit grid.
		if money.Cmp(currentMonthly, baselineMonthly, targetCurrency) <= 0 {
			continue
		}
		items = append(items, ReportAnnualGrowthItem{
			SubscriptionID:        sub.ID,
			Name:                  sub.Name,
			BaselineMonthlyAmount: baselineMonthly,
			CurrentMonthlyAmount:  currentMonthly,
			DeltaMonthlyAmount:    money.Diff(currentMonthly, baselineMonthly, targetCurrency),
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

// annualGrowthBaselineMonthlyAmounts returns, per subscription, the monthly
// amount recorded by that subscription's earliest comparable price-bearing
// event in its current currency epoch within the trailing year, converted to
// targetCurrency. It replaces a per-row lookup with a single ordered scan:
// events arrive oldest-first, and a currency switch clears any baseline from
// the abandoned epoch. The query is covered by
// idx_subscription_events_user_sub_created.
func (s *Service) annualGrowthBaselineMonthlyAmounts(
	userID uint,
	eligibleSubscriptionIDs map[uint]struct{},
	targetCurrency string,
	converter CurrencyConverter,
	now time.Time,
) (map[uint]float64, error) {
	if len(eligibleSubscriptionIDs) == 0 {
		return map[uint]float64{}, nil
	}

	oneYearAgo := normalizeDateUTC(now).AddDate(-1, 0, 0)
	var events []model.SubscriptionEvent
	if err := s.DB.Where(
		"user_id = ? AND type != ? AND previous_monthly_amount IS NOT NULL AND new_monthly_amount IS NOT NULL AND created_at >= ?",
		userID,
		subscriptionEventCreated,
		oneYearAgo,
	).Order("subscription_id ASC, created_at ASC, id ASC").Find(&events).Error; err != nil {
		return nil, err
	}

	type baselineCandidate struct {
		amount   float64
		currency string
	}
	candidates := make(map[uint]baselineCandidate)
	for _, event := range events {
		if event.SubscriptionID == nil || event.PreviousMonthlyAmount == nil {
			continue
		}
		subscriptionID := *event.SubscriptionID
		if _, eligible := eligibleSubscriptionIDs[subscriptionID]; !eligible {
			continue
		}
		if _, sameCurrency := priceChangeEventCurrency(event); !sameCurrency {
			// A currency switch starts a new comparison epoch. Discard any
			// baseline from the abandoned currency and keep scanning for the first
			// comparable event after the switch.
			delete(candidates, subscriptionID)
			continue
		}
		if _, ok := candidates[subscriptionID]; ok {
			continue
		}
		candidates[subscriptionID] = baselineCandidate{
			amount:   *event.PreviousMonthlyAmount,
			currency: event.PreviousCurrency,
		}
	}

	baselines := make(map[uint]float64, len(candidates))
	for subscriptionID, candidate := range candidates {
		baseline, err := convertHistoricalAmount(
			candidate.amount, candidate.currency, targetCurrency, converter,
		)
		if err != nil {
			return nil, err
		}
		baselines[subscriptionID] = baseline
	}
	return baselines, nil
}

// convertSubscriptionAmount converts a subscription's amount into
// targetCurrency. A converted amount is quantized to the target currency's
// minor unit immediately, so downstream aggregation never accumulates the
// sub-cent noise an exchange rate multiplication leaves behind.
func convertSubscriptionAmount(sub model.Subscription, targetCurrency string, converter CurrencyConverter) (float64, error) {
	if strings.EqualFold(sub.Currency, targetCurrency) {
		return roundDerivedAmount(sub.Amount, targetCurrency)
	}
	if converter == nil {
		return 0, ErrExchangeRateUnavailable
	}
	converted, ok := converter.Convert(sub.Amount, sub.Currency, targetCurrency)
	if !ok {
		return 0, ErrExchangeRateUnavailable
	}
	return roundDerivedAmount(converted, targetCurrency)
}

// convertHistoricalAmount converts an amount recorded on a subscription event
// into targetCurrency, quantizing the converted result to the target minor
// unit. Stored event amounts predating this policy may still carry drift; the
// money.Cmp-based gates on the read path absorb that.
func convertHistoricalAmount(amount float64, currency, targetCurrency string, converter CurrencyConverter) (float64, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	targetCurrency = strings.ToUpper(strings.TrimSpace(targetCurrency))
	if targetCurrency == "" {
		targetCurrency = "USD"
	}
	if currency == "" {
		return 0, ErrExchangeRateUnavailable
	}
	if strings.EqualFold(currency, targetCurrency) {
		return roundDerivedAmount(amount, targetCurrency)
	}
	if converter == nil {
		return 0, ErrExchangeRateUnavailable
	}
	converted, ok := converter.Convert(amount, currency, targetCurrency)
	if !ok {
		return 0, ErrExchangeRateUnavailable
	}
	return roundDerivedAmount(converted, targetCurrency)
}

// percentageDelta expects operands already quantized to their currency's minor
// unit so the percentage reflects the same numbers the caller reports. The
// percentage itself stays unrounded; the frontend formats it.
func percentageDelta(previousAmount, newAmount float64) float64 {
	if previousAmount <= 0 {
		return 0
	}
	return (newAmount - previousAmount) / previousAmount * 100
}

func (s *Service) reportCategoryLabels(userID uint) (map[uint]string, error) {
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

func (s *Service) reportPaymentMethodLabels(userID uint) (map[uint]string, error) {
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

func addReportBreakdown(
	items map[string]*reportBreakdownAccumulator,
	key, label string,
	monthlyAmount float64,
	targetCurrency string,
) error {
	item, ok := items[key]
	if !ok {
		item = &reportBreakdownAccumulator{
			key:   key,
			label: label,
		}
		items[key] = item
	}
	total, err := addAggregateAmounts(item.monthlyAmount, monthlyAmount, targetCurrency)
	if err != nil {
		return err
	}
	item.count++
	item.monthlyAmount = total
	return nil
}

// buildReportBreakdown quantizes each accumulated bucket to the target minor
// unit before emitting it. Percentages are derived from those rounded amounts
// but stay unrounded themselves — the frontend formats them.
func buildReportBreakdown(items map[string]*reportBreakdownAccumulator, totalMonthly float64, targetCurrency string) ([]ReportBreakdownItem, error) {
	result := make([]ReportBreakdownItem, 0, len(items))
	for _, item := range items {
		monthlyAmount, err := roundAggregateAmount(item.monthlyAmount, targetCurrency)
		if err != nil {
			return nil, err
		}
		percentage := 0.0
		if totalMonthly > 0 {
			percentage = monthlyAmount / totalMonthly * 100
		}
		result = append(result, ReportBreakdownItem{
			Key:           item.key,
			Label:         item.label,
			Count:         item.count,
			MonthlyAmount: monthlyAmount,
			YearlyAmount:  0,
			Percentage:    percentage,
		})
		result[len(result)-1].YearlyAmount, err = multiplyAggregateAmount(monthlyAmount, 12, targetCurrency)
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].MonthlyAmount == result[j].MonthlyAmount {
			return result[i].Label < result[j].Label
		}
		return result[i].MonthlyAmount > result[j].MonthlyAmount
	})
	return result, nil
}
