package service

import (
	"strings"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
)

func TestAdminCreateUserRejectsPasswordUnder8Characters(t *testing.T) {
	svc := NewAdminService(newTestDB(t))

	_, err := svc.CreateUser(CreateUserInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "short7!",
	})
	if err == nil {
		t.Fatal("CreateUser() error = nil, want validation error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "at least 8 characters") {
		t.Fatalf("CreateUser() error = %q, want 8-character validation error", err.Error())
	}
}

func TestAdminListUsersReportsCredentialFactorState(t *testing.T) {
	db := newTestDB(t)
	user := createLifecycleSecurityUser(t, db, "factor-user", "factor-user@example.com")
	if err := db.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"totp_enabled": true,
		"totp_secret":  "JBSWY3DPEHPK3PXP",
	}).Error; err != nil {
		t.Fatalf("failed to enable totp: %v", err)
	}
	for _, credentialID := range []string{"cred-list-1", "cred-list-2"} {
		passkey := model.PasskeyCredential{
			UserID:       user.ID,
			Name:         "Laptop",
			CredentialID: credentialID,
			Credential:   []byte("credential"),
		}
		if err := db.Create(&passkey).Error; err != nil {
			t.Fatalf("failed to create passkey: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		nextBillingDate := time.Now().UTC().AddDate(0, 0, i+1)
		subscription := model.Subscription{
			UserID:          user.ID,
			Name:            "Subscription",
			Amount:          9.99,
			Currency:        "USD",
			Status:          "active",
			RenewalMode:     "auto_renew",
			BillingType:     "recurring",
			NextBillingDate: &nextBillingDate,
		}
		if err := db.Create(&subscription).Error; err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}
	}

	users, err := NewAdminService(db).ListUsers()
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("ListUsers() count = %d, want 1", len(users))
	}
	if users[0].Username != "factor-user" {
		t.Fatalf("ListUsers()[0].Username = %q, want factor-user", users[0].Username)
	}
	if !users[0].TotpEnabled {
		t.Fatal("ListUsers()[0].TotpEnabled = false, want true")
	}
	if users[0].PasskeyCount != 2 {
		t.Fatalf("ListUsers()[0].PasskeyCount = %d, want 2", users[0].PasskeyCount)
	}
	if users[0].SubscriptionCount != 2 {
		t.Fatalf("ListUsers()[0].SubscriptionCount = %d, want 2", users[0].SubscriptionCount)
	}
}
