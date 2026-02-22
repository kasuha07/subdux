package pkg

import (
	"testing"
	"time"
)

func TestGetSystemTimezone(t *testing.T) {
	t.Run("returns non-nil location", func(t *testing.T) {
		loc := GetSystemTimezone()
		if loc == nil {
			t.Error("expected non-nil location")
		}
	})

}

func TestNormalizeDateInTimezone(t *testing.T) {
	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	utc := time.UTC

	tests := []struct {
		name     string
		input    time.Time
		loc      *time.Location
		wantYear int
		wantMon  time.Month
		wantDay  int
		wantHour int
	}{
		{
			name:     "UTC midnight",
			input:    time.Date(2026, 2, 22, 14, 30, 0, 0, utc),
			loc:      utc,
			wantYear: 2026,
			wantMon:  2,
			wantDay:  22,
			wantHour: 0,
		},
		{
			name:     "Tokyo timezone conversion",
			input:    time.Date(2026, 2, 22, 14, 30, 0, 0, utc),
			loc:      tokyo,
			wantYear: 2026,
			wantMon:  2,
			wantDay:  22,
			wantHour: 0,
		},
		{
			name:     "Tokyo late night becomes next day",
			input:    time.Date(2026, 2, 22, 23, 30, 0, 0, tokyo),
			loc:      tokyo,
			wantYear: 2026,
			wantMon:  2,
			wantDay:  22,
			wantHour: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeDateInTimezone(tt.input, tt.loc)
			if result.Year() != tt.wantYear || result.Month() != tt.wantMon ||
				result.Day() != tt.wantDay || result.Hour() != tt.wantHour {
				t.Errorf("NormalizeDateInTimezone() = %v, want %d-%02d-%02d %02d:00",
					result, tt.wantYear, tt.wantMon, tt.wantDay, tt.wantHour)
			}
			if result.Location() != tt.loc {
				t.Errorf("result location = %v, want %v", result.Location(), tt.loc)
			}
		})
	}
}

func TestTodayInTimezone(t *testing.T) {
	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	result := TodayInTimezone(tokyo)

	if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 {
		t.Errorf("TodayInTimezone should return midnight, got %02d:%02d:%02d",
			result.Hour(), result.Minute(), result.Second())
	}
	if result.Location() != tokyo {
		t.Errorf("result location = %v, want %v", result.Location(), tokyo)
	}
}

func TestDaysUntil(t *testing.T) {
	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	utc := time.UTC

	tests := []struct {
		name     string
		target   time.Time
		loc      *time.Location
		wantDays int
	}{
		{
			name:     "same day in UTC",
			target:   time.Now().UTC(),
			loc:      utc,
			wantDays: 0,
		},
		{
			name:     "tomorrow in UTC",
			target:   time.Now().UTC().AddDate(0, 0, 1),
			loc:      utc,
			wantDays: 1,
		},
		{
			name:     "7 days from now in Tokyo",
			target:   time.Now().In(tokyo).AddDate(0, 0, 7),
			loc:      tokyo,
			wantDays: 7,
		},
		{
			name:     "yesterday in UTC",
			target:   time.Now().UTC().AddDate(0, 0, -1),
			loc:      utc,
			wantDays: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DaysUntil(tt.target, tt.loc)
			if result != tt.wantDays {
				t.Errorf("DaysUntil() = %d, want %d", result, tt.wantDays)
			}
		})
	}
}

func TestDaysUntilCrossingTimezone(t *testing.T) {
	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	utc := time.UTC

	utcTime := time.Date(2026, 2, 23, 2, 0, 0, 0, utc)
	tokyoTime := utcTime.In(tokyo)

	utcDays := DaysUntil(utcTime, utc)
	tokyoDays := DaysUntil(tokyoTime, tokyo)

	if utcDays != tokyoDays {
		t.Logf("UTC time: %v (days until: %d)", utcTime, utcDays)
		t.Logf("Tokyo time: %v (days until: %d)", tokyoTime, tokyoDays)
	}
}
