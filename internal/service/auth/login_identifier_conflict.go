package auth

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// LoginIdentifierConflictField identifies the user field that lost a database
// uniqueness race. Preflight checks give friendlier errors in the usual case;
// PostgreSQL constraints cover writers that pass those checks concurrently.
func LoginIdentifierConflictField(err error) string {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return ""
	}
	switch pgErr.ConstraintName {
	case "idx_users_email_lower_unique", "idx_users_email", "users_email_username_collision":
		return "email"
	case "idx_users_username", "users_username_email_collision":
		return "username"
	default:
		return ""
	}
}

func mapLoginIdentifierWriteError(err error) error {
	switch LoginIdentifierConflictField(err) {
	case "email":
		return ErrEmailAlreadyRegistered
	case "username":
		return ErrUsernameAlreadyTaken
	default:
		return err
	}
}
