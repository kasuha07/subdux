package pkg

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func withSQLiteForeignKeysDisabled(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if err := db.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
		return fmt.Errorf("disable sqlite foreign keys: %w", err)
	}
	defer func() {
		_ = db.Exec("PRAGMA foreign_keys = ON").Error
	}()

	if err := db.Transaction(fn); err != nil {
		return err
	}

	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return fmt.Errorf("re-enable sqlite foreign keys: %w", err)
	}
	return validateSQLiteForeignKeys(db)
}

func validateSQLiteForeignKeys(db *gorm.DB) error {
	rows, err := db.Raw("PRAGMA foreign_key_check").Rows()
	if err != nil {
		return fmt.Errorf("run sqlite foreign_key_check: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		var table string
		var rowID int64
		var parent string
		var fkIndex int64
		if err := rows.Scan(&table, &rowID, &parent, &fkIndex); err != nil {
			return fmt.Errorf("scan sqlite foreign_key_check row: %w", err)
		}
		return fmt.Errorf("sqlite foreign key violation: table=%s rowid=%d parent=%s fk=%d", table, rowID, parent, fkIndex)
	}
	return nil
}

func rebuildSQLiteTable(db *gorm.DB, value interface{}) error {
	tableName, columnNames, err := sqliteModelMetadata(db, value)
	if err != nil {
		return err
	}
	legacyTableName := tableName + sqliteRebuildTableSuffix

	if db.Migrator().HasTable(legacyTableName) {
		if err := db.Migrator().DropTable(legacyTableName); err != nil {
			return fmt.Errorf("drop leftover legacy table %s: %w", legacyTableName, err)
		}
	}

	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", sqliteQuoteIdent(tableName), sqliteQuoteIdent(legacyTableName))).Error; err != nil {
		return fmt.Errorf("rename table %s: %w", tableName, err)
	}

	if err := dropSQLiteIndexesForTable(db, legacyTableName); err != nil {
		return err
	}

	if err := db.Migrator().CreateTable(value); err != nil {
		return fmt.Errorf("create rebuilt table %s: %w", tableName, err)
	}

	quotedColumns := sqliteJoinColumns(columnNames)
	copySQL := fmt.Sprintf(
		"INSERT INTO %s (%s) SELECT %s FROM %s",
		sqliteQuoteIdent(tableName),
		quotedColumns,
		quotedColumns,
		sqliteQuoteIdent(legacyTableName),
	)
	if err := db.Exec(copySQL).Error; err != nil {
		return fmt.Errorf("copy data into rebuilt table %s: %w", tableName, err)
	}

	if err := db.Migrator().DropTable(legacyTableName); err != nil {
		return fmt.Errorf("drop legacy table %s: %w", legacyTableName, err)
	}

	return nil
}

func sqliteModelMetadata(db *gorm.DB, value interface{}) (string, []string, error) {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(value); err != nil {
		return "", nil, fmt.Errorf("parse sqlite model metadata: %w", err)
	}

	columnNames := make([]string, 0, len(stmt.Schema.DBNames))
	for _, name := range stmt.Schema.DBNames {
		if name == "" {
			continue
		}
		columnNames = append(columnNames, name)
	}
	return stmt.Schema.Table, columnNames, nil
}

func dropSQLiteIndexesForTable(db *gorm.DB, tableName string) error {
	rows, err := db.Raw(fmt.Sprintf("PRAGMA index_list(%s)", sqliteQuoteIdent(tableName))).Rows()
	if err != nil {
		return fmt.Errorf("list sqlite indexes for %s: %w", tableName, err)
	}
	defer rows.Close()

	var indexNames []string
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return fmt.Errorf("scan sqlite index metadata for %s: %w", tableName, err)
		}
		if strings.HasPrefix(name, "sqlite_autoindex_") {
			continue
		}
		indexNames = append(indexNames, name)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate sqlite indexes for %s: %w", tableName, err)
	}

	for _, name := range indexNames {
		if err := db.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", sqliteQuoteIdent(name))).Error; err != nil {
			return fmt.Errorf("drop sqlite index %s: %w", name, err)
		}
	}

	return nil
}

func sqliteJoinColumns(columns []string) string {
	quoted := make([]string, 0, len(columns))
	for _, column := range columns {
		quoted = append(quoted, sqliteQuoteIdent(column))
	}
	return strings.Join(quoted, ", ")
}

func sqliteQuoteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
