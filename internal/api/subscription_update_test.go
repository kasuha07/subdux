package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"github.com/labstack/echo/v4"
)

// postUpdateJSON exercises the single-subscription Update handler, which owns
// the null-vs-omitted request-shape distinction for category_id,
// payment_method_id, notify_enabled, and notify_days_before.
func postUpdateJSON(t *testing.T, handler *SubscriptionHandler, userID, id uint, body string) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/subscriptions/%d", id), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := authedContext(e, rec, req, userID)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", id))
	if err := handler.Update(c); err != nil {
		APIErrorHandler(e.HTTPErrorHandler)(err, c)
	}
	return rec
}

func TestUpdateCategoryNullClearsCategory(t *testing.T) {
	handler, userID := newBatchHandler(t)
	sub := seedBatchHandlerSubscription(t, handler, userID, "Sub")

	cat := model.Category{UserID: userID, Name: "Music"}
	if err := handler.Service.DB.Create(&cat).Error; err != nil {
		t.Fatalf("create category failed: %v", err)
	}
	if _, err := handler.Service.Update(userID, sub.ID, subscriptionservice.UpdateSubscriptionInput{
		CategoryID:    &cat.ID,
		CategoryIDSet: true,
	}); err != nil {
		t.Fatalf("set category failed: %v", err)
	}

	rec := postUpdateJSON(t, handler, userID, sub.ID, `{"category_id":null}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var reloaded model.Subscription
	if err := handler.Service.DB.First(&reloaded, sub.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded.CategoryID != nil {
		t.Fatalf("category_id = %v, want nil after explicit null", *reloaded.CategoryID)
	}
}

func TestUpdateOmittedCategoryPreservesCategory(t *testing.T) {
	handler, userID := newBatchHandler(t)
	sub := seedBatchHandlerSubscription(t, handler, userID, "Sub")

	cat := model.Category{UserID: userID, Name: "Music"}
	if err := handler.Service.DB.Create(&cat).Error; err != nil {
		t.Fatalf("create category failed: %v", err)
	}
	if _, err := handler.Service.Update(userID, sub.ID, subscriptionservice.UpdateSubscriptionInput{
		CategoryID:    &cat.ID,
		CategoryIDSet: true,
	}); err != nil {
		t.Fatalf("set category failed: %v", err)
	}

	// Updating an unrelated field must leave the category untouched.
	rec := postUpdateJSON(t, handler, userID, sub.ID, `{"name":"Renamed"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var reloaded model.Subscription
	if err := handler.Service.DB.First(&reloaded, sub.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded.CategoryID == nil || *reloaded.CategoryID != cat.ID {
		t.Fatalf("category_id = %v, want %d preserved", reloaded.CategoryID, cat.ID)
	}
	if reloaded.Name != "Renamed" {
		t.Fatalf("name = %q, want Renamed", reloaded.Name)
	}
}

func TestUpdateNotifyEnabledNullDisablesNotifications(t *testing.T) {
	handler, userID := newBatchHandler(t)
	sub := seedBatchHandlerSubscription(t, handler, userID, "Sub")

	enabled := true
	if _, err := handler.Service.Update(userID, sub.ID, subscriptionservice.UpdateSubscriptionInput{
		NotifyEnabled:    &enabled,
		NotifyEnabledSet: true,
	}); err != nil {
		t.Fatalf("enable notifications failed: %v", err)
	}

	rec := postUpdateJSON(t, handler, userID, sub.ID, `{"notify_enabled":null}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var reloaded model.Subscription
	if err := handler.Service.DB.First(&reloaded, sub.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded.NotifyEnabled != nil {
		t.Fatalf("notify_enabled = %v, want nil after explicit null", *reloaded.NotifyEnabled)
	}
}
