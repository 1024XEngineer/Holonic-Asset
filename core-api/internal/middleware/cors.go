package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

var allowedMethods = strings.Join([]string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}, ",")

// CORS temporarily allows the API to be called from any origin.
func CORS() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			request := c.Request()
			headers := c.Response().Header()
			headers.Add(echo.HeaderVary, echo.HeaderOrigin)
			if request.Header.Get(echo.HeaderOrigin) == "" {
				return next(c)
			}

			headers.Set(echo.HeaderAccessControlAllowOrigin, "*")

			if request.Method != http.MethodOptions {
				return next(c)
			}

			headers.Set(echo.HeaderAccessControlAllowMethods, allowedMethods)
			headers.Set(echo.HeaderAccessControlAllowHeaders, echo.HeaderContentType)
			return c.NoContent(http.StatusNoContent)
		}
	}
}
