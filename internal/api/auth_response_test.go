package api

import (
	"encoding/json"
	"testing"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/service"
)

func TestMapAuthResponseOmitsRefreshToken(t *testing.T) {
	response := mapAuthResponse(&service.AuthResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		User: model.User{
			ID:          1,
			Username:    "alice",
			Email:       "alice@example.com",
			Role:        "user",
			Status:      "active",
			TotpEnabled: true,
		},
	})

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := payload["refresh_token"]; ok {
		t.Fatalf("auth response should not include refresh_token: %s", string(data))
	}
	if payload["access_token"] != "access-token" {
		t.Fatalf("access_token = %v, want access-token", payload["access_token"])
	}
	if payload["token"] != "access-token" {
		t.Fatalf("token = %v, want access-token", payload["token"])
	}
}

func TestMapLoginResponseOmitsRefreshToken(t *testing.T) {
	user := model.User{
		ID:       1,
		Username: "alice",
		Email:    "alice@example.com",
		Role:     "user",
		Status:   "active",
	}
	response := mapLoginResponse(&service.LoginResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		User:         &user,
	})

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := payload["refresh_token"]; ok {
		t.Fatalf("login response should not include refresh_token: %s", string(data))
	}
	if payload["access_token"] != "access-token" {
		t.Fatalf("access_token = %v, want access-token", payload["access_token"])
	}
	if payload["token"] != "access-token" {
		t.Fatalf("token = %v, want access-token", payload["token"])
	}
}

func TestMapOIDCSessionResponseOmitsRefreshToken(t *testing.T) {
	user := model.User{
		ID:       1,
		Username: "alice",
		Email:    "alice@example.com",
		Role:     "user",
		Status:   "active",
	}
	response := mapOIDCSessionResponse(&service.OIDCSessionResult{
		Purpose:      "login",
		Token:        "access-token",
		RefreshToken: "refresh-token",
		User:         &user,
	})

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := payload["refresh_token"]; ok {
		t.Fatalf("oidc response should not include refresh_token: %s", string(data))
	}
	if payload["access_token"] != "access-token" {
		t.Fatalf("access_token = %v, want access-token", payload["access_token"])
	}
	if payload["token"] != "access-token" {
		t.Fatalf("token = %v, want access-token", payload["token"])
	}
}
