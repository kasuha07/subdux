package service

import (
	"time"

	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"gorm.io/gorm"
)

// acquireBackgroundTaskLease tries to claim taskKey for ownerID until now+ttl.
// It returns true only to the caller that holds the lease, so a single instance
// runs a periodic task even when several share one database. The lease is
// reclaimable once expired or by its current owner (idempotent renewal).
func acquireBackgroundTaskLease(db *gorm.DB, ownerID, taskKey string, ttl time.Duration) (bool, error) {
	return serviceutil.AcquireBackgroundTaskLease(db, ownerID, taskKey, ttl)
}

// withBackgroundTaskLease runs fn only if ownerID can claim taskKey. When the
// lease is held elsewhere it returns nil without running, so callers can invoke
// it unconditionally on a timer.
func withBackgroundTaskLease(db *gorm.DB, ownerID, taskKey string, ttl time.Duration, run func() error) error {
	return serviceutil.WithBackgroundTaskLease(db, ownerID, taskKey, ttl, run)
}

// NewBackgroundTaskOwnerID returns a process-stable identifier for claiming
// background-task leases. It combines the hostname with random bytes so two
// instances on the same host do not collide.
func NewBackgroundTaskOwnerID() string {
	return serviceutil.NewBackgroundTaskOwnerID()
}
