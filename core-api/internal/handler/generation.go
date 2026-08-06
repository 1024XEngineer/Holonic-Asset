package handler

import (
	"context"
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
)

type GenerationHandler struct {
	runs generator.RunManager
}

func NewGenerationHandler(runs generator.RunManager) *GenerationHandler {
	return &GenerationHandler{runs: runs}
}

func (h *GenerationHandler) Create(
	ctx context.Context,
	request dto.CreateGenerationRequest,
) (dto.SuccessResponse[dto.CreateGenerationResponse], error) {
	runID, err := h.runs.Create(ctx, &generator.Request{
		ProjectID:        request.ProjectID,
		AssetID:          request.AssetID,
		Kind:             request.Kind,
		Prompt:           request.Prompt,
		TargetAssetPaths: request.TargetAssetPaths,
		Parameters:       request.Parameters,
	})
	if err != nil {
		return dto.SuccessResponse[dto.CreateGenerationResponse]{}, err
	}
	return dto.NewTypedSuccessResponse(dto.CreateGenerationResponse{GenerationRunID: runID}), nil
}

func (h *GenerationHandler) List(
	ctx context.Context,
	request dto.ListGenerationRunsRequest,
) (dto.SuccessResponse[dto.ListGenerationRunsResponse], error) {
	page, err := h.runs.List(ctx, &generator.RunListQuery{
		ProjectID: request.ProjectID,
		AssetID:   request.AssetID,
		Status:    request.Status,
		Limit:     request.Limit,
		Cursor:    request.Cursor,
	})
	if errors.Is(err, generator.ErrInvalidRunListStatus) ||
		errors.Is(err, generator.ErrInvalidRunListCursor) {
		return dto.SuccessResponse[dto.ListGenerationRunsResponse]{}, echo.ErrBadRequest
	}
	if err != nil {
		return dto.SuccessResponse[dto.ListGenerationRunsResponse]{}, err
	}
	if page == nil {
		return dto.NewTypedSuccessResponse(dto.ListGenerationRunsResponse{Items: []dto.GenerationRunListItemResponse{}}), nil
	}

	items := make([]dto.GenerationRunListItemResponse, len(page.Runs))
	for i := range page.Runs {
		run := page.Runs[i]
		items[i] = dto.GenerationRunListItemResponse{
			ID:        run.ID,
			ProjectID: run.ProjectID,
			AssetID:   run.AssetID,
			Kind:      run.Kind,
			Status:    dto.GenerationStatus(run.Status.String()),
		}
	}

	return dto.NewTypedSuccessResponse(dto.ListGenerationRunsResponse{
		Items:      items,
		NextCursor: page.NextCursor,
	}), nil
}

func (h *GenerationHandler) Get(
	ctx context.Context,
	request dto.GetGenerationRequest,
) (dto.SuccessResponse[dto.GetGenerationResponse], error) {
	run, err := h.runs.Get(ctx, request.GenerationRunID)
	if err != nil {
		return dto.SuccessResponse[dto.GetGenerationResponse]{}, err
	}

	return dto.NewTypedSuccessResponse(dto.GetGenerationResponse{
		ID:        run.ID,
		ProjectID: run.ProjectID,
		AssetID:   run.AssetID,
		Kind:      run.Kind,
		Status:    dto.GenerationStatus(run.Status.String()),
		Result:    run.Result,
		Error:     run.Error,
	}), nil
}

func (h *GenerationHandler) Cancel(
	ctx context.Context,
	request dto.CancelGenerationRequest,
) (dto.SuccessResponse[dto.CancelGenerationResponse], error) {
	err := h.runs.Cancel(ctx, request.GenerationRunID)
	if err != nil {
		return dto.SuccessResponse[dto.CancelGenerationResponse]{}, err
	}
	return dto.NewTypedSuccessResponse(dto.CancelGenerationResponse{Cancelled: true}), nil
}
