package catalog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"gorm.io/gorm"
)

// Client-facing catalog errors, typed with a serviceerr.Kind so the transport
// layer maps them to a status in one place. Messages are preserved verbatim
// from the inline errors they replace. The "in use" errors remain 400
// (KindInvalid), matching prior behavior.
var (
	ErrCurrencyInUse       = serviceerr.New(serviceerr.KindInvalid, "currency_is_in_use_by_existing_subscriptions", "currency is in use by existing subscriptions")
	ErrCategoryInUse       = serviceerr.New(serviceerr.KindInvalid, "category_is_in_use_by_existing_subscriptions", "category is in use by existing subscriptions")
	ErrPaymentMethodInUse  = serviceerr.New(serviceerr.KindInvalid, "payment_method_is_in_use_by_existing_subscriptions", "payment method is in use by existing subscriptions")
	ErrImageUploadDisabled = serviceutil.ErrImageUploadDisabled

	ErrCategoryNameLength      = serviceerr.New(serviceerr.KindInvalid, "name_must_be_1_30_characters", "name must be 1-30 characters")
	ErrCategoryNameExists      = serviceerr.New(serviceerr.KindConflict, "category_name_already_exists", "category name already exists")
	ErrCategoryNotFound        = serviceerr.New(serviceerr.KindNotFound, "category_not_found", "category not found")
	ErrCurrencyCodeLength      = serviceerr.New(serviceerr.KindInvalid, "code_must_be_1_10_characters", "code must be 1-10 characters")
	ErrCurrencyCodeUppercase   = serviceerr.New(serviceerr.KindInvalid, "code_must_contain_only_uppercase_letters", "code must contain only uppercase letters")
	ErrCurrencyCodeExists      = serviceerr.New(serviceerr.KindConflict, "currency_code_already_exists", "currency code already exists")
	ErrCurrencyNotFound        = serviceerr.New(serviceerr.KindNotFound, "currency_not_found", "currency not found")
	ErrCurrencyPreferredDelete = serviceerr.New(serviceerr.KindInvalid, "cannot_delete_your_preferred_currency", "cannot delete your preferred currency")
	ErrPaymentMethodNameLength = serviceerr.New(serviceerr.KindInvalid, "name_must_be_1_50_characters", "name must be 1-50 characters")
	ErrPaymentMethodNameExists = serviceerr.New(serviceerr.KindConflict, "payment_method_name_already_exists", "payment method name already exists")
	ErrPaymentMethodNotFound   = serviceerr.New(serviceerr.KindNotFound, "payment_method_not_found", "payment method not found")
)

type CategoryService struct {
	DB *gorm.DB
}

func NewCategoryService(db *gorm.DB) *CategoryService {
	return &CategoryService{DB: db}
}

type CreateCategoryInput struct {
	Name         string `json:"name"`
	DisplayOrder int    `json:"display_order"`
}

type UpdateCategoryInput struct {
	Name         *string `json:"name"`
	DisplayOrder *int    `json:"display_order"`
}

func (s *CategoryService) WithContext(ctx context.Context) *CategoryService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *CategoryService) List(userID uint) ([]model.Category, error) {
	var categories []model.Category
	err := s.DB.Where("user_id = ?", userID).Order("display_order ASC, id ASC").Find(&categories).Error
	return categories, err
}

func (s *CategoryService) Create(userID uint, input CreateCategoryInput) (*model.Category, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 30 {
		return nil, ErrCategoryNameLength
	}

	var existing model.Category
	err := s.DB.Where("user_id = ? AND name = ?", userID, name).First(&existing).Error
	if err == nil {
		return nil, ErrCategoryNameExists
	}

	category := model.Category{
		UserID:         userID,
		Name:           name,
		SystemKey:      nil,
		NameCustomized: true,
		DisplayOrder:   input.DisplayOrder,
	}

	if err := s.DB.Create(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (s *CategoryService) Update(userID, id uint, input UpdateCategoryInput) (*model.Category, error) {
	var category model.Category
	if err := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&category).Error; err != nil {
		return nil, ErrCategoryNotFound
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 30 {
			return nil, ErrCategoryNameLength
		}
		var existing model.Category
		err := s.DB.Where("user_id = ? AND name = ? AND id != ?", userID, name, id).First(&existing).Error
		if err == nil {
			return nil, ErrCategoryNameExists
		}
		category.Name = name
		category.NameCustomized = true
	}

	if input.DisplayOrder != nil {
		category.DisplayOrder = *input.DisplayOrder
	}

	if err := s.DB.Save(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (s *CategoryService) Delete(userID, id uint) error {
	var category model.Category
	if err := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&category).Error; err != nil {
		return ErrCategoryNotFound
	}

	var subscriptionsUsingCategory int64
	if err := s.DB.Model(&model.Subscription{}).
		Where("user_id = ? AND (category_id = ? OR category = ?)", userID, category.ID, category.Name).
		Count(&subscriptionsUsingCategory).Error; err != nil {
		return err
	}
	if subscriptionsUsingCategory > 0 {
		return ErrCategoryInUse
	}

	return s.DB.Delete(&category).Error
}

func (s *CategoryService) Reorder(userID uint, items []ReorderItem) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&model.Category{}).
				Where("id = ? AND user_id = ?", item.ID, userID).
				Update("display_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type CurrencyService struct {
	DB *gorm.DB
}

func NewCurrencyService(db *gorm.DB) *CurrencyService {
	return &CurrencyService{DB: db}
}

type CreateCurrencyInput struct {
	Code      string `json:"code"`
	Symbol    string `json:"symbol"`
	Alias     string `json:"alias"`
	SortOrder int    `json:"sort_order"`
}

type UpdateCurrencyInput struct {
	Symbol    *string `json:"symbol"`
	Alias     *string `json:"alias"`
	SortOrder *int    `json:"sort_order"`
}

type ReorderItem struct {
	ID        uint `json:"id"`
	SortOrder int  `json:"sort_order"`
}

func (s *CurrencyService) WithContext(ctx context.Context) *CurrencyService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *CurrencyService) List(userID uint) ([]model.UserCurrency, error) {
	var currencies []model.UserCurrency
	err := s.DB.Where("user_id = ?", userID).Order("sort_order ASC, id ASC").Find(&currencies).Error
	return currencies, err
}

func (s *CurrencyService) Create(userID uint, input CreateCurrencyInput) (*model.UserCurrency, error) {
	code := strings.ToUpper(strings.TrimSpace(input.Code))
	if code == "" || len(code) > 10 {
		return nil, ErrCurrencyCodeLength
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return nil, ErrCurrencyCodeUppercase
		}
	}

	var existing model.UserCurrency
	err := s.DB.Where("user_id = ? AND code = ?", userID, code).First(&existing).Error
	if err == nil {
		return nil, ErrCurrencyCodeExists
	}

	currency := model.UserCurrency{
		UserID:    userID,
		Code:      code,
		Symbol:    strings.TrimSpace(input.Symbol),
		Alias:     strings.TrimSpace(input.Alias),
		SortOrder: input.SortOrder,
	}

	if err := s.DB.Create(&currency).Error; err != nil {
		return nil, err
	}
	return &currency, nil
}

func (s *CurrencyService) Update(userID, id uint, input UpdateCurrencyInput) (*model.UserCurrency, error) {
	var currency model.UserCurrency
	if err := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&currency).Error; err != nil {
		return nil, ErrCurrencyNotFound
	}
	if input.Symbol != nil {
		currency.Symbol = strings.TrimSpace(*input.Symbol)
	}
	if input.Alias != nil {
		currency.Alias = strings.TrimSpace(*input.Alias)
	}
	if input.SortOrder != nil {
		currency.SortOrder = *input.SortOrder
	}
	if err := s.DB.Save(&currency).Error; err != nil {
		return nil, err
	}
	return &currency, nil
}

func (s *CurrencyService) Delete(userID, id uint, preferredCurrency string) error {
	var currency model.UserCurrency
	if err := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&currency).Error; err != nil {
		return ErrCurrencyNotFound
	}
	if strings.EqualFold(currency.Code, preferredCurrency) {
		return ErrCurrencyPreferredDelete
	}

	var subscriptionsUsingCurrency int64
	if err := s.DB.Model(&model.Subscription{}).
		Where("user_id = ? AND UPPER(currency) = ?", userID, strings.ToUpper(currency.Code)).
		Count(&subscriptionsUsingCurrency).Error; err != nil {
		return err
	}
	if subscriptionsUsingCurrency > 0 {
		return ErrCurrencyInUse
	}

	return s.DB.Delete(&currency).Error
}

func (s *CurrencyService) Reorder(userID uint, items []ReorderItem) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&model.UserCurrency{}).
				Where("id = ? AND user_id = ?", item.ID, userID).
				Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

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

func (s *PaymentMethodService) WithContext(ctx context.Context) *PaymentMethodService {
	clone := *s
	clone.DB = withContext(s.DB, ctx)
	return &clone
}

func (s *PaymentMethodService) List(userID uint) ([]model.PaymentMethod, error) {
	var methods []model.PaymentMethod
	err := s.DB.Where("user_id = ?", userID).Order("sort_order ASC, id ASC").Find(&methods).Error
	return methods, err
}

func (s *PaymentMethodService) Create(userID uint, input CreatePaymentMethodInput) (*model.PaymentMethod, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 50 {
		return nil, ErrPaymentMethodNameLength
	}

	var existing model.PaymentMethod
	err := s.DB.Where("user_id = ? AND name = ?", userID, name).First(&existing).Error
	if err == nil {
		return nil, ErrPaymentMethodNameExists
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
		return nil, ErrPaymentMethodNotFound
	}

	oldIcon := method.Icon
	shouldRemoveOldIcon := false

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 50 {
			return nil, ErrPaymentMethodNameLength
		}

		var existing model.PaymentMethod
		err := s.DB.Where("user_id = ? AND name = ? AND id != ?", userID, name, id).First(&existing).Error
		if err == nil {
			return nil, ErrPaymentMethodNameExists
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
		return ErrPaymentMethodNotFound
	}

	var subscriptionsUsingMethod int64
	if err := s.DB.Model(&model.Subscription{}).
		Where("user_id = ? AND payment_method_id = ?", userID, id).
		Count(&subscriptionsUsingMethod).Error; err != nil {
		return err
	}
	if subscriptionsUsingMethod > 0 {
		return ErrPaymentMethodInUse
	}

	if err := s.DB.Delete(&model.PaymentMethod{}, "id = ? AND user_id = ?", id, userID).Error; err != nil {
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

func (s *PaymentMethodService) AllowImageUpload() bool {
	var setting model.SystemSetting
	if err := s.DB.Where("key = ?", "allow_image_upload").First(&setting).Error; err == nil {
		return setting.Value == "true"
	}
	return true
}

func (s *PaymentMethodService) UploadPaymentMethodIcon(userID, methodID uint, file io.Reader, filename string, maxSize int64) (string, error) {
	if !s.AllowImageUpload() {
		return "", ErrImageUploadDisabled
	}

	method, err := s.GetByID(userID, methodID)
	if err != nil {
		return "", ErrPaymentMethodNotFound
	}

	sanitized, ext, err := serviceutil.SanitizeUploadedIcon(file, filename, maxSize)
	if err != nil {
		return "", err
	}

	iconDir := filepath.Join(pkg.GetDataPath(), "assets", "icons")
	if err := os.MkdirAll(iconDir, 0o750); err != nil {
		return "", errors.New("failed to create icon directory")
	}

	newFilename := fmt.Sprintf("%d_payment_%d_%d%s", userID, methodID, pkg.Now().UnixNano(), ext)
	destPath := filepath.Join(iconDir, newFilename)

	if err := os.WriteFile(destPath, sanitized, 0o600); err != nil {
		return "", errors.New("failed to save icon file")
	}

	iconValue := "file:" + newFilename
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

func withContext(db *gorm.DB, ctx context.Context) *gorm.DB {
	if db == nil {
		return nil
	}
	return db.WithContext(ctx)
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
