package router

import (
	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type TaxonomyRouter interface {
	SearchAssets(
		c *echox.Context,
		request dto.SearchAssetsRequest,
	) (*dto.AssetSearchResult, error)
}

// RegisterRoutes registers the public asset search route.
func RegisterTaxonomyRoutes(e *echo.Group, r TaxonomyRouter) {
	assets := e.Group("/projects/:projectId/assets")
	assets.GET("/search", echox.WrapReq(r.SearchAssets))
}
