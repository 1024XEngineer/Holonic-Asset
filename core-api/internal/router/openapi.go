package router

import (
	"errors"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"
)

const apiBasePath = "/api/v1"

func newOpenAPI(e *echo.Echo, group *echo.Group) huma.API {
	config := huma.DefaultConfig("Holonic Asset Core API", "0.1.0")
	config.Servers = []*huma.Server{{URL: apiBasePath}}

	// Keep existing response bodies stable while Huma is introduced
	// incrementally. The default schema-link transformer adds a $schema field.
	config.CreateHooks = nil

	return humaecho.NewV4WithGroup(e, group, config)
}

func openAPIError(err error) error {
	if err == nil {
		return nil
	}

	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		return huma.NewError(httpErr.Code, fmt.Sprint(httpErr.Message))
	}
	return err
}
