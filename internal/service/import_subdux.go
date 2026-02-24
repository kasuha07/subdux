package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

var ErrInvalidSubduxImportFormat = errors.New("invalid subdux export format")
var ErrSubduxImportTooLarge = errors.New("subdux export file is too large")

const maxSubduxImportItemsPerCollection = 5000

type SubduxImportRequest struct {
	Data    SubduxImportData `json:"data"`
	Confirm bool             `json:"confirm"`
}

type SubduxImportData struct {
	Currencies     []model.UserCurrency         `json:"currencies"`
	Categories     []model.Category             `json:"categories"`
	PaymentMethods []model.PaymentMethod        `json:"payment_methods"`
	Subscriptions  []model.Subscription         `json:"subscriptions"`
	Preference     *model.UserPreference        `json:"preference"`
	Notifications  SubduxNotificationImportData `json:"notifications"`
}

type SubduxNotificationImportData struct {
	Channels  []model.NotificationChannel  `json:"channels"`
	Templates []model.NotificationTemplate `json:"templates"`
	Policy    *model.NotificationPolicy    `json:"policy"`
}

type PreviewChannelChange struct {
	Type   string `json:"type"`
	IsNew  bool   `json:"is_new"`
	Config string `json:"config"`
}

type PreviewTemplateChange struct {
	ChannelType string `json:"channel_type"`
	Format      string `json:"format"`
	IsNew       bool   `json:"is_new"`
}

type PreviewPreferenceChange struct {
	WillCreate bool   `json:"will_create"`
	WillUpdate bool   `json:"will_update"`
	Current    string `json:"current"`
	Incoming   string `json:"incoming"`
}

type PreviewNotificationPolicyChange struct {
	WillCreate     bool `json:"will_create"`
	WillUpdate     bool `json:"will_update"`
	CurrentDays    int  `json:"current_days_before"`
	IncomingDays   int  `json:"incoming_days_before"`
	CurrentDueDay  bool `json:"current_notify_on_due_day"`
	IncomingDueDay bool `json:"incoming_notify_on_due_day"`
}

type SubduxImportPreview struct {
	Currencies     []PreviewCurrencyChange          `json:"currencies"`
	PaymentMethods []PreviewPaymentMethodChange     `json:"payment_methods"`
	Categories     []PreviewCategoryChange          `json:"categories"`
	Subscriptions  []PreviewSubscriptionChange      `json:"subscriptions"`
	Channels       []PreviewChannelChange           `json:"channels"`
	Templates      []PreviewTemplateChange          `json:"templates"`
	Preference     *PreviewPreferenceChange         `json:"preference,omitempty"`
	Policy         *PreviewNotificationPolicyChange `json:"policy,omitempty"`
}

type SubduxImportResponse struct {
	Preview *SubduxImportPreview `json:"preview,omitempty"`
	Result  *ImportResult        `json:"result,omitempty"`
}

func validateSubduxImportData(data SubduxImportData) error {
	if data.Currencies == nil || data.Categories == nil || data.PaymentMethods == nil || data.Subscriptions == nil {
		return ErrInvalidSubduxImportFormat
	}

	if len(data.Currencies) > maxSubduxImportItemsPerCollection ||
		len(data.Categories) > maxSubduxImportItemsPerCollection ||
		len(data.PaymentMethods) > maxSubduxImportItemsPerCollection ||
		len(data.Subscriptions) > maxSubduxImportItemsPerCollection ||
		len(data.Notifications.Channels) > maxSubduxImportItemsPerCollection ||
		len(data.Notifications.Templates) > maxSubduxImportItemsPerCollection {
		return ErrSubduxImportTooLarge
	}

	return nil
}

func canonicalChannelConfig(config string) string {
	trimmed := strings.TrimSpace(config)
	if trimmed == "" {
		return ""
	}

	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return trimmed
	}

	encoded, err := json.Marshal(parsed)
	if err != nil {
		return trimmed
	}
	return string(encoded)
}

func ptrIntSignature(value *int) string {
	if value == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *value)
}

func recurrenceSignature(sub model.Subscription) string {
	return strings.Join([]string{
		strings.ToLower(strings.TrimSpace(sub.RecurrenceType)),
		strings.ToLower(strings.TrimSpace(sub.IntervalUnit)),
		ptrIntSignature(sub.IntervalCount),
		ptrIntSignature(sub.MonthlyDay),
		ptrIntSignature(sub.YearlyMonth),
		ptrIntSignature(sub.YearlyDay),
	}, "|")
}

func findCategoryBySystemKeyOrName(tx *gorm.DB, userID uint, incoming model.Category) (*model.Category, error) {
	var category model.Category
	if incoming.SystemKey != nil && strings.TrimSpace(*incoming.SystemKey) != "" {
		err := tx.Where("user_id = ? AND system_key = ?", userID, strings.TrimSpace(*incoming.SystemKey)).First(&category).Error
		if err == nil {
			return &category, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	name := strings.TrimSpace(incoming.Name)
	if name == "" {
		return nil, nil
	}

	err := tx.Where("user_id = ? AND LOWER(name) = ?", userID, strings.ToLower(name)).First(&category).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func findPaymentMethodBySystemKeyOrName(tx *gorm.DB, userID uint, incoming model.PaymentMethod) (*model.PaymentMethod, error) {
	var paymentMethod model.PaymentMethod
	if incoming.SystemKey != nil && strings.TrimSpace(*incoming.SystemKey) != "" {
		err := tx.Where("user_id = ? AND system_key = ?", userID, strings.TrimSpace(*incoming.SystemKey)).First(&paymentMethod).Error
		if err == nil {
			return &paymentMethod, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	name := strings.TrimSpace(incoming.Name)
	if name == "" {
		return nil, nil
	}

	err := tx.Where("user_id = ? AND LOWER(name) = ?", userID, strings.ToLower(name)).First(&paymentMethod).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &paymentMethod, nil
}

func subscriptionExists(tx *gorm.DB, userID uint, sub model.Subscription) (bool, error) {
	query := tx.Model(&model.Subscription{}).
		Where("user_id = ? AND name = ? AND amount = ? AND currency = ? AND billing_type = ? AND recurrence_type = ? AND interval_unit = ?",
			userID,
			strings.TrimSpace(sub.Name),
			sub.Amount,
			strings.TrimSpace(sub.Currency),
			strings.TrimSpace(sub.BillingType),
			strings.TrimSpace(sub.RecurrenceType),
			strings.TrimSpace(sub.IntervalUnit),
		)

	if sub.IntervalCount == nil {
		query = query.Where("interval_count IS NULL")
	} else {
		query = query.Where("interval_count = ?", *sub.IntervalCount)
	}

	if sub.MonthlyDay == nil {
		query = query.Where("monthly_day IS NULL")
	} else {
		query = query.Where("monthly_day = ?", *sub.MonthlyDay)
	}

	if sub.YearlyMonth == nil {
		query = query.Where("yearly_month IS NULL")
	} else {
		query = query.Where("yearly_month = ?", *sub.YearlyMonth)
	}

	if sub.YearlyDay == nil {
		query = query.Where("yearly_day IS NULL")
	} else {
		query = query.Where("yearly_day = ?", *sub.YearlyDay)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func templateKey(template model.NotificationTemplate) string {
	channelType := ""
	if template.ChannelType != nil {
		channelType = strings.TrimSpace(*template.ChannelType)
	}
	return strings.Join([]string{
		strings.ToLower(channelType),
		strings.ToLower(strings.TrimSpace(template.Format)),
		strings.TrimSpace(template.Template),
	}, "|")
}

func (s *ImportService) ImportFromSubdux(userID uint, data SubduxImportData, confirm bool) (*SubduxImportResponse, error) {
	if err := validateSubduxImportData(data); err != nil {
		return nil, err
	}

	validator := NewTemplateValidator()
	result := &ImportResult{Errors: []string{}}
	preview := &SubduxImportPreview{
		Currencies:     []PreviewCurrencyChange{},
		PaymentMethods: []PreviewPaymentMethodChange{},
		Categories:     []PreviewCategoryChange{},
		Subscriptions:  []PreviewSubscriptionChange{},
		Channels:       []PreviewChannelChange{},
		Templates:      []PreviewTemplateChange{},
	}

	seenCurrencies := map[string]bool{}
	seenCategories := map[string]bool{}
	seenPaymentMethods := map[string]bool{}
	seenSubscriptions := map[string]bool{}
	seenChannels := map[string]bool{}
	seenTemplates := map[string]bool{}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		categoryIDMap := map[uint]uint{}
		paymentMethodIDMap := map[uint]uint{}

		for _, incoming := range data.Currencies {
			code := strings.ToUpper(strings.TrimSpace(incoming.Code))
			if code == "" || seenCurrencies[code] {
				continue
			}
			seenCurrencies[code] = true

			var existing model.UserCurrency
			err := tx.Where("user_id = ? AND code = ?", userID, code).First(&existing).Error
			isNew := err == gorm.ErrRecordNotFound
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}

			preview.Currencies = append(preview.Currencies, PreviewCurrencyChange{
				Code:   code,
				Symbol: strings.TrimSpace(incoming.Symbol),
				IsNew:  isNew,
			})

			if !confirm || !isNew {
				if confirm && !isNew {
					result.Skipped++
				}
				continue
			}

			created := model.UserCurrency{
				UserID:    userID,
				Code:      code,
				Symbol:    strings.TrimSpace(incoming.Symbol),
				Alias:     strings.TrimSpace(incoming.Alias),
				SortOrder: incoming.SortOrder,
			}
			if err := tx.Create(&created).Error; err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to create currency %q: %v", code, err))
				continue
			}
			result.Imported++
		}

		for _, incoming := range data.Categories {
			name := strings.TrimSpace(incoming.Name)
			if name == "" {
				if confirm {
					result.Skipped++
				}
				continue
			}

			key := strings.ToLower(name)
			if incoming.SystemKey != nil && strings.TrimSpace(*incoming.SystemKey) != "" {
				key = "system:" + strings.ToLower(strings.TrimSpace(*incoming.SystemKey))
			}
			if seenCategories[key] {
				continue
			}
			seenCategories[key] = true

			existing, err := findCategoryBySystemKeyOrName(tx, userID, incoming)
			if err != nil {
				return err
			}
			isNew := existing == nil

			preview.Categories = append(preview.Categories, PreviewCategoryChange{
				Name:  name,
				IsNew: isNew,
			})

			if incoming.ID != 0 && existing != nil {
				categoryIDMap[incoming.ID] = existing.ID
			}

			if !confirm || !isNew {
				if confirm && !isNew {
					result.Skipped++
				}
				continue
			}

			created := model.Category{
				UserID:         userID,
				Name:           name,
				SystemKey:      incoming.SystemKey,
				NameCustomized: incoming.NameCustomized,
				DisplayOrder:   incoming.DisplayOrder,
			}
			if err := tx.Create(&created).Error; err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to create category %q: %v", name, err))
				continue
			}
			if incoming.ID != 0 {
				categoryIDMap[incoming.ID] = created.ID
			}
			result.Imported++
		}

		for _, incoming := range data.PaymentMethods {
			name := strings.TrimSpace(incoming.Name)
			if name == "" {
				if confirm {
					result.Skipped++
				}
				continue
			}

			key := strings.ToLower(name)
			if incoming.SystemKey != nil && strings.TrimSpace(*incoming.SystemKey) != "" {
				key = "system:" + strings.ToLower(strings.TrimSpace(*incoming.SystemKey))
			}
			if seenPaymentMethods[key] {
				continue
			}
			seenPaymentMethods[key] = true

			existing, err := findPaymentMethodBySystemKeyOrName(tx, userID, incoming)
			if err != nil {
				return err
			}
			isNew := existing == nil

			preview.PaymentMethods = append(preview.PaymentMethods, PreviewPaymentMethodChange{
				Name:  name,
				IsNew: isNew,
				Matched: func() string {
					if existing == nil {
						return ""
					}
					return existing.Name
				}(),
			})

			if incoming.ID != 0 && existing != nil {
				paymentMethodIDMap[incoming.ID] = existing.ID
			}

			if !confirm || !isNew {
				if confirm && !isNew {
					result.Skipped++
				}
				continue
			}

			created := model.PaymentMethod{
				UserID:         userID,
				Name:           name,
				SystemKey:      incoming.SystemKey,
				NameCustomized: incoming.NameCustomized,
				Icon:           incoming.Icon,
				SortOrder:      incoming.SortOrder,
			}
			if err := tx.Create(&created).Error; err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to create payment method %q: %v", name, err))
				continue
			}
			if incoming.ID != 0 {
				paymentMethodIDMap[incoming.ID] = created.ID
			}
			result.Imported++
		}

		for _, incoming := range data.Notifications.Channels {
			channelType := strings.ToLower(strings.TrimSpace(incoming.Type))
			canonicalConfig := canonicalChannelConfig(incoming.Config)
			if channelType == "" {
				if confirm {
					result.Skipped++
				}
				continue
			}

			if !isValidChannelType(channelType) {
				if confirm {
					result.Errors = append(result.Errors, fmt.Sprintf("unsupported notification channel type %q", channelType))
					result.Skipped++
				}
				continue
			}

			if err := validateChannelConfig(channelType, canonicalConfig); err != nil {
				if confirm {
					result.Errors = append(result.Errors, fmt.Sprintf("invalid config for channel %q: %v", channelType, err))
					result.Skipped++
				}
				continue
			}

			key := strings.ToLower(channelType) + "|" + canonicalConfig
			if seenChannels[key] {
				continue
			}
			seenChannels[key] = true

			var existingChannels []model.NotificationChannel
			if err := tx.Where("user_id = ? AND type = ?", userID, channelType).Find(&existingChannels).Error; err != nil {
				return err
			}

			isNew := true
			for _, existing := range existingChannels {
				if canonicalChannelConfig(existing.Config) == canonicalConfig {
					isNew = false
					break
				}
			}

			preview.Channels = append(preview.Channels, PreviewChannelChange{
				Type:   channelType,
				IsNew:  isNew,
				Config: canonicalConfig,
			})

			if !confirm || !isNew {
				if confirm && !isNew {
					result.Skipped++
				}
				continue
			}

			created := model.NotificationChannel{
				UserID:  userID,
				Type:    channelType,
				Enabled: incoming.Enabled,
				Config:  canonicalConfig,
			}
			if err := tx.Create(&created).Error; err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to create notification channel %q: %v", channelType, err))
				continue
			}
			result.Imported++
		}

		for _, incoming := range data.Notifications.Templates {
			templateText := strings.TrimSpace(incoming.Template)
			if templateText == "" {
				if confirm {
					result.Skipped++
				}
				continue
			}

			var channelTypeVal *string
			if incoming.ChannelType != nil {
				trimmed := strings.TrimSpace(*incoming.ChannelType)
				if trimmed != "" {
					if !isValidChannelType(trimmed) {
						if confirm {
							result.Errors = append(result.Errors, fmt.Sprintf("unsupported notification template channel type %q", trimmed))
							result.Skipped++
						}
						continue
					}
					channelTypeVal = &trimmed
				}
			}
			format := strings.ToLower(strings.TrimSpace(incoming.Format))
			if format == "" {
				format = "plaintext"
			}
			if err := validator.ValidateFormat(format); err != nil {
				if confirm {
					result.Errors = append(result.Errors, fmt.Sprintf("invalid notification template format %q: %v", format, err))
					result.Skipped++
				}
				continue
			}
			if err := validator.ValidateTemplate(templateText); err != nil {
				if confirm {
					result.Errors = append(result.Errors, fmt.Sprintf("invalid notification template: %v", err))
					result.Skipped++
				}
				continue
			}

			templateForKey := model.NotificationTemplate{
				ChannelType: channelTypeVal,
				Format:      format,
				Template:    templateText,
			}
			key := templateKey(templateForKey)
			if seenTemplates[key] {
				continue
			}
			seenTemplates[key] = true

			query := tx.Where("user_id = ? AND format = ? AND template = ?", userID, format, templateText)
			if channelTypeVal == nil {
				query = query.Where("channel_type IS NULL")
			} else {
				query = query.Where("channel_type = ?", *channelTypeVal)
			}

			var existing model.NotificationTemplate
			err := query.First(&existing).Error
			isNew := err == gorm.ErrRecordNotFound
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}

			preview.Templates = append(preview.Templates, PreviewTemplateChange{
				ChannelType: func() string {
					if channelTypeVal == nil {
						return ""
					}
					return *channelTypeVal
				}(),
				Format: format,
				IsNew:  isNew,
			})

			if !confirm || !isNew {
				if confirm && !isNew {
					result.Skipped++
				}
				continue
			}

			created := model.NotificationTemplate{
				UserID:      userID,
				ChannelType: channelTypeVal,
				Format:      format,
				Template:    templateText,
			}
			if err := tx.Create(&created).Error; err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to create notification template: %v", err))
				continue
			}
			result.Imported++
		}

		for _, incoming := range data.Subscriptions {
			name := strings.TrimSpace(incoming.Name)
			if name == "" {
				if confirm {
					result.Skipped++
				}
				continue
			}

			incoming.Name = name
			incoming.Currency = strings.ToUpper(strings.TrimSpace(incoming.Currency))
			incoming.BillingType = strings.ToLower(strings.TrimSpace(incoming.BillingType))
			incoming.RecurrenceType = strings.ToLower(strings.TrimSpace(incoming.RecurrenceType))
			incoming.IntervalUnit = strings.ToLower(strings.TrimSpace(incoming.IntervalUnit))
			incoming.Category = strings.TrimSpace(incoming.Category)

			dedupKey := strings.Join([]string{
				strings.ToLower(incoming.Name),
				fmt.Sprintf("%f", incoming.Amount),
				strings.ToUpper(incoming.Currency),
				strings.ToLower(incoming.BillingType),
				recurrenceSignature(incoming),
			}, "|")

			isDuplicate := seenSubscriptions[dedupKey]
			if !isDuplicate {
				exists, err := subscriptionExists(tx, userID, incoming)
				if err != nil {
					return err
				}
				isDuplicate = exists
			}
			seenSubscriptions[dedupKey] = true

			previewSub := PreviewSubscriptionChange{
				Name:        incoming.Name,
				Amount:      incoming.Amount,
				Currency:    incoming.Currency,
				BillingType: incoming.BillingType,
				Category:    incoming.Category,
				Skipped:     isDuplicate,
			}
			if isDuplicate {
				previewSub.SkipReason = "duplicate"
			}
			preview.Subscriptions = append(preview.Subscriptions, previewSub)

			if !confirm || isDuplicate {
				if confirm && isDuplicate {
					result.Skipped++
				}
				continue
			}

			var categoryID *uint
			if incoming.CategoryID != nil {
				if mapped, ok := categoryIDMap[*incoming.CategoryID]; ok {
					categoryID = &mapped
				}
			}
			if categoryID == nil && incoming.Category != "" {
				var cat model.Category
				if err := tx.Where("user_id = ? AND LOWER(name) = ?", userID, strings.ToLower(incoming.Category)).First(&cat).Error; err == nil {
					categoryID = &cat.ID
				}
			}

			var paymentMethodID *uint
			if incoming.PaymentMethodID != nil {
				if mapped, ok := paymentMethodIDMap[*incoming.PaymentMethodID]; ok {
					paymentMethodID = &mapped
				}
			}

			created := model.Subscription{
				UserID:           userID,
				Name:             incoming.Name,
				Amount:           incoming.Amount,
				Currency:         incoming.Currency,
				Enabled:          incoming.Enabled,
				BillingType:      incoming.BillingType,
				RecurrenceType:   incoming.RecurrenceType,
				IntervalCount:    incoming.IntervalCount,
				IntervalUnit:     incoming.IntervalUnit,
				MonthlyDay:       incoming.MonthlyDay,
				YearlyMonth:      incoming.YearlyMonth,
				YearlyDay:        incoming.YearlyDay,
				NextBillingDate:  incoming.NextBillingDate,
				Category:         incoming.Category,
				CategoryID:       categoryID,
				PaymentMethodID:  paymentMethodID,
				NotifyEnabled:    incoming.NotifyEnabled,
				NotifyDaysBefore: incoming.NotifyDaysBefore,
				Icon:             incoming.Icon,
				URL:              incoming.URL,
				Notes:            incoming.Notes,
			}

			if err := tx.Create(&created).Error; err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to create subscription %q: %v", incoming.Name, err))
				continue
			}
			result.Imported++
		}

		if data.Preference != nil {
			incomingCurrency := strings.ToUpper(strings.TrimSpace(data.Preference.PreferredCurrency))
			if incomingCurrency != "" {
				var existing model.UserPreference
				err := tx.Where("user_id = ?", userID).First(&existing).Error
				willCreate := err == gorm.ErrRecordNotFound
				if err != nil && err != gorm.ErrRecordNotFound {
					return err
				}

				willUpdate := !willCreate && existing.PreferredCurrency != incomingCurrency
				preview.Preference = &PreviewPreferenceChange{
					WillCreate: willCreate,
					WillUpdate: willUpdate,
					Current: func() string {
						if willCreate {
							return ""
						}
						return existing.PreferredCurrency
					}(),
					Incoming: incomingCurrency,
				}

				if confirm {
					switch {
					case willCreate:
						created := model.UserPreference{UserID: userID, PreferredCurrency: incomingCurrency}
						if err := tx.Create(&created).Error; err != nil {
							result.Errors = append(result.Errors, fmt.Sprintf("failed to create preference: %v", err))
						} else {
							result.Imported++
						}
					case willUpdate:
						if err := tx.Model(&existing).Update("preferred_currency", incomingCurrency).Error; err != nil {
							result.Errors = append(result.Errors, fmt.Sprintf("failed to update preference: %v", err))
						} else {
							result.Imported++
						}
					default:
						result.Skipped++
					}
				}
			}
		}

		if data.Notifications.Policy != nil {
			incomingPolicy := data.Notifications.Policy
			if incomingPolicy.DaysBefore < 0 || incomingPolicy.DaysBefore > maxNotificationDaysBefore {
				if confirm {
					result.Errors = append(result.Errors, fmt.Sprintf("days_before must be between 0 and %d", maxNotificationDaysBefore))
					result.Skipped++
				}
			} else {
				var existing model.NotificationPolicy
				err := tx.Where("user_id = ?", userID).First(&existing).Error
				willCreate := err == gorm.ErrRecordNotFound
				if err != nil && err != gorm.ErrRecordNotFound {
					return err
				}

				willUpdate := !willCreate && (existing.DaysBefore != incomingPolicy.DaysBefore || existing.NotifyOnDueDay != incomingPolicy.NotifyOnDueDay)
				preview.Policy = &PreviewNotificationPolicyChange{
					WillCreate:     willCreate,
					WillUpdate:     willUpdate,
					CurrentDays:    existing.DaysBefore,
					IncomingDays:   incomingPolicy.DaysBefore,
					CurrentDueDay:  existing.NotifyOnDueDay,
					IncomingDueDay: incomingPolicy.NotifyOnDueDay,
				}

				if confirm {
					switch {
					case willCreate:
						created := model.NotificationPolicy{
							UserID:         userID,
							DaysBefore:     incomingPolicy.DaysBefore,
							NotifyOnDueDay: incomingPolicy.NotifyOnDueDay,
						}
						if err := tx.Create(&created).Error; err != nil {
							result.Errors = append(result.Errors, fmt.Sprintf("failed to create policy: %v", err))
						} else {
							result.Imported++
						}
					case willUpdate:
						updates := map[string]any{
							"days_before":       incomingPolicy.DaysBefore,
							"notify_on_due_day": incomingPolicy.NotifyOnDueDay,
						}
						if err := tx.Model(&existing).Updates(updates).Error; err != nil {
							result.Errors = append(result.Errors, fmt.Sprintf("failed to update policy: %v", err))
						} else {
							result.Imported++
						}
					default:
						result.Skipped++
					}
				}
			}
		}

		if !confirm {
			return errPreviewRollback
		}
		return nil
	})

	if err != nil && err != errPreviewRollback {
		return nil, err
	}

	if confirm {
		return &SubduxImportResponse{Result: result}, nil
	}
	return &SubduxImportResponse{Preview: preview}, nil
}
