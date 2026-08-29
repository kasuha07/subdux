package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	exchangerate "github.com/kasuha07/subdux/internal/service/exchangerate"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"github.com/labstack/echo/v4"
)

func newBatchHandler(t *testing.T) (*SubscriptionHandler, uint) {
	t.Helper()

	restoreClock := pkg.SetNowForTest(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	t.Cleanup(restoreClock)

	db := newBootstrapTestDB(t)
	user := model.User{Username: "tester", Email: "tester@example.com", Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	return NewSubscriptionHandler(subscriptionservice.NewService(db), exchangerate.NewService(db)), user.ID
}

func postBatchJSON(t *testing.T, handler *SubscriptionHandler, userID uint, body string) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions/batch", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := authedContext(e, rec, req, userID)
	if err := handler.Batch(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}
	return rec
}

func seedBatchHandlerSubscription(t *testing.T, handler *SubscriptionHandler, userID uint, name string) model.Subscription {
	t.Helper()

	monthly := 1
	sub, err := handler.Service.Create(userID, subscriptionservice.CreateSubscriptionInput{
		Name:            name,
		Amount:          10,
		Currency:        "USD",
		Status:          "active",
		RenewalMode:     "auto_renew",
		BillingType:     "recurring",
		RecurrenceType:  "interval",
		IntervalCount:   &monthly,
		IntervalUnit:    "month",
		NextBillingDate: "2026-05-15",
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}
	return *sub
}

func TestBatchDeleteReturnsAggregatedResult(t *testing.T) {
	handler, userID := newBatchHandler(t)
	first := seedBatchHandlerSubscription(t, handler, userID, "First")
	second := seedBatchHandlerSubscription(t, handler, userID, "Second")

	rec := postBatchJSON(t, handler, userID,
		`{"action":"delete","ids":[`+itoa(first.ID)+`,`+itoa(second.ID)+`]}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var result subscriptionservice.BatchSubscriptionResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal result failed: %v", err)
	}
	if result.Total != 2 || result.Succeeded != 2 || result.Failed != 0 || len(result.Failures) != 0 {
		t.Fatalf("result = %+v, want 2 succeeded", result)
	}

	var count int64
	if err := handler.Service.DB.Model(&model.Subscription{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("remaining subscriptions = %d, want 0", count)
	}
}

func TestBatchUpdateCategoryRejectsForeignCategory(t *testing.T) {
	handler, userID := newBatchHandler(t)
	sub := seedBatchHandlerSubscription(t, handler, userID, "Sub")

	other := model.User{Username: "other", Email: "other@example.com", Password: "x", Role: "user", Status: "active"}
	if err := handler.Service.DB.Create(&other).Error; err != nil {
		t.Fatalf("create other user failed: %v", err)
	}
	foreignCategory := model.Category{UserID: other.ID, Name: "Foreign"}
	if err := handler.Service.DB.Create(&foreignCategory).Error; err != nil {
		t.Fatalf("create foreign category failed: %v", err)
	}

	rec := postBatchJSON(t, handler, userID,
		`{"action":"update","ids":[`+itoa(sub.ID)+`],"category_id":`+itoa(foreignCategory.ID)+`}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %s)", rec.Code, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "category_not_found") {
		t.Fatalf("body = %s, want category_not_found", rec.Body.String())
	}
}

func TestBatchUpdateRejectsInvalidAction(t *testing.T) {
	handler, userID := newBatchHandler(t)
	rec := postBatchJSON(t, handler, userID, `{"action":"explode","ids":[1]}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %s)", rec.Code, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "invalid_batch_action") {
		t.Fatalf("body = %s, want invalid_batch_action", rec.Body.String())
	}
}

func TestBatchMarkRenewedReportsPartialFailures(t *testing.T) {
	handler, userID := newBatchHandler(t)
	autoSub := seedBatchHandlerSubscription(t, handler, userID, "Auto")
	manualSub := seedBatchHandlerSubscription(t, handler, userID, "Manual")

	manual := "manual_renew"
	if _, err := handler.Service.Update(userID, manualSub.ID, subscriptionservice.UpdateSubscriptionInput{RenewalMode: &manual}); err != nil {
		t.Fatalf("switch to manual renew failed: %v", err)
	}

	rec := postBatchJSON(t, handler, userID,
		`{"action":"mark_renewed","ids":[`+itoa(autoSub.ID)+`,`+itoa(manualSub.ID)+`]}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var result subscriptionservice.BatchSubscriptionResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal result failed: %v", err)
	}
	if result.Total != 2 || result.Succeeded != 1 || result.Failed != 1 {
		t.Fatalf("result = %+v, want 1 succeeded / 1 failed", result)
	}
	if len(result.Failures) != 1 || result.Failures[0].ID != autoSub.ID {
		t.Fatalf("failures = %+v, want the auto-renew subscription rejected", result.Failures)
	}
}

func itoa(value uint) string {
	return fmt.Sprintf("%d", value)
}
