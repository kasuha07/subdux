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

	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	backupsettings "github.com/kasuha07/subdux/internal/service/settings"
	"github.com/yeka/zip"
	"gorm.io/gorm"
)

const (
	KeyScheduleEnabled    = "backup_schedule_enabled"
	KeyTimeOfDay          = "backup_time_of_day"
	KeyIncludeAssets      = "backup_include_assets"
	KeyEncryptEnabled     = "backup_encrypt_enabled"
	KeyEncryptionPassword = "backup_encryption_password"
	KeyLocalDir           = "backup_local_dir"
	KeyRetentionCount     = "backup_retention_count"
	KeyLastRunAt          = "backup_last_run_at"
	KeyLastStatus         = "backup_last_status"
	KeyLastError          = "backup_last_error"
)

const (
	StatusOK     = "success"
	StatusFailed = "failed"

	backupTaskKey      = "scheduled_backup"
	backupLeaseTTL     = 30 * time.Minute
	minBackupRetention = 1
	maxBackupRetention = 1000
)

var (
	ErrInvalidBackupTimeOfDay           = errors.New("backup time of day must be in HH:MM 24-hour format")
	ErrInvalidBackupRetentionCount      = errors.New("backup retention count must be between 1 and 1000")
	ErrInvalidBackupLocalDir            = errors.New("backup local directory must be an absolute path or a clean relative path without '..' segments")
	ErrBackupEncryptionPasswordRequired = errors.New("encryption password is required when backup encryption is enabled")
)

var backupTimeOfDayPattern = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

// backupFileNamePattern matches the timestamped local backup filenames produced
// by CreateLocalBackup. Retention and listing operate only on files matching
// this pattern so unrelated files in the directory are never touched.
var backupFileNamePattern = regexp.MustCompile(`^subdux-backup-.*\.zip$`)

// LocalBackupInfo describes a single local backup file for the listing endpoint.
type LocalBackupInfo struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
	Encrypted  bool   `json:"encrypted"`
}

type UpdateSettingsInput struct {
	ScheduleEnabled    *bool
	TimeOfDay          *string
	IncludeAssets      *bool
	EncryptEnabled     *bool
	EncryptionPassword *string
	LocalDir           *string
	RetentionCount     *int64
}

type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// writeBackupZipFromDB writes a backup archive at archivePath containing the
// SQLite database at dbPath (stored as "subdux.db") and, when includeAssets is
// set, the assets tree. When encryptPassword is non-empty every entry is
// encrypted with WinZip AES-256; otherwise entries are stored as plain deflate
// entries with byte-identical internal structure. This single routine backs
// both the download path (plain) and the scheduled local-backup path.
func writeBackupZipFromDB(archivePath string, dbPath string, includeAssets bool, encryptPassword string) error {
	file, err := os.Create(archivePath) // #nosec G304 -- archivePath is generated under a server-controlled backup directory.
	if err != nil {
		return err
	}
	defer file.Close()

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

type runtimeConfig struct {
	ScheduleEnabled bool
	TimeOfDay       string
	IncludeAssets   bool
	EncryptEnabled  bool
	EncryptPassword string
	LocalDir        string
	RetentionCount  int64
}

func (s *Service) loadRuntimeConfig() (runtimeConfig, error) {
	cfg := runtimeConfig{
		TimeOfDay:      "03:00",
		RetentionCount: 7,
	}

	scheduleEnabled, err := backupsettings.GetBool(context.Background(), s.DB, KeyScheduleEnabled, false)
	if err != nil {
		return cfg, err
	}
	cfg.ScheduleEnabled = scheduleEnabled

	timeOfDay, err := backupsettings.GetString(context.Background(), s.DB, KeyTimeOfDay, "03:00")
	if err != nil {
		return cfg, err
	}
	if strings.TrimSpace(timeOfDay) != "" {
		cfg.TimeOfDay = timeOfDay
	}

	includeAssets, err := backupsettings.GetBool(context.Background(), s.DB, KeyIncludeAssets, false)
	if err != nil {
		return cfg, err
	}
	cfg.IncludeAssets = includeAssets

	encryptEnabled, err := backupsettings.GetBool(context.Background(), s.DB, KeyEncryptEnabled, false)
	if err != nil {
		return cfg, err
	}
	cfg.EncryptEnabled = encryptEnabled

	localDir, err := backupsettings.GetString(context.Background(), s.DB, KeyLocalDir, "")
	if err != nil {
		return cfg, err
	}
	cfg.LocalDir = strings.TrimSpace(localDir)

	retentionCount, err := backupsettings.GetInt(context.Background(), s.DB, KeyRetentionCount, 7)
	if err != nil {
		return cfg, err
	}
	if retentionCount >= minBackupRetention {
		cfg.RetentionCount = int64(retentionCount)
	}

	if encryptEnabled {
		storedPassword, err := backupsettings.GetString(context.Background(), s.DB, KeyEncryptionPassword, "")
		if err != nil {
			return cfg, err
		}
		cfg.EncryptPassword = storedPassword
	}

	return cfg, nil
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

// BackupDB produces an on-demand backup and returns the path of the file to
// serve. When password is empty the historical behavior is preserved: a raw
// SQLite .db file when includeAssets is false, or a plain .zip when true.
// When password is non-empty encryption requires a zip container, so the DB is
// always bundled into a WinZip AES-256 .zip (with assets honored) regardless of
// includeAssets. The password is trimmed before deciding, so all-whitespace is
// treated as empty.
func (s *Service) BackupDB(includeAssets bool, password string) (string, error) {
	password = strings.TrimSpace(password)

	timestamp := pkg.Now().Format("20060102-150405")
	backupPath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-backup-%s.db", timestamp))

	if err := s.DB.Exec("VACUUM INTO ?", backupPath).Error; err != nil {
		return "", err
	}

	if !includeAssets && password == "" {
		return backupPath, nil
	}

	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-backup-%s.zip", timestamp))
	if err := writeBackupZipFromDB(archivePath, backupPath, includeAssets, password); err != nil {
		_ = os.Remove(backupPath)
		_ = os.Remove(archivePath)
		return "", err
	}

	_ = os.Remove(backupPath)

	return archivePath, nil
}

// CreateLocalBackup builds a local backup archive using the saved configuration
// and returns the absolute path of the created file. Retention is applied after
// a successful write.
func (s *Service) CreateLocalBackup() (string, error) {
	cfg, err := s.loadRuntimeConfig()
	if err != nil {
		return "", err
	}

	if cfg.EncryptEnabled && cfg.EncryptPassword == "" {
		return "", ErrBackupEncryptionPasswordRequired
	}

	dir, err := resolveBackupDir(cfg.LocalDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}

	// The timestamp is second-precision, so two concurrent runs (e.g. the
	// manual POST /backup/run firing in the same clock-second as the scheduled
	// run) would otherwise resolve to the same paths and race on os.Create. A
	// random token makes every run's temp DB and archive paths unique. The
	// suffix is compatible with backupFileNamePattern and the mod-time based
	// listing/retention ordering.
	token, err := newBackupToken()
	if err != nil {
		return "", err
	}
	timestamp := pkg.Now().Format("20060102-150405")
	dbTempPath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-backup-%s-%s.db", timestamp, token))
	if err := s.DB.Exec("VACUUM INTO ?", dbTempPath).Error; err != nil {
		return "", err
	}
	defer os.Remove(dbTempPath)

	archivePath := filepath.Join(dir, fmt.Sprintf("subdux-backup-%s-%s.zip", timestamp, token))
	password := ""
	if cfg.EncryptEnabled {
		password = cfg.EncryptPassword
	}
	if err := writeBackupZipFromDB(archivePath, dbTempPath, cfg.IncludeAssets, password); err != nil {
		_ = os.Remove(archivePath)
		return "", err
	}

	if err := s.applyLocalBackupRetention(dir, cfg.RetentionCount); err != nil {
		return archivePath, err
	}

	return archivePath, nil
}

// applyLocalBackupRetention deletes local backup files beyond the newest keep
// files. It only ever considers non-recursive entries in dir whose basename
// matches backupFileNamePattern, so unrelated files are never removed.
func (s *Service) applyLocalBackupRetention(dir string, keep int64) error {
	if keep < minBackupRetention {
		keep = minBackupRetention
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	type backupEntry struct {
		path    string
		modTime time.Time
	}
	matches := make([]backupEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !backupFileNamePattern.MatchString(entry.Name()) {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		matches = append(matches, backupEntry{
			path:    filepath.Join(dir, entry.Name()),
			modTime: info.ModTime(),
		})
	}

	if int64(len(matches)) <= keep {
		return nil
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime.After(matches[j].modTime)
	})

	for _, stale := range matches[keep:] {
		if err := os.Remove(stale.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	return nil
}

// ListLocalBackups returns the resolved backup directory and the local backup
// files it contains, newest first. A single unreadable file does not fail the
// whole listing.
func (s *Service) ListLocalBackups() (string, []LocalBackupInfo, error) {
	localDir, err := backupsettings.GetString(context.Background(), s.DB, KeyLocalDir, "")
	if err != nil {
		return "", nil, err
	}
	dir, err := resolveBackupDir(localDir)
	if err != nil {
		return "", nil, err
	}

	items := make([]LocalBackupInfo, 0)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return dir, items, nil
		}
		return dir, nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !backupFileNamePattern.MatchString(entry.Name()) {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		items = append(items, LocalBackupInfo{
			Name:       entry.Name(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().Format(time.RFC3339),
			Encrypted:  backupArchiveIsEncrypted(fullPath),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ModifiedAt > items[j].ModifiedAt
	})

	return dir, items, nil
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

// RunScheduledBackup runs a lease-guarded scheduled backup. It is safe to call
// on a short timer: it only performs work when the schedule is enabled and a
// backup is due for the current day, and it records the run status regardless
// of outcome.
func (s *Service) RunScheduledBackup(ownerID string) error {
	return serviceutil.WithBackgroundTaskLease(s.DB, ownerID, backupTaskKey, backupLeaseTTL, func() error {
		cfg, err := s.loadRuntimeConfig()
		if err != nil {
			return err
		}
		if !cfg.ScheduleEnabled {
			return nil
		}

		loc := pkg.GetSystemTimezone()
		now := pkg.NowInSystemTimezone()

		lastRunRaw, err := backupsettings.GetString(context.Background(), s.DB, KeyLastRunAt, "")
		if err != nil {
			return err
		}
		lastRunAt := parseBackupLastRunAt(lastRunRaw)

		if !backupDue(now, cfg.TimeOfDay, lastRunAt, loc) {
			return nil
		}

		if _, backupErr := s.CreateLocalBackup(); backupErr != nil {
			// Record the failure status/error but do NOT stamp the last-run
			// timestamp: backupDue gates on the last successful run, so a
			// transient failure must not suppress retries for the rest of the
			// day.
			if statusErr := s.recordBackupRunFailure(backupErr.Error()); statusErr != nil {
				return statusErr
			}
			return backupErr
		}

		return s.recordBackupRunSuccess(now)
	})
}

// parseBackupLastRunAt parses the persisted last-run timestamp. An empty or
// unparseable value yields the zero time, meaning "never run".
func parseBackupLastRunAt(raw string) time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}
	}
	return parsed
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

// recordBackupRunSuccess persists the runtime status keys for a successful
// scheduled run, stamping the last-run timestamp that backupDue uses to gate
// the once-per-day schedule. These keys are runtime-written status fields and
// are intentionally not part of UpdateSettingsInput.
func (s *Service) recordBackupRunSuccess(runAt time.Time) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := backupsettings.SaveString(tx, KeyLastRunAt, runAt.Format(time.RFC3339)); err != nil {
			return err
		}
		if err := backupsettings.SaveString(tx, KeyLastStatus, StatusOK); err != nil {
			return err
		}
		return backupsettings.SaveString(tx, KeyLastError, "")
	})
}

// recordBackupRunFailure persists the failure status and error for a scheduled
// run without touching KeyLastRunAt. Because backupDue gates on the last
// successful run, leaving the timestamp untouched allows retries to proceed on
// subsequent ticks the same day once the failure condition clears.
func (s *Service) recordBackupRunFailure(runErr string) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := backupsettings.SaveString(tx, KeyLastStatus, StatusFailed); err != nil {
			return err
		}
		return backupsettings.SaveString(tx, KeyLastError, runErr)
	})
}

// ApplySettings validates and persists the user-editable backup settings within
// the caller's transaction.
func ApplySettings(tx *gorm.DB, input UpdateSettingsInput) error {
	if input.TimeOfDay != nil {
		trimmed := strings.TrimSpace(*input.TimeOfDay)
		if !backupTimeOfDayPattern.MatchString(trimmed) {
			return ErrInvalidBackupTimeOfDay
		}
		if err := backupsettings.SaveString(tx, KeyTimeOfDay, trimmed); err != nil {
			return err
		}
	}

	if input.RetentionCount != nil {
		count := *input.RetentionCount
		if count < minBackupRetention || count > maxBackupRetention {
			return ErrInvalidBackupRetentionCount
		}
		if err := backupsettings.SaveString(tx, KeyRetentionCount, strconv.FormatInt(count, 10)); err != nil {
			return err
		}
	}

	if input.LocalDir != nil {
		normalized, err := normalizeBackupLocalDir(*input.LocalDir)
		if err != nil {
			return err
		}
		if err := backupsettings.SaveString(tx, KeyLocalDir, normalized); err != nil {
			return err
		}
	}

	if input.IncludeAssets != nil {
		if err := backupsettings.SaveBool(tx, KeyIncludeAssets, *input.IncludeAssets); err != nil {
			return err
		}
	}

	if input.ScheduleEnabled != nil {
		if err := backupsettings.SaveBool(tx, KeyScheduleEnabled, *input.ScheduleEnabled); err != nil {
			return err
		}
	}

	if input.EncryptionPassword != nil {
		if err := backupsettings.SaveEncrypted(tx, KeyEncryptionPassword, *input.EncryptionPassword); err != nil {
			return err
		}
	}

	if input.EncryptEnabled != nil {
		if *input.EncryptEnabled {
			if err := ensureBackupEncryptionPasswordAvailable(tx, input); err != nil {
				return err
			}
		}
		if err := backupsettings.SaveBool(tx, KeyEncryptEnabled, *input.EncryptEnabled); err != nil {
			return err
		}
	}

	return nil
}

// ensureBackupEncryptionPasswordAvailable confirms a password is available when
// enabling encryption: either provided in this request or already stored.
func ensureBackupEncryptionPasswordAvailable(tx *gorm.DB, input UpdateSettingsInput) error {
	if input.EncryptionPassword != nil && strings.TrimSpace(*input.EncryptionPassword) != "" {
		return nil
	}

	stored, err := backupsettings.GetString(context.Background(), tx, KeyEncryptionPassword, "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(stored) != "" {
		return nil
	}

	return ErrBackupEncryptionPasswordRequired
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
