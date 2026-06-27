package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/pkg/logging"
)

const (
	sqliteBusyPrimaryCode   = 5
	sqliteLockedPrimaryCode = 6
)

type sqliteErrorCode interface {
	Code() int
}

func writeInternalServerError(c echo.Context, err error) error {
	logger := logging.FromContext(c.Request().Context())

	if isTransientSQLiteBusyError(err) {
		logger.Warn("transient database busy error", slog.Any("error", err))
		c.Response().Header().Set("Retry-After", "1")
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "database is busy, retry later"})
	}

	if err != nil {
		logger.Error("internal server error", slog.Any("error", err))
	}

	return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
}

func isTransientSQLiteBusyError(err error) bool {
	if err == nil {
		return false
	}

	var coded sqliteErrorCode
	if errors.As(err, &coded) {
		switch coded.Code() & 0xff {
		case sqliteBusyPrimaryCode, sqliteLockedPrimaryCode:
			return true
		}
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "sqlite_busy") ||
		strings.Contains(message, "sqlite_locked") ||
		strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database table is locked")
}
