// Package httpx holds transport-layer helpers shared across the API handler
// packages: the JSON error envelope, request binding, and error classification
// used by the central error handler. It has no dependency on any handler or
// service package, so every handler subpackage can import it without cycles.
package httpx

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	sqliteBusyPrimaryCode   = 5
	sqliteLockedPrimaryCode = 6
)

type sqliteErrorCode interface {
	Code() int
}

// WriteError renders the frozen error envelope: {"error": message}.
func WriteError(c echo.Context, status int, message string) error {
	return c.JSON(status, echo.Map{"error": message})
}

// BindJSON binds the request body into dst, writing a 400 with the (optional)
// custom message and returning false on failure.
func BindJSON(c echo.Context, dst any, invalidMessages ...string) bool {
	if err := c.Bind(dst); err != nil {
		_ = WriteError(c, http.StatusBadRequest, bindJSONInvalidMessage(invalidMessages...))
		return false
	}
	return true
}

// BindOptionalJSON is like BindJSON but tolerates an empty body (io.EOF).
func BindOptionalJSON(c echo.Context, dst any, invalidMessages ...string) bool {
	if err := c.Bind(dst); err != nil && !errors.Is(err, io.EOF) {
		_ = WriteError(c, http.StatusBadRequest, bindJSONInvalidMessage(invalidMessages...))
		return false
	}
	return true
}

// BindLimitedJSON binds a size-limited body, distinguishing a too-large payload
// (413) from a malformed one (400).
func BindLimitedJSON(c echo.Context, dst any, tooLargeMessage, invalidMessage string) bool {
	if err := c.Bind(dst); err != nil {
		if IsRequestTooLargeError(err) || strings.Contains(err.Error(), "request body too large") {
			_ = WriteError(c, http.StatusRequestEntityTooLarge, tooLargeMessage)
			return false
		}
		_ = WriteError(c, http.StatusBadRequest, invalidMessage)
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

// ParseUintParam parses a uint path parameter, writing a 400 and returning false
// on a malformed value.
func ParseUintParam(c echo.Context, name string, invalidMessages ...string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, strconv.IntSize)
	if err != nil {
		message := "invalid " + name
		if len(invalidMessages) > 0 {
			message = invalidMessages[0]
		}
		_ = WriteError(c, http.StatusBadRequest, message)
		return 0, false
	}
	return uint(value), true
}

// IsRequestTooLargeError reports whether err is an HTTP body-size-limit error.
func IsRequestTooLargeError(err error) bool {
	if err == nil {
		return false
	}

	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "request body too large")
}

// IsTransientSQLiteBusyError reports whether err is a retryable SQLite
// busy/locked condition, which the caller maps to 503 with a Retry-After hint.
func IsTransientSQLiteBusyError(err error) bool {
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
