package notification

import (
	"strconv"
	"time"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

// invalidQuietHoursTimeError is returned when a quiet-hours boundary is a
// non-empty string that is not a valid "HH:MM" time.
func invalidQuietHoursTimeError() error {
	return serviceerr.NewCode(
		serviceerr.KindInvalid,
		"quiet_hours_time_invalid",
		"quiet hours times must be in HH:MM 24-hour format",
		nil,
	)
}

// parseHM parses an "HH:MM" 24-hour string into minutes since midnight.
// It requires the exact 5-character zero-padded form (e.g. "08:00") that the
// frontend <input type="time"> control emits and the model column stores.
func parseHM(value string) (int, bool) {
	if len(value) != 5 || value[2] != ':' {
		return 0, false
	}
	h, err := strconv.Atoi(value[:2])
	if err != nil || h < 0 || h > 23 {
		return 0, false
	}
	m, err := strconv.Atoi(value[3:])
	if err != nil || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// ValidQuietHoursTime reports whether value is a parseable "HH:MM" time. It is
// exported so callers that persist quiet-hours boundaries directly (e.g. the
// importer) can validate them with the same rule as UpdatePolicy.
func ValidQuietHoursTime(value string) bool {
	_, ok := parseHM(value)
	return ok
}

// quietHoursDeferUntil reports whether t falls inside the configured quiet-hours
// window and, if so, returns the absolute instant the window ends. When t is
// outside the window — or the window is unset (unparseable times or start ==
// end) — it returns (false, t). The window is treated as half-open [start, end):
// a notification produced exactly at the end minute is already outside it.
//
// startHM/endHM are "HH:MM" in loc's wall-clock time. A window where start > end
// (e.g. 22:00–08:00) spans midnight.
func quietHoursDeferUntil(t time.Time, startHM, endHM string, loc *time.Location) (bool, time.Time) {
	if loc == nil {
		loc = time.UTC
	}
	startMin, ok := parseHM(startHM)
	if !ok {
		return false, t
	}
	endMin, ok := parseHM(endHM)
	if !ok {
		return false, t
	}
	if startMin == endMin {
		return false, t
	}

	local := t.In(loc)
	curMin := local.Hour()*60 + local.Minute()
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	endToday := dayStart.Add(time.Duration(endMin) * time.Minute)
	endTomorrow := dayStart.AddDate(0, 0, 1).Add(time.Duration(endMin) * time.Minute)

	if startMin < endMin {
		// Same-day window, e.g. 01:00–06:00.
		if curMin >= startMin && curMin < endMin {
			return true, endToday
		}
		return false, t
	}

	// Cross-midnight window, e.g. 22:00–08:00.
	if curMin >= startMin {
		// Evening side of midnight: the window ends the next morning.
		return true, endTomorrow
	}
	if curMin < endMin {
		// Early-morning side of midnight: the window ends later today.
		return true, endToday
	}
	return false, t
}
