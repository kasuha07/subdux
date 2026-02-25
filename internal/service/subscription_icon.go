package service

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
)

func (s *SubscriptionService) GetMaxIconFileSize() int64 {
	var setting model.SystemSetting
	if err := s.DB.Where("key = ?", "max_icon_file_size").First(&setting).Error; err == nil {
		if v, err := strconv.ParseInt(setting.Value, 10, 64); err == nil {
			return v
		}
	}
	return 65536
}

func (s *SubscriptionService) AllowImageUpload() bool {
	var setting model.SystemSetting
	if err := s.DB.Where("key = ?", "allow_image_upload").First(&setting).Error; err == nil {
		return setting.Value == "true"
	}
	return true
}

func (s *SubscriptionService) UploadSubscriptionIcon(userID, subID uint, file io.Reader, filename string, maxSize int64) (string, error) {
	if !s.AllowImageUpload() {
		return "", ErrImageUploadDisabled
	}

	sub, err := s.GetByID(userID, subID)
	if err != nil {
		return "", errors.New("subscription not found")
	}

	sanitized, ext, err := sanitizeUploadedIcon(file, filename, maxSize)
	if err != nil {
		return "", err
	}

	iconDir := filepath.Join(pkg.GetDataPath(), "assets", "icons")
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		return "", errors.New("failed to create icon directory")
	}

	newFilename := fmt.Sprintf("%d_%d_%d%s", userID, subID, time.Now().UnixNano(), ext)
	destPath := filepath.Join(iconDir, newFilename)

	if err := os.WriteFile(destPath, sanitized, 0644); err != nil {
		return "", errors.New("failed to save icon file")
	}

	s.removeManagedIconFile(sub.Icon)

	iconValue := "file:" + newFilename
	if err := s.DB.Model(&model.Subscription{}).Where("id = ? AND user_id = ?", subID, userID).Update("icon", iconValue).Error; err != nil {
		os.Remove(destPath)
		return "", err
	}

	return iconValue, nil
}

func (s *SubscriptionService) removeManagedIconFile(icon string) {
	if path, ok := managedIconFilePath(icon); ok {
		_ = os.Remove(path)
	}
}

func managedIconFilePath(icon string) (string, bool) {
	const iconPrefix = "file:"
	if !strings.HasPrefix(icon, iconPrefix) {
		return "", false
	}

	filename := strings.TrimPrefix(icon, iconPrefix)
	if filename == "" {
		return "", false
	}
	if strings.Contains(filename, "/") || strings.Contains(filename, `\`) {
		return "", false
	}
	if filepath.Base(filename) != filename {
		return "", false
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".ico" {
		return "", false
	}

	return filepath.Join(pkg.GetDataPath(), "assets", "icons", filename), true
}
