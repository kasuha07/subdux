package api

import (
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"github.com/labstack/echo/v4"
	"strconv"
)

func parseRevisionQuery(c echo.Context) (uint64, error) {
	raw := c.QueryParam("revision")
	if raw == "" {
		return 0, serviceutil.ErrRevisionRequired
	}
	revision, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || revision == 0 {
		return 0, serviceutil.ErrRevisionRequired
	}
	return revision, nil
}
