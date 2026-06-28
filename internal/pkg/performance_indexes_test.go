package pkg

import (
	"strings"
	"testing"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

// queryPlanDetail runs EXPLAIN QUERY PLAN for query and returns the concatenated
// planner detail text (one line per plan node). SQLite reports index usage as
// "SEARCH <table> USING INDEX <name>" and a full scan as "SCAN <table>", so the
// returned string is enough to assert which access path the planner chose.
func queryPlanDetail(t *testing.T, db *gorm.DB, query string, args ...interface{}) string {
	t.Helper()

	rows, err := db.Raw("EXPLAIN QUERY PLAN "+query, args...).Rows()
	if err != nil {
		t.Fatalf("explain query plan error = %v", err)
	}
	defer rows.Close()

	var details []string
	for rows.Next() {
		var (
			id      int
			parent  int
			notused int
			detail  string
		)
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatalf("scan query plan row error = %v", err)
		}
		details = append(details, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate query plan rows error = %v", err)
	}
	return strings.Join(details, "\n")
}

// TestHotQueriesUseCompositeIndexes pins the access path of the hottest read
// queries to their composite indexes. Under SQLite's single-writer queue an
// accidental full-table SCAN lengthens the serialized queue for every caller,
// so this test fails loudly if a future query-shape change stops hitting the
// index it was designed for.
func TestHotQueriesUseCompositeIndexes(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := configureSQLiteDatabase(db); err != nil {
		t.Fatalf("configureSQLiteDatabase() error = %v", err)
	}
	if err := runSchemaMigrations(db); err != nil {
		t.Fatalf("runSchemaMigrations() error = %v", err)
	}

	cases := []struct {
		name  string
		index string
		query string
		args  []interface{}
	}{
		{
			name:  "lifecycle reconcile / dashboard summary",
			index: "idx_subscriptions_user_status_billing",
			query: "SELECT * FROM subscriptions WHERE user_id = ? AND status = ? AND billing_type = ?",
			args:  []interface{}{1, "active", "recurring"},
		},
		{
			name:  "subscription price history",
			index: "idx_subscription_events_user_sub_created",
			query: "SELECT * FROM subscription_events WHERE user_id = ? AND subscription_id = ? ORDER BY created_at ASC",
			args:  []interface{}{1, 1},
		},
		{
			name:  "notification failure scan",
			index: "idx_notification_logs_user_status_sent",
			query: "SELECT * FROM notification_logs WHERE user_id = ? AND status = ? AND sent_at >= ?",
			args:  []interface{}{1, "failed", "2026-01-01"},
		},
		{
			name:  "notification recovery lookup",
			index: "idx_notification_logs_user_sub_channel_sent",
			query: "SELECT id FROM notification_logs WHERE user_id = ? AND subscription_id = ? AND channel_type = ? AND status = ? AND sent_at > ?",
			args:  []interface{}{1, 1, "email", "sent", "2026-01-01"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := queryPlanDetail(t, db, tc.query, tc.args...)
			if !strings.Contains(plan, tc.index) {
				t.Fatalf("query plan did not use index %s\nplan:\n%s", tc.index, plan)
			}
			if strings.Contains(plan, "SCAN ") {
				t.Fatalf("query plan performs a full table scan\nplan:\n%s", plan)
			}
		})
	}
}

// TestSubscriptionNextBillingIndexExists guards the ordering index used by the
// subscription list query. Its leading sort key is an expression
// (next_billing_date IS NULL), so the planner may not always consume the index
// for ordering; asserting existence keeps the index from silently disappearing
// across schema edits while avoiding a brittle EXPLAIN assertion.
func TestSubscriptionNextBillingIndexExists(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := configureSQLiteDatabase(db); err != nil {
		t.Fatalf("configureSQLiteDatabase() error = %v", err)
	}
	if err := runSchemaMigrations(db); err != nil {
		t.Fatalf("runSchemaMigrations() error = %v", err)
	}

	if !db.Migrator().HasIndex(&model.Subscription{}, "idx_subscriptions_user_next_billing") {
		t.Fatal("expected index idx_subscriptions_user_next_billing to exist")
	}
}
