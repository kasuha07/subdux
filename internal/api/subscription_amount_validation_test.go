package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	exchangerate "github.com/kasuha07/subdux/internal/service/exchangerate"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"github.com/labstack/echo/v4"
)

// The amounts below are spelled out rather than derived from
// contract.MaxSubscriptionAmount so a change to the bound fails these
// transport-level expectations loudly instead of moving with it.
func newAmountValidationHandler(t *testing.T) (*SubscriptionHandler, uint) {
	t.Helper()

	db := newBootstrapTestDB(t)
	user := model.User{Username: "tester", Email: "tester@example.com", Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	return NewSubscriptionHandler(subscriptionservice.NewService(db), exchangerate.NewService(db)), user.ID
}

func postSubscriptionJSON(t *testing.T, handler *SubscriptionHandler, userID uint, body string) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	if err := handler.Create(authedContext(e, rec, req, userID)); err != nil {
		t.Fatalf("Create handler returned error: %v", err)
	}
	return rec
}

func putSubscriptionJSON(t *testing.T, handler *SubscriptionHandler, userID uint, body string) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/subscriptions/1", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := authedContext(e, rec, req, userID)
	c.SetParamNames("id")
	c.SetParamValues("1")
	if err := handler.Update(c); err != nil {
		t.Fatalf("Update handler returned error: %v", err)
	}
	return rec
}

func TestCreateSubscriptionRejectsOutOfRangeAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount string
		want   string
	}{
		{name: "negative", amount: "-1", want: "amount_must_not_be_negative"},
		{name: "above the maximum", amount: "900000000000.01", want: "amount_too_large"},
		{name: "far above the maximum", amount: "1.8e306", want: "amount_too_large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userID := newAmountValidationHandler(t)
			rec := postSubscriptionJSON(t, handler, userID, `{"name":"Video Pro","amount":`+tt.amount+`,"currency":"USD"}`)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if !hasErrorCode(rec.Body.String(), tt.want) {
				t.Fatalf("body = %s, want error code %q", rec.Body.String(), tt.want)
			}
		})
	}
}

func TestCreateSubscriptionAcceptsMaximumAmount(t *testing.T) {
	handler, userID := newAmountValidationHandler(t)
	rec := postSubscriptionJSON(t, handler, userID,
		`{"name":"Video Pro","amount":900000000000,"currency":"USD","status":"active","renewal_mode":"auto_renew","billing_type":"recurring","recurrence_type":"interval","interval_count":1,"interval_unit":"month","next_billing_date":"2026-04-15"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestUpdateSubscriptionRejectsOutOfRangeAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount string
		want   string
	}{
		{name: "negative", amount: "-1", want: "amount_must_not_be_negative"},
		{name: "above the maximum", amount: "900000000000.01", want: "amount_too_large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userID := newAmountValidationHandler(t)
			rec := putSubscriptionJSON(t, handler, userID, `{"amount":`+tt.amount+`}`)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if !hasErrorCode(rec.Body.String(), tt.want) {
				t.Fatalf("body = %s, want error code %q", rec.Body.String(), tt.want)
			}
		})
	}
}
