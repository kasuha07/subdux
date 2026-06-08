package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shiroha/subdux/internal/model"
)

func validSubscriptionInputWithURL(rawURL string) CreateSubscriptionInput {
	intervalCount := 1
	return CreateSubscriptionInput{
		Name:            "Netflix",
		Amount:          12.99,
		Currency:        "USD",
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02"),
		URL:             rawURL,
	}
}

func TestNormalizeSubscriptionURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "empty", raw: "", want: ""},
		{name: "trimmed https", raw: "  https://example.com/manage?plan=pro  ", want: "https://example.com/manage?plan=pro"},
		{name: "http", raw: "http://example.com", want: "http://example.com"},
		{name: "host required", raw: "https:///path", wantErr: true},
		{name: "javascript rejected", raw: "javascript:alert(1)", wantErr: true},
		{name: "data rejected", raw: "data:text/html,<script>alert(1)</script>", wantErr: true},
		{name: "whitespace rejected", raw: "https://example.com/a b", wantErr: true},
		{name: "control rejected", raw: "https://example.com/\npath", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeSubscriptionURL(tt.raw)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidSubscriptionURL) {
					t.Fatalf("normalizeSubscriptionURL() error = %v, want ErrInvalidSubscriptionURL", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeSubscriptionURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeSubscriptionURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateSubscriptionNormalizesURL(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	svc := NewSubscriptionService(db)

	sub, err := svc.Create(user.ID, validSubscriptionInputWithURL("  https://example.com/manage  "))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if sub.URL != "https://example.com/manage" {
		t.Fatalf("Create() URL = %q, want trimmed URL", sub.URL)
	}
}

func TestCreateSubscriptionRejectsUnsafeURL(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	svc := NewSubscriptionService(db)

	_, err := svc.Create(user.ID, validSubscriptionInputWithURL("javascript:alert(1)"))
	if !errors.Is(err, ErrInvalidSubscriptionURL) {
		t.Fatalf("Create() error = %v, want ErrInvalidSubscriptionURL", err)
	}
}

func TestUpdateSubscriptionRejectsUnsafeURL(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	svc := NewSubscriptionService(db)

	sub, err := svc.Create(user.ID, validSubscriptionInputWithURL("https://example.com"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	unsafeURL := "data:text/html,<script>alert(1)</script>"
	_, err = svc.Update(user.ID, sub.ID, UpdateSubscriptionInput{URL: &unsafeURL})
	if !errors.Is(err, ErrInvalidSubscriptionURL) {
		t.Fatalf("Update() error = %v, want ErrInvalidSubscriptionURL", err)
	}
}

func TestImportFromSubduxSkipsUnsafeSubscriptionURL(t *testing.T) {
	db := newImportTestDB(t)
	user := createTestUser(t, db)
	svc := NewImportService(db)
	data := sampleSubduxImportData()
	data.Subscriptions[0].URL = "javascript:alert(1)"

	resp, err := svc.ImportFromSubdux(user.ID, data, true)
	if err != nil {
		t.Fatalf("ImportFromSubdux() error = %v", err)
	}
	if resp.Result == nil {
		t.Fatal("ImportFromSubdux() result = nil")
	}
	if resp.Result.Skipped != 1 {
		t.Fatalf("ImportFromSubdux() skipped = %d, want 1", resp.Result.Skipped)
	}
	if len(resp.Result.Errors) != 1 || !strings.Contains(resp.Result.Errors[0], "invalid url") {
		t.Fatalf("ImportFromSubdux() errors = %#v, want invalid url error", resp.Result.Errors)
	}
	var count int64
	if err := db.Model(&model.Subscription{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count subscriptions: %v", err)
	}
	if count != 0 {
		t.Fatalf("subscription count = %d, want 0", count)
	}
}

func TestImportFromWallosSkipsUnsafeSubscriptionURL(t *testing.T) {
	db := newImportTestDB(t)
	user := createTestUser(t, db)
	svc := NewImportService(db)

	resp, err := svc.ImportFromWallos(user.ID, []WallosSubscription{{
		Name:         "Netflix",
		Price:        "12.99 USD",
		PaymentCycle: "Monthly",
		NextPayment:  time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02"),
		Active:       "1",
		URL:          "data:text/html,<script>alert(1)</script>",
	}}, true)
	if err != nil {
		t.Fatalf("ImportFromWallos() error = %v", err)
	}
	if resp.Result == nil {
		t.Fatal("ImportFromWallos() result = nil")
	}
	if resp.Result.Imported != 0 || resp.Result.Skipped != 1 {
		t.Fatalf("ImportFromWallos() imported/skipped = %d/%d, want 0/1", resp.Result.Imported, resp.Result.Skipped)
	}
	if len(resp.Result.Errors) != 1 || !strings.Contains(resp.Result.Errors[0], "invalid url") {
		t.Fatalf("ImportFromWallos() errors = %#v, want invalid url error", resp.Result.Errors)
	}
}
