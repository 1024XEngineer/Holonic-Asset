package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	exportmodule "github.com/1024XEngineer/Holonic-Asset/internal/module/export"
)

type ExportHandler struct{ service *exportmodule.Service }

func NewExportHandler(service *exportmodule.Service) *ExportHandler {
	return &ExportHandler{service: service}
}

func (h *ExportHandler) Create(ctx context.Context, request dto.CreateAssetExportRequest) (dto.SuccessResponse[dto.CreateAssetExportResponse], error) {
	response, err := h.service.Create(ctx, exportmodule.CreateRequest(request))
	if err != nil {
		return dto.SuccessResponse[dto.CreateAssetExportResponse]{}, exportHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.CreateAssetExportResponse(response)), nil
}

func (h *ExportHandler) Get(ctx context.Context, exportID uint) (dto.SuccessResponse[dto.AssetExportResponse], error) {
	response, err := h.service.Get(ctx, exportID)
	if err != nil {
		return dto.SuccessResponse[dto.AssetExportResponse]{}, exportHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.AssetExportResponse(response)), nil
}

func exportHandlerError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, exportmodule.ErrInvalidRequest) || errors.Is(err, exportmodule.ErrUnsupportedAsset) {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	return err
}
