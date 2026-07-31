package handler

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type Handler struct {
	AssetManager domain.Manager
}

func NewHandler(manager domain.Manager) *Handler {
	return &Handler{AssetManager: manager}
}

func (h *Handler) GetAssets(x *echox.Context, request dto.GetAssetsRequest) (dto.Response, error) {
	projectID, err := parseAssetPathID(x, "project_id")
	if err != nil {
		return dto.Response{}, err
	}

	assets, err := h.AssetManager.GetAssets(x, projectID, domain.AssetListFilter{
		Query: request.Query,
		Tags:  request.Tags,
		Types: request.Types,
	})
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

	asset, err := h.AssetManager.GetDetail(x, assetID)
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
		Content:     asset.Content,
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

func (h *Handler) Record(x *echox.Context, asset dto.RecordAssetRequest) (dto.Response, error) {
	if asset.AssetID == 0 {
		return dto.Response{}, echo.ErrBadRequest
	}
	record, err := h.AssetManager.CreateRecord(x, &domain.AssetRecord{AssetID: asset.AssetID})
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.RecordAssetResponse{
		RecordID:  record.ID,
		AssetID:   record.AssetID,
		Version:   record.Version,
		ContentID: record.ContentID,
		CreatedAt: record.CreatedAt,
	}), nil
}

func (h *Handler) Records(x *echox.Context) (dto.Response, error) {
	assetID, err := parseAssetPathID(x, "asset_id")
	if err != nil {
		return dto.Response{}, err
	}
	records, err := h.AssetManager.GetRecordHistory(x, assetID)
	if err != nil {
		return dto.Response{}, err
	}
	items := make([]dto.AssetRecordResponse, len(records))
	for index, record := range records {
		items[index] = dto.AssetRecordResponse{
			RecordID:  record.ID,
			AssetID:   record.AssetID,
			Version:   record.Version,
			ContentID: record.ContentID,
			CreatedAt: record.CreatedAt,
			Content:   record.Content,
		}
	}
	return dto.NewSuccessResponse(dto.GetAssetRecordsResponse{Records: items}), nil
}

func (h *Handler) CopyAsset(ctx *echox.Context, asset dto.CopyAssetRequest) (dto.Response, error) {
	if asset.AssetID == 0 {
		return dto.Response{}, echo.ErrBadRequest
	}
	newAssetID, err := h.AssetManager.Copy(ctx, asset.AssetID, 0)
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.CopyAssetResponse{NewAssetID: newAssetID}), nil
}

func (h *Handler) RollBackAsset(ctx *echox.Context, asset dto.RollBackAssetRequest) (dto.Response, error) {
	if asset.AssetID == 0 || asset.Version == 0 {
		return dto.Response{}, echo.ErrBadRequest
	}
	record, err := h.AssetManager.RollBackRecord(ctx, asset.AssetID, asset.Version)
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.RollBackAssetResponse{
		AssetID:   record.AssetID,
		Version:   record.Version,
		ContentID: record.ContentID,
	}), nil
}

func (h *Handler) Delete(ctx *echox.Context, req dto.DeleteAssetRequest) (dto.Response, error) {
	if req.AssetID == 0 {
		return dto.Response{}, echo.ErrBadRequest
	}
	if err := h.AssetManager.Delete(ctx, req.AssetID); err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(struct{}{}), nil
}

func (h *Handler) UpdateAsset(ctx *echox.Context, req dto.UpdateAssetRequest) (dto.Response, error) {
	asset, err := h.AssetManager.UpdateAsset(ctx, req.AssetID, &domain.AssetUpdate{
		Name:        req.Name,
		ProjectID:   req.ProjectID,
		Type:        req.Type,
		Description: req.Description,
		Tags:        req.Tags,
		Attributes:  req.Attributes,
	})
	if err != nil {
		return dto.Response{}, err
	}
	return dto.NewSuccessResponse(dto.UpdateAssetResponse{
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
