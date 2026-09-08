package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestMutationRoutesRequireRevision(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)
	for _, tc := range []struct{ method, path, body string }{
		{"PUT", "/api/subscriptions/1", `{"name":"edit"}`},
		{"DELETE", "/api/subscriptions/1", ``},
		{"POST", "/api/subscriptions/1/mark-renewed", `{}`},
		{"POST", "/api/subscriptions/batch", `{"action":"delete","ids":[1]}`},
		{"PUT", "/api/categories/1", `{"name":"edit"}`},
		{"PUT", "/api/payment-methods/1", `{"name":"edit"}`},
		{"PUT", "/api/currencies/1", `{"alias":"edit"}`},
		{"PUT", "/api/categories/reorder", `[{"id":1,"sort_order":0}]`},
		{"PUT", "/api/payment-methods/reorder", `[{"id":1,"sort_order":0}]`},
		{"PUT", "/api/currencies/reorder", `[{"id":1,"sort_order":0}]`},
		{"PUT", "/api/notifications/channels/1", `{"enabled":false}`},
		{"PUT", "/api/notifications/templates/1", `{"template":"edit"}`},
		{"PUT", "/api/notifications/policy", `{"days_before":5}`},
		{"PUT", "/api/admin/settings", `{"site_name":"edit"}`},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "revision_required") {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestSubscriptionHTTPRevisionConflictPreservesWinner(t *testing.T) {
	handler, userID := newBatchHandler(t)
	sub := seedBatchHandlerSubscription(t, handler, userID, "original")
	for _, tc := range []struct {
		body   string
		status int
	}{
		{fmt.Sprintf(`{"name":"winner","revision":%d}`, sub.Revision), http.StatusOK},
		{fmt.Sprintf(`{"name":"stale","revision":%d}`, sub.Revision), http.StatusConflict},
		{`{"name":"missing"}`, http.StatusBadRequest},
		{`{"name":"zero","revision":0}`, http.StatusBadRequest},
	} {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPut, "/api/subscriptions/1", strings.NewReader(tc.body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := authedContext(e, rec, req, userID)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(sub.ID))
		if err := handler.Update(c); err != nil {
			APIErrorHandler(e.HTTPErrorHandler)(err, c)
		}
		if rec.Code != tc.status {
			t.Fatalf("status=%d want=%d body=%s", rec.Code, tc.status, rec.Body.String())
		}
		if tc.status == http.StatusConflict {
			var body map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body["error_code"] != "revision_conflict" {
				t.Fatalf("body=%v", body)
			}
		}
	}
	fresh, err := handler.Service.GetByID(userID, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Name != "winner" || fresh.Revision != sub.Revision+1 {
		t.Fatalf("stored=%+v", fresh)
	}
}
