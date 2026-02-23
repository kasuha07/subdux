package api

import "testing"

func TestValidateIconManagedAssetPath(t *testing.T) {
	if !validateIcon("file:1_2_3.png") {
		t.Fatal("validateIcon() should accept valid managed icon path")
	}
	if !validateIcon("file:1_2_3.ico") {
		t.Fatal("validateIcon() should accept valid managed ico path")
	}
}

func TestValidateIconRejectsAssetTraversal(t *testing.T) {
	tests := []string{
		"file:../../../data/subdux.db",
		"file:nested/icon.png",
		`file:..\..\data\subdux.db`,
		"file:icon.svg",
	}

	for _, icon := range tests {
		if validateIcon(icon) {
			t.Fatalf("validateIcon() should reject icon path %q", icon)
		}
	}
}

func TestValidateIconAcceptsIconGoValues(t *testing.T) {
	tests := []string{
		"lg:paypal",
		"lg:googlecloud",
		"bl:icbc",
		"bl:bankofchina",
		"ccp:paypal",
	}

	for _, icon := range tests {
		if !validateIcon(icon) {
			t.Fatalf("validateIcon() should accept IconGo icon value %q", icon)
		}
	}
}

func TestValidateIconRejectsUnsupportedIconValue(t *testing.T) {
	tests := []string{
		"si:paypal",
		"http://example.com/logo.png",
		"https://example.com/logo.png",
		"lg:PayPal",
		"LG:paypal",
		"lg:pay/pal",
		"file:abc",
	}

	for _, icon := range tests {
		if validateIcon(icon) {
			t.Fatalf("validateIcon() should reject icon value %q", icon)
		}
	}
}

func TestValidateSubscriptionIconAllowsExternalURL(t *testing.T) {
	tests := []string{
		"http://example.com/logo.png",
		"https://cdn.example.com/path/icon.jpg",
	}

	for _, icon := range tests {
		if !validateSubscriptionIcon(icon) {
			t.Fatalf("validateSubscriptionIcon() should accept external url %q", icon)
		}
	}
}

func TestValidateSubscriptionIconRejectsInvalidExternalURL(t *testing.T) {
	tests := []string{
		"javascript:alert(1)",
		"ftp://example.com/icon.png",
		"https:///icon.png",
	}

	for _, icon := range tests {
		if validateSubscriptionIcon(icon) {
			t.Fatalf("validateSubscriptionIcon() should reject invalid external url %q", icon)
		}
	}
}
