package service

import (
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
)

var (
	ErrCurrencyInUse       = catalogservice.ErrCurrencyInUse
	ErrCategoryInUse       = catalogservice.ErrCategoryInUse
	ErrPaymentMethodInUse  = catalogservice.ErrPaymentMethodInUse
	ErrImageUploadDisabled = serviceutil.ErrImageUploadDisabled
)
