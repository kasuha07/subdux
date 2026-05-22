package pkg

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const sqliteBusyTimeoutMilliseconds = 5000

// GetDataPath returns the root data directory from the DATA_PATH environment
// variable, falling back to "data" when unset. The database, assets, and any
// other persistent files are stored under this directory.
func GetDataPath() string {
	if p := os.Getenv("DATA_PATH"); p != "" {
		return p
	}
	return "data"
}

func InitDB() *gorm.DB {
	dataPath := GetDataPath()
	if err := prepareDataPathRuntimeOwnership(dataPath); err != nil {
		log.Fatalf("Failed to prepare runtime ownership for data directory %q: %v", dataPath, err)
	}
	if err := ensureDataPathWritable(dataPath); err != nil {
		log.Fatalf("Failed to prepare data directory %q: %v", dataPath, err)
	}

	dbPath := filepath.Join(dataPath, "subdux.db")
	db, err := openSQLiteDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	return db
}

func openSQLiteDatabase(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := configureSQLiteDatabase(db); err != nil {
		return nil, err
	}
	if err := runSchemaMigrations(db); err != nil {
		return nil, err
	}

	return db, nil
}

func configureSQLiteDatabase(db *gorm.DB) error {
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return fmt.Errorf("enable sqlite foreign keys: %w", err)
	}

	journalMode, err := readSQLiteStringPragma(db, "PRAGMA journal_mode = WAL")
	if err != nil {
		return fmt.Errorf("enable sqlite wal mode: %w", err)
	}
	if strings.ToLower(journalMode) != "wal" {
		return fmt.Errorf("unexpected sqlite journal mode %q", journalMode)
	}

	if err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d", sqliteBusyTimeoutMilliseconds)).Error; err != nil {
		return fmt.Errorf("set sqlite busy_timeout: %w", err)
	}

	foreignKeys, err := readSQLiteIntPragma(db, "PRAGMA foreign_keys")
	if err != nil {
		return fmt.Errorf("verify sqlite foreign_keys pragma: %w", err)
	}
	if foreignKeys != 1 {
		return fmt.Errorf("sqlite foreign_keys pragma is %d, want 1", foreignKeys)
	}

	busyTimeout, err := readSQLiteIntPragma(db, "PRAGMA busy_timeout")
	if err != nil {
		return fmt.Errorf("verify sqlite busy_timeout pragma: %w", err)
	}
	if busyTimeout < sqliteBusyTimeoutMilliseconds {
		return fmt.Errorf("sqlite busy_timeout pragma is %dms, want at least %dms", busyTimeout, sqliteBusyTimeoutMilliseconds)
	}

	return nil
}

func readSQLiteIntPragma(db *gorm.DB, query string) (int, error) {
	var value int
	if err := db.Raw(query).Row().Scan(&value); err != nil {
		return 0, err
	}
	return value, nil
}

func readSQLiteStringPragma(db *gorm.DB, query string) (string, error) {
	var value string
	if err := db.Raw(query).Row().Scan(&value); err != nil {
		return "", err
	}
	return value, nil
}

func ensureDataPathWritable(dataPath string) error {
	info, err := os.Stat(dataPath)
	switch {
	case err == nil:
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory")
		}
	case os.IsNotExist(err):
		if err := os.MkdirAll(dataPath, 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	default:
		return fmt.Errorf("failed to inspect directory: %w", err)
	}

	probe, err := os.CreateTemp(dataPath, ".subdux-write-check-*")
	if err != nil {
		return fmt.Errorf("directory is not writable: %w", err)
	}
	probePath := probe.Name()

	if err := probe.Close(); err != nil {
		_ = os.Remove(probePath)
		return fmt.Errorf("failed to close write probe: %w", err)
	}

	if err := os.Remove(probePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean up write probe: %w", err)
	}

	return nil
}
