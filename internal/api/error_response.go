package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func writeInternalServerError(c echo.Context, err error) error {
	if err != nil {
		c.Logger().Error(err)
	}

	return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
}
