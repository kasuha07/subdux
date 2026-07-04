package admin

import (
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/servicetest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := servicetest.NewDB(t)
	if err := db.AutoMigrate(
		&model.APIKey{},
		&model.CalendarToken{},
		&model.NotificationOutbox{},
		&model.UserBackupCode{},
	); err != nil {
		t.Fatalf("failed to migrate admin test tables: %v", err)
	}
	return db
}
