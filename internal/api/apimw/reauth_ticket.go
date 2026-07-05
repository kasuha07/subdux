package apimw

import (
	"net/http"
	"strings"

	"github.com/kasuha07/subdux/internal/api/httpx"
	"github.com/labstack/echo/v4"
)

// ReauthTicketHeader carries the single-use step-up ("reauth") ticket. It lives
// in a header (rather than a request body) so sensitive endpoints can consume it
// before parsing/buffering the body — e.g. the restore upload is gated behind a
// proven-present admin before any multipart data is read.
const ReauthTicketHeader = "X-Reauth-Ticket"

// ReauthTicketFromRequest extracts the trimmed step-up ticket from the request.
func ReauthTicketFromRequest(c echo.Context) string {
	return strings.TrimSpace(c.Request().Header.Get(ReauthTicketHeader))
}

// WriteReauthError maps a step-up failure to HTTP 400 with the error message. It
// never returns 401 (so a failed re-auth does not trip the frontend's
// token-refresh/logout flow) and never 500 (the client-correctable step-up stays
// a bad-request from the caller's perspective). This all-400 policy is specific
// to the reauth flow, which is why these do not pass through the central
// Kind-based error handler.
func WriteReauthError(c echo.Context, err error) error {
	return httpx.WriteErrorFrom(c, http.StatusBadRequest, err)
}
