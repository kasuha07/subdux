package service

import (
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	"gorm.io/gorm"
)

type CategoryService = catalogservice.CategoryService
type CreateCategoryInput = catalogservice.CreateCategoryInput
type UpdateCategoryInput = catalogservice.UpdateCategoryInput

func NewCategoryService(db *gorm.DB) *CategoryService {
	return catalogservice.NewCategoryService(db)
}
