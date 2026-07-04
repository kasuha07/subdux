package service

import (
	idempotencyservice "github.com/kasuha07/subdux/internal/service/idempotency"
	"gorm.io/gorm"
)

type IdempotencyService = idempotencyservice.Service

func NewIdempotencyService(db *gorm.DB) *IdempotencyService {
	return idempotencyservice.NewService(db)
}
