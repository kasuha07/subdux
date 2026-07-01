package api

import (
	"archive/zip"
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	yekazip "github.com/yeka/zip"
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

	_, err := prepareRestorePayloadFromZipWithLimits(zipPath, "", backupRestoreLimits{
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
	pngData := mustEncodeRestorePNG(t)
	zipPath := writeRestoreZip(t, map[string][]byte{
		"subdux.db":          sqliteFileHeader,
		"assets/icons/a.png": pngData,
	})

	_, err := prepareRestorePayloadFromZipWithLimits(zipPath, "", backupRestoreLimits{
		maxDatabaseExtractedSize: 128,
		maxAssetsExtractedSize:   int64(len(pngData) - 1),
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

func TestPrepareRestorePayloadFromZipCountsOriginalAssetSize(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())
	pngData := mustEncodeRestorePNG(t)
	pngWithPayload := append(bytes.Clone(pngData), bytes.Repeat([]byte("x"), 64)...)
	zipPath := writeRestoreZip(t, map[string][]byte{
		"subdux.db":          sqliteFileHeader,
		"assets/icons/a.png": pngWithPayload,
		"assets/icons/b.png": pngWithPayload,
	})

	_, err := prepareRestorePayloadFromZipWithLimits(zipPath, "", backupRestoreLimits{
		maxDatabaseExtractedSize: 128,
		maxAssetsExtractedSize:   int64(len(pngWithPayload) + len(pngData)),
		maxAssetEntries:          4,
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
	pngData := mustEncodeRestorePNG(t)
	zipPath := writeRestoreZip(t, map[string][]byte{
		"subdux.db":          sqliteFileHeader,
		"assets/icons/1.png": pngData,
		"assets/icons/2.png": pngData,
		"assets/icons/3.png": pngData,
	})

	_, err := prepareRestorePayloadFromZipWithLimits(zipPath, "", backupRestoreLimits{
		maxDatabaseExtractedSize: 128,
		maxAssetsExtractedSize:   4096,
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
	pngData := append(mustEncodeRestorePNG(t), []byte("<script>evil()</script>")...)
	zipPath := writeRestoreZip(t, map[string][]byte{
		"subdux.db":          sqliteFileHeader,
		"assets/icons/a.png": pngData,
	})

	payload, err := prepareRestorePayloadFromZipWithLimits(zipPath, "", backupRestoreLimits{
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
	if bytes.Contains(contents, []byte("<script>")) {
		t.Fatal("restored asset should be sanitized and strip appended script payload")
	}
	if _, err := png.Decode(bytes.NewReader(contents)); err != nil {
		t.Fatalf("restored asset should decode as png: %v", err)
	}
}

func TestPrepareRestorePayloadFromZipRejectsNonImageAsset(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())
	zipPath := writeRestoreZip(t, map[string][]byte{
		"subdux.db":            sqliteFileHeader,
		"assets/icons/pwn.png": []byte("<script>evil()</script>"),
	})

	_, err := prepareRestorePayloadFromZipWithLimits(zipPath, "", backupRestoreLimits{
		maxDatabaseExtractedSize: 128,
		maxAssetsExtractedSize:   128,
		maxAssetEntries:          2,
	})
	if !errors.Is(err, errInvalidBackup) {
		t.Fatalf("prepareRestorePayloadFromZipWithLimits() error = %v, want invalid backup", err)
	}
	if !strings.Contains(err.Error(), "invalid asset image") {
		t.Fatalf("error = %q, want invalid asset image", err.Error())
	}
	assertNoRestoreAssetsTempDirs(t)
}

func TestPrepareRestorePayloadFromZipRejectsExecutableAssetPath(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())
	zipPath := writeRestoreZip(t, map[string][]byte{
		"subdux.db":         sqliteFileHeader,
		"assets/pwn.html":   []byte("<script>evil()</script>"),
		"assets/icons/a.js": []byte("alert(1)"),
	})

	_, err := prepareRestorePayloadFromZipWithLimits(zipPath, "", backupRestoreLimits{
		maxDatabaseExtractedSize: 128,
		maxAssetsExtractedSize:   128,
		maxAssetEntries:          2,
	})
	if !errors.Is(err, errInvalidBackup) {
		t.Fatalf("prepareRestorePayloadFromZipWithLimits() error = %v, want invalid backup", err)
	}
	if !strings.Contains(err.Error(), "unsupported assets entry") {
		t.Fatalf("error = %q, want unsupported assets entry", err.Error())
	}
	assertNoRestoreAssetsTempDirs(t)
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

func mustEncodeRestorePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.NRGBA{R: 42, G: 120, B: 220, A: 255})
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatalf("failed to encode png: %v", err)
	}
	return out.Bytes()
}

// writeEncryptedRestoreZip builds a WinZip AES-256 encrypted archive whose
// entries are all protected with the given password, matching what the download
// path produces via writeBackupZipFromDB.
func writeEncryptedRestoreZip(t *testing.T, password string, files map[string][]byte) string {
	t.Helper()

	zipPath := filepath.Join(t.TempDir(), "backup.zip")
	out, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create encrypted zip: %v", err)
	}
	zipWriter := yekazip.NewWriter(out)

	for name, contents := range files {
		writer, err := zipWriter.Encrypt(name, password, yekazip.AES256Encryption)
		if err != nil {
			t.Fatalf("failed to create encrypted zip entry %q: %v", name, err)
		}
		if _, err := writer.Write(contents); err != nil {
			t.Fatalf("failed to write encrypted zip entry %q: %v", name, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close encrypted zip writer: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("failed to close encrypted zip file: %v", err)
	}

	return zipPath
}

func TestPrepareRestorePayloadPlaintextZipStillRestores(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())
	zipPath := writeRestoreZip(t, map[string][]byte{
		"subdux.db": sqliteFileHeader,
	})

	payload, err := prepareRestorePayload(zipPath, "")
	if err != nil {
		t.Fatalf("prepareRestorePayload() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = os.Remove(payload.dbFilePath) })
	if payload.dbFilePath == "" {
		t.Fatal("payload.dbFilePath is empty")
	}
	if !isSQLiteBackupFile(payload.dbFilePath) {
		t.Fatal("restored db is not a valid SQLite file")
	}
}

func TestPrepareRestorePayloadEncryptedZipCorrectPassword(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())
	const password = "restore-secret"
	pngData := mustEncodeRestorePNG(t)
	zipPath := writeEncryptedRestoreZip(t, password, map[string][]byte{
		"subdux.db":          sqliteFileHeader,
		"assets/icons/a.png": pngData,
	})

	payload, err := prepareRestorePayload(zipPath, password)
	if err != nil {
		t.Fatalf("prepareRestorePayload() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(payload.dbFilePath)
		if payload.assetsDirPath != "" {
			_ = os.RemoveAll(payload.assetsDirPath)
		}
	})

	if !isSQLiteBackupFile(payload.dbFilePath) {
		t.Fatal("decrypted db is not a valid SQLite file")
	}
	if !payload.replaceAssetsDir {
		t.Fatal("payload.replaceAssetsDir = false, want true")
	}
	assetPath := filepath.Join(payload.assetsDirPath, "icons", "a.png")
	if _, err := os.Stat(assetPath); err != nil {
		t.Fatalf("decrypted asset stat error = %v, want nil", err)
	}
}

func TestPrepareRestorePayloadEncryptedZipWrongPassword(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())
	zipPath := writeEncryptedRestoreZip(t, "right-password", map[string][]byte{
		"subdux.db": sqliteFileHeader,
	})

	_, err := prepareRestorePayload(zipPath, "wrong-password")
	if !errors.Is(err, errBackupInvalidPassword) {
		t.Fatalf("prepareRestorePayload() error = %v, want errBackupInvalidPassword", err)
	}
}

func TestPrepareRestorePayloadEncryptedZipMissingPassword(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())
	zipPath := writeEncryptedRestoreZip(t, "some-password", map[string][]byte{
		"subdux.db": sqliteFileHeader,
	})

	_, err := prepareRestorePayload(zipPath, "")
	if !errors.Is(err, errBackupPasswordRequired) {
		t.Fatalf("prepareRestorePayload() error = %v, want errBackupPasswordRequired", err)
	}
}
