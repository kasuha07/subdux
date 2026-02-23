package service

import (
	"errors"
	"strings"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
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

func (s *CategoryService) List(userID uint) ([]model.Category, error) {
	var categories []model.Category
	err := s.DB.Where("user_id = ?", userID).Order("display_order ASC, id ASC").Find(&categories).Error
	return categories, err
}

func (s *CategoryService) Create(userID uint, input CreateCategoryInput) (*model.Category, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 30 {
		return nil, errors.New("name must be 1-30 characters")
	}

	var existing model.Category
	err := s.DB.Where("user_id = ? AND name = ?", userID, name).First(&existing).Error
	if err == nil {
		return nil, errors.New("category name already exists")
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
		return nil, errors.New("category not found")
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 30 {
			return nil, errors.New("name must be 1-30 characters")
		}
		var existing model.Category
		err := s.DB.Where("user_id = ? AND name = ? AND id != ?", userID, name, id).First(&existing).Error
		if err == nil {
			return nil, errors.New("category name already exists")
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
		return errors.New("category not found")
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
