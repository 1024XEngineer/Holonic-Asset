package handler

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
)

type GenerationHandler struct {
	runs generator.RunManager
}

func NewGenerationHandler(runs generator.RunManager) *GenerationHandler {
	return &GenerationHandler{runs: runs}
}

func (h *GenerationHandler) Create(
	ctx *echox.Context,
	request dto.CreateGenerationRequest,
) (dto.CreateGenerationResponse, error) {
	runID, err := h.runs.Create(ctx, &generator.Request{
		ProjectID:         request.ProjectID,
		AssetID:           request.AssetID,
		Kind:              request.Kind,
		Prompt:            request.Prompt,
		ReferenceMediaIDs: request.ReferenceMediaIDs,
		TargetAssetPaths:  request.TargetAssetPaths,
		Parameters:        request.Parameters,
	})
	return dto.CreateGenerationResponse{GenerationRunID: runID}, err
}

func (h *GenerationHandler) List(
	ctx *echox.Context,
	request dto.ListGenerationRunsRequest,
) (dto.ListGenerationRunsResponse, error) {
	page, err := h.runs.List(ctx, &generator.RunListQuery{
		ProjectID: request.ProjectID,
		AssetID:   request.AssetID,
		Status:    request.Status,
		Limit:     request.Limit,
		Cursor:    request.Cursor,
	})
	if errors.Is(err, generator.ErrInvalidRunListStatus) {
		return dto.ListGenerationRunsResponse{}, echo.ErrBadRequest
	}
	if err != nil {
		return dto.ListGenerationRunsResponse{}, err
	}
	if page == nil {
		return dto.ListGenerationRunsResponse{Items: []dto.GenerationRunListItemResponse{}}, nil
	}

	items := make([]dto.GenerationRunListItemResponse, len(page.Runs))
	for i := range page.Runs {
		run := page.Runs[i]
		items[i] = dto.GenerationRunListItemResponse{
			ID:        run.ID,
			ProjectID: run.ProjectID,
			AssetID:   run.AssetID,
			Kind:      run.Kind,
			Status:    run.Status,
		}
	}

	return dto.ListGenerationRunsResponse{
		Items:      items,
		NextCursor: page.NextCursor,
	}, nil
}

func (h *GenerationHandler) Get(
	ctx *echox.Context,
	request dto.GetGenerationRequest,
) (dto.GetGenerationResponse, error) {
	run, err := h.runs.Get(ctx, request.GenerationRunID)
	if err != nil {
		return dto.GetGenerationResponse{}, err
	}

	return dto.GetGenerationResponse{
		ID:        run.ID,
		ProjectID: run.ProjectID,
		AssetID:   run.AssetID,
		Kind:      run.Kind,
		Status:    run.Status,
		Result:    run.Result,
		Error:     run.Error,
	}, nil
}

func (h *GenerationHandler) Cancel(
	ctx *echox.Context,
	request dto.CancelGenerationRequest,
) (dto.CancelGenerationResponse, error) {
	err := h.runs.Cancel(ctx, request.GenerationRunID)
	return dto.CancelGenerationResponse{Cancelled: err == nil}, err
}
