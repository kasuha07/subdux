package serviceerr

import "testing"

func TestNewCodePreservesParams(t *testing.T) {
	err := NewCode(KindInvalid, "backup_file_is_too_large_max_mb", "backup file is too large (max 32 MB)", map[string]any{
		"max_mb": 32,
	})

	if err.Code != "backup_file_is_too_large_max_mb" {
		t.Fatalf("Code = %q, want explicit code", err.Code)
	}
	if got := err.Params["max_mb"]; got != 32 {
		t.Fatalf("Params[max_mb] = %v, want 32", got)
	}
}
