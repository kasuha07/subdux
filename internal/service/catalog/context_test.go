package catalog_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	"gorm.io/gorm"
)

// newContextCancellationTestDB provisions an isolated SQLite database with the
// handful of tables touched by the services exercised below.
func newContextCancellationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-context-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.Category{}, &model.PaymentMethod{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

// TestWithContextCancelsInFlightQueries proves that binding a request context to
// a service via WithContext propagates cancellation down to GORM: once the
// context is cancelled, queries fail fast with context.Canceled instead of
// running to completion. This is the behavior that lets a client disconnect or
// the HTTP write timeout actually abort database work.
func TestWithContextCancelsInFlightQueries(t *testing.T) {
	db := newContextCancellationTestDB(t)

	const userID = uint(1)
	if err := db.Create(&model.Category{UserID: userID, Name: "Streaming"}).Error; err != nil {
		t.Fatalf("failed to seed category: %v", err)
	}

	svc := catalogservice.NewCategoryService(db)

	// Baseline: a live context returns the seeded data without error.
	liveCategories, err := svc.WithContext(context.Background()).List(userID)
	if err != nil {
		t.Fatalf("List() with live context returned error: %v", err)
	}
	if len(liveCategories) != 1 {
		t.Fatalf("List() with live context returned %d categories, want 1", len(liveCategories))
	}

	// Cancelled context: the bound session must refuse to run the query.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := svc.WithContext(ctx).List(userID); !errors.Is(err, context.Canceled) {
		t.Fatalf("List() with cancelled context error = %v, want context.Canceled", err)
	}

	// A write path must be cancelled too, so slow writes under the 60s write
	// timeout can be interrupted rather than committed after the client is gone.
	_, err = svc.WithContext(ctx).Create(userID, catalogservice.CreateCategoryInput{Name: "Should Not Persist"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() with cancelled context error = %v, want context.Canceled", err)
	}

	// The cancelled write must not have persisted: a fresh, live query still
	// sees only the seeded row.
	remaining, err := svc.WithContext(context.Background()).List(userID)
	if err != nil {
		t.Fatalf("List() after cancelled write returned error: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("cancelled Create() leaked a row: got %d categories, want 1", len(remaining))
	}
}

// TestWithContextPreservesParentHandle confirms WithContext returns an
// independent shallow copy: rebinding the context must not mutate the original
// service's database handle, so concurrent requests cannot disturb one another.
func TestWithContextPreservesParentHandle(t *testing.T) {
	db := newContextCancellationTestDB(t)
	svc := catalogservice.NewPaymentMethodService(db)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Bind a cancelled context to a clone.
	_ = svc.WithContext(ctx)

	// The parent service must still operate on a live handle.
	if _, err := svc.WithContext(context.Background()).List(uint(1)); err != nil {
		t.Fatalf("parent service handle was disturbed by clone: %v", err)
	}
}
