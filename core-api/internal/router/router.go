package router

import "github.com/labstack/echo/v4"

// Register assembles and returns all routes.
func Register(
	as AssetRouter,
	pr ProjectRouter,
	gr GenerationRouter,
	ur UploadRouter,
) *echo.Echo {
	e := echo.New()
	api := e.Group(apiBasePath)
	openAPI := newOpenAPI(e, api)
	if as != nil {
		RegisterAssetRoutes(openAPI, as)
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
	return e
}
