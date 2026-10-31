package idempotency

import (
	"fmt"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/servicetest"
	"gorm.io/gorm"
)

func newIdempotencyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := servicetest.NewDB(t)
	if err := db.AutoMigrate(&model.MCPIdempotencyKey{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestLookupTreatsExpiredRecordAsAbsent(t *testing.T) {
	db := newIdempotencyTestDB(t)
	user := servicetest.CreateUser(t, db)
	service := NewService(db)

	old := model.MCPIdempotencyKey{UserID: user.ID, IdempotencyKey: "k", ToolName: "update_subscription", RequestHash: "h", CreatedAt: time.Now().Add(-RetentionWindow - time.Minute)}
	if err := db.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	record, err := service.Lookup(user.ID, "k")
	if err != nil || record != nil {
		t.Fatalf("Lookup() = %+v, %v; want expired record treated as absent", record, err)
	}
	// The key is reusable: saving it again does not hit the unique index.
	if err := service.Save(&model.MCPIdempotencyKey{UserID: user.ID, IdempotencyKey: "k", ToolName: "update_subscription", RequestHash: "h2"}); err != nil {
		t.Fatalf("Save() after expiry error = %v", err)
	}
	record, err = service.Lookup(user.ID, "k")
	if err != nil || record == nil || record.RequestHash != "h2" {
		t.Fatalf("Lookup() = %+v, %v; want the fresh record", record, err)
	}
}

func TestSaveEvictsOldestRecordsBeyondPerUserCap(t *testing.T) {
	db := newIdempotencyTestDB(t)
	user := servicetest.CreateUser(t, db)
	service := NewService(db)

	base := time.Now().Add(-time.Hour)
	rows := make([]model.MCPIdempotencyKey, MaxRecordsPerUser)
	for i := range rows {
		rows[i] = model.MCPIdempotencyKey{UserID: user.ID, IdempotencyKey: fmt.Sprintf("k-%d", i), ToolName: "t", RequestHash: "h", CreatedAt: base.Add(time.Duration(i) * time.Millisecond)}
	}
	if err := db.CreateInBatches(rows, 500).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.Save(&model.MCPIdempotencyKey{UserID: user.ID, IdempotencyKey: "newest", ToolName: "t", RequestHash: "h"}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.MCPIdempotencyKey{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != MaxRecordsPerUser {
		t.Fatalf("records = %d, want %d", count, MaxRecordsPerUser)
	}
	if record, _ := service.Lookup(user.ID, "k-0"); record != nil {
		t.Fatal("oldest record was not evicted")
	}
	if record, _ := service.Lookup(user.ID, "newest"); record == nil {
		t.Fatal("new record missing")
	}
}

func TestCleanupDeletesOnlyExpiredRecords(t *testing.T) {
	db := newIdempotencyTestDB(t)
	user := servicetest.CreateUser(t, db)
	now := time.Now()
	for key, created := range map[string]time.Time{"old": now.Add(-RetentionWindow - time.Minute), "fresh": now.Add(-time.Minute)} {
		if err := db.Create(&model.MCPIdempotencyKey{UserID: user.ID, IdempotencyKey: key, ToolName: "t", RequestHash: "h", CreatedAt: created}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := NewService(db).Cleanup(now); err != nil {
		t.Fatal(err)
	}
	var keys []string
	if err := db.Model(&model.MCPIdempotencyKey{}).Pluck("idempotency_key", &keys).Error; err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0] != "fresh" {
		t.Fatalf("remaining keys = %v, want [fresh]", keys)
	}
}
