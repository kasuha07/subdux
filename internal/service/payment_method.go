package service

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

type PaymentMethodService struct {
	DB *gorm.DB
}

func NewPaymentMethodService(db *gorm.DB) *PaymentMethodService {
	return &PaymentMethodService{DB: db}
}

type CreatePaymentMethodInput struct {
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
}

type UpdatePaymentMethodInput struct {
	Name      *string `json:"name"`
	Icon      *string `json:"icon"`
	SortOrder *int    `json:"sort_order"`
}

func (s *PaymentMethodService) List(userID uint) ([]model.PaymentMethod, error) {
	var methods []model.PaymentMethod
	err := s.DB.Where("user_id = ?", userID).Order("sort_order ASC, id ASC").Find(&methods).Error
	return methods, err
}

func (s *PaymentMethodService) Create(userID uint, input CreatePaymentMethodInput) (*model.PaymentMethod, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 50 {
		return nil, errors.New("name must be 1-50 characters")
	}

	var existing model.PaymentMethod
	err := s.DB.Where("user_id = ? AND name = ?", userID, name).First(&existing).Error
	if err == nil {
		return nil, errors.New("payment method name already exists")
	}

	method := model.PaymentMethod{
		UserID:         userID,
		Name:           name,
		SystemKey:      nil,
		NameCustomized: true,
		Icon:           strings.TrimSpace(input.Icon),
		SortOrder:      input.SortOrder,
	}

	if err := s.DB.Create(&method).Error; err != nil {
		return nil, err
	}
	return &method, nil
}

func (s *PaymentMethodService) Update(userID, id uint, input UpdatePaymentMethodInput) (*model.PaymentMethod, error) {
	method, err := s.GetByID(userID, id)
	if err != nil {
		return nil, errors.New("payment method not found")
	}

	oldIcon := method.Icon
	shouldRemoveOldIcon := false

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 50 {
			return nil, errors.New("name must be 1-50 characters")
		}

		var existing model.PaymentMethod
		err := s.DB.Where("user_id = ? AND name = ? AND id != ?", userID, name, id).First(&existing).Error
		if err == nil {
			return nil, errors.New("payment method name already exists")
		}
		method.Name = name
		method.NameCustomized = true
	}

	if input.Icon != nil {
		nextIcon := strings.TrimSpace(*input.Icon)
		shouldRemoveOldIcon = oldIcon != "" && oldIcon != nextIcon
		method.Icon = nextIcon
	}

	if input.SortOrder != nil {
		method.SortOrder = *input.SortOrder
	}

	if err := s.DB.Save(method).Error; err != nil {
		return nil, err
	}

	if shouldRemoveOldIcon {
		s.removeManagedIconFile(oldIcon)
	}

	return method, nil
}

func (s *PaymentMethodService) Delete(userID, id uint) error {
	method, err := s.GetByID(userID, id)
	if err != nil {
		return errors.New("payment method not found")
	}

	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Subscription{}).
			Where("user_id = ? AND payment_method_id = ?", userID, id).
			Update("payment_method_id", nil).Error; err != nil {
			return err
		}
		return tx.Delete(&model.PaymentMethod{}, "id = ? AND user_id = ?", id, userID).Error
	}); err != nil {
		return err
	}

	s.removeManagedIconFile(method.Icon)
	return nil
}

func (s *PaymentMethodService) Reorder(userID uint, items []ReorderItem) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&model.PaymentMethod{}).
				Where("id = ? AND user_id = ?", item.ID, userID).
				Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *PaymentMethodService) GetByID(userID, id uint) (*model.PaymentMethod, error) {
	var method model.PaymentMethod
	err := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&method).Error
	if err != nil {
		return nil, err
	}
	return &method, nil
}

func (s *PaymentMethodService) GetMaxIconFileSize() int64 {
	var setting model.SystemSetting
	if err := s.DB.Where("key = ?", "max_icon_file_size").First(&setting).Error; err == nil {
		if v, err := strconv.ParseInt(setting.Value, 10, 64); err == nil {
			return v
		}
	}
	return 65536
}

func (s *PaymentMethodService) UploadPaymentMethodIcon(userID, methodID uint, file io.Reader, filename string, maxSize int64) (string, error) {
	method, err := s.GetByID(userID, methodID)
	if err != nil {
		return "", errors.New("payment method not found")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return "", errors.New("only PNG and JPG images are supported")
	}

	buf, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		return "", errors.New("failed to read file")
	}
	if int64(len(buf)) > maxSize {
		return "", errors.New("file size exceeds limit")
	}

	contentType := http.DetectContentType(buf)
	if contentType != "image/png" && contentType != "image/jpeg" {
		return "", errors.New("only PNG and JPG images are supported")
	}

	if ext == ".jpeg" {
		ext = ".jpg"
	}

	iconDir := filepath.Join(pkg.GetDataPath(), "assets", "icons")
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		return "", errors.New("failed to create icon directory")
	}

	newFilename := fmt.Sprintf("%d_payment_%d_%d%s", userID, methodID, time.Now().UnixNano(), ext)
	destPath := filepath.Join(iconDir, newFilename)

	if err := os.WriteFile(destPath, buf, 0644); err != nil {
		return "", errors.New("failed to save icon file")
	}

	iconValue := "assets/icons/" + newFilename
	if err := s.DB.Model(&model.PaymentMethod{}).
		Where("id = ? AND user_id = ?", methodID, userID).
		Update("icon", iconValue).Error; err != nil {
		_ = os.Remove(destPath)
		return "", err
	}

	s.removeManagedIconFile(method.Icon)
	return iconValue, nil
}

func (s *PaymentMethodService) removeManagedIconFile(icon string) {
	if path, ok := managedIconFilePath(icon); ok {
		_ = os.Remove(path)
	}
}
