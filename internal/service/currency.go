package service

import (
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	"gorm.io/gorm"
)

type CurrencyService = catalogservice.CurrencyService
type CreateCurrencyInput = catalogservice.CreateCurrencyInput
type UpdateCurrencyInput = catalogservice.UpdateCurrencyInput
type ReorderItem = catalogservice.ReorderItem

func NewCurrencyService(db *gorm.DB) *CurrencyService {
	return catalogservice.NewCurrencyService(db)
}
