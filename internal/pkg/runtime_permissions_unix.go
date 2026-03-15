//go:build unix

package pkg

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

const defaultRuntimeUID = 65532
const defaultRuntimeGID = 65532

func prepareDataPathRuntimeOwnership(dataPath string) error {
	if os.Geteuid() != 0 {
		return nil
	}

	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := filepath.WalkDir(dataPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := os.Lchown(path, defaultRuntimeUID, defaultRuntimeGID); err != nil {
			return fmt.Errorf("failed to chown %q: %w", path, err)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := syscall.Setgroups([]int{}); err != nil {
		return fmt.Errorf("failed to clear supplemental groups: %w", err)
	}
	if err := syscall.Setgid(defaultRuntimeGID); err != nil {
		return fmt.Errorf("failed to drop gid to %d: %w", defaultRuntimeGID, err)
	}
	if err := syscall.Setuid(defaultRuntimeUID); err != nil {
		return fmt.Errorf("failed to drop uid to %d: %w", defaultRuntimeUID, err)
	}
	if os.Getegid() != defaultRuntimeGID || os.Geteuid() != defaultRuntimeUID {
		return fmt.Errorf("failed to confirm dropped privileges, current uid=%d gid=%d", os.Geteuid(), os.Getegid())
	}

	return nil
}
