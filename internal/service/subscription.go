package service

import (
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
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
	Name            string  `json:"name"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	BillingCycle    string  `json:"billing_cycle"`
	NextBillingDate string  `json:"next_billing_date"`
	Category        string  `json:"category"`
	Icon            string  `json:"icon"`
	URL             string  `json:"url"`
	Notes           string  `json:"notes"`
	Color           string  `json:"color"`
}

type UpdateSubscriptionInput struct {
	Name            *string  `json:"name"`
	Amount          *float64 `json:"amount"`
	Currency        *string  `json:"currency"`
	BillingCycle    *string  `json:"billing_cycle"`
	NextBillingDate *string  `json:"next_billing_date"`
	Category        *string  `json:"category"`
	Icon            *string  `json:"icon"`
	URL             *string  `json:"url"`
	Notes           *string  `json:"notes"`
	Status          *string  `json:"status"`
	Color           *string  `json:"color"`
}

type DashboardSummary struct {
	TotalMonthly     float64              `json:"total_monthly"`
	TotalYearly      float64              `json:"total_yearly"`
	ActiveCount      int64                `json:"active_count"`
	UpcomingRenewals []model.Subscription `json:"upcoming_renewals"`
	Currency         string               `json:"currency"`
}

func (s *SubscriptionService) List(userID uint) ([]model.Subscription, error) {
	var subs []model.Subscription
	err := s.DB.Where("user_id = ?", userID).Order("next_billing_date ASC").Find(&subs).Error
	return subs, err
}

func (s *SubscriptionService) GetByID(userID, id uint) (*model.Subscription, error) {
	var sub model.Subscription
	err := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&sub).Error
	return &sub, err
}

func (s *SubscriptionService) Create(userID uint, input CreateSubscriptionInput) (*model.Subscription, error) {
	var nextBilling time.Time
	if input.NextBillingDate != "" {
		parsed, err := time.Parse("2006-01-02", input.NextBillingDate)
		if err != nil {
			return nil, err
		}
		nextBilling = parsed
	} else {
		nextBilling = time.Now()
	}

	currency := input.Currency
	if currency == "" {
		currency = "USD"
	}

	sub := model.Subscription{
		UserID:          userID,
		Name:            input.Name,
		Amount:          input.Amount,
		Currency:        currency,
		BillingCycle:    input.BillingCycle,
		NextBillingDate: nextBilling,
		Category:        input.Category,
		Icon:            input.Icon,
		URL:             input.URL,
		Notes:           input.Notes,
		Status:          "active",
		Color:           input.Color,
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
		updates["currency"] = *input.Currency
	}
	if input.BillingCycle != nil {
		updates["billing_cycle"] = *input.BillingCycle
	}
	if input.Category != nil {
		updates["category"] = *input.Category
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
	if input.Status != nil {
		updates["status"] = *input.Status
	}
	if input.Color != nil {
		updates["color"] = *input.Color
	}
	if input.NextBillingDate != nil {
		parsed, err := time.Parse("2006-01-02", *input.NextBillingDate)
		if err != nil {
			return nil, err
		}
		updates["next_billing_date"] = parsed
	}

	if err := s.DB.Model(sub).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.GetByID(userID, id)
}

func (s *SubscriptionService) Delete(userID, id uint) error {
	return s.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Subscription{}).Error
}

func (s *SubscriptionService) GetDashboardSummary(userID uint, targetCurrency string, converter CurrencyConverter) (*DashboardSummary, error) {
	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND status = ?", userID, "active").Find(&subs).Error; err != nil {
		return nil, err
	}

	if targetCurrency == "" {
		targetCurrency = "USD"
	}

	var totalMonthly float64
	for _, sub := range subs {
		amount := sub.Amount
		if converter != nil && sub.Currency != targetCurrency {
			amount = converter.Convert(amount, sub.Currency, targetCurrency)
		}
		switch sub.BillingCycle {
		case "weekly":
			totalMonthly += amount * 4.33
		case "monthly":
			totalMonthly += amount
		case "yearly":
			totalMonthly += amount / 12
		}
	}

	sevenDays := time.Now().AddDate(0, 0, 7)
	var upcoming []model.Subscription
	s.DB.Where("user_id = ? AND status = ? AND next_billing_date <= ?", userID, "active", sevenDays).
		Order("next_billing_date ASC").Find(&upcoming)

	return &DashboardSummary{
		TotalMonthly:     totalMonthly,
		TotalYearly:      totalMonthly * 12,
		ActiveCount:      int64(len(subs)),
		UpcomingRenewals: upcoming,
		Currency:         targetCurrency,
	}, nil
}
