package pkg

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	postgresBackupFormat  = "subdux-postgres-json-v1"
	maxPostgresBackupSize = 32 << 20
)

type postgresBackupTable struct {
	name          string
	columns       []string
	primaryFields []string
	schema        *schema.Schema
}

// WritePostgresBackup writes a portable JSON snapshot of the application
// tables. PostgreSQL serializes each row with to_jsonb, retaining native JSON,
// timestamp, numeric and bytea representations for the matching restore path.
func WritePostgresBackup(db *gorm.DB, dst io.Writer) error {
	if !IsPostgres(db) {
		return fmt.Errorf("PostgreSQL backup requires a PostgreSQL connection")
	}

	tables, err := postgresBackupTables(db)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if _, err := io.WriteString(dst, `{"format":"`+postgresBackupFormat+`","tables":{`); err != nil {
			return err
		}
		for i, table := range tables {
			if i > 0 {
				if _, err := io.WriteString(dst, ","); err != nil {
					return err
				}
			}
			encodedName, _ := json.Marshal(table.name)
			if _, err := dst.Write(encodedName); err != nil {
				return err
			}
			if _, err := io.WriteString(dst, ":["); err != nil {
				return err
			}

			rows, err := tx.Raw("SELECT to_jsonb(source_row) FROM " + quotePostgresIdentifier(table.name) + " AS source_row").Rows()
			if err != nil {
				return fmt.Errorf("read PostgreSQL backup table %s: %w", table.name, err)
			}
			rowCount := 0
			for rows.Next() {
				var row []byte
				if err := rows.Scan(&row); err != nil {
					_ = rows.Close()
					return fmt.Errorf("read PostgreSQL backup row from %s: %w", table.name, err)
				}
				if rowCount > 0 {
					if _, err := io.WriteString(dst, ","); err != nil {
						_ = rows.Close()
						return err
					}
				}
				if !json.Valid(row) {
					_ = rows.Close()
					return fmt.Errorf("PostgreSQL returned invalid JSON for table %s", table.name)
				}
				if _, err := dst.Write(row); err != nil {
					_ = rows.Close()
					return err
				}
				rowCount++
			}
			if err := rows.Err(); err != nil {
				_ = rows.Close()
				return fmt.Errorf("iterate PostgreSQL backup table %s: %w", table.name, err)
			}
			if err := rows.Close(); err != nil {
				return fmt.Errorf("close PostgreSQL backup rows for %s: %w", table.name, err)
			}
			if _, err := io.WriteString(dst, "]"); err != nil {
				return err
			}
		}
		_, err := io.WriteString(dst, "}}\n")
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
}

// ImportSQLiteBackupToPostgres migrates an SQLite backup into an already
// migrated PostgreSQL database. It copies the uploaded SQLite file before
// running the current SQLite migrations, then transfers current model columns
// through the same JSON restore contract used by PostgreSQL-native backups.
func ImportSQLiteBackupToPostgres(postgresDB *gorm.DB, sqlitePath string) error {
	if !IsPostgres(postgresDB) {
		return fmt.Errorf("SQLite import requires a PostgreSQL destination")
	}
	sourcePath, cleanup, err := copySQLiteBackupForImport(sqlitePath)
	if err != nil {
		return err
	}
	defer cleanup()

	sourceDB, err := openSQLiteDatabase(sourcePath)
	if err != nil {
		return fmt.Errorf("open and migrate SQLite backup: %w", err)
	}
	sourceSQL, err := sourceDB.DB()
	if err != nil {
		return fmt.Errorf("access SQLite backup connection: %w", err)
	}
	defer func() {
		_ = sourceSQL.Close()
		_ = os.Remove(sourcePath + "-wal")
		_ = os.Remove(sourcePath + "-shm")
		_ = os.Remove(sourcePath + "-journal")
	}()

	var snapshot bytes.Buffer
	if err := WriteSQLiteBackupForPostgres(sourceDB, &snapshot); err != nil {
		return fmt.Errorf("encode SQLite backup for PostgreSQL: %w", err)
	}
	if err := sourceSQL.Close(); err != nil {
		return fmt.Errorf("close migrated SQLite backup: %w", err)
	}
	return RestorePostgresBackup(postgresDB, bytes.NewReader(snapshot.Bytes()))
}

// WriteSQLiteBackupForPostgres exports current application columns from an
// SQLite connection into the JSON format accepted by RestorePostgresBackup.
// Blob values are encoded as PostgreSQL's \x-prefixed bytea text format.
func WriteSQLiteBackupForPostgres(db *gorm.DB, dst io.Writer) error {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "sqlite" {
		return fmt.Errorf("SQLite-to-PostgreSQL export requires an SQLite connection")
	}
	tables, err := postgresBackupTables(db)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if _, err := io.WriteString(dst, `{"format":"`+postgresBackupFormat+`","tables":{`); err != nil {
			return err
		}
		for tableIndex, table := range tables {
			if tableIndex > 0 {
				if _, err := io.WriteString(dst, ","); err != nil {
					return err
				}
			}
			encodedName, _ := json.Marshal(table.name)
			if _, err := dst.Write(encodedName); err != nil {
				return err
			}
			if _, err := io.WriteString(dst, ":["); err != nil {
				return err
			}

			quotedColumns := make([]string, 0, len(table.columns))
			for _, column := range table.columns {
				quotedColumns = append(quotedColumns, quotePostgresIdentifier(column))
			}
			query := "SELECT " + strings.Join(quotedColumns, ", ") + " FROM " + quotePostgresIdentifier(table.name)
			rows, err := tx.Raw(query).Rows()
			if err != nil {
				return fmt.Errorf("read SQLite backup table %s: %w", table.name, err)
			}
			rowCount := 0
			for rows.Next() {
				values := make([]any, len(table.columns))
				destinations := make([]any, len(values))
				for i := range values {
					destinations[i] = &values[i]
				}
				if err := rows.Scan(destinations...); err != nil {
					_ = rows.Close()
					return fmt.Errorf("read SQLite backup row from %s: %w", table.name, err)
				}
				row := make(map[string]any, len(values))
				for i, column := range table.columns {
					field := table.schema.FieldsByDBName[column]
					converted, err := sqliteBackupValueForPostgres(field, values[i])
					if err != nil {
						_ = rows.Close()
						return fmt.Errorf("convert SQLite backup value %s.%s: %w", table.name, column, err)
					}
					row[column] = converted
				}
				encodedRow, err := json.Marshal(row)
				if err != nil {
					_ = rows.Close()
					return fmt.Errorf("encode SQLite backup row from %s: %w", table.name, err)
				}
				if rowCount > 0 {
					if _, err := io.WriteString(dst, ","); err != nil {
						_ = rows.Close()
						return err
					}
				}
				if _, err := dst.Write(encodedRow); err != nil {
					_ = rows.Close()
					return err
				}
				rowCount++
			}
			if err := rows.Err(); err != nil {
				_ = rows.Close()
				return fmt.Errorf("iterate SQLite backup table %s: %w", table.name, err)
			}
			if err := rows.Close(); err != nil {
				return fmt.Errorf("close SQLite backup rows for %s: %w", table.name, err)
			}
			if _, err := io.WriteString(dst, "]"); err != nil {
				return err
			}
		}
		_, err := io.WriteString(dst, "}}\n")
		return err
	})
}

func sqliteBackupValueForPostgres(field *schema.Field, value any) (any, error) {
	if value == nil || field == nil {
		return value, nil
	}
	fieldType := field.FieldType
	for fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}
	if fieldType.Kind() == reflect.Bool {
		switch typed := value.(type) {
		case bool:
			return typed, nil
		case int64:
			return typed != 0, nil
		case int:
			return typed != 0, nil
		case []byte:
			parsed, err := strconv.ParseBool(string(typed))
			return parsed, err
		case string:
			parsed, err := strconv.ParseBool(typed)
			return parsed, err
		default:
			return nil, fmt.Errorf("unsupported SQLite boolean value type %T", value)
		}
	}
	if fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.Uint8 {
		if blob, ok := value.([]byte); ok {
			return `\x` + hex.EncodeToString(blob), nil
		}
	}
	if blob, ok := value.([]byte); ok {
		return string(blob), nil
	}
	return value, nil
}

func copySQLiteBackupForImport(sourcePath string) (string, func(), error) {
	source, err := os.Open(sourcePath) // #nosec G304 -- sourcePath comes from an admin-uploaded backup already validated by the restore path.
	if err != nil {
		return "", func() {}, err
	}
	defer source.Close()
	tempFile, err := os.CreateTemp("", "subdux-postgres-import-*.db")
	if err != nil {
		return "", func() {}, err
	}
	tempPath := tempFile.Name()
	cleanup := func() {
		_ = os.Remove(tempPath)
		_ = os.Remove(tempPath + "-wal")
		_ = os.Remove(tempPath + "-shm")
		_ = os.Remove(tempPath + "-journal")
	}
	if err := tempFile.Chmod(0o600); err != nil {
		_ = tempFile.Close()
		cleanup()
		return "", func() {}, err
	}
	if _, err := io.Copy(tempFile, source); err != nil {
		_ = tempFile.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := tempFile.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return tempPath, cleanup, nil
}

// RestorePostgresBackup validates the complete snapshot before deleting any
// current rows, restores it transactionally, and reloads the JWT key while
// holding the same lock used by token issuance and verification.
func RestorePostgresBackup(db *gorm.DB, src io.Reader) error {
	if !IsPostgres(db) {
		return fmt.Errorf("PostgreSQL restore requires a PostgreSQL connection")
	}
	limited, err := io.ReadAll(io.LimitReader(src, maxPostgresBackupSize+1))
	if err != nil {
		return fmt.Errorf("read PostgreSQL backup: %w", err)
	}
	if len(limited) > maxPostgresBackupSize {
		return fmt.Errorf("PostgreSQL backup exceeds the %d byte limit", maxPostgresBackupSize)
	}

	var snapshot struct {
		Format string                     `json:"format"`
		Tables map[string]json.RawMessage `json:"tables"`
	}
	decoder := json.NewDecoder(bytes.NewReader(limited))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&snapshot); err != nil {
		return fmt.Errorf("decode PostgreSQL backup: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			return fmt.Errorf("PostgreSQL backup has trailing JSON data")
		}
		return fmt.Errorf("read PostgreSQL backup trailer: %w", err)
	}
	if snapshot.Format != postgresBackupFormat {
		return fmt.Errorf("unsupported PostgreSQL backup format %q", snapshot.Format)
	}
	if snapshot.Tables == nil {
		return fmt.Errorf("PostgreSQL backup has no table data")
	}

	tables, err := postgresBackupTables(db)
	if err != nil {
		return err
	}
	knownTables := make(map[string]struct{}, len(tables))
	for _, table := range tables {
		knownTables[table.name] = struct{}{}
		if !db.Migrator().HasTable(table.name) {
			return fmt.Errorf("PostgreSQL restore table %s is missing from the current schema", table.name)
		}
	}
	for tableName := range snapshot.Tables {
		if _, ok := knownTables[tableName]; !ok {
			return fmt.Errorf("PostgreSQL backup contains unknown table %s", tableName)
		}
	}

	jwtSecretMu.Lock()
	defer jwtSecretMu.Unlock()
	if err := db.Transaction(func(tx *gorm.DB) error {
		quotedTables := make([]string, 0, len(tables))
		for _, table := range tables {
			quotedTables = append(quotedTables, quotePostgresIdentifier(table.name))
		}
		if err := tx.Exec("TRUNCATE TABLE " + strings.Join(quotedTables, ", ") + " RESTART IDENTITY CASCADE").Error; err != nil {
			return fmt.Errorf("clear PostgreSQL application tables: %w", err)
		}

		for _, table := range tables {
			rawRows, ok := snapshot.Tables[table.name]
			if !ok {
				continue
			}
			var rows []map[string]json.RawMessage
			if trimmed := bytes.TrimSpace(rawRows); len(trimmed) == 0 || trimmed[0] != '[' {
				return fmt.Errorf("PostgreSQL backup table %s is not a JSON array", table.name)
			}
			if err := json.Unmarshal(rawRows, &rows); err != nil {
				return fmt.Errorf("decode PostgreSQL backup table %s: %w", table.name, err)
			}
			if len(rows) == 0 {
				continue
			}

			var availableColumns []string
			if err := tx.Raw(`
				SELECT column_name FROM information_schema.columns
				WHERE table_schema = current_schema() AND table_name = ?
				ORDER BY ordinal_position
			`, table.name).Scan(&availableColumns).Error; err != nil {
				return fmt.Errorf("inspect PostgreSQL restore table %s: %w", table.name, err)
			}
			available := make(map[string]struct{}, len(availableColumns))
			for _, column := range availableColumns {
				available[column] = struct{}{}
			}

			columnSet := make(map[string]struct{})
			for _, row := range rows {
				for column := range row {
					if _, ok := available[column]; !ok {
						return fmt.Errorf("PostgreSQL backup table %s has unknown column %s", table.name, column)
					}
					columnSet[column] = struct{}{}
				}
			}
			columns := make([]string, 0, len(columnSet))
			for column := range columnSet {
				columns = append(columns, column)
			}
			sort.Strings(columns)
			if len(columns) == 0 {
				return fmt.Errorf("PostgreSQL backup table %s contains rows without columns", table.name)
			}

			quotedColumns := make([]string, 0, len(columns))
			for _, column := range columns {
				quotedColumns = append(quotedColumns, quotePostgresIdentifier(column))
			}
			insertSQL := "INSERT INTO " + quotePostgresIdentifier(table.name) +
				" (" + strings.Join(quotedColumns, ", ") + ") SELECT " + strings.Join(quotedColumns, ", ") +
				" FROM jsonb_populate_recordset(NULL::" + quotePostgresIdentifier(table.name) + ", ?::jsonb)"
			if err := tx.Exec(insertSQL, string(rawRows)).Error; err != nil {
				return fmt.Errorf("restore PostgreSQL table %s: %w", table.name, err)
			}

			for _, primaryField := range table.primaryFields {
				var sequence sql.NullString
				if err := tx.Raw("SELECT pg_get_serial_sequence(?, ?)", table.name, primaryField).Row().Scan(&sequence); err != nil {
					return fmt.Errorf("find PostgreSQL sequence for %s.%s: %w", table.name, primaryField, err)
				}
				if !sequence.Valid {
					continue
				}
				var maxID sql.NullInt64
				if err := tx.Raw("SELECT MAX(" + quotePostgresIdentifier(primaryField) + ") FROM " + quotePostgresIdentifier(table.name)).Row().Scan(&maxID); err != nil {
					return fmt.Errorf("read PostgreSQL sequence value for %s.%s: %w", table.name, primaryField, err)
				}
				if maxID.Valid {
					var restoredSequenceValue int64
					if err := tx.Raw("SELECT setval(?::regclass, ?, true)", sequence.String, maxID.Int64).Row().Scan(&restoredSequenceValue); err != nil {
						return fmt.Errorf("restore PostgreSQL sequence for %s.%s: %w", table.name, primaryField, err)
					}
				} else {
					var resetSequenceValue int64
					if err := tx.Raw("SELECT setval(?::regclass, 1, false)", sequence.String).Row().Scan(&resetSequenceValue); err != nil {
						return fmt.Errorf("reset empty PostgreSQL sequence for %s.%s: %w", table.name, primaryField, err)
					}
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if err := initJWTSecretLocked(db); err != nil {
		return fmt.Errorf("reload JWT secret after PostgreSQL restore: %w", err)
	}
	return nil
}

func postgresBackupTables(db *gorm.DB) ([]postgresBackupTable, error) {
	models := model.ApplicationModels()
	tables := make([]postgresBackupTable, 0, len(models))
	for _, value := range models {
		statement := &gorm.Statement{DB: db}
		if err := statement.Parse(value); err != nil {
			return nil, fmt.Errorf("parse application backup model %T: %w", value, err)
		}
		table := postgresBackupTable{
			name:   statement.Schema.Table,
			schema: statement.Schema,
		}
		for _, column := range statement.Schema.DBNames {
			table.columns = append(table.columns, column)
		}
		for _, field := range statement.Schema.PrimaryFields {
			table.primaryFields = append(table.primaryFields, field.DBName)
		}
		if _, err := quotePostgresIdentifierChecked(table.name); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func quotePostgresIdentifier(value string) string {
	quoted, _ := quotePostgresIdentifierChecked(value)
	return quoted
}

func quotePostgresIdentifierChecked(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("empty PostgreSQL identifier")
	}
	for i, r := range value {
		if !(r == '_' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9') {
			return "", fmt.Errorf("invalid PostgreSQL identifier %q", value)
		}
	}
	return `"` + value + `"`, nil
}
