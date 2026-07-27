package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type TaxonomyHandler struct {
	service service.AssetDiscoveryService
}

func NewTaxonomyHandler(assetDiscoveryService service.AssetDiscoveryService) *TaxonomyHandler {
	return &TaxonomyHandler{service: assetDiscoveryService}
}

func (h *TaxonomyHandler) SearchAssets(
	c *echox.Context,
	request dto.SearchAssetsRequest,
) (*dto.AssetSearchResult, error) {
	if request.ProjectID == 0 {
		return nil, echo.ErrBadRequest
	}
	result, err := h.service.SearchAssets(c, &service.SearchAssetsRequest{
		ProjectID: request.ProjectID,
		Query:     request.Query,
		Tags:      request.Tags,
		Types:     request.Types,
	})
	if err != nil {
		return nil, err
	}

	items := make([]dto.AssetSearchItem, len(result.Assets))
	for index, item := range result.Assets {
		items[index] = dto.AssetSearchItem{
			ID:          item.ID,
			Name:        item.Name,
			ProjectID:   item.ProjectID,
			Type:        item.Type,
			Description: item.Description,
			Tags:        item.Tags,
			Version:     item.Version,
		}
	}
	return &dto.AssetSearchResult{Assets: items}, nil
}
