package handler

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type Handler struct {
	AssetService         service.AssetService
	AssetResourceService service.AssetResourceService
	AssetVerionService   service.AssetVersionService
}

func NewHandler(as service.AssetService, rs service.AssetResourceService, vs service.AssetVersionService) *Handler {
	return &Handler{
		AssetService:         as,
		AssetResourceService: rs,
		AssetVerionService:   vs,
	}
}

func (h *Handler) GetAssets(x *echox.Context) (dto.Response, error) {
	projectID, err := parseAssetPathID(x, "project_id")
	if err != nil {
		return dto.Response{}, err
	}

	assets, err := h.AssetService.GetAssets(x, projectID)
	if err != nil {
		return dto.Response{}, err
	}

	items := make([]dto.AssetListItemResponse, len(assets))
	for index, asset := range assets {
		items[index] = dto.AssetListItemResponse{
			AssetID:     asset.ID,
			Name:        asset.Name,
			ProjectID:   asset.ProjectID,
			Type:        asset.Type,
			Description: asset.Description,
			Tags:        asset.Tags,
			Version:     asset.Version,
		}
	}

	return dto.NewSuccessResponse(dto.GetAssetsResponse{Assets: items}), nil
}

func (h *Handler) Detail(x *echox.Context) (dto.Response, error) {
	assetID, err := parseAssetPathID(x, "asset_id")
	if err != nil {
		return dto.Response{}, err
	}

	asset, err := h.AssetService.GetDetail(x, assetID)
	if err != nil {
		return dto.Response{}, err
	}

	return dto.NewSuccessResponse(dto.AssetDetailResponse{
		AssetID:     asset.ID,
		Name:        asset.Name,
		ProjectID:   asset.ProjectID,
		Type:        asset.Type,
		Description: asset.Description,
		Tags:        asset.Tags,
		Attributes:  asset.Attributes,
		Version:     asset.Version,
	}), nil
}

func parseAssetPathID(x *echox.Context, name string) (uint, error) {
	value, err := strconv.ParseUint(x.Param(name), 10, strconv.IntSize)
	if err != nil || value == 0 {
		return 0, echo.ErrBadRequest
	}
	return uint(value), nil
}

func (h *Handler) GetProtoTypeResource(x *echox.Context) (dto.Response, error) {
	resources, err := h.AssetResourceService.GetProtoTypeResource(x, 0, 0)
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.GetAssetResourcesResponse{Resources: resources}), nil
}

func (h *Handler) GetAnimations(x *echox.Context) (dto.Response, error) {
	resources, err := h.AssetResourceService.GetAnimations(x, 0, 0)
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.GetAssetResourcesResponse{Resources: resources}), nil
}

func (h *Handler) GetItemResources(x *echox.Context) (dto.Response, error) {
	resources, err := h.AssetResourceService.GetItemResources(x, 0, 0)
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.GetAssetResourcesResponse{Resources: resources}), nil
}

func (h *Handler) Record(x *echox.Context, asset dto.RecordAssetRequest) (dto.Response, error) {
	_, err := h.AssetVerionService.CreateRecord(x, &domain.AssetVersion{AssetID: asset.AssetID})
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse([]dto.RecordAssetResponse{}), nil
}

func (h *Handler) CreateCharacterAsset(ctx *echox.Context, asset dto.CreateCharacterAssetRequest) (dto.Response, error) {
	id, err := h.AssetService.CreateCharacterAsset(ctx, asset.Asset)
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.CreateCharacterAssetResponse{ID: id}), nil
}

func (h *Handler) CreateObjectAsset(ctx *echox.Context, asset dto.CreateObjectAssetRequest) (dto.Response, error) {
	id, err := h.AssetService.CreateObjectAsset(ctx, asset.Asset)
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.CreateObjectAssetResponse{ID: id}), nil
}

func (h *Handler) CreateTileSetAsset(ctx *echox.Context, asset dto.CreateTileSetAssetRequest) (dto.Response, error) {
	id, err := h.AssetService.CreateTileSetAsset(ctx, asset.Asset)
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.CreateTileSetAssetResponse{ID: id}), nil
}

func (h *Handler) CopyAsset(ctx *echox.Context, asset dto.CopyAssetRequest) (dto.Response, error) {
	newAssetID, err := h.AssetVerionService.Copy(ctx, asset.AssetID, 0)
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.CopyAssetResponse{NewAssetID: newAssetID}), nil
}

func (h *Handler) RollBackAsset(ctx *echox.Context, asset dto.RollBackAssetRequest) (dto.Response, error) {
	_, err := h.AssetVerionService.RollBackVersion(ctx, asset.AssetID, 0)
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.RollBackAssetResponse{}), nil
}

func (h *Handler) Tags(ctx *echox.Context, req dto.AddTagsRequest) (dto.Response, error) {
	tags, err := h.AssetService.UpdateTags(ctx, req.AssetID, req.Tags)
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.AddTagsResponse{Tags: tags}), nil
}

func (h *Handler) CreateAnimation(ctx *echox.Context, req dto.CreateAnimationRequest) (dto.Response, error) {
	id, err := h.AssetResourceService.CreateAnimationResource(ctx, &domain.AssetResource{AssetID: req.AssetID, Name: req.Name, Type: domain.AssetResourceType(req.Type)})
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.CreateAnimationResponse{ID: id}), nil
}
