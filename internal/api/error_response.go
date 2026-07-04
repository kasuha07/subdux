package api

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/kasuha07/subdux/internal/pkg/logging"
	"github.com/labstack/echo/v4"
)

const (
	sqliteBusyPrimaryCode   = 5
	sqliteLockedPrimaryCode = 6
)

type sqliteErrorCode interface {
	Code() int
}

func bindJSON(c echo.Context, dst any, invalidMessages ...string) bool {
	if err := c.Bind(dst); err != nil {
		_ = writeError(c, http.StatusBadRequest, bindJSONInvalidMessage(invalidMessages...))
		return false
	}
	return true
}

func bindOptionalJSON(c echo.Context, dst any, invalidMessages ...string) bool {
	if err := c.Bind(dst); err != nil && !errors.Is(err, io.EOF) {
		_ = writeError(c, http.StatusBadRequest, bindJSONInvalidMessage(invalidMessages...))
		return false
	}
	return true
}

func bindLimitedJSON(c echo.Context, dst any, tooLargeMessage, invalidMessage string) bool {
	if err := c.Bind(dst); err != nil {
		if isRequestTooLargeError(err) || strings.Contains(err.Error(), "request body too large") {
			_ = writeError(c, http.StatusRequestEntityTooLarge, tooLargeMessage)
			return false
		}
		_ = writeError(c, http.StatusBadRequest, invalidMessage)
		return false
	}
	return true
}

func bindJSONInvalidMessage(invalidMessages ...string) string {
	if len(invalidMessages) > 0 {
		return invalidMessages[0]
	}
	return "invalid request body"
}

func parseUintParam(c echo.Context, name string, invalidMessages ...string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, strconv.IntSize)
	if err != nil {
		message := "invalid " + name
		if len(invalidMessages) > 0 {
			message = invalidMessages[0]
		}
		_ = writeError(c, http.StatusBadRequest, message)
		return 0, false
	}
	return uint(value), true
}

func writeError(c echo.Context, status int, message string) error {
	return c.JSON(status, echo.Map{"error": message})
}

func writeServiceError(c echo.Context, err error, cases ...serviceErrorCase) error {
	for _, serviceCase := range cases {
		if serviceCase.matches(err) {
			return writeError(c, serviceCase.status, err.Error())
		}
	}
	return writeInternalServerError(c, err)
}

type serviceErrorCase struct {
	status int
	match  func(error) bool
}

func serviceError(status int, targets ...error) serviceErrorCase {
	return serviceErrorCase{
		status: status,
		match: func(err error) bool {
			for _, target := range targets {
				if errors.Is(err, target) {
					return true
				}
			}
			return false
		},
	}
}

func serviceErrorMessage(status int, messages ...string) serviceErrorCase {
	return serviceErrorCase{
		status: status,
		match: func(err error) bool {
			if err == nil {
				return false
			}
			for _, message := range messages {
				if err.Error() == message {
					return true
				}
			}
			return false
		},
	}
}

func serviceErrorFunc(status int, match func(error) bool) serviceErrorCase {
	return serviceErrorCase{status: status, match: match}
}

func (c serviceErrorCase) matches(err error) bool {
	return c.match != nil && c.match(err)
}

func writeInternalServerError(c echo.Context, err error) error {
	logger := logging.FromContext(c.Request().Context())

	if isTransientSQLiteBusyError(err) {
		logger.Warn("transient database busy error", slog.Any("error", err))
		c.Response().Header().Set("Retry-After", "1")
		return writeError(c, http.StatusServiceUnavailable, "database is busy, retry later")
	}

	if err != nil {
		logger.Error("internal server error", slog.Any("error", err))
	}

	return writeError(c, http.StatusInternalServerError, "internal server error")
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
