package router

import (
	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
)

type AssetRouter interface {
	GetAssets(x *echox.Context, req dto.GetAssetsRequest) (dto.Response, error)
	Detail(x *echox.Context) (dto.Response, error)
	Record(x *echox.Context, req dto.RecordAssetRequest) (dto.Response, error)
	Records(x *echox.Context) (dto.Response, error)
	CopyAsset(ctx *echox.Context, req dto.CopyAssetRequest) (dto.Response, error)
	RollBackAsset(ctx *echox.Context, req dto.RollBackAssetRequest) (dto.Response, error)

	UpdateAsset(ctx *echox.Context, req dto.UpdateAssetRequest) (dto.Response, error)
	Delete(ctx *echox.Context, req dto.DeleteAssetRequest) (dto.Response, error)
}

// RegisterRoutes registers all HTTP routes.
func RegisterAssetRoutes(e *echo.Group, r AssetRouter) {
	project := e.Group("/projects")

	project.GET("/:project_id/assets", echox.WrapReq(r.GetAssets))

	asset := e.Group("/asset")

	asset.GET("/:asset_id/records", echox.Wrap(r.Records))

	asset.GET("/:asset_id", echox.Wrap(r.Detail))

	asset.POST("/save", echox.WrapReq(r.Record))

	asset.POST("/copy", echox.WrapReq(r.CopyAsset))

	asset.POST("/rollback", echox.WrapReq(r.RollBackAsset))

	asset.POST("/update", echox.WrapReq(r.UpdateAsset))

	asset.DELETE("/delete", echox.WrapReq(r.Delete))
}
