package calendar

import (
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/servicetest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := servicetest.NewDB(t)
	if err := db.AutoMigrate(&model.CalendarToken{}); err != nil {
		t.Fatalf("failed to migrate calendar tokens: %v", err)
	}
	return db
}

func createTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()
	return servicetest.CreateUser(t, db)
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("parse date %q: %v", value, err)
	}
	return parsed
}
