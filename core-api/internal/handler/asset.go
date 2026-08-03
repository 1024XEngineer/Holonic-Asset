package handler

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type Handler struct {
	AssetManager domain.Manager
}

func NewHandler(manager domain.Manager) *Handler {
	return &Handler{AssetManager: manager}
}

func (h *Handler) GetAssets(
	x context.Context,
	request dto.GetAssetsRequest,
) (dto.SuccessResponse[dto.GetAssetsResponse], error) {
	projectID := request.ProjectID
	if projectID == 0 {
		return dto.SuccessResponse[dto.GetAssetsResponse]{}, echo.ErrBadRequest
	}

	assets, err := h.AssetManager.GetAssets(x, projectID, domain.AssetListFilter{
		Query: request.Query,
		Tags:  request.Tags,
		Types: request.Types,
	})
	if err != nil {
		return dto.SuccessResponse[dto.GetAssetsResponse]{}, err
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

	return dto.NewTypedSuccessResponse(dto.GetAssetsResponse{Assets: items}), nil
}

func (h *Handler) Detail(
	x context.Context,
	request dto.AssetDetailRequest,
) (dto.SuccessResponse[dto.AssetDetailResponse], error) {
	assetID := request.AssetID
	if assetID == 0 {
		return dto.SuccessResponse[dto.AssetDetailResponse]{}, echo.ErrBadRequest
	}

	asset, err := h.AssetManager.GetDetail(x, assetID)
	if err != nil {
		return dto.SuccessResponse[dto.AssetDetailResponse]{}, err
	}

	return dto.NewTypedSuccessResponse(dto.AssetDetailResponse{
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

func (h *Handler) Record(
	x context.Context,
	asset dto.RecordAssetRequest,
) (dto.SuccessResponse[dto.RecordAssetResponse], error) {
	if asset.AssetID == 0 {
		return dto.SuccessResponse[dto.RecordAssetResponse]{}, echo.ErrBadRequest
	}
	record, err := h.AssetManager.CreateRecord(x, &domain.AssetRecord{AssetID: asset.AssetID})
	if err != nil {
		return dto.SuccessResponse[dto.RecordAssetResponse]{}, err
	}
	return dto.NewTypedSuccessResponse(dto.RecordAssetResponse{
		RecordID:  record.ID,
		AssetID:   record.AssetID,
		Version:   record.Version,
		ContentID: record.ContentID,
		CreatedAt: record.CreatedAt,
	}), nil
}

func (h *Handler) Records(
	x context.Context,
	request dto.GetAssetRecordsRequest,
) (dto.SuccessResponse[dto.GetAssetRecordsResponse], error) {
	assetID := request.AssetID
	if assetID == 0 {
		return dto.SuccessResponse[dto.GetAssetRecordsResponse]{}, echo.ErrBadRequest
	}
	records, err := h.AssetManager.GetRecordHistory(x, assetID)
	if err != nil {
		return dto.SuccessResponse[dto.GetAssetRecordsResponse]{}, err
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
	return dto.NewTypedSuccessResponse(dto.GetAssetRecordsResponse{Records: items}), nil
}

func (h *Handler) CopyAsset(
	ctx context.Context,
	asset dto.CopyAssetRequest,
) (dto.SuccessResponse[dto.CopyAssetResponse], error) {
	if asset.AssetID == 0 {
		return dto.SuccessResponse[dto.CopyAssetResponse]{}, echo.ErrBadRequest
	}
	newAssetID, err := h.AssetManager.Copy(ctx, asset.AssetID, 0)
	if err != nil {
		return dto.SuccessResponse[dto.CopyAssetResponse]{}, err
	}
	return dto.NewTypedSuccessResponse(dto.CopyAssetResponse{NewAssetID: newAssetID}), nil
}

func (h *Handler) RollBackAsset(
	ctx context.Context,
	asset dto.RollBackAssetRequest,
) (dto.SuccessResponse[dto.RollBackAssetResponse], error) {
	if asset.AssetID == 0 || asset.Version == 0 {
		return dto.SuccessResponse[dto.RollBackAssetResponse]{}, echo.ErrBadRequest
	}
	record, err := h.AssetManager.RollBackRecord(ctx, asset.AssetID, asset.Version)
	if err != nil {
		return dto.SuccessResponse[dto.RollBackAssetResponse]{}, err
	}
	return dto.NewTypedSuccessResponse(dto.RollBackAssetResponse{
		AssetID:   record.AssetID,
		Version:   record.Version,
		ContentID: record.ContentID,
	}), nil
}

func (h *Handler) UpdateAsset(
	ctx context.Context,
	req dto.UpdateAssetRequest,
) (dto.SuccessResponse[dto.UpdateAssetResponse], error) {
	if req.AssetID == 0 {
		return dto.SuccessResponse[dto.UpdateAssetResponse]{}, echo.ErrBadRequest
	}
	asset, err := h.AssetManager.UpdateAsset(ctx, req.AssetID, &domain.AssetUpdate{
		Name:        req.Name,
		ProjectID:   req.ProjectID,
		Type:        req.Type,
		Description: req.Description,
		Tags:        req.Tags,
		Attributes:  req.Attributes,
	})
	if err != nil {
		return dto.SuccessResponse[dto.UpdateAssetResponse]{}, err
	}
	return dto.NewTypedSuccessResponse(dto.UpdateAssetResponse{
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
