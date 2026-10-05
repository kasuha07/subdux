package migrations

import (
	"strings"

	"gorm.io/gorm"
)

// migrateOIDCConnectionIssuer binds every OIDC link to the issuer that asserted
// its subject. OIDC only guarantees subject uniqueness within one issuer, so a
// link keyed by subject alone would let a different identity provider log in as
// a linked user by asserting the same subject.
//
// Existing links were all created against the issuer configured today (the
// application only ever trusts one), so they are attributed to it. When no
// issuer is configured the links stay unattributed: they can no longer match
// any login until the user links their account again. Fixed SQL keeps this
// migration independent of future model changes.
func migrateOIDCConnectionIssuer(db *gorm.DB) error {
	if !db.Migrator().HasTable("oidc_connections") {
		return nil
	}

	if !db.Migrator().HasColumn("oidc_connections", "issuer") {
		columnType := "text"
		if db.Dialector.Name() == "postgres" {
			columnType = "varchar(2048)"
		}
		if err := db.Exec("ALTER TABLE oidc_connections ADD COLUMN issuer " + columnType + " NOT NULL DEFAULT ''").Error; err != nil {
			return err
		}
	}

	if db.Migrator().HasTable("system_settings") {
		var values []string
		if err := db.Raw("SELECT value FROM system_settings WHERE key = ?", "oidc_issuer_url").Scan(&values).Error; err != nil {
			return err
		}
		issuer := ""
		if len(values) > 0 {
			issuer = strings.TrimSpace(values[0])
		}
		if issuer != "" {
			if err := db.Exec("UPDATE oidc_connections SET issuer = ? WHERE issuer = ''", issuer).Error; err != nil {
				return err
			}
		}
	}

	for _, statement := range []string{
		"DROP INDEX IF EXISTS idx_oidc_provider_subject",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_oidc_provider_issuer_subject ON oidc_connections (provider, issuer, subject)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
