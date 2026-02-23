package service

import (
	"errors"
	"strings"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

type CurrencyService struct {
	DB *gorm.DB
}

func NewCurrencyService(db *gorm.DB) *CurrencyService {
	return &CurrencyService{DB: db}
}

type CreateCurrencyInput struct {
	Code      string `json:"code"`
	Symbol    string `json:"symbol"`
	Alias     string `json:"alias"`
	SortOrder int    `json:"sort_order"`
}

type UpdateCurrencyInput struct {
	Symbol    *string `json:"symbol"`
	Alias     *string `json:"alias"`
	SortOrder *int    `json:"sort_order"`
}

type ReorderItem struct {
	ID        uint `json:"id"`
	SortOrder int  `json:"sort_order"`
}

func (s *CurrencyService) List(userID uint) ([]model.UserCurrency, error) {
	var currencies []model.UserCurrency
	err := s.DB.Where("user_id = ?", userID).Order("sort_order ASC, id ASC").Find(&currencies).Error
	return currencies, err
}

func (s *CurrencyService) Create(userID uint, input CreateCurrencyInput) (*model.UserCurrency, error) {
	code := strings.ToUpper(strings.TrimSpace(input.Code))
	if code == "" || len(code) > 10 {
		return nil, errors.New("code must be 1-10 characters")
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return nil, errors.New("code must contain only uppercase letters")
		}
	}

	var existing model.UserCurrency
	err := s.DB.Where("user_id = ? AND code = ?", userID, code).First(&existing).Error
	if err == nil {
		return nil, errors.New("currency code already exists")
	}

	currency := model.UserCurrency{
		UserID:    userID,
		Code:      code,
		Symbol:    strings.TrimSpace(input.Symbol),
		Alias:     strings.TrimSpace(input.Alias),
		SortOrder: input.SortOrder,
	}

	if err := s.DB.Create(&currency).Error; err != nil {
		return nil, err
	}
	return &currency, nil
}

func (s *CurrencyService) Update(userID, id uint, input UpdateCurrencyInput) (*model.UserCurrency, error) {
	var currency model.UserCurrency
	if err := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&currency).Error; err != nil {
		return nil, errors.New("currency not found")
	}
	if input.Symbol != nil {
		currency.Symbol = strings.TrimSpace(*input.Symbol)
	}
	if input.Alias != nil {
		currency.Alias = strings.TrimSpace(*input.Alias)
	}
	if input.SortOrder != nil {
		currency.SortOrder = *input.SortOrder
	}
	if err := s.DB.Save(&currency).Error; err != nil {
		return nil, err
	}
	return &currency, nil
}

func (s *CurrencyService) Delete(userID, id uint, preferredCurrency string) error {
	var currency model.UserCurrency
	if err := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&currency).Error; err != nil {
		return errors.New("currency not found")
	}
	if strings.EqualFold(currency.Code, preferredCurrency) {
		return errors.New("cannot delete your preferred currency")
	}

	var subscriptionsUsingCurrency int64
	if err := s.DB.Model(&model.Subscription{}).
		Where("user_id = ? AND UPPER(currency) = ?", userID, strings.ToUpper(currency.Code)).
		Count(&subscriptionsUsingCurrency).Error; err != nil {
		return err
	}
	if subscriptionsUsingCurrency > 0 {
		return ErrCurrencyInUse
	}

	return s.DB.Delete(&currency).Error
}

func (s *CurrencyService) Reorder(userID uint, items []ReorderItem) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&model.UserCurrency{}).
				Where("id = ? AND user_id = ?", item.ID, userID).
				Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
