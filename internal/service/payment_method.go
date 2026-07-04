package service

import (
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	"gorm.io/gorm"
)

type PaymentMethodService = catalogservice.PaymentMethodService
type CreatePaymentMethodInput = catalogservice.CreatePaymentMethodInput
type UpdatePaymentMethodInput = catalogservice.UpdatePaymentMethodInput

func NewPaymentMethodService(db *gorm.DB) *PaymentMethodService {
	return catalogservice.NewPaymentMethodService(db)
}
