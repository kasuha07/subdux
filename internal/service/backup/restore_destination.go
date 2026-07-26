package backup

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

var (
	// ErrBackupArchiveNotFound is returned when the requested archive is not in
	// the destination's own listing. Membership in List is the authoritative
	// name gate: it makes this path incapable of fetching anything the browse
	// UI did not show, which the per-target name checks alone cannot guarantee
	// (they only prove a name is well formed, not that it is a backup we put
	// there).
	ErrBackupArchiveNotFound = serviceerr.New(
		serviceerr.KindNotFound,
		"backup_archive_not_found",
		"backup archive not found at this destination",
	)

	// ErrBackupArchiveDownloadFailed wraps a transport failure while fetching an
	// archive. It is typed so the central error handler reports a dependency
	// failure instead of writeRestoreBackupError's generic 500 "failed to
	// restore backup" — which would blame the restore for an endpoint that
	// never answered.
	ErrBackupArchiveDownloadFailed = serviceerr.New(
		serviceerr.KindUnavailable,
		"backup_archive_download_failed",
		"the backup archive could not be downloaded from the destination",
	)
)

// maxDestinationRestoreArchiveSize bounds the archive a destination restore will
// pull down. It is deliberately the same 32 MiB the upload path allows
// (maxBackupUploadSize in internal/api/admin_backup.go), so "restorable" means
// one thing regardless of where the archive came from. It cannot import that
// constant — the service layer does not depend on the API layer — so the two
// are kept equal by this comment and by
// TestRestoreBackupDestinationRejectsOversizedArchive in internal/api, which
// asserts the reported max_mb.
const maxDestinationRestoreArchiveSize = 32 << 20

// RestoreDestinationBackup restores the database from an archive that already
// lives at a saved destination, so an admin can roll back without first pulling
// the file down by hand and re-uploading it.
//
// The destination row is read before RestoreBackup runs, because RestoreBackup
// closes the SQL handle before replacing the database file.
func (s *Service) RestoreDestinationBackup(
	ctx context.Context,
	id uint,
	archiveName string,
	password string,
) (RestoreResult, error) {
	var destination model.BackupDestination
	if err := s.DB.First(&destination, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return RestoreResult{}, ErrBackupDestinationNotFound
		}
		return RestoreResult{}, err
	}

	target, err := newBackupTarget(destination, s.DB)
	if err != nil {
		return RestoreResult{}, err
	}

	objects, err := target.List(ctx)
	if err != nil {
		// A destination that never answered must not be blamed on the restore;
		// joining the typed sentinel first makes errors.As resolve 503
		// backup_archive_download_failed instead of the generic 500 the
		// untyped error would otherwise render as.
		return RestoreResult{}, errors.Join(ErrBackupArchiveDownloadFailed, err)
	}
	var listed *BackupObject
	for i := range objects {
		if objects[i].Name == archiveName {
			listed = &objects[i]
			break
		}
	}
	if listed == nil {
		return RestoreResult{}, ErrBackupArchiveNotFound
	}
	// Refuse before any transfer when the destination already says the archive
	// is too big; the streaming cap below is what actually holds if it lies.
	if listed.Size > maxDestinationRestoreArchiveSize {
		return RestoreResult{}, errBackupArchiveTooLarge(maxDestinationRestoreArchiveSize)
	}

	archivePath, cleanup, err := downloadArchiveToTemp(ctx, target, archiveName, maxDestinationRestoreArchiveSize)
	if err != nil {
		return RestoreResult{}, err
	}
	defer cleanup()

	return s.RestoreBackup(archivePath, password)
}

// downloadArchiveToTemp streams one archive into a private 0700 temp directory
// and returns the staged path plus its cleanup. It takes a BackupTarget rather
// than a destination id so the size cap can be exercised against a target that
// misreports its own object sizes.
//
// The staged file is given a fixed name instead of the archive's: nothing the
// remote returns should ever name a path on this server, even after the name
// checks above. RestoreBackup sniffs content (isSQLiteBackupFile /
// isZipBackupFile), never the extension, so the name is free.
func downloadArchiveToTemp(
	ctx context.Context,
	target BackupTarget,
	name string,
	maxBytes int64,
) (string, func(), error) {
	reader, size, err := target.Get(ctx, name)
	if err != nil {
		return "", nil, errors.Join(ErrBackupArchiveDownloadFailed, err)
	}
	defer reader.Close()

	if size > maxBytes {
		return "", nil, errBackupArchiveTooLarge(maxBytes)
	}

	dir, err := newPrivateBackupTempDir()
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	archivePath := filepath.Join(dir, "subdux-destination-restore.zip")
	file, err := os.OpenFile(archivePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G304 -- archivePath is a fixed name inside a freshly created private temp dir.
	if err != nil {
		cleanup()
		return "", nil, err
	}

	// The authoritative cap: a destination that under-reports its object size
	// must not be able to fill the disk. Read one byte past the limit so an
	// exact-limit archive still restores.
	written, err := io.Copy(file, io.LimitReader(reader, maxBytes+1))
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		cleanup()
		return "", nil, errors.Join(ErrBackupArchiveDownloadFailed, err)
	}
	if written > maxBytes {
		cleanup()
		return "", nil, errBackupArchiveTooLarge(maxBytes)
	}

	return archivePath, cleanup, nil
}

// errBackupArchiveTooLarge reuses the upload path's public contract code so the
// admin sees the same sentence for the same condition on both restore routes.
// It reports the cap it was actually asked to enforce, so tests that exercise
// a small cap assert a truthful max_mb rather than a hardcoded 32.
func errBackupArchiveTooLarge(maxBytes int64) error {
	return serviceerr.NewCode(
		serviceerr.KindInvalid,
		"backup_file_is_too_large_max_mb",
		"backup archive exceeds the restore size limit",
		map[string]any{"max_mb": maxBytes >> 20},
	)
}
