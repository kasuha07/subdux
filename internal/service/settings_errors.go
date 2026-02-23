package service

import "errors"

var (
	ErrCurrencyInUse       = errors.New("currency is in use by existing subscriptions")
	ErrCategoryInUse       = errors.New("category is in use by existing subscriptions")
	ErrPaymentMethodInUse  = errors.New("payment method is in use by existing subscriptions")
	ErrImageUploadDisabled = errors.New("image uploads are disabled by administrator")
)
