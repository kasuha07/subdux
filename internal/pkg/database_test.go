package pkg

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
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

func TestSQLiteDatabaseDSNAppliesPragmasToNewConnections(t *testing.T) {
	dsn, err := sqliteDatabaseDSN(filepath.Join(t.TempDir(), "subdux.db"))
	if err != nil {
		t.Fatalf("sqliteDatabaseDSN() error = %v", err)
	}

	db, err := sql.Open(sqlite.DriverName, dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	assertSQLiteConnectionPragmas(t, db)

	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("db.Conn() error = %v", err)
	}
	if err := conn.Raw(func(any) error { return driver.ErrBadConn }); err != nil && !errors.Is(err, driver.ErrBadConn) {
		t.Fatalf("mark connection bad error = %v", err)
	}
	if err := conn.Close(); err != nil && !errors.Is(err, sql.ErrConnDone) {
		t.Fatalf("conn.Close() error = %v", err)
	}

	assertSQLiteConnectionPragmas(t, db)
}

func TestOpenSQLiteDatabaseLimitsConnectionPool(t *testing.T) {
	db, err := openSQLiteDatabase(filepath.Join(t.TempDir(), "subdux.db"))
	if err != nil {
		t.Fatalf("openSQLiteDatabase() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	defer sqlDB.Close()

	if got, want := sqlDB.Stats().MaxOpenConnections, 1; got != want {
		t.Fatalf("MaxOpenConnections = %d, want %d", got, want)
	}
}

func assertSQLiteConnectionPragmas(t *testing.T, db *sql.DB) {
	t.Helper()

	foreignKeys := querySQLiteIntPragma(t, db, "PRAGMA foreign_keys")
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	busyTimeout := querySQLiteIntPragma(t, db, "PRAGMA busy_timeout")
	if busyTimeout < sqliteBusyTimeoutMilliseconds {
		t.Fatalf("busy_timeout = %d, want at least %d", busyTimeout, sqliteBusyTimeoutMilliseconds)
	}

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("read journal_mode pragma error = %v", err)
	}
	if strings.ToLower(journalMode) != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}
}

func querySQLiteIntPragma(t *testing.T, db *sql.DB, query string) int {
	t.Helper()

	var value int
	if err := db.QueryRow(query).Scan(&value); err != nil {
		t.Fatalf("%s error = %v", query, err)
	}
	return value
}
