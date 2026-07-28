package router

import "github.com/labstack/echo/v4"

// Register assembles and returns all routes.
func Register(
	as AssetRouter,
	pr ProjectRouter,
	gr GenerationRouter,
	ur UploadRouter,
	tr TaxonomyRouter,
) *echo.Echo {
	e := echo.New()
	api := e.Group("/api/v1")
	if as != nil {
		RegisterAssetRoutes(api, as)
	}
	if pr != nil {
		RegisterProjectRoutes(api, pr)
	}
	if gr != nil {
		RegisterGenerationRoutes(api, gr)
	}
	if ur != nil {
		RegisterUploadRoutes(api, ur)
	}
	if tr != nil {
		RegisterTaxonomyRoutes(api, tr)
	}
	return e
}
