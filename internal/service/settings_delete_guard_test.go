package service

import (
	"errors"
	"testing"

	"github.com/shiroha/subdux/internal/model"
)

func TestCurrencyDeleteBlockedWhenUsedBySubscription(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	currencyService := NewCurrencyService(db)

	currency, err := currencyService.Create(user.ID, CreateCurrencyInput{
		Code:   "EUR",
		Symbol: "€",
		Alias:  "Euro",
	})
	if err != nil {
		t.Fatalf("create currency failed: %v", err)
	}

	sub := model.Subscription{
		UserID:   user.ID,
		Name:     "Netflix",
		Amount:   9.99,
		Currency: "EUR",
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	err = currencyService.Delete(user.ID, currency.ID, "USD")
	if !errors.Is(err, ErrCurrencyInUse) {
		t.Fatalf("delete currency error = %v, want %v", err, ErrCurrencyInUse)
	}

	var existing model.UserCurrency
	if err := db.Where("id = ? AND user_id = ?", currency.ID, user.ID).First(&existing).Error; err != nil {
		t.Fatalf("currency should still exist: %v", err)
	}
}

func TestCategoryDeleteBlockedWhenUsedBySubscription(t *testing.T) {
	tests := []struct {
		name             string
		withCategoryID   bool
		withCategoryName bool
	}{
		{
			name:           "blocks when subscription references category_id",
			withCategoryID: true,
		},
		{
			name:             "blocks when subscription references legacy category name",
			withCategoryName: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			user := createTestUser(t, db)
			categoryService := NewCategoryService(db)

			category, err := categoryService.Create(user.ID, CreateCategoryInput{Name: "Video"})
			if err != nil {
				t.Fatalf("create category failed: %v", err)
			}

			sub := model.Subscription{
				UserID:   user.ID,
				Name:     "Netflix",
				Amount:   9.99,
				Currency: "USD",
			}
			if tc.withCategoryID {
				sub.CategoryID = &category.ID
			}
			if tc.withCategoryName {
				sub.Category = category.Name
			}

			if err := db.Create(&sub).Error; err != nil {
				t.Fatalf("create subscription failed: %v", err)
			}

			err = categoryService.Delete(user.ID, category.ID)
			if !errors.Is(err, ErrCategoryInUse) {
				t.Fatalf("delete category error = %v, want %v", err, ErrCategoryInUse)
			}
		})
	}
}

func TestPaymentMethodDeleteBlockedWhenUsedBySubscription(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	paymentMethodService := NewPaymentMethodService(db)

	method, err := paymentMethodService.Create(user.ID, CreatePaymentMethodInput{
		Name: "Credit Card",
	})
	if err != nil {
		t.Fatalf("create payment method failed: %v", err)
	}

	sub := model.Subscription{
		UserID:          user.ID,
		Name:            "Netflix",
		Amount:          9.99,
		Currency:        "USD",
		PaymentMethodID: &method.ID,
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	err = paymentMethodService.Delete(user.ID, method.ID)
	if !errors.Is(err, ErrPaymentMethodInUse) {
		t.Fatalf("delete payment method error = %v, want %v", err, ErrPaymentMethodInUse)
	}

	var existing model.PaymentMethod
	if err := db.Where("id = ? AND user_id = ?", method.ID, user.ID).First(&existing).Error; err != nil {
		t.Fatalf("payment method should still exist: %v", err)
	}
}
