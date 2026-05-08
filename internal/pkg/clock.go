package pkg

import (
	"sync"
	"time"
)

// Clock supplies the current wall-clock time.
//
// Production code uses the real system clock. Tests can temporarily replace it
// with a fixed clock through SetClockForTest or SetNowForTest so date-sensitive
// logic remains deterministic.
type Clock interface {
	Now() time.Time
}

type clockFunc func() time.Time

func (f clockFunc) Now() time.Time {
	return f()
}

var (
	clockMu      sync.RWMutex
	currentClock Clock = clockFunc(time.Now)
)

// Now returns the current time from the process-wide clock.
func Now() time.Time {
	clockMu.RLock()
	clock := currentClock
	clockMu.RUnlock()

	if clock == nil {
		return time.Now()
	}
	return clock.Now()
}

// NowUTC returns the current time normalized to UTC.
func NowUTC() time.Time {
	return Now().UTC()
}

// NowIn returns the current time in the provided location. A nil location falls
// back to time.Local to avoid panics from time.Time.In(nil).
func NowIn(loc *time.Location) time.Time {
	if loc == nil {
		loc = time.Local
	}
	return Now().In(loc)
}

// NowInSystemTimezone returns the current time in the configured system
// timezone (TZ environment variable when set, otherwise time.Local).
func NowInSystemTimezone() time.Time {
	return NowIn(GetSystemTimezone())
}

// SetClockForTest replaces the process-wide clock and returns a restore
// function. It is intended for tests; callers should defer or Cleanup the
// returned function.
func SetClockForTest(clock Clock) func() {
	if clock == nil {
		clock = clockFunc(time.Now)
	}

	clockMu.Lock()
	previous := currentClock
	currentClock = clock
	clockMu.Unlock()

	return func() {
		clockMu.Lock()
		currentClock = previous
		clockMu.Unlock()
	}
}

// SetNowForTest fixes the process-wide clock at a single instant and returns a
// restore function. It is intended for date-sensitive tests.
func SetNowForTest(now time.Time) func() {
	return SetClockForTest(clockFunc(func() time.Time {
		return now
	}))
}
