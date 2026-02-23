package service

import (
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type defaultCategoryTemplate struct {
	SystemKey     string
	CanonicalName string
	DisplayOrder  int
}

type defaultPaymentMethodTemplate struct {
	SystemKey     string
	CanonicalName string
	Icon          string
	SortOrder     int
}

type defaultCurrencyTemplate struct {
	Code      string
	Symbol    string
	SortOrder int
}

var defaultCategoryTemplates = []defaultCategoryTemplate{
	{SystemKey: "video", CanonicalName: "Video", DisplayOrder: 0},
	{SystemKey: "music", CanonicalName: "Music", DisplayOrder: 1},
	{SystemKey: "productivity", CanonicalName: "Productivity", DisplayOrder: 2},
	{SystemKey: "cloud", CanonicalName: "Cloud", DisplayOrder: 3},
	{SystemKey: "shopping", CanonicalName: "Shopping", DisplayOrder: 4},
	{SystemKey: "finance", CanonicalName: "Finance", DisplayOrder: 5},
	{SystemKey: "education", CanonicalName: "Education", DisplayOrder: 6},
	{SystemKey: "health", CanonicalName: "Health", DisplayOrder: 7},
	{SystemKey: "news", CanonicalName: "News", DisplayOrder: 8},
	{SystemKey: "other", CanonicalName: "Other", DisplayOrder: 9},
}

var defaultPaymentMethodTemplates = []defaultPaymentMethodTemplate{
	{SystemKey: "credit_card", CanonicalName: "Credit Card", Icon: "💳", SortOrder: 0},
	{SystemKey: "debit_card", CanonicalName: "Debit Card", Icon: "💳", SortOrder: 1},
	{SystemKey: "paypal", CanonicalName: "PayPal", Icon: "lg:paypal", SortOrder: 2},
	{SystemKey: "bank_transfer", CanonicalName: "Bank Transfer", Icon: "🏦", SortOrder: 3},
	{SystemKey: "cash", CanonicalName: "Cash", Icon: "💵", SortOrder: 4},
}

var defaultCurrencyTemplates = []defaultCurrencyTemplate{
	{Code: "USD", Symbol: "$", SortOrder: 0},
	{Code: "EUR", Symbol: "€", SortOrder: 1},
	{Code: "GBP", Symbol: "£", SortOrder: 2},
	{Code: "CNY", Symbol: "¥", SortOrder: 3},
	{Code: "JPY", Symbol: "¥", SortOrder: 4},
}

const defaultNotificationTemplate = "Your subscription {{.SubscriptionName}} ({{.Amount}} {{.Currency}}) will be billed in {{.DaysUntil}} days on {{.BillingDate}}. Payment method: {{.PaymentMethod}}. URL: {{.URL}}. Remark: {{.Remark}}."

func SeedUserDefaults(tx *gorm.DB, userID uint) error {
	if err := seedDefaultCategories(tx, userID); err != nil {
		return err
	}
	if err := seedDefaultPaymentMethods(tx, userID); err != nil {
		return err
	}
	if err := seedDefaultCurrencies(tx, userID); err != nil {
		return err
	}
	if err := seedDefaultNotificationTemplate(tx, userID); err != nil {
		return err
	}
	return seedDefaultPreference(tx, userID)
}

func seedDefaultCategories(tx *gorm.DB, userID uint) error {
	categories := make([]model.Category, 0, len(defaultCategoryTemplates))
	for _, template := range defaultCategoryTemplates {
		categories = append(categories, model.Category{
			UserID:         userID,
			Name:           template.CanonicalName,
			SystemKey:      stringPtr(template.SystemKey),
			NameCustomized: false,
			DisplayOrder:   template.DisplayOrder,
		})
	}

	if len(categories) == 0 {
		return nil
	}

	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&categories).Error
}

func seedDefaultPaymentMethods(tx *gorm.DB, userID uint) error {
	methods := make([]model.PaymentMethod, 0, len(defaultPaymentMethodTemplates))
	for _, template := range defaultPaymentMethodTemplates {
		methods = append(methods, model.PaymentMethod{
			UserID:         userID,
			Name:           template.CanonicalName,
			SystemKey:      stringPtr(template.SystemKey),
			NameCustomized: false,
			Icon:           template.Icon,
			SortOrder:      template.SortOrder,
		})
	}

	if len(methods) == 0 {
		return nil
	}

	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&methods).Error
}

func seedDefaultCurrencies(tx *gorm.DB, userID uint) error {
	currencies := make([]model.UserCurrency, 0, len(defaultCurrencyTemplates))
	for _, template := range defaultCurrencyTemplates {
		currencies = append(currencies, model.UserCurrency{
			UserID:    userID,
			Code:      template.Code,
			Symbol:    template.Symbol,
			Alias:     "",
			SortOrder: template.SortOrder,
		})
	}

	if len(currencies) == 0 {
		return nil
	}

	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&currencies).Error
}

func seedDefaultPreference(tx *gorm.DB, userID uint) error {
	preference := model.UserPreference{
		UserID:            userID,
		PreferredCurrency: "USD",
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&preference).Error
}

func seedDefaultNotificationTemplate(tx *gorm.DB, userID uint) error {
	var count int64
	if err := tx.Model(&model.NotificationTemplate{}).
		Where("user_id = ? AND channel_type IS NULL", userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	tmpl := model.NotificationTemplate{
		UserID:      userID,
		Format:      "plaintext",
		Template:    defaultNotificationTemplate,
		ChannelType: nil,
	}
	return tx.Create(&tmpl).Error
}

func stringPtr(value string) *string {
	return &value
}
