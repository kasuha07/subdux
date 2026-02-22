package pkg

import (
	"os"
	"sync"
	"time"
)

var (
	systemTimezone     *time.Location
	systemTimezoneOnce sync.Once
)

// GetSystemTimezone returns the system's default timezone.
// Priority: TZ environment variable > system local timezone.
// Result is cached after first call.
func GetSystemTimezone() *time.Location {
	systemTimezoneOnce.Do(func() {
		// Try TZ environment variable first (Docker/container support)
		if tzEnv := os.Getenv("TZ"); tzEnv != "" {
			if loc, err := time.LoadLocation(tzEnv); err == nil {
				systemTimezone = loc
				return
			}
		}

		// Fall back to system local timezone
		systemTimezone = time.Local
	})
	return systemTimezone
}

// GetSystemTimezoneName returns the IANA timezone name (e.g., "Asia/Shanghai", "UTC").
func GetSystemTimezoneName() string {
	loc := GetSystemTimezone()
	// time.Local.String() returns "Local" which is not IANA format
	// We need to get the actual zone name
	if loc == time.UTC {
		return "UTC"
	}

	// Try to get zone name from current time
	now := time.Now().In(loc)
	zone, _ := now.Zone()

	// If zone is an abbreviation (e.g., "CST"), try to get IANA name from TZ env
	if tzEnv := os.Getenv("TZ"); tzEnv != "" {
		return tzEnv
	}

	// Last resort: return zone abbreviation or "Local"
	if zone != "" {
		return zone
	}
	return "Local"
}

// NormalizeDateInTimezone normalizes a time to start-of-day (00:00:00) in the given timezone.
func NormalizeDateInTimezone(t time.Time, loc *time.Location) time.Time {
	// Convert to target timezone first
	inZone := t.In(loc)
	// Create start-of-day in that timezone
	return time.Date(inZone.Year(), inZone.Month(), inZone.Day(), 0, 0, 0, 0, loc)
}

// TodayInTimezone returns the current date at 00:00:00 in the given timezone.
func TodayInTimezone(loc *time.Location) time.Time {
	return NormalizeDateInTimezone(time.Now(), loc)
}

// DaysUntil calculates the number of days from now until the target date,
// both normalized to the given timezone.
func DaysUntil(target time.Time, loc *time.Location) int {
	today := TodayInTimezone(loc)
	targetDate := NormalizeDateInTimezone(target, loc)
	diff := targetDate.Sub(today)
	return int(diff.Hours() / 24)
}
