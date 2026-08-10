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
			if request.Header.Get(echo.HeaderOrigin) == "" {
				return next(c)
			}

			headers := c.Response().Header()
			headers.Set(echo.HeaderAccessControlAllowOrigin, "*")
			headers.Set(echo.HeaderAccessControlExposeHeaders, "*")

			if request.Method != http.MethodOptions {
				return next(c)
			}

			headers.Set(echo.HeaderAccessControlAllowMethods, allowedMethods)
			if requestedHeaders := request.Header.Get(echo.HeaderAccessControlRequestHeaders); requestedHeaders != "" {
				headers.Set(echo.HeaderAccessControlAllowHeaders, requestedHeaders)
			}
			return c.NoContent(http.StatusNoContent)
		}
	}
}
