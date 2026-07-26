package pkg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"

	"gorm.io/gorm"
)

// ErrDatabaseNotReopenable is returned by ReopenDatabase when the handle was
// not created by openSQLiteDatabase and therefore carries no switchable pool.
// Tests that call gorm.Open directly land here; callers must treat it as
// "reopen unavailable, ask for a process restart" rather than a failure of the
// operation that triggered the reopen.
var ErrDatabaseNotReopenable = errors.New("database handle was not opened through InitDB and cannot be reopened in place")

// switchableConnPool is a gorm connection pool that indirects every call
// through an atomically replaceable *sql.DB.
//
// Restoring a backup swaps the SQLite file underneath a running process: the
// live pool must be closed before the file is replaced, and a brand-new pool
// must be opened against the restored file afterwards. Without this
// indirection the swap would be impossible, because the *gorm.DB handle is
// captured at startup and shared by every service, handler, and background
// worker; each one would keep pointing at the closed pool and fail with
// "sql: database is closed" until the process restarted. Holders keep their
// *gorm.DB, and ReopenDatabase atomically stores a fresh *sql.DB here.
//
// The implemented interface set is exactly what gorm v1.31.2 asserts on a
// ConnPool along the code paths this project uses:
//
//   - gorm.ConnPool (PrepareContext/ExecContext/QueryContext/QueryRowContext)
//     is the base interface every query goes through.
//   - gorm.TxBeginner (BeginTx returning *sql.Tx) is required because gorm
//     wraps default Create/Update/Delete callbacks in a transaction
//     (SkipDefaultTransaction is not set). gorm type-switches on this in
//     DB.Begin and falls back to ErrInvalidTransaction at RUNTIME, so a
//     missing BeginTx would compile fine and break every write.
//   - gorm.GetDBConnector (GetDBConn) is what DB.DB() resolves through, which
//     keeps restore, pool configuration, and shutdown pointed at the live pool.
//   - interface{ Ping() error } is asserted by gorm.Open when automatic ping is
//     enabled. gorm.Open runs before this wrapper is installed, but Ping is
//     cheap to provide and keeps the wrapper substitutable for a raw *sql.DB.
//
// Deliberately not implemented: gorm.TxCommitter (Commit/Rollback) and the
// gorm.Tx interface — this is a pool, not a transaction, and gorm probes
// TxCommitter to detect nested transactions; claiming it would corrupt that
// detection. gorm.ConnPoolBeginner is redundant because TxBeginner is matched
// first in both DB.Begin and PreparedStmtDB. The PreparedStmtDB/PreparedStmtTX
// paths are unreachable here because PrepareStmt is not enabled.
type switchableConnPool struct {
	current atomic.Pointer[sql.DB]
}

var (
	_ gorm.ConnPool             = (*switchableConnPool)(nil)
	_ gorm.TxBeginner           = (*switchableConnPool)(nil)
	_ gorm.GetDBConnector       = (*switchableConnPool)(nil)
	_ interface{ Ping() error } = (*switchableConnPool)(nil)
)

// newSwitchableConnPool wraps the pool that gorm opened. The pool is never nil
// after construction: swap only ever stores another live *sql.DB.
func newSwitchableConnPool(sqlDB *sql.DB) *switchableConnPool {
	pool := &switchableConnPool{}
	pool.swap(sqlDB)
	return pool
}

func (p *switchableConnPool) swap(sqlDB *sql.DB) {
	p.current.Store(sqlDB)
}

func (p *switchableConnPool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return p.current.Load().PrepareContext(ctx, query)
}

func (p *switchableConnPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return p.current.Load().ExecContext(ctx, query, args...)
}

func (p *switchableConnPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return p.current.Load().QueryContext(ctx, query, args...)
}

func (p *switchableConnPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return p.current.Load().QueryRowContext(ctx, query, args...)
}

func (p *switchableConnPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return p.current.Load().BeginTx(ctx, opts)
}

func (p *switchableConnPool) GetDBConn() (*sql.DB, error) {
	return p.current.Load(), nil
}

func (p *switchableConnPool) Ping() error {
	return p.current.Load().Ping()
}

// switchableConnPoolOf finds the wrapper installed by openSQLiteDatabase.
// Session clones (WithContext, Transaction) copy Statement.ConnPool and share
// Config, so both are checked.
func switchableConnPoolOf(db *gorm.DB) (*switchableConnPool, bool) {
	if db == nil {
		return nil, false
	}
	if db.Statement != nil {
		if pool, ok := db.Statement.ConnPool.(*switchableConnPool); ok {
			return pool, true
		}
	}
	if db.Config != nil {
		if pool, ok := db.Config.ConnPool.(*switchableConnPool); ok {
			return pool, true
		}
	}
	return nil, false
}

// ReopenDatabase reconnects db to the SQLite file at its configured path after
// the file was replaced on disk, without restarting the process.
//
// The caller is expected to have closed the previous pool (db.DB().Close())
// and swapped the database file already. Reopening goes through
// openSQLiteDatabase so the restored file gets the same DSN pragmas, the same
// single-connection pool sizing, the same pragma verification and — crucially
// — schema migrations, because a restored backup may predate the running
// binary's schema. The temporary *gorm.DB used for that work is discarded; only
// its *sql.DB is kept and stored into db's existing pool wrapper, so every
// holder of the shared *gorm.DB keeps working.
//
// The JWT secret is reloaded afterwards: when it is not pinned by the
// JWT_SECRET environment variable it lives in the restored database's
// system_settings, and skipping the reload would validate tokens against the
// pre-restore secret.
//
// Known limitation, deliberately not addressed here: the CORS allow-list is
// read once at startup (cmd/server/main.go loadCORSOrigins) and installed into
// the echo middleware chain, so a restore that changes the stored origins still
// needs a restart to take effect.
//
// Returns ErrDatabaseNotReopenable when db carries no pool wrapper.
func ReopenDatabase(db *gorm.DB) error {
	pool, ok := switchableConnPoolOf(db)
	if !ok {
		return ErrDatabaseNotReopenable
	}

	reopened, err := openSQLiteDatabase(filepath.Join(GetDataPath(), "subdux.db"))
	if err != nil {
		return fmt.Errorf("reopen database: %w", err)
	}
	sqlDB, err := reopened.DB()
	if err != nil {
		return fmt.Errorf("access reopened database pool: %w", err)
	}

	pool.swap(sqlDB)

	if err := InitJWTSecret(db); err != nil {
		return fmt.Errorf("reload JWT secret after reopening database: %w", err)
	}

	return nil
}
