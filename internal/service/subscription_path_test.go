package service

import (
	"path/filepath"
	"testing"
)

func TestManagedIconFilePath(t *testing.T) {
	tests := []struct {
		name     string
		icon     string
		wantPath string
		wantOK   bool
	}{
		{
			name:     "valid png icon path",
			icon:     "file:1_2_3.png",
			wantPath: filepath.Join("data", "assets", "icons", "1_2_3.png"),
			wantOK:   true,
		},
		{
			name:     "valid jpg icon path",
			icon:     "file:9_8_7.jpg",
			wantPath: filepath.Join("data", "assets", "icons", "9_8_7.jpg"),
			wantOK:   true,
		},
		{
			name:   "reject traversal outside data directory",
			icon:   "file:../../data/subdux.db",
			wantOK: false,
		},
		{
			name:   "reject traversal under icons prefix",
			icon:   "file:../../../data/subdux.db",
			wantOK: false,
		},
		{
			name:   "reject nested path",
			icon:   "file:nested/icon.png",
			wantOK: false,
		},
		{
			name:   "reject windows separator",
			icon:   `file:..\..\data\subdux.db`,
			wantOK: false,
		},
		{
			name:   "reject empty filename",
			icon:   "file:",
			wantOK: false,
		},
		{
			name:   "reject non-image extension",
			icon:   "file:icon.svg",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotOK := managedIconFilePath(tt.icon)
			if gotOK != tt.wantOK {
				t.Fatalf("managedIconFilePath() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotPath != tt.wantPath {
				t.Fatalf("managedIconFilePath() path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}
