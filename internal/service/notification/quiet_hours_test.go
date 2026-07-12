package notification

import (
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
)

func TestParseHM(t *testing.T) {
	cases := []struct {
		in      string
		wantMin int
		wantOK  bool
	}{
		{"00:00", 0, true},
		{"08:00", 480, true},
		{"23:59", 1439, true},
		{"22:30", 1350, true},
		{"", 0, false},
		{"8:00", 0, false},   // not zero-padded
		{"08:0", 0, false},   // too short
		{"08-00", 0, false},  // wrong separator
		{"24:00", 0, false},  // hour out of range
		{"12:60", 0, false},  // minute out of range
		{"ab:cd", 0, false},  // non-numeric
		{"08:00 ", 0, false}, // trailing space / wrong length
	}
	for _, tc := range cases {
		gotMin, gotOK := parseHM(tc.in)
		if gotOK != tc.wantOK || (tc.wantOK && gotMin != tc.wantMin) {
			t.Errorf("parseHM(%q) = (%d, %v), want (%d, %v)", tc.in, gotMin, gotOK, tc.wantMin, tc.wantOK)
		}
	}
}

func TestQuietHoursDeferUntil(t *testing.T) {
	loc := time.UTC
	day := func(h, m int) time.Time {
		return time.Date(2026, 3, 15, h, m, 0, 0, loc)
	}

	cases := []struct {
		name       string
		now        time.Time
		start, end string
		wantIn     bool
		wantUntil  time.Time
	}{
		{name: "same-day inside", now: day(3, 0), start: "01:00", end: "06:00", wantIn: true, wantUntil: day(6, 0)},
		{name: "same-day before window", now: day(0, 30), start: "01:00", end: "06:00", wantIn: false},
		{name: "same-day after window", now: day(6, 30), start: "01:00", end: "06:00", wantIn: false},
		{name: "same-day at start boundary", now: day(1, 0), start: "01:00", end: "06:00", wantIn: true, wantUntil: day(6, 0)},
		{name: "same-day at end boundary excluded", now: day(6, 0), start: "01:00", end: "06:00", wantIn: false},

		{name: "cross-midnight evening side", now: day(23, 0), start: "22:00", end: "08:00", wantIn: true, wantUntil: day(8, 0).AddDate(0, 0, 1)},
		{name: "cross-midnight at start boundary", now: day(22, 0), start: "22:00", end: "08:00", wantIn: true, wantUntil: day(8, 0).AddDate(0, 0, 1)},
		{name: "cross-midnight morning side", now: day(2, 0), start: "22:00", end: "08:00", wantIn: true, wantUntil: day(8, 0)},
		{name: "cross-midnight daytime outside", now: day(12, 0), start: "22:00", end: "08:00", wantIn: false},
		{name: "cross-midnight at end boundary excluded", now: day(8, 0), start: "22:00", end: "08:00", wantIn: false},

		{name: "unset equal start end", now: day(3, 0), start: "08:00", end: "08:00", wantIn: false},
		{name: "unset invalid start", now: day(3, 0), start: "", end: "08:00", wantIn: false},
		{name: "unset invalid end", now: day(3, 0), start: "22:00", end: "nope", wantIn: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotIn, gotUntil := quietHoursDeferUntil(tc.now, tc.start, tc.end, loc)
			if gotIn != tc.wantIn {
				t.Fatalf("inWindow = %v, want %v", gotIn, tc.wantIn)
			}
			if !tc.wantIn {
				if !gotUntil.Equal(tc.now) {
					t.Fatalf("until = %v, want passthrough %v", gotUntil, tc.now)
				}
				return
			}
			if !gotUntil.Equal(tc.wantUntil) {
				t.Fatalf("until = %v, want %v", gotUntil, tc.wantUntil)
			}
		})
	}
}

func TestQuietHoursDeferUntilNilLocation(t *testing.T) {
	now := time.Date(2026, 3, 15, 23, 0, 0, 0, time.UTC)
	in, until := quietHoursDeferUntil(now, "22:00", "08:00", nil)
	if !in {
		t.Fatal("inWindow = false, want true with nil location falling back to UTC")
	}
	want := time.Date(2026, 3, 16, 8, 0, 0, 0, time.UTC)
	if !until.Equal(want) {
		t.Fatalf("until = %v, want %v", until, want)
	}
}

func TestUpdatePolicySavesQuietHours(t *testing.T) {
	db := newNotificationDaysBeforeTestDB(t)
	user := createNotificationDaysBeforeTestUser(t, db)
	service := NewService(db, nil, nil)

	enabled := true
	start := "22:00"
	end := "08:00"
	policy, err := service.UpdatePolicy(user.ID, UpdatePolicyInput{
		QuietHoursEnabled: &enabled,
		QuietHoursStart:   &start,
		QuietHoursEnd:     &end,
	})
	if err != nil {
		t.Fatalf("UpdatePolicy() error = %v", err)
	}
	if !policy.QuietHoursEnabled || policy.QuietHoursStart != "22:00" || policy.QuietHoursEnd != "08:00" {
		t.Fatalf("policy quiet hours = (%v, %q, %q), want (true, 22:00, 08:00)",
			policy.QuietHoursEnabled, policy.QuietHoursStart, policy.QuietHoursEnd)
	}

	// Reload to confirm persistence.
	var reloaded model.NotificationPolicy
	if err := db.Where("user_id = ?", user.ID).First(&reloaded).Error; err != nil {
		t.Fatalf("reload policy error = %v", err)
	}
	if !reloaded.QuietHoursEnabled || reloaded.QuietHoursStart != "22:00" || reloaded.QuietHoursEnd != "08:00" {
		t.Fatalf("reloaded quiet hours = (%v, %q, %q), want (true, 22:00, 08:00)",
			reloaded.QuietHoursEnabled, reloaded.QuietHoursStart, reloaded.QuietHoursEnd)
	}
}

func TestUpdatePolicyRejectsInvalidQuietHoursTime(t *testing.T) {
	db := newNotificationDaysBeforeTestDB(t)
	user := createNotificationDaysBeforeTestUser(t, db)
	service := NewService(db, nil, nil)

	bad := "9:00"
	_, err := service.UpdatePolicy(user.ID, UpdatePolicyInput{QuietHoursStart: &bad})
	if err == nil {
		t.Fatal("UpdatePolicy() error = nil, want quiet hours validation error")
	}
}

func TestEnqueueDefersDeliveryDuringQuietHours(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	createNotificationOutboxTemplate(t, db, user.ID)

	now := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)

	notifyDate := normalizeDateUTC(now)
	createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"https://example.com/hook"}`)

	// Build a quiet-hours window that always contains `now` in the system
	// timezone, so the assertion holds regardless of the host's local zone.
	loc := pkg.GetSystemTimezone()
	local := now.In(loc)
	start := local.Add(-1 * time.Hour).Format("15:04")
	end := local.Add(2 * time.Hour).Format("15:04")
	if err := db.Create(&model.NotificationPolicy{
		UserID:            user.ID,
		DaysBefore:        0,
		NotifyOnDueDay:    true,
		QuietHoursEnabled: true,
		QuietHoursStart:   start,
		QuietHoursEnd:     end,
	}).Error; err != nil {
		t.Fatalf("failed to create notification policy: %v", err)
	}

	svc := NewService(db, NewNotificationTemplateService(db, NewTemplateValidator()), NewTemplateRenderer(NewTemplateValidator()))
	if err := svc.EnqueuePendingNotifications(); err != nil {
		t.Fatalf("EnqueuePendingNotifications() error = %v", err)
	}

	var jobs []model.NotificationOutbox
	if err := db.Find(&jobs).Error; err != nil {
		t.Fatalf("load outbox jobs failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("outbox job count = %d, want 1", len(jobs))
	}
	job := jobs[0]
	if !job.ScheduledFor.After(now) {
		t.Fatalf("scheduled_for = %v, want deferred after now %v", job.ScheduledFor, now)
	}
	if !job.ScheduledFor.Equal(job.NextAttemptAt) {
		t.Fatalf("scheduled_for %v != next_attempt_at %v", job.ScheduledFor, job.NextAttemptAt)
	}
}

func TestEnqueueDeliversImmediatelyWhenQuietHoursDisabled(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	createNotificationOutboxTemplate(t, db, user.ID)

	now := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)

	notifyDate := normalizeDateUTC(now)
	createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"https://example.com/hook"}`)

	// Quiet hours configured but disabled -> no deferral.
	if err := db.Create(&model.NotificationPolicy{
		UserID:            user.ID,
		DaysBefore:        0,
		NotifyOnDueDay:    true,
		QuietHoursEnabled: false,
		QuietHoursStart:   "00:00",
		QuietHoursEnd:     "23:59",
	}).Error; err != nil {
		t.Fatalf("failed to create notification policy: %v", err)
	}

	svc := NewService(db, NewNotificationTemplateService(db, NewTemplateValidator()), NewTemplateRenderer(NewTemplateValidator()))
	if err := svc.EnqueuePendingNotifications(); err != nil {
		t.Fatalf("EnqueuePendingNotifications() error = %v", err)
	}

	var job model.NotificationOutbox
	if err := db.First(&job).Error; err != nil {
		t.Fatalf("load outbox job failed: %v", err)
	}
	if !job.ScheduledFor.Equal(now.UTC()) {
		t.Fatalf("scheduled_for = %v, want now %v", job.ScheduledFor, now.UTC())
	}
}

func TestUpdatePolicyDisablingQuietHoursReleasesStaleDelay(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	now := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)

	if err := db.Create(&model.NotificationPolicy{
		UserID: user.ID, QuietHoursEnabled: true, QuietHoursStart: "08:00", QuietHoursEnd: "12:00",
	}).Error; err != nil {
		t.Fatalf("create notification policy failed: %v", err)
	}
	job := model.NotificationOutbox{
		DedupeKey: "stale-quiet-delay", UserID: user.ID, SubscriptionID: 1,
		ChannelType: "webhook", TriggerType: notificationTriggerDueDay,
		NotifyDate: normalizeDateUTC(now), ScheduledFor: now.Add(3 * time.Hour),
		Status: notificationOutboxStatusPending, MaxAttempts: 5,
		NextAttemptAt: now.Add(3 * time.Hour), Message: "hello",
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create outbox job failed: %v", err)
	}

	disabled := false
	if _, err := NewService(db, nil, nil).UpdatePolicy(user.ID, UpdatePolicyInput{QuietHoursEnabled: &disabled}); err != nil {
		t.Fatalf("UpdatePolicy() error = %v", err)
	}
	var saved model.NotificationOutbox
	if err := db.First(&saved, job.ID).Error; err != nil {
		t.Fatalf("load outbox job failed: %v", err)
	}
	if !saved.NextAttemptAt.Equal(now.UTC()) {
		t.Fatalf("next_attempt_at = %v, want %v", saved.NextAttemptAt, now.UTC())
	}
	if !saved.ScheduledFor.Equal(now.UTC()) {
		t.Fatalf("scheduled_for = %v, want %v", saved.ScheduledFor, now.UTC())
	}
}
