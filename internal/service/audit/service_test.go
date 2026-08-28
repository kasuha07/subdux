package audit

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"gorm.io/gorm"
)

type auditTestClock struct {
	now time.Time
}

func (c *auditTestClock) Now() time.Time {
	return c.now
}

func newAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "audit-test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.AuditEvent{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

func createAuditTestUser(t *testing.T, db *gorm.DB, name string) model.User {
	t.Helper()

	user := model.User{
		Username: name,
		Email:    name + "@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user %q: %v", name, err)
	}
	return user
}

func createAuditTestEvent(t *testing.T, svc *Service, userID uint, toolName string) {
	t.Helper()

	if _, err := svc.Create(CreateEventInput{
		UserID:       userID,
		KeyID:        1,
		ToolName:     toolName,
		ResourceType: ResourceSubscription,
		Action:       "create",
		Status:       StatusSuccess,
	}); err != nil {
		t.Fatalf("create audit event %q: %v", toolName, err)
	}
}

func TestCreateRetainsLatestThirtyEventsPerUser(t *testing.T) {
	db := newAuditTestDB(t)
	user := createAuditTestUser(t, db, "audit-user")
	other := createAuditTestUser(t, db, "other-audit-user")
	base := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	clock := &auditTestClock{now: base}
	restoreClock := pkg.SetClockForTest(clock)
	t.Cleanup(restoreClock)

	svc := NewService(db)
	for i := 0; i <= auditEventRetentionLimit; i++ {
		clock.now = base.Add(time.Duration(i) * time.Minute)
		createAuditTestEvent(t, svc, user.ID, fmt.Sprintf("write-%02d", i))
	}
	clock.now = base.Add(time.Duration(auditEventRetentionLimit+1) * time.Minute)
	createAuditTestEvent(t, svc, other.ID, "other-write")

	var userCount, otherCount int64
	if err := db.Model(&model.AuditEvent{}).Where("user_id = ?", user.ID).Count(&userCount).Error; err != nil {
		t.Fatalf("count user events: %v", err)
	}
	if err := db.Model(&model.AuditEvent{}).Where("user_id = ?", other.ID).Count(&otherCount).Error; err != nil {
		t.Fatalf("count other events: %v", err)
	}
	if userCount != auditEventRetentionLimit || otherCount != 1 {
		t.Fatalf("event counts = %d/%d, want %d/1", userCount, otherCount, auditEventRetentionLimit)
	}

	events, err := svc.List(EventFilter{UserID: &user.ID, Limit: maxAuditEventListLimit + 1})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(events) != auditEventRetentionLimit {
		t.Fatalf("List() count = %d, want %d", len(events), auditEventRetentionLimit)
	}
	for i, event := range events {
		wantToolName := fmt.Sprintf("write-%02d", auditEventRetentionLimit-i)
		if event.ToolName != wantToolName {
			t.Fatalf("event %d tool = %q, want newest-first %q", i, event.ToolName, wantToolName)
		}
	}
}

func TestListUsesStableLatestFirstOrder(t *testing.T) {
	db := newAuditTestDB(t)
	user := createAuditTestUser(t, db, "stable-audit-user")
	now := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)

	svc := NewService(db)
	createAuditTestEvent(t, svc, user.ID, "first")
	createAuditTestEvent(t, svc, user.ID, "second")

	events, err := svc.List(EventFilter{UserID: &user.ID, Limit: 2})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("List() count = %d, want 2", len(events))
	}
	if events[0].OccurredAt != events[1].OccurredAt || events[0].EventID < events[1].EventID {
		t.Fatalf("events are not stably ordered by occurred_at DESC, event_id DESC: %#v", events)
	}
}
