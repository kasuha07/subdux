package pkg

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsureDataPathWritableCreatesMissingDirectory(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "missing")

	if err := ensureDataPathWritable(dataPath); err != nil {
		t.Fatalf("ensureDataPathWritable() error = %v", err)
	}

	info, err := os.Stat(dataPath)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v", dataPath, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q should be a directory", dataPath)
	}
}

func TestEnsureDataPathWritableRejectsFilePath(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", filePath, err)
	}

	err := ensureDataPathWritable(filePath)
	if err == nil {
		t.Fatal("ensureDataPathWritable() should fail when DATA_PATH points to a file")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("ensureDataPathWritable() error = %v, want not-a-directory detail", err)
	}
}

func TestEnsureDataPathWritableRejectsReadOnlyDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on Windows")
	}

	dataPath := filepath.Join(t.TempDir(), "readonly")
	if err := os.Mkdir(dataPath, 0o555); err != nil {
		t.Fatalf("os.Mkdir(%q) error = %v", dataPath, err)
	}

	probe, err := os.CreateTemp(dataPath, "preflight-*")
	if err == nil {
		_ = probe.Close()
		_ = os.Remove(probe.Name())
		t.Skip("current process can still write to 0555 directories")
	}

	err = ensureDataPathWritable(dataPath)
	if err == nil {
		t.Fatal("ensureDataPathWritable() should fail when DATA_PATH is not writable")
	}
	if !strings.Contains(err.Error(), "not writable") {
		t.Fatalf("ensureDataPathWritable() error = %v, want writable detail", err)
	}
}
