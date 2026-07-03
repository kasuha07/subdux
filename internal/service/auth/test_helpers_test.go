package auth

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/servicetest"
	"gorm.io/gorm"
)

const (
	ReauthOperationBackup  = "backup"
	ReauthOperationRestore = "restore"
)

type notificationTestRoundTripper func(req *http.Request) (*http.Response, error)

func (fn notificationTestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type mutableClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *mutableClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *mutableClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return servicetest.NewDB(t)
}

func createTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()
	return servicetest.CreateUser(t, db)
}

func seedSystemSetting(t *testing.T, db *gorm.DB, key string, value string) {
	t.Helper()
	if err := db.Where("key = ?", key).
		Assign(model.SystemSetting{Value: value}).
		FirstOrCreate(&model.SystemSetting{Key: key}).Error; err != nil {
		t.Fatalf("failed to seed setting %q: %v", key, err)
	}
}
