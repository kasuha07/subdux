package api

import "testing"

func TestValidateIconManagedAssetPath(t *testing.T) {
	if !validateIcon("assets/icons/1_2_3.png") {
		t.Fatal("validateIcon() should accept valid managed icon path")
	}
}

func TestValidateIconRejectsAssetTraversal(t *testing.T) {
	tests := []string{
		"assets/../../data/subdux.db",
		"assets/icons/../../../data/subdux.db",
		"assets/icons/nested/icon.png",
	}

	for _, icon := range tests {
		if validateIcon(icon) {
			t.Fatalf("validateIcon() should reject icon path %q", icon)
		}
	}
}
