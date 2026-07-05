// Package httpx holds transport-layer helpers shared across the API handler
// packages: the JSON error envelope, localized contract metadata, request
// binding, and error classification used by the central error handler.
package httpx

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/labstack/echo/v4"
)

const (
	sqliteBusyPrimaryCode   = 5
	sqliteLockedPrimaryCode = 6
)

type sqliteErrorCode interface {
	Code() int
}

type errorResponse struct {
	ErrorCode   string         `json:"error_code,omitempty"`
	ErrorParams map[string]any `json:"error_params,omitempty"`
}

type messageResponse struct {
	MessageCode   string         `json:"message_code,omitempty"`
	MessageParams map[string]any `json:"message_params,omitempty"`
}

// WriteError renders the stable error envelope used by localized clients.
func WriteError(c echo.Context, status int, code string) error {
	return WriteErrorCode(c, status, code, nil)
}

// WriteErrorFrom renders err with its typed service metadata when available.
func WriteErrorFrom(c echo.Context, status int, err error) error {
	if err == nil {
		return WriteError(c, status, fallbackErrorCodeForStatus(status))
	}

	var typed *serviceerr.Error
	if errors.As(err, &typed) && typed != nil {
		return WriteErrorCode(c, status, typed.Code, typed.Params)
	}

	return WriteError(c, status, fallbackErrorCodeForStatus(status))
}

// WriteErrorCode renders an error envelope with an explicit stable code and
// interpolation params for localized clients.
func WriteErrorCode(c echo.Context, status int, code string, params map[string]any) error {
	return c.JSON(status, errorResponse{
		ErrorCode:   normalizeCode(code),
		ErrorParams: cloneParams(params),
	})
}

// WriteMessage renders the stable success envelope used by localized clients.
func WriteMessage(c echo.Context, status int, code string) error {
	return WriteMessageFields(c, status, code, nil)
}

// WriteMessageFields renders a success message envelope with extra response
// fields.
func WriteMessageFields(c echo.Context, status int, code string, fields map[string]any) error {
	return WriteMessageCodeFields(c, status, code, nil, fields)
}

// WriteMessageCodeFields renders a success message envelope with an explicit
// stable code, interpolation params, and extra response fields.
func WriteMessageCodeFields(
	c echo.Context,
	status int,
	code string,
	params map[string]any,
	fields map[string]any,
) error {
	payload := echo.Map{"message_code": normalizeCode(code)}
	if cloned := cloneParams(params); len(cloned) > 0 {
		payload["message_params"] = cloned
	}
	for key, value := range fields {
		payload[key] = value
	}
	return c.JSON(status, payload)
}

// BindJSON binds the request body into dst, writing a 400 with the (optional)
// stable error code and returning false on failure.
func BindJSON(c echo.Context, dst any, invalidCodes ...string) bool {
	if err := c.Bind(dst); err != nil {
		_ = WriteError(c, http.StatusBadRequest, bindJSONInvalidCode(invalidCodes...))
		return false
	}
	return true
}

// BindOptionalJSON is like BindJSON but tolerates an empty body (io.EOF).
func BindOptionalJSON(c echo.Context, dst any, invalidCodes ...string) bool {
	if err := c.Bind(dst); err != nil && !errors.Is(err, io.EOF) {
		_ = WriteError(c, http.StatusBadRequest, bindJSONInvalidCode(invalidCodes...))
		return false
	}
	return true
}

// BindLimitedJSON binds a size-limited body, distinguishing a too-large payload
// (413) from a malformed one (400).
func BindLimitedJSON(c echo.Context, dst any, tooLargeCode, invalidCode string) bool {
	if err := c.Bind(dst); err != nil {
		if IsRequestTooLargeError(err) || strings.Contains(err.Error(), "request body too large") {
			_ = WriteError(c, http.StatusRequestEntityTooLarge, tooLargeCode)
			return false
		}
		_ = WriteError(c, http.StatusBadRequest, invalidCode)
		return false
	}
	return true
}

func bindJSONInvalidCode(invalidCodes ...string) string {
	if len(invalidCodes) > 0 {
		return invalidCodes[0]
	}
	return "invalid_request_body"
}

// ParseUintParam parses a uint path parameter, writing a 400 and returning false
// on a malformed value.
func ParseUintParam(c echo.Context, name string, invalidCodes ...string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, strconv.IntSize)
	if err != nil {
		code := "invalid_" + name
		if len(invalidCodes) > 0 {
			code = invalidCodes[0]
		}
		_ = WriteError(c, http.StatusBadRequest, code)
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

func normalizeCode(code string) string {
	code = strings.TrimSpace(code)
	if code != "" {
		return code
	}
	return "message"
}

func fallbackErrorCodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "request_failed"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusTooManyRequests:
		return "too_many_requests"
	case http.StatusServiceUnavailable:
		return "service_unavailable"
	default:
		return "internal_server_error"
	}
}

func cloneParams(params map[string]any) map[string]any {
	if len(params) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}
