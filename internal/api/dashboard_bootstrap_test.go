package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
	"gorm.io/gorm"
)

func newBootstrapTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-bootstrap-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Subscription{},
		&model.SubscriptionEvent{},
		&model.Category{},
		&model.PaymentMethod{},
		&model.UserCurrency{},
		&model.UserPreference{},
		&model.ExchangeRate{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func authedContext(e *echo.Echo, rec *httptest.ResponseRecorder, req *http.Request, userID uint) echo.Context {
	c := e.NewContext(req, rec)
	c.Set("user", &jwt.Token{Claims: &pkg.JWTClaims{UserID: userID}})
	return c
}

func TestDashboardBootstrapAggregatesEverySectionWithoutWriting(t *testing.T) {
	restoreClock := pkg.SetNowForTest(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	t.Cleanup(restoreClock)

	db := newBootstrapTestDB(t)
	subService := service.NewSubscriptionService(db)

	user := model.User{Username: "tester", Email: "tester@example.com", Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	monthly := 1
	if _, err := subService.Create(user.ID, service.CreateSubscriptionInput{
		Name:            "Active Sub",
		Amount:          12,
		Currency:        "USD",
		Status:          "active",
		RenewalMode:     "auto_renew",
		BillingType:     "recurring",
		RecurrenceType:  "interval",
		IntervalCount:   &monthly,
		IntervalUnit:    "month",
		NextBillingDate: "2026-04-15",
	}); err != nil {
		t.Fatalf("create active subscription failed: %v", err)
	}
	// An ended subscription belongs in the full list but not the active summary.
	// Insert it directly: the create path only accepts active recurring input.
	endedAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&model.Subscription{
		UserID:      user.ID,
		Name:        "Retired Sub",
		Amount:      99,
		Currency:    "USD",
		Status:      "ended",
		RenewalMode: "cancel_at_period_end",
		BillingType: "recurring",
		EndsAt:      &endedAt,
	}).Error; err != nil {
		t.Fatalf("create ended subscription failed: %v", err)
	}

	if err := db.Create(&model.Category{UserID: user.ID, Name: "Tools"}).Error; err != nil {
		t.Fatalf("create category failed: %v", err)
	}
	if err := db.Create(&model.PaymentMethod{UserID: user.ID, Name: "Card"}).Error; err != nil {
		t.Fatalf("create payment method failed: %v", err)
	}
	if err := db.Create(&model.UserCurrency{UserID: user.ID, Code: "USD"}).Error; err != nil {
		t.Fatalf("create currency failed: %v", err)
	}

	// The bootstrap is a read: it must not write to the subscriptions table.
	var subscriptionWrites int
	countWrite := func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "subscriptions" {
			subscriptionWrites++
		}
	}
	if err := db.Callback().Update().After("gorm:update").Register("test:count_sub_update", countWrite); err != nil {
		t.Fatalf("register update counter failed: %v", err)
	}
	if err := db.Callback().Create().After("gorm:create").Register("test:count_sub_create", countWrite); err != nil {
		t.Fatalf("register create counter failed: %v", err)
	}

	handler := NewDashboardBootstrapHandler(
		subService,
		service.NewExchangeRateService(db),
		service.NewCurrencyService(db),
		service.NewCategoryService(db),
		service.NewPaymentMethodService(db),
	)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/bootstrap", nil)
	rec := httptest.NewRecorder()
	c := authedContext(e, rec, req, user.ID)

	if err := handler.Get(c); err != nil {
		t.Fatalf("Bootstrap handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body dashboardBootstrapResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if len(body.Subscriptions) != 2 {
		t.Fatalf("subscriptions = %d, want 2 (full list, all statuses)", len(body.Subscriptions))
	}
	if body.Summary == nil {
		t.Fatal("summary is nil")
	}
	if body.Summary.ActiveCount != 1 {
		t.Fatalf("summary.active_count = %d, want 1 (active subset only)", body.Summary.ActiveCount)
	}
	if len(body.Categories) != 1 {
		t.Fatalf("categories = %d, want 1", len(body.Categories))
	}
	if len(body.PaymentMethods) != 1 {
		t.Fatalf("payment_methods = %d, want 1", len(body.PaymentMethods))
	}
	if len(body.Currencies) != 1 {
		t.Fatalf("currencies = %d, want 1", len(body.Currencies))
	}
	if body.PreferredCurrency == "" {
		t.Fatal("preferred_currency is empty")
	}

	if subscriptionWrites != 0 {
		t.Fatalf("bootstrap issued %d subscription writes, want 0 (read path must be write-free)", subscriptionWrites)
	}
}
