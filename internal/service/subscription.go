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
	CategoryID      *uint   `json:"category_id"`
	PaymentMethodID *uint   `json:"payment_method_id"`
	Icon            string  `json:"icon"`
	URL             string  `json:"url"`
	Notes           string  `json:"notes"`
}

type UpdateSubscriptionInput struct {
	Name            *string  `json:"name"`
	Amount          *float64 `json:"amount"`
	Currency        *string  `json:"currency"`
	BillingCycle    *string  `json:"billing_cycle"`
	NextBillingDate *string  `json:"next_billing_date"`
	Category        *string  `json:"category"`
	CategoryID      *uint    `json:"category_id"`
	PaymentMethodID *uint    `json:"payment_method_id"`
	Icon            *string  `json:"icon"`
	URL             *string  `json:"url"`
	Notes           *string  `json:"notes"`
	Status          *string  `json:"status"`
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

	var paymentMethodID *uint
	if input.PaymentMethodID != nil && *input.PaymentMethodID != 0 {
		if err := s.validatePaymentMethod(userID, *input.PaymentMethodID); err != nil {
			return nil, err
		}
		paymentMethodID = input.PaymentMethodID
	}

	sub := model.Subscription{
		UserID:          userID,
		Name:            input.Name,
		Amount:          input.Amount,
		Currency:        currency,
		BillingCycle:    input.BillingCycle,
		NextBillingDate: nextBilling,
		Category:        input.Category,
		CategoryID:      input.CategoryID,
		PaymentMethodID: paymentMethodID,
		Icon:            input.Icon,
		URL:             input.URL,
		Notes:           input.Notes,
		Status:          "active",
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
	if input.Status != nil {
		updates["status"] = *input.Status
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

	iconDir := filepath.Join("data", "assets", "icons")
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

	return filepath.Join("data", "assets", "icons", filename), true
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
