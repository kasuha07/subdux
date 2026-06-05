package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
	"gorm.io/gorm"
)

func newMCPTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.SystemSetting{},
		&model.APIKey{},
		&model.Subscription{},
		&model.SubscriptionEvent{},
		&model.SubscriptionActionSnooze{},
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

func createMCPTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	user := model.User{
		Username: "mcp-user",
		Email:    "mcp@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func newMCPTestHandler(db *gorm.DB) *MCPHandler {
	return NewMCPHandler(
		service.NewAPIKeyService(db),
		service.NewSubscriptionService(db),
		service.NewExchangeRateService(db),
		service.NewCurrencyService(db),
		service.NewCategoryService(db),
		service.NewPaymentMethodService(db),
	)
}

func createMCPAPIKey(t *testing.T, db *gorm.DB, user model.User, scopes []string) string {
	t.Helper()

	resp, err := service.NewAPIKeyService(db).Create(user.ID, user.Role, service.CreateAPIKeyInput{
		Name:   "Agent",
		Scopes: scopes,
	})
	if err != nil {
		t.Fatalf("failed to create api key: %v", err)
	}
	return resp.Key
}

func enableMCP(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := db.Where("key = ?", "mcp_enabled").
		Assign(model.SystemSetting{Value: "true"}).
		FirstOrCreate(&model.SystemSetting{Key: "mcp_enabled"}).Error; err != nil {
		t.Fatalf("failed to enable mcp: %v", err)
	}
}

func performMCPRequest(t *testing.T, handler *MCPHandler, apiKey string, body map[string]interface{}) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	req.Host = "localhost:8080"
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.HandlePost(c); err != nil {
		t.Fatalf("HandlePost() error = %v", err)
	}

	var decoded map[string]interface{}
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("failed to decode response %q: %v", rec.Body.String(), err)
		}
	}
	return rec, decoded
}

func TestMCPRequiresAPIKey(t *testing.T) {
	db := newMCPTestDB(t)
	handler := newMCPTestHandler(db)

	rec, _ := performMCPRequest(t, handler, "", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
	})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMCPInitializeAndListTools(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	rec, resp := performMCPRequest(t, handler, apiKey, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("MCP-Protocol-Version") != mcpProtocolVersion {
		t.Fatalf("MCP-Protocol-Version header = %q, want %q", rec.Header().Get("MCP-Protocol-Version"), mcpProtocolVersion)
	}

	result := resp["result"].(map[string]interface{})
	if result["protocolVersion"] != mcpProtocolVersion {
		t.Fatalf("protocolVersion = %v, want %s", result["protocolVersion"], mcpProtocolVersion)
	}

	rec, resp = performMCPRequest(t, handler, apiKey, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	tools := resp["result"].(map[string]interface{})["tools"].([]interface{})
	var foundCreate bool
	for _, item := range tools {
		tool := item.(map[string]interface{})
		if tool["name"] == "create_subscription" {
			foundCreate = true
			break
		}
	}
	if !foundCreate {
		t.Fatalf("tools/list missing create_subscription: %#v", tools)
	}
}

func TestSetupRoutesRegistersMCPAtRoot(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	enableMCP(t, db)
	if err := pkg.InitJWTSecret(db); err != nil {
		t.Fatalf("failed to initialize jwt secret: %v", err)
	}

	e := echo.New()
	SetupRoutes(context.Background(), e, db, service.NewBackgroundTaskMonitor())

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-API-Key", apiKey)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := resp["result"]; !ok {
		t.Fatalf("response missing result: %#v", resp)
	}
}

func TestSetupRoutesMCPDisabledByDefault(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	if err := pkg.InitJWTSecret(db); err != nil {
		t.Fatalf("failed to initialize jwt secret: %v", err)
	}

	e := echo.New()
	SetupRoutes(context.Background(), e, db, service.NewBackgroundTaskMonitor())

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-API-Key", apiKey)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestSiteInfoIncludesMCPEnabled(t *testing.T) {
	db := newMCPTestDB(t)
	enableMCP(t, db)
	if err := pkg.InitJWTSecret(db); err != nil {
		t.Fatalf("failed to initialize jwt secret: %v", err)
	}

	e := echo.New()
	SetupRoutes(context.Background(), e, db, service.NewBackgroundTaskMonitor())

	req := httptest.NewRequest(http.MethodGet, "/api/site-info", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["mcp_enabled"] != true {
		t.Fatalf("mcp_enabled = %v, want true", resp["mcp_enabled"])
	}
}

func TestMCPCreateAndListSubscriptionWithAPIKey(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	rec, resp := performMCPRequest(t, handler, apiKey, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "create_subscription",
			"arguments": map[string]interface{}{
				"name":              "Claude Pro",
				"amount":            20,
				"currency":          "USD",
				"next_billing_date": "2026-06-15",
			},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	result := resp["result"].(map[string]interface{})
	if result["isError"] == true {
		t.Fatalf("create_subscription returned tool error: %#v", result)
	}
	created := result["structuredContent"].(map[string]interface{})
	if created["name"] != "Claude Pro" {
		t.Fatalf("created name = %v, want Claude Pro", created["name"])
	}
	if created["next_billing_date"] != "2026-06-15" {
		t.Fatalf("next_billing_date = %v, want 2026-06-15", created["next_billing_date"])
	}

	rec, resp = performMCPRequest(t, handler, apiKey, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "list_subscriptions",
			"arguments": map[string]interface{}{},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	listResult := resp["result"].(map[string]interface{})
	structured := listResult["structuredContent"].(map[string]interface{})
	subs := structured["subscriptions"].([]interface{})
	if len(subs) != 1 {
		t.Fatalf("subscription count = %d, want 1", len(subs))
	}
}

func TestMCPListReferenceToolsReturnStructuredObjects(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	tests := []struct {
		name string
		key  string
	}{
		{name: "list_categories", key: "categories"},
		{name: "list_payment_methods", key: "payment_methods"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, resp := performMCPRequest(t, handler, apiKey, map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/call",
				"params": map[string]interface{}{
					"name":      tt.name,
					"arguments": map[string]interface{}{},
				},
			})
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			result := resp["result"].(map[string]interface{})
			structured := result["structuredContent"].(map[string]interface{})
			items := structured[tt.key].([]interface{})
			if len(items) != 0 {
				t.Fatalf("%s count = %d, want 0", tt.key, len(items))
			}
		})
	}
}

func TestMCPReadOnlyAPIKeyCannotWrite(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, []string{service.APIKeyScopeRead})
	handler := newMCPTestHandler(db)

	rec, resp := performMCPRequest(t, handler, apiKey, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "create_subscription",
			"arguments": map[string]interface{}{
				"name":              "Claude Pro",
				"amount":            20,
				"next_billing_date": "2026-06-15",
			},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	result := resp["result"].(map[string]interface{})
	if result["isError"] != true {
		t.Fatalf("isError = %v, want true; response = %#v", result["isError"], result)
	}
}

func TestMCPRejectsInvalidToolArgumentType(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	rec, resp := performMCPRequest(t, handler, apiKey, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "create_subscription",
			"arguments": map[string]interface{}{
				"name":              "Claude Pro",
				"amount":            "not-a-number",
				"next_billing_date": "2026-06-15",
			},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	errPayload := resp["error"].(map[string]interface{})
	if int(errPayload["code"].(float64)) != -32602 {
		t.Fatalf("error code = %v, want -32602", errPayload["code"])
	}
}

func TestMCPRejectsInvalidDashboardCurrency(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	rec, resp := performMCPRequest(t, handler, apiKey, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "get_dashboard_summary",
			"arguments": map[string]interface{}{
				"currency": "INVALID",
			},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	errPayload := resp["error"].(map[string]interface{})
	if int(errPayload["code"].(float64)) != -32602 {
		t.Fatalf("error code = %v, want -32602", errPayload["code"])
	}
	if errPayload["message"] != "currency not found" {
		t.Fatalf("error message = %v, want currency not found", errPayload["message"])
	}
}

func TestMCPUpdateSubscriptionClearsNullableReferences(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	category := model.Category{UserID: user.ID, Name: "Video"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("failed to create category: %v", err)
	}
	method := model.PaymentMethod{UserID: user.ID, Name: "Visa"}
	if err := db.Create(&method).Error; err != nil {
		t.Fatalf("failed to create payment method: %v", err)
	}

	intervalCount := 1
	sub, err := service.NewSubscriptionService(db).Create(user.ID, service.CreateSubscriptionInput{
		Name:            "Claude Pro",
		Amount:          20,
		Currency:        "USD",
		BillingType:     "recurring",
		RecurrenceType:  "interval",
		IntervalCount:   &intervalCount,
		IntervalUnit:    "month",
		NextBillingDate: "2026-06-15",
		CategoryID:      &category.ID,
		PaymentMethodID: &method.ID,
	})
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	rec, resp := performMCPRequest(t, handler, apiKey, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "update_subscription",
			"arguments": map[string]interface{}{
				"id":                sub.ID,
				"category_id":       nil,
				"payment_method_id": nil,
			},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	result := resp["result"].(map[string]interface{})
	if result["isError"] == true {
		t.Fatalf("update_subscription returned tool error: %#v", result)
	}
	updated := result["structuredContent"].(map[string]interface{})
	if updated["category_id"] != nil {
		t.Fatalf("category_id = %v, want nil", updated["category_id"])
	}
	if updated["payment_method_id"] != nil {
		t.Fatalf("payment_method_id = %v, want nil", updated["payment_method_id"])
	}

	var stored model.Subscription
	if err := db.Where("id = ? AND user_id = ?", sub.ID, user.ID).First(&stored).Error; err != nil {
		t.Fatalf("failed to reload subscription: %v", err)
	}
	if stored.CategoryID != nil {
		t.Fatalf("stored category_id = %v, want nil", *stored.CategoryID)
	}
	if stored.PaymentMethodID != nil {
		t.Fatalf("stored payment_method_id = %v, want nil", *stored.PaymentMethodID)
	}
}

func TestMCPMissingSubscriptionIDsReturnToolErrors(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	for _, id := range []int{-1, 0} {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			rec, resp := performMCPRequest(t, handler, apiKey, map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/call",
				"params": map[string]interface{}{
					"name": "get_subscription",
					"arguments": map[string]interface{}{
						"id": id,
					},
				},
			})
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if _, ok := resp["error"]; ok {
				t.Fatalf("response has RPC error, want tool error: %#v", resp)
			}

			result := resp["result"].(map[string]interface{})
			if result["isError"] != true {
				t.Fatalf("isError = %v, want true; response = %#v", result["isError"], result)
			}
			structured := result["structuredContent"].(map[string]interface{})
			if structured["error"] != "subscription not found" {
				t.Fatalf("tool error = %v, want subscription not found", structured["error"])
			}
		})
	}
}

func TestMCPRejectsCrossOriginRequest(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	req.Host = "localhost:8080"
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set(echo.HeaderOrigin, "https://evil.example.com")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.HandlePost(c); err != nil {
		t.Fatalf("HandlePost() error = %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
