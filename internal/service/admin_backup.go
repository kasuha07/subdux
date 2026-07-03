package service

import (
	svcbackup "github.com/kasuha07/subdux/internal/service/backup"
)

// BackupDB produces an on-demand backup and returns the path of the file to
// serve. When password is empty the historical behavior is preserved: a raw
// SQLite .db file when includeAssets is false, or a plain .zip when true.
// When password is non-empty encryption requires a zip container, so the DB is
// always bundled into a WinZip AES-256 .zip (with assets honored) regardless of
// includeAssets. The password is trimmed before deciding, so all-whitespace is
// treated as empty.
func (s *AdminService) BackupDB(includeAssets bool, password string) (string, error) {
	return svcbackup.NewService(s.DB).BackupDB(includeAssets, password)
}
