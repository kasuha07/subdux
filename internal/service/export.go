package service

import (
	"errors"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

type ExportService struct {
	DB *gorm.DB
}

func NewExportService(db *gorm.DB) *ExportService {
	return &ExportService{DB: db}
}

type UserExportData struct {
	ExportedAt     time.Time                `json:"exported_at"`
	User           UserExportInfo           `json:"user"`
	Subscriptions  []model.Subscription     `json:"subscriptions"`
	Categories     []model.Category         `json:"categories"`
	PaymentMethods []model.PaymentMethod    `json:"payment_methods"`
	Currencies     []model.UserCurrency     `json:"currencies"`
	Preference     *model.UserPreference    `json:"preference"`
	Notifications  UserNotificationExport   `json:"notifications"`
}

type UserExportInfo struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type UserNotificationExport struct {
	Channels  []model.NotificationChannel  `json:"channels"`
	Policy    *model.NotificationPolicy    `json:"policy"`
	Templates []model.NotificationTemplate `json:"templates"`
}

func (s *ExportService) ExportUserData(userID uint) (*UserExportData, error) {
	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	var subs []model.Subscription
	if err := s.DB.Where("user_id = ?", userID).Find(&subs).Error; err != nil {
		return nil, err
	}
	if subs == nil {
		subs = []model.Subscription{}
	}

	var categories []model.Category
	if err := s.DB.Where("user_id = ?", userID).Find(&categories).Error; err != nil {
		return nil, err
	}
	if categories == nil {
		categories = []model.Category{}
	}

	var paymentMethods []model.PaymentMethod
	if err := s.DB.Where("user_id = ?", userID).Find(&paymentMethods).Error; err != nil {
		return nil, err
	}
	if paymentMethods == nil {
		paymentMethods = []model.PaymentMethod{}
	}

	var currencies []model.UserCurrency
	if err := s.DB.Where("user_id = ?", userID).Find(&currencies).Error; err != nil {
		return nil, err
	}
	if currencies == nil {
		currencies = []model.UserCurrency{}
	}

	var preference model.UserPreference
	var prefPtr *model.UserPreference
	if err := s.DB.Where("user_id = ?", userID).First(&preference).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else {
		prefPtr = &preference
	}

	var channels []model.NotificationChannel
	if err := s.DB.Where("user_id = ?", userID).Find(&channels).Error; err != nil {
		return nil, err
	}
	if channels == nil {
		channels = []model.NotificationChannel{}
	}

	var policy model.NotificationPolicy
	var policyPtr *model.NotificationPolicy
	if err := s.DB.Where("user_id = ?", userID).First(&policy).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else {
		policyPtr = &policy
	}

	var templates []model.NotificationTemplate
	if err := s.DB.Where("user_id = ?", userID).Find(&templates).Error; err != nil {
		return nil, err
	}
	if templates == nil {
		templates = []model.NotificationTemplate{}
	}

	var calTokens []model.CalendarToken
	if err := s.DB.Where("user_id = ?", userID).Find(&calTokens).Error; err != nil {
		return nil, err
	}
	for i := range calTokens {
		calTokens[i].MaskToken()
	}

	return &UserExportData{
		ExportedAt: time.Now().UTC(),
		User: UserExportInfo{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
		Subscriptions:  subs,
		Categories:     categories,
		PaymentMethods: paymentMethods,
		Currencies:     currencies,
		Preference:     prefPtr,
		Notifications: UserNotificationExport{
			Channels:  channels,
			Policy:    policyPtr,
			Templates: templates,
		},
	}, nil
}
