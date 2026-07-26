package backup

import (
	"context"
	"io"
	"time"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

// ErrUnknownBackupDestinationType is returned when a destination row carries a
// type that has no registered target implementation.
var ErrUnknownBackupDestinationType = serviceerr.New(serviceerr.KindInvalid, "unknown_backup_destination_type", "unknown backup destination type")

// BackupObject describes a single stored backup archive at a destination. It is
// the destination-agnostic unit that retention and listing operate on, so the
// same ordering logic applies whether the bytes live on local disk, in S3, or
// on a WebDAV server.
type BackupObject struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	Encrypted  bool      `json:"encrypted"`
}

// BackupTarget is a place a produced backup archive can be delivered to. The
// archive is fully built (and, when configured, already AES-256 encrypted)
// before it reaches a target, so a target only moves opaque bytes and never
// sees plaintext database contents. This is what keeps third-party transports
// (S3, WebDAV) out of the plaintext trust boundary.
//
// Retention is applied by the orchestration layer, which calls List, orders by
// ModifiedAt, and Deletes the excess — keeping per-destination retention logic
// in one place regardless of the underlying store.
type BackupTarget interface {
	// Type returns the destination type identifier (e.g. "local").
	Type() string
	// Put stores the archive under name, reading exactly size bytes from r.
	// Implementations should stream rather than buffer the whole archive.
	Put(ctx context.Context, name string, r io.Reader, size int64) error
	// List returns the backup archives currently held by this target, in no
	// guaranteed order. Each returned archive must have a non-zero ModifiedAt;
	// retention cannot safely order an archive whose modification time is
	// unavailable.
	List(ctx context.Context) ([]BackupObject, error)
	// Get opens the archive stored under name for reading. The returned
	// ReadCloser streams the archive bytes and MUST be closed by the caller;
	// closing it also releases the per-operation timeout the implementation
	// armed. size is the archive's length as the target reports it, or -1 when
	// the target cannot report one — callers must treat an unreported size as
	// unbounded and enforce their own limit while copying.
	//
	// Implementations apply the same name gate their Put and Delete apply, so a
	// caller can never read a resource outside the archive namespace.
	Get(ctx context.Context, name string) (io.ReadCloser, int64, error)
	// Delete removes a single archive by name. Removing a name that no longer
	// exists is not an error.
	Delete(ctx context.Context, name string) error
	// RetentionCount is the number of newest archives to keep at this target.
	RetentionCount() int
}

// backupObjectReader couples a target's response body to the context that
// bounds it. A per-operation timeout is armed inside Get but must stay armed
// while the archive streams, so the cancel func fires on Close rather than on
// return — the ordinary `defer cancel()` would tear down the transfer the
// moment Get handed the reader back.
type backupObjectReader struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *backupObjectReader) Close() error {
	err := r.ReadCloser.Close()
	r.cancel()
	return err
}
