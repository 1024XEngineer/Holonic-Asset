package router

import (
	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type AssetRouter interface {
	GetAssets(x *echox.Context) (dto.Response, error)
	Detail(x *echox.Context) (dto.Response, error)
	GetProtoTypeResource(x *echox.Context) (dto.Response, error)
	GetAnimations(x *echox.Context) (dto.Response, error)
	GetItemResources(x *echox.Context) (dto.Response, error)
	Record(x *echox.Context, req dto.RecordAssetRequest) (dto.Response, error)
	CreateCharacterAsset(ctx *echox.Context, req dto.CreateCharacterAssetRequest) (dto.Response, error)
	CreateObjectAsset(ctx *echox.Context, req dto.CreateObjectAssetRequest) (dto.Response, error)
	CreateTileSetAsset(ctx *echox.Context, req dto.CreateTileSetAssetRequest) (dto.Response, error)
	CreateAnimation(ctx *echox.Context, req dto.CreateAnimationRequest) (dto.Response, error)
	CopyAsset(ctx *echox.Context, req dto.CopyAssetRequest) (dto.Response, error)
	RollBackAsset(ctx *echox.Context, req dto.RollBackAssetRequest) (dto.Response, error)

	Tags(ctx *echox.Context, req dto.AddTagsRequest) (dto.Response, error)
}

// RegisterRoutes registers all HTTP routes.
func RegisterAssetRoutes(e *echo.Group, r AssetRouter) {
	project := e.Group("/projects")

	project.GET("/:project_id/assets", echox.Wrap(r.GetAssets))

	asset := e.Group("/asset")

	asset.GET("/:asset_id", echox.Wrap(r.Detail))

	asset.GET("/:asset_id/prototype", echox.Wrap(r.GetProtoTypeResource))

	asset.GET("/:asset_id/animations", echox.Wrap(r.GetAnimations))

	asset.GET("/:asset_id/items", echox.Wrap(r.GetItemResources))

	asset.POST("/save", echox.WrapReq(r.Record))

	asset.POST("/characters", echox.WrapReq(r.CreateCharacterAsset))

	asset.POST("/objects", echox.WrapReq(r.CreateObjectAsset))

	asset.POST("/tilesets", echox.WrapReq(r.CreateTileSetAsset))

	asset.POST("/animations", echox.WrapReq(r.CreateAnimation))

	asset.POST("/copy", echox.WrapReq(r.CopyAsset))

	asset.POST("/rollback", echox.WrapReq(r.RollBackAsset))

	asset.POST("/tags", echox.WrapReq(r.Tags))
}
