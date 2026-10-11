package importer

import (
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/servicetest"
)

func TestImportFromSubduxKeepsCompleteScheduledPrice(t *testing.T) {
	db := newImportTestDB(t)
	user := servicetest.CreateUser(t, db)
	svc := NewService(db)
	data := sampleSubduxImportData()
	pendingAmount := 30.0
	pendingFrom := time.Date(2099, 1, 15, 0, 0, 0, 0, time.UTC)
	data.Subscriptions[0].PendingAmount = &pendingAmount
	data.Subscriptions[0].PendingFrom = &pendingFrom

	if _, err := svc.ImportFromSubdux(user.ID, data, true); err != nil {
		t.Fatalf("ImportFromSubdux() error = %v", err)
	}

	var imported model.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&imported).Error; err != nil {
		t.Fatalf("load imported subscription: %v", err)
	}
	if imported.PendingAmount == nil || *imported.PendingAmount != 30 ||
		imported.PendingFrom == nil || !imported.PendingFrom.Equal(pendingFrom) {
		t.Fatalf("imported pending price = %v/%v, want 30 from %v", imported.PendingAmount, imported.PendingFrom, pendingFrom)
	}
}

func TestImportFromSubduxDropsIncompleteOrInvalidScheduledPrice(t *testing.T) {
	pendingFrom := time.Date(2099, 1, 15, 0, 0, 0, 0, time.UTC)
	negative := -1.0
	valid := 30.0
	for name, tc := range map[string]struct {
		amount *float64
		from   *time.Time
	}{
		"missing date":    {amount: &valid},
		"missing amount":  {from: &pendingFrom},
		"negative amount": {amount: &negative, from: &pendingFrom},
	} {
		t.Run(name, func(t *testing.T) {
			db := newImportTestDB(t)
			user := servicetest.CreateUser(t, db)
			data := sampleSubduxImportData()
			data.Subscriptions[0].PendingAmount = tc.amount
			data.Subscriptions[0].PendingFrom = tc.from

			if _, err := NewService(db).ImportFromSubdux(user.ID, data, true); err != nil {
				t.Fatalf("ImportFromSubdux() error = %v", err)
			}
			var imported model.Subscription
			if err := db.Where("user_id = ?", user.ID).First(&imported).Error; err != nil {
				t.Fatalf("subscription was not imported: %v", err)
			}
			if imported.PendingAmount != nil || imported.PendingFrom != nil {
				t.Fatalf("imported pending price = %v/%v, want dropped", imported.PendingAmount, imported.PendingFrom)
			}
		})
	}
}
