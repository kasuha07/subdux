package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// PostgreSQL uses an expression index for email lookup and a trigger for the
// cross-column rule, which cannot be expressed by a unique index on users.
// Existing cross-column collisions remain untouched; only new claims are
// rejected. SQLite keeps the service-level checks.
func migratePostgresLoginIdentifiers(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}

	var duplicate struct {
		Email string
	}
	result := db.Raw(`SELECT lower(email) AS email FROM users GROUP BY lower(email) HAVING count(*) > 1 LIMIT 1`).Scan(&duplicate)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return fmt.Errorf("cannot create case-insensitive users email index: existing users share an email after case folding; resolve the duplicate accounts manually")
	}

	for _, statement := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower_unique ON users (lower(email))`,
		`CREATE INDEX IF NOT EXISTS idx_users_username_lower ON users (lower(username))`,
		`CREATE FUNCTION subdux_check_user_login_identifiers() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    email_key text := lower(NEW.email);
    username_key text := lower(NEW.username);
    check_email boolean := TG_OP = 'INSERT';
    check_username boolean := TG_OP = 'INSERT';
    first_key text;
    second_key text;
BEGIN
    IF TG_OP = 'UPDATE' THEN
        check_email := lower(NEW.email) IS DISTINCT FROM lower(OLD.email);
        check_username := lower(NEW.username) IS DISTINCT FROM lower(OLD.username);
    END IF;
    IF NOT check_email AND NOT check_username THEN
        RETURN NEW;
    END IF;

    -- Every writer claims the same two identifier keys in the same order.
    -- Hash collisions only serialize unrelated writers; they cannot admit a
    -- conflicting pair. The second writer sees the first committed row.
    first_key := least(email_key, username_key);
    second_key := greatest(email_key, username_key);
    PERFORM pg_advisory_xact_lock(1789378244, hashtext(first_key));
    IF second_key <> first_key THEN
        PERFORM pg_advisory_xact_lock(1789378244, hashtext(second_key));
    END IF;

    IF check_email AND EXISTS (
        SELECT 1 FROM users WHERE id <> NEW.id AND lower(username) = email_key
    ) THEN
        RAISE EXCEPTION 'email conflicts with another username'
            USING ERRCODE = '23505', CONSTRAINT = 'users_email_username_collision';
    END IF;
    IF check_username AND EXISTS (
        SELECT 1 FROM users WHERE id <> NEW.id AND lower(email) = username_key
    ) THEN
        RAISE EXCEPTION 'username conflicts with another email'
            USING ERRCODE = '23505', CONSTRAINT = 'users_username_email_collision';
    END IF;
    RETURN NEW;
END
$$`,
		`CREATE TRIGGER trg_users_login_identifiers
            BEFORE INSERT OR UPDATE OF email, username ON users
            FOR EACH ROW EXECUTE FUNCTION subdux_check_user_login_identifiers()`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
