package service

import (
	"sort"
	"sync"
	"time"

	"github.com/shiroha/subdux/internal/pkg"
)

const (
	BackgroundTaskStatusIdle      = "idle"
	BackgroundTaskStatusRunning   = "running"
	BackgroundTaskStatusSucceeded = "succeeded"
	BackgroundTaskStatusFailed    = "failed"
)

type BackgroundTaskMonitor struct {
	mu    sync.RWMutex
	tasks map[string]*backgroundTaskState
}

type backgroundTaskState struct {
	Key                      string
	Name                     string
	Description              string
	Interval                 time.Duration
	Status                   string
	Running                  bool
	LastStartedAt            *time.Time
	LastFinishedAt           *time.Time
	NextRunAt                *time.Time
	LastDurationMilliseconds int64
	LastError                string
	SuccessCount             int64
	FailureCount             int64
}

type BackgroundTaskSnapshot struct {
	Key                      string     `json:"key"`
	Name                     string     `json:"name"`
	Description              string     `json:"description"`
	IntervalSeconds          int64      `json:"interval_seconds"`
	Status                   string     `json:"status"`
	Running                  bool       `json:"running"`
	LastStartedAt            *time.Time `json:"last_started_at"`
	LastFinishedAt           *time.Time `json:"last_finished_at"`
	NextRunAt                *time.Time `json:"next_run_at"`
	LastDurationMilliseconds int64      `json:"last_duration_ms"`
	LastError                string     `json:"last_error"`
	SuccessCount             int64      `json:"success_count"`
	FailureCount             int64      `json:"failure_count"`
}

func NewBackgroundTaskMonitor() *BackgroundTaskMonitor {
	return &BackgroundTaskMonitor{tasks: make(map[string]*backgroundTaskState)}
}

func (m *BackgroundTaskMonitor) Register(key, name, description string, interval time.Duration) {
	if m == nil || key == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.tasks[key]; ok {
		existing.Name = name
		existing.Description = description
		existing.Interval = interval
		return
	}

	m.tasks[key] = &backgroundTaskState{
		Key:         key,
		Name:        name,
		Description: description,
		Interval:    interval,
		Status:      BackgroundTaskStatusIdle,
	}
}

func (m *BackgroundTaskMonitor) Run(key string, run func() error) error {
	if m == nil {
		return run()
	}

	startedAt := pkg.NowUTC()
	m.mu.Lock()
	task := m.ensureTaskLocked(key)
	task.Status = BackgroundTaskStatusRunning
	task.Running = true
	task.LastStartedAt = &startedAt
	task.LastError = ""
	m.mu.Unlock()

	err := run()
	finishedAt := pkg.NowUTC()

	m.mu.Lock()
	defer m.mu.Unlock()
	task = m.ensureTaskLocked(key)
	task.Running = false
	task.LastFinishedAt = &finishedAt
	task.LastDurationMilliseconds = finishedAt.Sub(startedAt).Milliseconds()
	if task.Interval > 0 {
		nextRunAt := finishedAt.Add(task.Interval)
		task.NextRunAt = &nextRunAt
	}
	if err != nil {
		task.Status = BackgroundTaskStatusFailed
		task.LastError = err.Error()
		task.FailureCount++
		return err
	}

	task.Status = BackgroundTaskStatusSucceeded
	task.SuccessCount++
	return nil
}

func (m *BackgroundTaskMonitor) List() []BackgroundTaskSnapshot {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshots := make([]BackgroundTaskSnapshot, 0, len(m.tasks))
	for _, task := range m.tasks {
		snapshots = append(snapshots, snapshotBackgroundTask(task))
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Key < snapshots[j].Key
	})
	return snapshots
}

func (m *BackgroundTaskMonitor) ensureTaskLocked(key string) *backgroundTaskState {
	if task, ok := m.tasks[key]; ok {
		return task
	}

	task := &backgroundTaskState{Key: key, Name: key, Status: BackgroundTaskStatusIdle}
	m.tasks[key] = task
	return task
}

func snapshotBackgroundTask(task *backgroundTaskState) BackgroundTaskSnapshot {
	return BackgroundTaskSnapshot{
		Key:                      task.Key,
		Name:                     task.Name,
		Description:              task.Description,
		IntervalSeconds:          int64(task.Interval.Seconds()),
		Status:                   task.Status,
		Running:                  task.Running,
		LastStartedAt:            cloneTimePtr(task.LastStartedAt),
		LastFinishedAt:           cloneTimePtr(task.LastFinishedAt),
		NextRunAt:                cloneTimePtr(task.NextRunAt),
		LastDurationMilliseconds: task.LastDurationMilliseconds,
		LastError:                task.LastError,
		SuccessCount:             task.SuccessCount,
		FailureCount:             task.FailureCount,
	}
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
