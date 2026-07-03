package service

import (
	"errors"

	"github.com/kasuha07/subdux/internal/service/serviceutil"
)

var (
	ErrCurrencyInUse       = errors.New("currency is in use by existing subscriptions")
	ErrCategoryInUse       = errors.New("category is in use by existing subscriptions")
	ErrPaymentMethodInUse  = errors.New("payment method is in use by existing subscriptions")
	ErrImageUploadDisabled = serviceutil.ErrImageUploadDisabled
)
