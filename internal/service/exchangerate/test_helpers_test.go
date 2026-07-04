package exchangerate

import (
	"context"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/servicetest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := servicetest.NewDB(t)
	if err := db.AutoMigrate(&model.ExchangeRate{}); err != nil {
		t.Fatalf("failed to migrate exchange rate table: %v", err)
	}
	return db
}

func seedSystemSetting(t *testing.T, db *gorm.DB, key string, value string) {
	t.Helper()
	if err := db.Where("key = ?", key).
		Assign(model.SystemSetting{Value: value}).
		FirstOrCreate(&model.SystemSetting{Key: key}).Error; err != nil {
		t.Fatalf("failed to seed setting %q: %v", key, err)
	}
}

func TestWithContextSharesStatefulCache(t *testing.T) {
	db := newTestDB(t)
	parent := NewService(db)

	clone := parent.WithContext(context.Background())
	if clone.cache != parent.cache {
		t.Fatal("WithContext duplicated the rate cache; clone must share the parent's *rateCache")
	}

	clone.cache.mu.Lock()
	clone.cache.rates[rateCacheKey("EUR")] = 0.9
	clone.cache.mu.Unlock()

	if got := parent.Convert(100, "USD", "EUR"); got != 90 {
		t.Fatalf("parent.Convert via shared cache = %v, want 90", got)
	}
}
