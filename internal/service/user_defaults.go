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
	{SystemKey: "other", CanonicalName: "Other", DisplayOrder: 4},
}

var defaultPaymentMethodTemplates = []defaultPaymentMethodTemplate{
	{SystemKey: "alipay", CanonicalName: "支付宝", Icon: "custom:alipay", SortOrder: 0},
	{SystemKey: "wechatpay", CanonicalName: "微信", Icon: "custom:wechatpay", SortOrder: 1},
	{SystemKey: "paypal", CanonicalName: "Paypal", Icon: "lg:paypal", SortOrder: 2},
}

var defaultCurrencyTemplates = []defaultCurrencyTemplate{
	{Code: "CNY", Symbol: "¥", SortOrder: 0},
	{Code: "USD", Symbol: "$", SortOrder: 1},
	{Code: "EUR", Symbol: "€", SortOrder: 2},
	{Code: "GBP", Symbol: "£", SortOrder: 3},
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
		PreferredCurrency: "CNY",
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
