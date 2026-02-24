package service

import (
	"reflect"
	"testing"
	"time"
)

func TestUniqueUserIDs(t *testing.T) {
	tests := []struct {
		name    string
		input   []uint
		wantIDs []uint
	}{
		{
			name:    "empty input",
			input:   nil,
			wantIDs: nil,
		},
		{
			name:    "preserves first-seen order and removes duplicates",
			input:   []uint{3, 1, 3, 2, 1, 4},
			wantIDs: []uint{3, 1, 2, 4},
		},
		{
			name:    "already unique",
			input:   []uint{10, 20, 30},
			wantIDs: []uint{10, 20, 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueUserIDs(tt.input)
			if !reflect.DeepEqual(got, tt.wantIDs) {
				t.Fatalf("uniqueUserIDs() = %#v, want %#v", got, tt.wantIDs)
			}
		})
	}
}

func TestNotificationWorkerCount(t *testing.T) {
	tests := []struct {
		name      string
		userCount int
		want      int
	}{
		{name: "zero users", userCount: 0, want: 0},
		{name: "one user", userCount: 1, want: 1},
		{name: "below max", userCount: maxParallelUserNotificationChecks - 1, want: maxParallelUserNotificationChecks - 1},
		{name: "at max", userCount: maxParallelUserNotificationChecks, want: maxParallelUserNotificationChecks},
		{name: "above max", userCount: maxParallelUserNotificationChecks + 10, want: maxParallelUserNotificationChecks},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := notificationWorkerCount(tt.userCount)
			if got != tt.want {
				t.Fatalf("notificationWorkerCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNotificationDispatchWorkerCount(t *testing.T) {
	tests := []struct {
		name     string
		jobCount int
		want     int
	}{
		{name: "zero jobs", jobCount: 0, want: 0},
		{name: "one job", jobCount: 1, want: 1},
		{name: "below max", jobCount: maxParallelNotificationDispatchesPerUser - 1, want: maxParallelNotificationDispatchesPerUser - 1},
		{name: "at max", jobCount: maxParallelNotificationDispatchesPerUser, want: maxParallelNotificationDispatchesPerUser},
		{name: "above max", jobCount: maxParallelNotificationDispatchesPerUser + 10, want: maxParallelNotificationDispatchesPerUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := notificationDispatchWorkerCount(tt.jobCount)
			if got != tt.want {
				t.Fatalf("notificationDispatchWorkerCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestShouldScheduleNotificationDispatch(t *testing.T) {
	scheduled := make(map[string]struct{})
	notifyDate := time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC)

	if ok := shouldScheduleNotificationDispatch(scheduled, 7, "webhook", notifyDate); !ok {
		t.Fatal("first dispatch scheduling returned false, want true")
	}

	if ok := shouldScheduleNotificationDispatch(scheduled, 7, "webhook", notifyDate); ok {
		t.Fatal("duplicate dispatch scheduling returned true, want false")
	}

	if ok := shouldScheduleNotificationDispatch(scheduled, 7, "smtp", notifyDate); !ok {
		t.Fatal("different channel scheduling returned false, want true")
	}
}
