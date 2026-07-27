package internal

import (
	"github.com/labstack/echo/v4"

	asset "github.com/1024XEngineer/Holonic-Asset/internal/asset/router"
	generation "github.com/1024XEngineer/Holonic-Asset/internal/generation/router"
	media "github.com/1024XEngineer/Holonic-Asset/internal/media/router"
	project "github.com/1024XEngineer/Holonic-Asset/internal/project/router"
	taxonomy "github.com/1024XEngineer/Holonic-Asset/internal/taxonomy/router"
)

// Register assembles and returns all routes.
func Register(
	as asset.AssetRouter,
	pr project.ProjectRouter,
	gr generation.GenerationRouter,
	mr media.MediaRouter,
	tr taxonomy.TaxonomyRouter,
) *echo.Echo {
	e := echo.New()
	api := e.Group("/api/v1")
	if as != nil {
		asset.RegisterRoutes(api, as)
	}
	if pr != nil {
		project.RegisterRoutes(api, pr)
	}
	if gr != nil {
		generation.RegisterRoutes(api, gr)
	}
	if mr != nil {
		media.RegisterRoutes(api, mr)
	}
	if tr != nil {
		taxonomy.RegisterRoutes(api, tr)
	}

	return e
}
