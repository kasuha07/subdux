package service

import (
	"context"

	"gorm.io/gorm"
)

// This file centralizes the context-binding helpers for services whose only
// mutable state is the GORM handle (plus immutable configuration such as a
// validator). Each WithContext returns a shallow copy of the service with its
// database handle bound to ctx via gorm.DB.WithContext, so that GORM propagates
// cancellation to in-flight SQL when the request context is cancelled — for
// example on client disconnect or when the HTTP write timeout fires.
//
// The copy is safe because these structs hold no locks or other by-value
// synchronization primitives; transactions and free helper functions that take
// a *gorm.DB inherit the bound context automatically from the session they are
// handed. Services that own in-memory caches or session stores (AuthService,
// ExchangeRateService, NotificationService) define WithContext alongside their
// own declarations so the shared state is preserved.

// withContext binds db to ctx, tolerating a nil handle so context-less unit
// tests that construct services without a database keep working.
func withContext(db *gorm.DB, ctx context.Context) *gorm.DB {
	if db == nil {
		return nil
	}
	return db.WithContext(ctx)
}

func (s *SubscriptionService) WithContext(ctx context.Context) *SubscriptionService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *CategoryService) WithContext(ctx context.Context) *CategoryService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *PaymentMethodService) WithContext(ctx context.Context) *PaymentMethodService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *CurrencyService) WithContext(ctx context.Context) *CurrencyService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *AdminService) WithContext(ctx context.Context) *AdminService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *AuditService) WithContext(ctx context.Context) *AuditService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *CalendarService) WithContext(ctx context.Context) *CalendarService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *ExportService) WithContext(ctx context.Context) *ExportService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *ImportService) WithContext(ctx context.Context) *ImportService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *TOTPService) WithContext(ctx context.Context) *TOTPService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *SystemSettingsService) WithContext(ctx context.Context) *SystemSettingsService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *IconProxyService) WithContext(ctx context.Context) *IconProxyService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *NotificationTemplateService) WithContext(ctx context.Context) *NotificationTemplateService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *APIKeyService) WithContext(ctx context.Context) *APIKeyService {
	clone := *s
	clone.db = withContext(s.db, ctx)
	return &clone
}
