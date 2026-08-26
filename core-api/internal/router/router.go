package router

import (
	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/middleware"
)

type Authentication struct {
	Router         AuthRouter
	Middleware     echo.MiddlewareFunc
	AllowedOrigins []string
}

// Register assembles and returns all routes.
func Register(
	as AssetRouter,
	pr ProjectRouter,
	gr GenerationRouter,
	ur UploadRouter,
	tr ProjectTagRouter,
	authentication ...Authentication,
) *echo.Echo {
	return register(as, pr, gr, ur, tr, nil, authentication...)
}

// RegisterWithExport assembles the API including the asynchronous export routes.
func RegisterWithExport(
	as AssetRouter,
	pr ProjectRouter,
	gr GenerationRouter,
	ur UploadRouter,
	tr ProjectTagRouter,
	er ExportRouter,
	authentication ...Authentication,
) *echo.Echo {
	return register(as, pr, gr, ur, tr, er, authentication...)
}

func register(
	as AssetRouter,
	pr ProjectRouter,
	gr GenerationRouter,
	ur UploadRouter,
	tr ProjectTagRouter,
	er ExportRouter,
	authentication ...Authentication,
) *echo.Echo {
	e := echo.New()
	api := e.Group(apiBasePath)
	secured := len(authentication) > 0 && authentication[0].Middleware != nil
	allowedOrigins := []string{"*"}
	if len(authentication) > 0 && len(authentication[0].AllowedOrigins) > 0 {
		allowedOrigins = authentication[0].AllowedOrigins
	}
	e.Use(middleware.CORS(allowedOrigins...))
	if secured {
		api.Use(skipPath(authentication[0].Middleware, apiBasePath+authLoginPath))
	}
	openAPI := newOpenAPI(e, api, secured)
	if len(authentication) > 0 && authentication[0].Router != nil {
		RegisterAuthRoutes(openAPI, authentication[0].Router)
	}
	if as != nil {
		RegisterAssetRoutes(openAPI, as)
	}
	if tr != nil {
		RegisterProjectTagRoutes(openAPI, tr)
	}
	if pr != nil {
		RegisterProjectRoutes(openAPI, pr)
	}
	if gr != nil {
		RegisterGenerationRoutes(openAPI, gr)
	}
	if ur != nil {
		RegisterUploadRoutes(openAPI, ur)
	}
	if er != nil {
		RegisterExportRoutes(openAPI, er)
	}
	return e
}

func skipPath(mw echo.MiddlewareFunc, path string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		secured := mw(next)
		return func(c echo.Context) error {
			if c.Request().URL.Path == path {
				return next(c)
			}
			return secured(c)
		}
	}
}
