package service

import (
	"strings"
	"testing"

	"github.com/shiroha/subdux/internal/model"
)

func TestGenerateCalendarTokenStoresHashAndListHidesToken(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.CalendarToken{}); err != nil {
		t.Fatalf("failed to migrate calendar tokens: %v", err)
	}

	user := createTestUser(t, db)
	svc := NewCalendarService(db)

	created, err := svc.GenerateToken(user.ID, "Personal")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if created.Token == "" {
		t.Fatal("GenerateToken() returned empty token")
	}

	var stored model.CalendarToken
	if err := db.First(&stored, created.ID).Error; err != nil {
		t.Fatalf("failed to load stored calendar token: %v", err)
	}
	if stored.Token == created.Token {
		t.Fatal("calendar token should not be stored in plaintext")
	}
	if stored.Token != hashCalendarToken(created.Token) {
		t.Fatalf("stored token hash = %q, want %q", stored.Token, hashCalendarToken(created.Token))
	}

	userID, err := svc.ValidateToken(created.Token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if userID != user.ID {
		t.Fatalf("ValidateToken() userID = %d, want %d", userID, user.ID)
	}

	listed, err := svc.ListTokens(user.ID)
	if err != nil {
		t.Fatalf("ListTokens() error = %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("ListTokens() len = %d, want 1", len(listed))
	}
	if listed[0].Token != "" {
		t.Fatalf("ListTokens() token = %q, want empty", listed[0].Token)
	}
}

func TestValidateCalendarTokenMigratesLegacyPlaintextToken(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.CalendarToken{}); err != nil {
		t.Fatalf("failed to migrate calendar tokens: %v", err)
	}

	user := createTestUser(t, db)
	legacyToken := strings.Repeat("a", 64)
	stored := model.CalendarToken{UserID: user.ID, Token: legacyToken, Name: "Legacy"}
	if err := db.Create(&stored).Error; err != nil {
		t.Fatalf("failed to create legacy calendar token: %v", err)
	}

	svc := NewCalendarService(db)
	userID, err := svc.ValidateToken(legacyToken)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if userID != user.ID {
		t.Fatalf("ValidateToken() userID = %d, want %d", userID, user.ID)
	}

	var migrated model.CalendarToken
	if err := db.First(&migrated, stored.ID).Error; err != nil {
		t.Fatalf("failed to reload migrated token: %v", err)
	}
	if migrated.Token != hashCalendarToken(legacyToken) {
		t.Fatalf("migrated token hash = %q, want %q", migrated.Token, hashCalendarToken(legacyToken))
	}
}
