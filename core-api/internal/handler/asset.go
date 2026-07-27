package handler

import (
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

func (h *Handler) GetAssets(x *echox.Context) ([]dto.GetAssetsResponse, error) {
	_, err := h.AssetService.GetAssets(x, 1)
	return nil, err
}

func (h *Handler) Detail(x *echox.Context) (dto.AssetDetailResponse, error) {
	_, err := h.AssetService.GetDetail(x, 0)
	return dto.AssetDetailResponse{}, err
}

func (h *Handler) GetProtoTypeResource(x *echox.Context) (dto.GetAssetResourcesResponse, error) {
	_, err := h.AssetResourceService.GetProtoTypeResource(x, 0, 0)
	return dto.GetAssetResourcesResponse{}, err
}

func (h *Handler) GetAnimations(x *echox.Context) (dto.GetAssetResourcesResponse, error) {
	_, err := h.AssetResourceService.GetAnimations(x, 0, 0)
	return dto.GetAssetResourcesResponse{}, err
}

func (h *Handler) GetItemResources(x *echox.Context) (dto.GetAssetResourcesResponse, error) {
	_, err := h.AssetResourceService.GetItemResources(x, 0, 0)
	return dto.GetAssetResourcesResponse{}, err
}

func (h *Handler) Record(x *echox.Context, asset dto.RecordAssetRequest) ([]dto.RecordAssetResponse, error) {
	_, err := h.AssetVerionService.CreateRecord(x, &domain.AssetVersion{AssetID: asset.AssetID})
	return []dto.RecordAssetResponse{}, err
}

func (h *Handler) CreateCharacterAsset(ctx *echox.Context, asset dto.CreateCharacterAssetRequest) (dto.CreateCharacterAssetResponse, error) {
	_, err := h.AssetService.CreateCharacterAsset(ctx, asset.Asset)
	return dto.CreateCharacterAssetResponse{}, err
}

func (h *Handler) CreateObjectAsset(ctx *echox.Context, asset dto.CreateObjectAssetRequest) (dto.CreateObjectAssetResponse, error) {
	_, err := h.AssetService.CreateObjectAsset(ctx, asset.Asset)
	return dto.CreateObjectAssetResponse{}, err
}

func (h *Handler) CreateTileSetAsset(ctx *echox.Context, asset dto.CreateTileSetAssetRequest) (dto.CreateTileSetAssetResponse, error) {
	_, err := h.AssetService.CreateTileSetAsset(ctx, asset.Asset)
	return dto.CreateTileSetAssetResponse{}, err
}

func (h *Handler) CopyAsset(ctx *echox.Context, asset dto.CopyAssetRequest) (dto.CopyAssetResponse, error) {
	_, err := h.AssetVerionService.Copy(ctx, asset.AssetID, 0)
	return dto.CopyAssetResponse{}, err
}

func (h *Handler) RollBackAsset(ctx *echox.Context, asset dto.RollBackAssetRequest) (dto.RollBackAssetResponse, error) {
	_, err := h.AssetVerionService.RollBackVersion(ctx, asset.AssetID, 0)
	return dto.RollBackAssetResponse{}, err
}

func (h *Handler) Tags(ctx *echox.Context, req dto.AddTagsRequest) (dto.AddTagsResponse, error) {
	_, err := h.AssetService.UpdateTags(ctx, req.AssetID, req.Tags)
	return dto.AddTagsResponse{}, err
}

func (h *Handler) CreateAnimation(ctx *echox.Context, req dto.CreateAnimationRequest) (dto.CreateAnimationResponse, error) {
	_, err := h.AssetResourceService.CreateAnimationResource(ctx, &domain.AssetResource{AssetID: req.AssetID, Name: req.Name, Type: domain.AssetResourceType(req.Type)})
	return dto.CreateAnimationResponse{}, err
}
