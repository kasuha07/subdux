package pkg

import (
	"fmt"

	"gorm.io/gorm"
)

func createMissingTables(db *gorm.DB) error {
	for _, value := range applicationModels {
		if db.Migrator().HasTable(value) {
			continue
		}
		if err := db.Migrator().CreateTable(value); err != nil {
			return fmt.Errorf("create missing table: %w", err)
		}
	}
	return nil
}
