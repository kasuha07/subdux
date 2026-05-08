package service

import (
	"errors"
	"testing"
	"time"
)

func TestBackgroundTaskMonitorRunRecordsSuccess(t *testing.T) {
	monitor := NewBackgroundTaskMonitor()
	monitor.Register("job", "Job", "Does work", time.Hour)

	if err := monitor.Run("job", func() error { return nil }); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	tasks := monitor.List()
	if len(tasks) != 1 {
		t.Fatalf("task count = %d, want 1", len(tasks))
	}
	task := tasks[0]
	if task.Status != BackgroundTaskStatusSucceeded {
		t.Fatalf("status = %q, want %q", task.Status, BackgroundTaskStatusSucceeded)
	}
	if task.SuccessCount != 1 || task.FailureCount != 0 {
		t.Fatalf("counts = success %d failure %d, want 1/0", task.SuccessCount, task.FailureCount)
	}
	if task.LastStartedAt == nil || task.LastFinishedAt == nil || task.NextRunAt == nil {
		t.Fatalf("expected timestamps to be populated: %#v", task)
	}
	if task.IntervalSeconds != int64(time.Hour.Seconds()) {
		t.Fatalf("interval seconds = %d, want %d", task.IntervalSeconds, int64(time.Hour.Seconds()))
	}
}

func TestBackgroundTaskMonitorRunRecordsFailure(t *testing.T) {
	monitor := NewBackgroundTaskMonitor()
	monitor.Register("job", "Job", "Does work", 0)
	expected := errors.New("boom")

	if err := monitor.Run("job", func() error { return expected }); !errors.Is(err, expected) {
		t.Fatalf("Run() error = %v, want %v", err, expected)
	}

	tasks := monitor.List()
	if len(tasks) != 1 {
		t.Fatalf("task count = %d, want 1", len(tasks))
	}
	task := tasks[0]
	if task.Status != BackgroundTaskStatusFailed {
		t.Fatalf("status = %q, want %q", task.Status, BackgroundTaskStatusFailed)
	}
	if task.LastError != expected.Error() {
		t.Fatalf("last error = %q, want %q", task.LastError, expected.Error())
	}
	if task.SuccessCount != 0 || task.FailureCount != 1 {
		t.Fatalf("counts = success %d failure %d, want 0/1", task.SuccessCount, task.FailureCount)
	}
}
