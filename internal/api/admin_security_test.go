package api

import (
	"archive/zip"
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCreateUserRejectsPasswordOver72Bytes(t *testing.T) {
	e := echo.New()
	body := `{"username":"alice","email":"alice@example.com","password":"` + strings.Repeat("a", 73) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := &AdminHandler{}
	if err := handler.CreateUser(c); err != nil {
		t.Fatalf("CreateUser() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "72 bytes") {
		t.Fatalf("expected error message to mention 72 bytes, got %s", rec.Body.String())
	}
}

func TestCreateUserRejectsPasswordUnder8Characters(t *testing.T) {
	e := echo.New()
	body := `{"username":"alice","email":"alice@example.com","password":"short7!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := &AdminHandler{}
	if err := handler.CreateUser(c); err != nil {
		t.Fatalf("CreateUser() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "at least 8 characters") {
		t.Fatalf("expected error message to mention 8 characters, got %s", rec.Body.String())
	}
}

func TestIsRequestTooLargeError(t *testing.T) {
	if !isRequestTooLargeError(&http.MaxBytesError{Limit: 1}) {
		t.Fatal("expected MaxBytesError to be detected")
	}
	if isRequestTooLargeError(nil) {
		t.Fatal("nil error should not be treated as request-too-large")
	}
}

func TestPrepareRestorePayloadFromZipRejectsOversizedDatabaseEntry(t *testing.T) {
	zipPath := writeRestoreZip(t, map[string][]byte{
		"subdux.db": append(sqliteFileHeader, bytes.Repeat([]byte("x"), 32)...),
	})

	_, err := prepareRestorePayloadFromZipWithLimits(zipPath, backupRestoreLimits{
		maxDatabaseExtractedSize: int64(len(sqliteFileHeader) + 8),
		maxAssetsExtractedSize:   128,
		maxAssetEntries:          4,
	})
	if !errors.Is(err, errInvalidBackup) {
		t.Fatalf("prepareRestorePayloadFromZipWithLimits() error = %v, want invalid backup", err)
	}
	if !strings.Contains(err.Error(), "database exceeds extracted size limit") {
		t.Fatalf("error = %q, want database size limit", err.Error())
	}
}

func TestPrepareRestorePayloadFromZipRejectsOversizedAssetsTotal(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())
	zipPath := writeRestoreZip(t, map[string][]byte{
		"subdux.db":           sqliteFileHeader,
		"assets/icons/a.png":  bytes.Repeat([]byte("a"), 10),
		"assets/icons/b.png":  bytes.Repeat([]byte("b"), 10),
		"assets/icons/c.png":  bytes.Repeat([]byte("c"), 10),
		"assets/icons/d.png":  bytes.Repeat([]byte("d"), 10),
		"assets/icons/e.png":  bytes.Repeat([]byte("e"), 10),
		"assets/icons/ok.png": []byte("ok"),
	})

	_, err := prepareRestorePayloadFromZipWithLimits(zipPath, backupRestoreLimits{
		maxDatabaseExtractedSize: 128,
		maxAssetsExtractedSize:   32,
		maxAssetEntries:          8,
	})
	if !errors.Is(err, errInvalidBackup) {
		t.Fatalf("prepareRestorePayloadFromZipWithLimits() error = %v, want invalid backup", err)
	}
	if !strings.Contains(err.Error(), "assets exceed extracted size limit") {
		t.Fatalf("error = %q, want assets size limit", err.Error())
	}
	assertNoRestoreAssetsTempDirs(t)
}

func TestPrepareRestorePayloadFromZipRejectsTooManyAssetEntries(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())
	zipPath := writeRestoreZip(t, map[string][]byte{
		"subdux.db":          sqliteFileHeader,
		"assets/icons/1.png": []byte("1"),
		"assets/icons/2.png": []byte("2"),
		"assets/icons/3.png": []byte("3"),
	})

	_, err := prepareRestorePayloadFromZipWithLimits(zipPath, backupRestoreLimits{
		maxDatabaseExtractedSize: 128,
		maxAssetsExtractedSize:   128,
		maxAssetEntries:          2,
	})
	if !errors.Is(err, errInvalidBackup) {
		t.Fatalf("prepareRestorePayloadFromZipWithLimits() error = %v, want invalid backup", err)
	}
	if !strings.Contains(err.Error(), "too many assets") {
		t.Fatalf("error = %q, want too many assets", err.Error())
	}
	assertNoRestoreAssetsTempDirs(t)
}

func TestPrepareRestorePayloadFromZipAcceptsBackupWithinExtractedLimits(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())
	zipPath := writeRestoreZip(t, map[string][]byte{
		"subdux.db":          sqliteFileHeader,
		"assets/icons/a.png": []byte("icon"),
	})

	payload, err := prepareRestorePayloadFromZipWithLimits(zipPath, backupRestoreLimits{
		maxDatabaseExtractedSize: 128,
		maxAssetsExtractedSize:   128,
		maxAssetEntries:          2,
	})
	if err != nil {
		t.Fatalf("prepareRestorePayloadFromZipWithLimits() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(payload.dbFilePath)
		if payload.assetsDirPath != "" {
			_ = os.RemoveAll(payload.assetsDirPath)
		}
	})

	if payload.dbFilePath == "" {
		t.Fatal("payload.dbFilePath is empty")
	}
	if _, err := os.Stat(payload.dbFilePath); err != nil {
		t.Fatalf("restored db file stat error = %v, want nil", err)
	}
	if !payload.replaceAssetsDir {
		t.Fatal("payload.replaceAssetsDir = false, want true")
	}
	if payload.assetsDirPath == "" {
		t.Fatal("payload.assetsDirPath is empty")
	}
	assetPath := filepath.Join(payload.assetsDirPath, "icons", "a.png")
	contents, err := os.ReadFile(assetPath)
	if err != nil {
		t.Fatalf("restored asset read error = %v, want nil", err)
	}
	if string(contents) != "icon" {
		t.Fatalf("restored asset = %q, want icon", string(contents))
	}
}

func writeRestoreZip(t *testing.T, files map[string][]byte) string {
	t.Helper()

	zipPath := filepath.Join(t.TempDir(), "backup.zip")
	out, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip: %v", err)
	}
	zipWriter := zip.NewWriter(out)

	for name, contents := range files {
		writer, err := zipWriter.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %q: %v", name, err)
		}
		if _, err := writer.Write(contents); err != nil {
			t.Fatalf("failed to write zip entry %q: %v", name, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("failed to close zip file: %v", err)
	}

	return zipPath
}

func assertNoRestoreAssetsTempDirs(t *testing.T) {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(os.Getenv("DATA_PATH"), ".subdux-restore-assets-*"))
	if err != nil {
		t.Fatalf("failed to glob restore asset temp dirs: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("restore asset temp dirs were not cleaned up: %v", matches)
	}
}
