package backup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"github.com/yeka/zip"
	"gorm.io/gorm"
)

const (
	StatusPending      = "pending"
	StatusOK           = "success"
	StatusPartial      = "partial"
	StatusFailed       = "failed"
	StatusSuperseded   = "superseded"
	StatusNotAttempted = "not_attempted"

	backupTaskKey      = "scheduled_backup"
	backupLeaseTTL     = 30 * time.Minute
	minBackupRetention = 1
	maxBackupRetention = 1000

	// backupRunRetryInterval is the minimum gap between resume attempts for an
	// incomplete scheduled run. The scheduler ticks every minute and the task
	// lease is re-acquirable by its current owner, so without this spacing a
	// destination that keeps failing would be retried ~60 times an hour, and a
	// run that failed before staging would rebuild the entire database snapshot
	// on every one of those attempts.
	backupRunRetryInterval = 15 * time.Minute

	// backupRunResumeWindow bounds how long one incomplete run may keep owning
	// the schedule. Past it the staged archive is a stale snapshot, so retrying
	// the same bytes is worth less than taking a fresh backup: the run is
	// superseded and the next due tick starts a new one.
	backupRunResumeWindow = 24 * time.Hour

	defaultRetentionCount  = 7
	defaultBackupTimeOfDay = "03:00"
	backupTempDirPattern   = "subdux-backup-*"
	backupStagingDirName   = "backup-staging"
)

var (
	ErrInvalidBackupTimeOfDay           = serviceerr.New(serviceerr.KindInvalid, "backup_time_of_day_must_be_in_hh_mm_24_hour_format", "backup time of day must be in HH:MM 24-hour format")
	ErrInvalidBackupRetentionCount      = serviceerr.New(serviceerr.KindInvalid, "backup_retention_count_must_be_between_1_and_1000", "backup retention count must be between 1 and 1000")
	ErrInvalidBackupLocalDir            = serviceerr.New(serviceerr.KindInvalid, "backup_local_directory_must_be_an_absolute_path_or_a_clean_relative_path_without_segments", "backup local directory must be an absolute path or a clean relative path without '..' segments")
	ErrBackupEncryptionPasswordRequired = serviceerr.New(serviceerr.KindInvalid, "encryption_password_is_required_when_backup_encryption_is_enabled", "encryption password is required when backup encryption is enabled")
	ErrNoEnabledBackupDestination       = serviceerr.New(serviceerr.KindInvalid, "no_enabled_backup_destination_is_configured", "no enabled backup destination is configured")
	ErrAllBackupDestinationsFailed      = serviceerr.New(serviceerr.KindInternal, "all_backup_destinations_failed", "all backup destinations failed")
	ErrBackupArchiveUnavailable         = serviceerr.New(serviceerr.KindInternal, "backup_archive_is_unavailable_for_retry", "the persisted backup archive is unavailable for retry")
)

var backupTimeOfDayPattern = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

// backupFileNamePattern matches the timestamped local backup filenames produced
// by CreateLocalBackup. Retention and listing operate only on files matching
// this pattern so unrelated files in the directory are never touched.
var backupFileNamePattern = regexp.MustCompile(`^subdux-backup-.*\.zip$`)

type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// WithContext returns a shallow copy of the service whose DB handle is bound to
// ctx, mirroring the other domain services so callers (e.g. the admin handler)
// can scope a backup operation to the request context without reaching into the
// underlying *gorm.DB.
func (s *Service) WithContext(ctx context.Context) *Service {
	if s.DB == nil {
		return s
	}
	clone := *s
	clone.DB = s.DB.WithContext(ctx)
	return &clone
}

// writeBackupZipFromDB writes a backup archive at archivePath containing the
// SQLite database at dbPath (stored as "subdux.db") and, when includeAssets is
// set, the assets tree. When encryptPassword is non-empty every entry is
// encrypted with WinZip AES-256; otherwise entries are stored as plain deflate
// entries with byte-identical internal structure. This single routine backs
// both the download path (plain) and the scheduled local-backup path.
func writeBackupZipFromDB(archivePath string, dbPath string, includeAssets bool, encryptPassword string) error {
	file, err := os.OpenFile(archivePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- archivePath is generated under a server-controlled backup directory.
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return err
	}

	zipWriter := zip.NewWriter(file)

	if err := addFileToBackupZip(zipWriter, dbPath, "subdux.db", encryptPassword); err != nil {
		_ = zipWriter.Close()
		return err
	}

	if includeAssets {
		if err := addAssetsToBackupZip(zipWriter, encryptPassword); err != nil {
			_ = zipWriter.Close()
			return err
		}
	}

	return zipWriter.Close()
}

// backupZipEntry returns the writer for a new archive entry, encrypting it when
// a password is supplied and storing it plainly otherwise. Both branches go
// through the yeka writer so plain and encrypted archives share one code path.
func backupZipEntry(zipWriter *zip.Writer, archivePath string, encryptPassword string) (io.Writer, error) {
	if encryptPassword != "" {
		return zipWriter.Encrypt(archivePath, encryptPassword, zip.AES256Encryption)
	}
	return zipWriter.Create(archivePath)
}

func addFileToBackupZip(zipWriter *zip.Writer, sourcePath string, archivePath string, encryptPassword string) error {
	sourceFile, err := os.Open(sourcePath) // #nosec G304 -- sourcePath is generated from the DB backup temp file or a walked assets directory.
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	targetFile, err := backupZipEntry(zipWriter, archivePath, encryptPassword)
	if err != nil {
		return err
	}

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return err
	}

	return nil
}

func addAssetsToBackupZip(zipWriter *zip.Writer, encryptPassword string) error {
	assetsRoot := filepath.Join(pkg.GetDataPath(), "assets")
	if err := addDirectoryToBackupZip(zipWriter, "assets/"); err != nil {
		return err
	}

	info, err := os.Stat(assetsRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	return filepath.Walk(assetsRoot, func(path string, fileInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fileInfo.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(assetsRoot, path)
		if err != nil {
			return err
		}

		archivePath := filepath.ToSlash(filepath.Join("assets", relativePath))
		return addFileToBackupZip(zipWriter, path, archivePath, encryptPassword)
	})
}

func addDirectoryToBackupZip(zipWriter *zip.Writer, archivePath string) error {
	header := &zip.FileHeader{
		Name: archivePath,
	}
	header.SetMode(os.ModeDir | 0o755)

	_, err := zipWriter.CreateHeader(header)
	return err
}

// resolveBackupDir returns the absolute target directory for local backups,
// defaulting to <DATA_PATH>/backups when no directory is configured.
func resolveBackupDir(localDir string) (string, error) {
	dir := strings.TrimSpace(localDir)
	if dir == "" {
		dir = filepath.Join(pkg.GetDataPath(), "backups")
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return absDir, nil
}

// newBackupToken returns a short random hex token appended to backup file names
// so concurrent runs (manual + scheduled in the same second-precision clock
// second) never resolve to the same temp DB or archive path.
func newBackupToken() (string, error) {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

// newPrivateBackupTempDir creates the short-lived directory used for backup
// snapshots and archives. MkdirTemp already requests 0700, and the explicit
// Chmod keeps that permission an invariant even when the platform or runtime
// applies different defaults.
func newPrivateBackupTempDir() (string, error) {
	dir, err := os.MkdirTemp("", backupTempDirPattern)
	if err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func backupStagingDir() string {
	return filepath.Join(pkg.GetDataPath(), backupStagingDirName)
}

func isPrivateStagedArchivePath(path string) bool {
	cleanPath := filepath.Clean(path)
	cleanDir := filepath.Clean(backupStagingDir())
	if filepath.Dir(cleanPath) != cleanDir {
		return false
	}
	parts := strings.SplitN(filepath.Base(cleanPath), "-", 2)
	if len(parts) != 2 {
		return false
	}
	if _, err := strconv.ParseUint(parts[0], 10, 64); err != nil {
		return false
	}
	return backupFileNamePattern.MatchString(parts[1])
}

// stageBackupArchive moves a scheduled archive into a private, persistent
// spool before external delivery. A scheduled retry must use these exact bytes;
// rebuilding the database snapshot under the same filename would make one run
// inconsistent across destinations.
func (s *Service) stageBackupArchive(runID uint, archiveName, archivePath string) (string, error) {
	if strings.ContainsAny(archiveName, `/\\`) || filepath.Base(archiveName) != archiveName || !backupFileNamePattern.MatchString(archiveName) {
		return "", ErrBackupArchiveUnavailable
	}
	dir := backupStagingDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	stagedPath := filepath.Join(dir, fmt.Sprintf("%d-%s", runID, archiveName))
	if err := os.Rename(archivePath, stagedPath); err != nil {
		if copyErr := copyBackupArchive(archivePath, stagedPath); copyErr != nil {
			return "", copyErr
		}
		if removeErr := os.Remove(archivePath); removeErr != nil {
			_ = os.Remove(stagedPath)
			return "", removeErr
		}
	}
	if err := os.Chmod(stagedPath, 0o600); err != nil {
		_ = os.Remove(stagedPath)
		return "", err
	}
	result := s.DB.Model(&model.BackupRun{}).Where("id = ?", runID).Updates(map[string]any{"archive_path": stagedPath})
	if result.Error != nil {
		_ = os.Remove(stagedPath)
		return "", result.Error
	}
	if result.RowsAffected != 1 {
		_ = os.Remove(stagedPath)
		return "", ErrBackupRunNotFound
	}
	return stagedPath, nil
}

func copyBackupArchive(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath) // #nosec G304 -- both paths are generated by the backup service.
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- targetPath is the private backup spool.
	if err != nil {
		return err
	}
	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		_ = os.Remove(targetPath)
		return err
	}
	if err := target.Chmod(0o600); err != nil {
		_ = target.Close()
		_ = os.Remove(targetPath)
		return err
	}
	return target.Close()
}

func (s *Service) cleanupBackupRunArchive(runID uint) error {
	var run model.BackupRun
	if err := s.DB.First(&run, runID).Error; err != nil {
		return err
	}
	archivePath := strings.TrimSpace(run.ArchivePath)
	if archivePath == "" {
		return nil
	}
	if err := os.Remove(archivePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		if bookkeepingErr := s.markGlobalBookkeepingFailure(runID, err); bookkeepingErr != nil {
			return errors.Join(err, bookkeepingErr)
		}
		return err
	}
	result := s.DB.Model(&model.BackupRun{}).Where("id = ?", runID).Update("archive_path", "")
	if result.Error != nil {
		if bookkeepingErr := s.markGlobalBookkeepingFailure(runID, result.Error); bookkeepingErr != nil {
			return errors.Join(result.Error, bookkeepingErr)
		}
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrBackupRunNotFound
	}
	return nil
}

// BackupDB produces an on-demand backup and returns the path of the file to
// serve. When password is empty the historical behavior is preserved: a raw
// SQLite .db file when includeAssets is false, or a plain .zip when true.
// When password is non-empty encryption requires a zip container, so the DB is
// always bundled into a WinZip AES-256 .zip (with assets honored) regardless of
// includeAssets. The password is trimmed before deciding, so all-whitespace is
// treated as empty.
//
// Every download is recorded as a backup run (source "download") so the admin
// history lists it next to destination runs. The run is created before the
// snapshot and finalized with the outcome, so a failed build is visible too.
// The name carries a random token like destination archives, which keeps the
// archive_name unique index safe for concurrent downloads.
func (s *Service) BackupDB(includeAssets bool, password string) (string, error) {
	password = strings.TrimSpace(password)
	encrypted := password != ""
	zipArchive := includeAssets || encrypted

	token, err := newBackupToken()
	if err != nil {
		return "", err
	}
	extension := ".db"
	if zipArchive {
		extension = ".zip"
	}
	archiveName := fmt.Sprintf("subdux-backup-%s-%s%s", pkg.Now().Format("20060102-150405"), token, extension)

	state, err := s.beginBackupRun(backupRunSourceDownload, archiveName, nil)
	if err != nil {
		return "", err
	}

	archivePath, cleanup, err := s.buildDownloadArchive(archiveName, includeAssets, password)
	if err != nil {
		result := downloadRunResult(state.run, encrypted, 0, err)
		if finalizeErr := s.finalizeBackupRun(state.run.ID, result, false); finalizeErr != nil {
			return "", errors.Join(err, finalizeErr)
		}
		return "", err
	}

	var sizeBytes int64
	if info, statErr := os.Stat(archivePath); statErr == nil {
		sizeBytes = info.Size()
	}
	result := downloadRunResult(state.run, encrypted, sizeBytes, nil)
	if finalizeErr := s.finalizeBackupRun(state.run.ID, result, false); finalizeErr != nil {
		// The archive was built, but without its record the download would be an
		// unrecorded backup. Treat the failed status write like a failed build:
		// discard the archive and report the error.
		cleanup()
		return "", finalizeErr
	}
	return archivePath, nil
}

// buildDownloadArchive writes the download archive named archiveName into a
// private temp directory. The caller owns the directory on success (the HTTP
// handler removes it after serving the file); cleanup removes it on the error
// paths where the caller never learns the path.
func (s *Service) buildDownloadArchive(archiveName string, includeAssets bool, password string) (string, func(), error) {
	tempDir, err := newPrivateBackupTempDir()
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	if !includeAssets && password == "" {
		backupPath := filepath.Join(tempDir, archiveName)
		if err := s.DB.Exec("VACUUM INTO ?", backupPath).Error; err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := os.Chmod(backupPath, 0o600); err != nil {
			cleanup()
			return "", func() {}, err
		}
		return backupPath, cleanup, nil
	}

	dbTempPath := filepath.Join(tempDir, strings.TrimSuffix(archiveName, ".zip")+".db")
	if err := s.DB.Exec("VACUUM INTO ?", dbTempPath).Error; err != nil {
		cleanup()
		return "", func() {}, err
	}
	if err := os.Chmod(dbTempPath, 0o600); err != nil {
		cleanup()
		return "", func() {}, err
	}

	archivePath := filepath.Join(tempDir, archiveName)
	if err := writeBackupZipFromDB(archivePath, dbTempPath, includeAssets, password); err != nil {
		cleanup()
		return "", func() {}, err
	}
	_ = os.Remove(dbTempPath)

	return archivePath, cleanup, nil
}

// downloadRunResult shapes the terminal record of a download-sourced run.
// Delivery means "the archive was produced for the admin's browser"; retention
// never applies because nothing is stored server-side, and there is no
// per-destination bookkeeping to fail.
func downloadRunResult(run model.BackupRun, encrypted bool, sizeBytes int64, buildErr error) BackupRunResult {
	result := BackupRunResult{
		RunID:                   run.ID,
		ArchiveName:             run.ArchiveName,
		Status:                  StatusOK,
		DeliveryStatus:          StatusOK,
		RetentionStatus:         StatusNotAttempted,
		BookkeepingStatus:       StatusOK,
		GlobalBookkeepingStatus: StatusOK,
		SizeBytes:               sizeBytes,
		Encrypted:               encrypted,
	}
	if buildErr != nil {
		result.Status = StatusFailed
		result.DeliveryStatus = StatusFailed
		result.Error = buildErr.Error()
	}
	return result
}

// DestinationRunResult captures the per-destination outcome of a fan-out run.
// Delivery, retention, and bookkeeping are independent stages: a successful
// delivery remains successful even when later cleanup or status persistence
// fails.
type DestinationRunResult struct {
	DestinationID     uint   `json:"destination_id"`
	Type              string `json:"type"`
	DeliveryStatus    string `json:"delivery_status"`
	Success           bool   `json:"success"` // archive delivery succeeded
	Error             string `json:"error,omitempty"`
	RetentionStatus   string `json:"retention_status"`
	RetentionError    string `json:"retention_error,omitempty"`
	BookkeepingStatus string `json:"bookkeeping_status"`
	BookkeepingError  string `json:"bookkeeping_error,omitempty"`
}

// BackupRunResult summarizes a single scheduled/manual run: the archive name
// shared across all destinations and one entry per enabled destination.
// SizeBytes and Encrypted describe the archive itself and are persisted on the
// run row so the admin history can show them without re-reading archives.
type BackupRunResult struct {
	RunID                   uint                   `json:"run_id"`
	ArchiveName             string                 `json:"archive_name"`
	Status                  string                 `json:"status"`
	DeliveryStatus          string                 `json:"delivery_status"`
	RetentionStatus         string                 `json:"retention_status"`
	BookkeepingStatus       string                 `json:"bookkeeping_status"`
	GlobalBookkeepingStatus string                 `json:"global_bookkeeping_status"`
	SizeBytes               int64                  `json:"size_bytes"`
	Encrypted               bool                   `json:"encrypted"`
	Error                   string                 `json:"error,omitempty"`
	Results                 []DestinationRunResult `json:"results"`
}

// RunDestinationBackup runs one destination's plan immediately, using that
// destination's own archive settings. It is the manual counterpart to a
// scheduled firing and deliberately does not touch last_scheduled_run_at, so an
// ad-hoc run never consumes the destination's slot for the day.
func (s *Service) RunDestinationBackup(ctx context.Context, id uint) (BackupRunResult, error) {
	var destination model.BackupDestination
	if err := s.DB.First(&destination, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return BackupRunResult{}, ErrBackupDestinationNotFound
		}
		return BackupRunResult{}, err
	}

	schedule, err := destinationScheduleFromStored(destination.Config)
	if err != nil {
		return BackupRunResult{}, err
	}

	return s.runBackup(
		ctx,
		backupRunSourceManual,
		[]model.BackupDestination{destination},
		schedule.archiveSpec(),
		nil,
	)
}

// runBackup starts a new run over destinations, or continues the run in resumed
// when the caller has already resolved one. A resume reuses the persisted
// archive name and destination stage rows, so completed delivery stages are
// never retried as a new archive merely because retention or bookkeeping was
// incomplete. The caller resolves the resumable run rather than this function so
// the scheduler can apply its retry spacing to the same run it then hands over
// here.
//
// Every destination in one call must share spec: the run produces a single
// archive, so its contents cannot vary per destination.
func (s *Service) runBackup(
	ctx context.Context,
	source string,
	destinations []model.BackupDestination,
	spec archiveSpec,
	resumed *backupRunState,
) (BackupRunResult, error) {
	var state backupRunState
	var err error
	if resumed != nil {
		state = *resumed
	}
	if state.run.ID == 0 {
		if len(destinations) == 0 {
			return BackupRunResult{}, ErrNoEnabledBackupDestination
		}
		archiveName, nameErr := newBackupArchiveName()
		if nameErr != nil {
			return BackupRunResult{}, nameErr
		}
		state, err = s.beginBackupRun(source, archiveName, destinations)
		if err != nil {
			return BackupRunResult{}, err
		}
	}

	var archivePath string
	cleanup := func() {}
	if source == backupRunSourceScheduled && strings.TrimSpace(state.run.ArchivePath) != "" {
		archivePath = state.run.ArchivePath
		if !isPrivateStagedArchivePath(archivePath) {
			err = ErrBackupArchiveUnavailable
		} else if _, statErr := os.Stat(archivePath); statErr != nil {
			err = fmt.Errorf("%w: %v", ErrBackupArchiveUnavailable, statErr)
		}
	} else {
		var builtPath string
		builtPath, cleanup, err = s.buildBackupArchiveNamed(spec, state.run.ArchiveName)
		archivePath = builtPath
		if err == nil && source == backupRunSourceScheduled {
			var stageErr error
			archivePath, stageErr = s.stageBackupArchive(state.run.ID, state.run.ArchiveName, builtPath)
			cleanup()
			cleanup = func() {}
			err = stageErr
		}
	}
	if err != nil {
		result := BackupRunResult{
			RunID:                   state.run.ID,
			ArchiveName:             state.run.ArchiveName,
			Status:                  StatusFailed,
			DeliveryStatus:          StatusFailed,
			RetentionStatus:         StatusNotAttempted,
			BookkeepingStatus:       StatusOK,
			GlobalBookkeepingStatus: StatusPending,
			Encrypted:               spec.Password != "",
			Error:                   err.Error(),
		}
		if finalizeErr := s.finalizeBackupRun(state.run.ID, result, source == backupRunSourceScheduled); finalizeErr != nil {
			return result, errors.Join(err, finalizeErr)
		}
		return result, err
	}
	if source == backupRunSourceManual {
		defer cleanup()
	}

	// The archive size is display metadata for the run history; an unreadable
	// size is recorded as 0 ("unknown") rather than failing a deliverable run.
	var archiveSize int64
	if info, statErr := os.Stat(archivePath); statErr == nil {
		archiveSize = info.Size()
	}

	// deliverToDestination resolves and revision-checks the destination row
	// itself, so the loop stays a plain fan-out: every outcome, including a
	// deleted or replaced destination, is recorded on the run-destination row by
	// that one code path.
	for i := range state.destinations {
		_, _ = s.deliverToDestination(ctx, archivePath, state.run.ArchiveName, &state.destinations[i])
	}

	result := aggregateBackupRunResult(state.run, state.destinations)
	result.SizeBytes = archiveSize
	result.Encrypted = spec.Password != ""
	if finalizeErr := s.finalizeBackupRun(state.run.ID, result, source == backupRunSourceScheduled); finalizeErr != nil {
		result.BookkeepingStatus = StatusFailed
		if result.Status == StatusOK {
			result.Status = StatusPartial
		}
		result.Error = joinErrors([]error{errors.New(result.Error), finalizeErr})
		if result.Status == StatusFailed {
			return result, errorForBackupRunResult(result)
		}
		return result, nil
	}
	if source == backupRunSourceManual {
		result.GlobalBookkeepingStatus = StatusOK
	}
	if result.Status == StatusFailed {
		return result, errorForBackupRunResult(result)
	}
	return result, nil
}

func newBackupArchiveName() (string, error) {
	token, err := newBackupToken()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("subdux-backup-%s-%s.zip", pkg.Now().Format("20060102-150405"), token), nil
}

func errorForBackupRunResult(result BackupRunResult) error {
	parts := []error{ErrAllBackupDestinationsFailed}
	if strings.TrimSpace(result.Error) != "" {
		parts = append(parts, errors.New(result.Error))
	}
	return errors.Join(parts...)
}

func joinErrors(errs []error) string {
	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil && strings.TrimSpace(err.Error()) != "" {
			messages = append(messages, err.Error())
		}
	}
	return strings.Join(messages, "; ")
}

// buildBackupArchiveNamed writes a backup archive under archiveName into a
// private temp directory and returns its path plus a cleanup closure that
// removes both the archive and the intermediate DB snapshot. The caller supplies
// the name so a resumed run rebuilds under the name already persisted on the
// run row, keeping delivery idempotent across retries.
func (s *Service) buildBackupArchiveNamed(spec archiveSpec, archiveName string) (string, func(), error) {
	tempDir, err := newPrivateBackupTempDir()
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	token, err := newBackupToken()
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	timestamp := pkg.Now().Format("20060102-150405")

	dbTempPath := filepath.Join(tempDir, fmt.Sprintf("subdux-backup-%s-%s.db", timestamp, token))
	if err := s.DB.Exec("VACUUM INTO ?", dbTempPath).Error; err != nil {
		cleanup()
		return "", func() {}, err
	}
	if err := os.Chmod(dbTempPath, 0o600); err != nil {
		cleanup()
		return "", func() {}, err
	}

	archivePath := filepath.Join(tempDir, archiveName)

	if err := writeBackupZipFromDB(archivePath, dbTempPath, spec.IncludeAssets, spec.Password); err != nil {
		cleanup()
		return "", func() {}, err
	}

	return archivePath, cleanup, nil
}

// deliverToDestination advances one destination through delivery, retention,
// and bookkeeping. The persisted run-destination row is updated after each
// stage, so a scheduled retry can resume at the first incomplete stage. The
// destination row is loaded and revision-checked here, which makes this the
// single place that decides what a stale, deleted, or unbuildable destination
// does to the run.
func (s *Service) deliverToDestination(ctx context.Context, archivePath, archiveName string, runDestination *model.BackupRunDestination) (DestinationRunResult, []error) {
	var bookkeepingErrors []error
	destinationID := runDestination.DestinationID
	if runDestination.RetentionStatus == "" {
		runDestination.RetentionStatus = StatusNotAttempted
	}
	if runDestination.BookkeepingStatus == "" {
		runDestination.BookkeepingStatus = StatusPending
	}

	persist := func(updates map[string]any) {
		if err := s.persistRunDestination(runDestination.ID, updates); err != nil {
			bookkeepingErrors = appendBackupBookkeepingError(bookkeepingErrors, destinationID, err)
		}
	}
	finish := func() (DestinationRunResult, []error) {
		runDestination.BookkeepingStatus = StatusOK
		runDestination.BookkeepingError = ""
		if len(bookkeepingErrors) > 0 {
			runDestination.BookkeepingStatus = StatusFailed
			runDestination.BookkeepingError = joinErrors(bookkeepingErrors)
		}
		before := len(bookkeepingErrors)
		persist(map[string]any{
			"bookkeeping_status": runDestination.BookkeepingStatus,
			"bookkeeping_error":  runDestination.BookkeepingError,
		})
		if len(bookkeepingErrors) > before {
			runDestination.BookkeepingStatus = StatusFailed
			runDestination.BookkeepingError = joinErrors(bookkeepingErrors)
		}
		result := destinationRunResult(*runDestination)
		result.BookkeepingStatus = runDestination.BookkeepingStatus
		result.BookkeepingError = runDestination.BookkeepingError
		return result, bookkeepingErrors
	}

	// loadBackupDestinationForRun is the revision gate for active runs. Keep its
	// error on the same failure path as target construction below.
	destination, err := s.loadBackupDestinationForRun(*runDestination)
	var target BackupTarget
	if err == nil {
		target, err = newBackupTarget(destination, s.DB)
	}
	if err != nil {
		runDestination.DeliveryStatus = StatusFailed
		runDestination.DeliveryError = err.Error()
		runDestination.RetentionStatus = StatusNotAttempted
		runDestination.RetentionError = ""
		persist(map[string]any{
			"delivery_status":  runDestination.DeliveryStatus,
			"delivery_error":   runDestination.DeliveryError,
			"retention_status": runDestination.RetentionStatus,
			"retention_error":  runDestination.RetentionError,
		})
		// Skip the destination summary write when there is no row this run may
		// write to. ErrBackupDestinationChanged means the row was replaced after
		// this run's snapshot, so the newer row must not inherit the old run's
		// failure; ErrBackupDestinationNotFound means the row is gone entirely.
		// Either way the run-destination row above already carries the outcome.
		if !errors.Is(err, ErrBackupDestinationChanged) && !errors.Is(err, ErrBackupDestinationNotFound) {
			if statusErr := s.recordDestinationOutcome(destinationID, runDestination.DeliveryStatus, runDestination.DeliveryError, runDestination.RetentionStatus, runDestination.RetentionError); statusErr != nil {
				bookkeepingErrors = appendBackupBookkeepingError(bookkeepingErrors, destinationID, statusErr)
			}
		}
		return finish()
	}

	if runDestination.DeliveryStatus != StatusOK {
		alreadyDelivered := false
		if runDestination.DeliveryAttempted {
			objects, listErr := target.List(ctx)
			if listErr != nil {
				runDestination.DeliveryStatus = StatusFailed
				runDestination.DeliveryError = fmt.Sprintf("delivery state could not be confirmed: %v", listErr)
				runDestination.RetentionStatus = StatusNotAttempted
				runDestination.RetentionError = ""
				persist(map[string]any{
					"delivery_status":  runDestination.DeliveryStatus,
					"delivery_error":   runDestination.DeliveryError,
					"retention_status": runDestination.RetentionStatus,
					"retention_error":  runDestination.RetentionError,
				})
				if statusErr := s.recordDestinationOutcome(destinationID, runDestination.DeliveryStatus, runDestination.DeliveryError, runDestination.RetentionStatus, runDestination.RetentionError); statusErr != nil {
					bookkeepingErrors = appendBackupBookkeepingError(bookkeepingErrors, destinationID, statusErr)
				}
				return finish()
			}
			for _, object := range objects {
				if object.Name == archiveName {
					alreadyDelivered = true
					break
				}
			}
		}

		if !runDestination.DeliveryAttempted {
			runDestination.DeliveryAttempted = true
			persist(map[string]any{"delivery_attempted": true})
			if len(bookkeepingErrors) > 0 {
				return finish()
			}
		}
		if !alreadyDelivered {
			if err := s.deliver(ctx, target, archivePath, archiveName); err != nil {
				runDestination.DeliveryStatus = StatusFailed
				runDestination.DeliveryError = err.Error()
				runDestination.RetentionStatus = StatusNotAttempted
				runDestination.RetentionError = ""
				persist(map[string]any{
					"delivery_status":  runDestination.DeliveryStatus,
					"delivery_error":   runDestination.DeliveryError,
					"retention_status": runDestination.RetentionStatus,
					"retention_error":  runDestination.RetentionError,
				})
				if statusErr := s.recordDestinationOutcome(destinationID, runDestination.DeliveryStatus, runDestination.DeliveryError, runDestination.RetentionStatus, runDestination.RetentionError); statusErr != nil {
					bookkeepingErrors = appendBackupBookkeepingError(bookkeepingErrors, destinationID, statusErr)
				}
				return finish()
			}
		}

		now := pkg.Now()
		runDestination.DeliveryStatus = StatusOK
		runDestination.DeliveryError = ""
		runDestination.DeliveredAt = &now
		persist(map[string]any{
			"delivery_status": runDestination.DeliveryStatus,
			"delivery_error":  "",
			"delivered_at":    now,
		})
	}

	if runDestination.RetentionStatus != StatusOK {
		if err := applyTargetRetention(ctx, target); err != nil {
			runDestination.RetentionStatus = StatusFailed
			runDestination.RetentionError = err.Error()
		} else {
			runDestination.RetentionStatus = StatusOK
			runDestination.RetentionError = ""
		}
		now := pkg.Now()
		runDestination.RetentionFinishedAt = &now
		persist(map[string]any{
			"retention_status":      runDestination.RetentionStatus,
			"retention_error":       runDestination.RetentionError,
			"retention_finished_at": now,
		})
	}

	if statusErr := s.recordDestinationOutcome(destination.ID, runDestination.DeliveryStatus, runDestination.DeliveryError, runDestination.RetentionStatus, runDestination.RetentionError); statusErr != nil {
		bookkeepingErrors = appendBackupBookkeepingError(bookkeepingErrors, destination.ID, statusErr)
	}
	return finish()
}

// deliver uploads the archive to one already-constructed target. Retention is
// deliberately separate because a successful upload must not be retried merely
// because listing or deleting stale objects failed afterward. The target is
// passed in rather than rebuilt so one delivery attempt uses exactly one client
// (which for S3 means one connection pool, not two).
func (s *Service) deliver(ctx context.Context, target BackupTarget, archivePath, archiveName string) error {
	archive, err := os.Open(archivePath) // #nosec G304 -- archivePath is a server-generated temp file created by buildBackupArchiveNamed, or the private staging spool.
	if err != nil {
		return err
	}
	defer archive.Close()

	info, err := archive.Stat()
	if err != nil {
		return err
	}

	return target.Put(ctx, archiveName, archive, info.Size())
}

// applyTargetRetention lists the archives at a target and deletes all but the
// newest RetentionCount by modification time. Listing errors abort retention
// (the upload already succeeded); individual delete failures are surfaced so a
// destination that cannot prune is not silently reported as fully healthy.
func applyTargetRetention(ctx context.Context, target BackupTarget) error {
	keep := target.RetentionCount()
	if keep < minBackupRetention {
		keep = minBackupRetention
	}

	objects, err := target.List(ctx)
	if err != nil {
		return err
	}
	if len(objects) <= keep {
		return nil
	}
	for _, object := range objects {
		if object.ModifiedAt.IsZero() {
			// An unknown timestamp cannot safely participate in retention
			// ordering: it might be newer than every known object. Fail closed
			// before deleting anything rather than treating it as the oldest.
			return fmt.Errorf("backup object %q has no modification time", object.Name)
		}
	}

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].ModifiedAt.After(objects[j].ModifiedAt)
	})

	for _, stale := range objects[keep:] {
		if err := target.Delete(ctx, stale.Name); err != nil {
			return err
		}
	}
	return nil
}

// ListDestinationBackups returns the archives held at one destination, newest
// first, by delegating to the destination's target. It is the generalized,
// destination-aware replacement for directory-only listing.
func (s *Service) ListDestinationBackups(ctx context.Context, id uint) ([]BackupObject, error) {
	var destination model.BackupDestination
	if err := s.DB.First(&destination, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBackupDestinationNotFound
		}
		return nil, err
	}

	target, err := newBackupTarget(destination, s.DB)
	if err != nil {
		return nil, err
	}

	objects, err := target.List(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].ModifiedAt.After(objects[j].ModifiedAt)
	})
	return objects, nil
}

// backupArchiveIsEncrypted reports whether the archive's first regular entry is
// AES-encrypted. Detection failures default to false rather than failing the
// listing.
func backupArchiveIsEncrypted(archivePath string) bool {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return false
	}
	defer reader.Close()

	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		return entry.IsEncrypted()
	}
	return false
}

// RunScheduledBackup runs a lease-guarded scheduled backup pass. Each enabled
// destination is its own plan, so one pass may start several runs: incomplete
// runs are resumed first, then every destination that has come due and is not
// already owned by a resumed run is grouped onto a shared archive.
//
// Only a destination whose delivery succeeds advances its last_scheduled_run_at,
// so a partial run leaves the unsatisfied destinations due. Their persisted
// stage rows make the retry target only the unfinished delivery/retention work.
func (s *Service) RunScheduledBackup(ownerID string) error {
	return serviceutil.WithBackgroundTaskLease(s.DB, ownerID, backupTaskKey, backupLeaseTTL, func() error {
		resumables, err := s.findResumableScheduledRuns()
		if err != nil {
			return err
		}

		var errs []error

		// A destination inside an incomplete run is spoken for: starting a second
		// run for it would deliver two archives for the same due window and race
		// the first run's retention. Resuming that run is the only way it makes
		// progress.
		claimed := make(map[uint]struct{})
		for _, resumable := range resumables {
			for _, runDestination := range resumable.destinations {
				claimed[runDestination.DestinationID] = struct{}{}
			}
		}

		for i := range resumables {
			resumable := &resumables[i]
			// Space the retries: a failing run must not consume a build and
			// delivery cycle on every scheduler tick.
			if !scheduledRunResumeDue(resumable.run, pkg.Now()) {
				continue
			}
			spec, specErr := s.archiveSpecForResumedRun(*resumable)
			if specErr != nil {
				errs = append(errs, specErr)
				continue
			}
			errs = append(errs, s.runScheduledGroup(nil, spec, resumable))
		}

		due, dueErr := s.dueScheduledDestinations(claimed)
		errs = append(errs, dueErr)
		for _, group := range due {
			errs = append(errs, s.runScheduledGroup(group.destinations, group.spec, nil))
		}

		return errors.Join(errs...)
	})
}

// scheduledGroup is a set of due destinations that agree on archive contents and
// therefore share one archive for this pass.
type scheduledGroup struct {
	spec         archiveSpec
	destinations []model.BackupDestination
}

// dueScheduledDestinations returns the destinations whose plans have come due,
// grouped onto shared archives. Destinations in claimed are skipped because an
// incomplete run already owns them.
//
// A destination whose stored config cannot be read or whose schedule is invalid
// is reported rather than skipped silently: it is configured to be backed up and
// is not being backed up, which the operator must see.
func (s *Service) dueScheduledDestinations(claimed map[uint]struct{}) ([]scheduledGroup, error) {
	destinations, err := s.listEnabledDestinations()
	if err != nil {
		return nil, err
	}

	loc := pkg.GetSystemTimezone()
	now := pkg.NowInSystemTimezone()

	var errs []error
	var order []archiveSpec
	grouped := make(map[archiveSpec][]model.BackupDestination)
	for _, destination := range destinations {
		if _, busy := claimed[destination.ID]; busy {
			continue
		}
		schedule, scheduleErr := destinationScheduleFromStored(destination.Config)
		if scheduleErr != nil {
			errs = append(errs, fmt.Errorf("backup destination %d schedule: %w", destination.ID, scheduleErr))
			continue
		}
		if !backupDue(now, schedule.TimeOfDay, lastScheduledRunAt(destination), loc) {
			continue
		}
		spec := schedule.archiveSpec()
		if _, seen := grouped[spec]; !seen {
			order = append(order, spec)
		}
		grouped[spec] = append(grouped[spec], destination)
	}

	groups := make([]scheduledGroup, 0, len(order))
	for _, spec := range order {
		groups = append(groups, scheduledGroup{spec: spec, destinations: grouped[spec]})
	}
	return groups, errors.Join(errs...)
}

// runScheduledGroup performs one scheduled run and its bookkeeping: recording
// which destinations satisfied their schedule, then releasing the staged archive
// once nothing is left to retry.
func (s *Service) runScheduledGroup(destinations []model.BackupDestination, spec archiveSpec, resumed *backupRunState) error {
	result, backupErr := s.runBackup(context.Background(), backupRunSourceScheduled, destinations, spec, resumed)
	if result.RunID == 0 {
		return backupErr
	}

	if statusErr := s.recordScheduledRunOutcome(result); statusErr != nil {
		_ = s.markGlobalBookkeepingFailure(result.RunID, statusErr)
		return errors.Join(backupErr, statusErr)
	}
	if bookkeepingErr := s.markGlobalBookkeepingSuccess(result.RunID); bookkeepingErr != nil {
		return errors.Join(backupErr, bookkeepingErr)
	}
	if result.Status == StatusOK {
		if cleanupErr := s.cleanupBackupRunArchive(result.RunID); cleanupErr != nil {
			return errors.Join(backupErr, cleanupErr)
		}
	}
	return backupErr
}

// recordScheduledRunOutcome marks each destination that actually received the
// archive as having satisfied its schedule. A destination whose delivery failed
// is deliberately left alone so it stays due and is retried.
func (s *Service) recordScheduledRunOutcome(result BackupRunResult) error {
	now := pkg.Now()
	var errs []error
	for _, item := range result.Results {
		if !item.Success {
			continue
		}
		dbResult := s.DB.Model(&model.BackupDestination{}).
			Where("id = ?", item.DestinationID).
			Update("last_scheduled_run_at", now)
		if dbResult.Error != nil {
			errs = append(errs, dbResult.Error)
		}
	}
	return errors.Join(errs...)
}

// archiveSpecForResumedRun re-derives the archive settings of an in-flight run
// from its destinations' current configs. findResumableScheduledRuns has already
// superseded any run whose destination revisions moved, so the stored config
// still describes the archive this run was started to produce — which is why the
// spec is not persisted on the run row alongside the (secret) password.
func (s *Service) archiveSpecForResumedRun(state backupRunState) (archiveSpec, error) {
	for _, runDestination := range state.destinations {
		var destination model.BackupDestination
		if err := s.DB.First(&destination, runDestination.DestinationID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return archiveSpec{}, err
		}
		schedule, err := destinationScheduleFromStored(destination.Config)
		if err != nil {
			return archiveSpec{}, err
		}
		return schedule.archiveSpec(), nil
	}
	return archiveSpec{}, ErrNoEnabledBackupDestination
}

// lastScheduledRunAt is the due-check anchor for one destination. A destination
// that has never completed a scheduled run yields the zero time, which
// backupDue reads as "due as soon as today's firing time passes".
func lastScheduledRunAt(destination model.BackupDestination) time.Time {
	if destination.LastScheduledRunAt == nil {
		return time.Time{}
	}
	return *destination.LastScheduledRunAt
}

// backupDue reports whether a scheduled backup should run at now given the
// configured HH:MM time of day and the last run timestamp, all evaluated in
// loc. A backup is due when the current time is at or after today's scheduled
// moment and no successful run has been recorded for today yet.
func backupDue(now time.Time, timeOfDay string, lastRunAt time.Time, loc *time.Location) bool {
	if loc == nil {
		loc = time.Local
	}

	match := backupTimeOfDayPattern.FindStringSubmatch(strings.TrimSpace(timeOfDay))
	if match == nil {
		return false
	}
	hour, _ := strconv.Atoi(match[1])
	minute, _ := strconv.Atoi(match[2])

	nowLocal := now.In(loc)
	scheduledToday := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), hour, minute, 0, 0, loc)
	if nowLocal.Before(scheduledToday) {
		return false
	}

	if lastRunAt.IsZero() {
		return true
	}

	lastRunLocal := pkg.NormalizeDateInTimezone(lastRunAt, loc)
	today := pkg.NormalizeDateInTimezone(nowLocal, loc)
	return lastRunLocal.Before(today)
}

// normalizeBackupLocalDir validates and normalizes a configured backup
// directory. Empty means "use the default"; non-empty must be an absolute path
// or a clean relative path with no ".." segments.
func normalizeBackupLocalDir(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed), nil
	}

	cleaned := filepath.Clean(trimmed)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", ErrInvalidBackupLocalDir
	}
	for _, segment := range strings.Split(filepath.ToSlash(cleaned), "/") {
		if segment == ".." {
			return "", ErrInvalidBackupLocalDir
		}
	}
	return cleaned, nil
}
