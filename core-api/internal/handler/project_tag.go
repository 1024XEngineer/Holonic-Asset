package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func (h *Handler) CreateProjectTag(
	ctx context.Context,
	request dto.CreateProjectTagRequest,
) (dto.SuccessResponse[dto.CreateProjectTagResponse], error) {
	tag, err := h.AssetManager.CreateProjectTag(ctx, domain.ProjectTag{
		ProjectID:   request.ProjectID,
		Name:        request.Name,
		Description: request.Description,
		Color:       request.Color,
	})
	if err != nil {
		return dto.SuccessResponse[dto.CreateProjectTagResponse]{}, projectTagHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.CreateProjectTagResponse{Tag: projectTagResponse(tag)}), nil
}

func (h *Handler) ListProjectTags(
	ctx context.Context,
	request dto.ListProjectTagsRequest,
) (dto.SuccessResponse[dto.ListProjectTagsResponse], error) {
	tags, err := h.AssetManager.ListProjectTags(ctx, request.ProjectID)
	if err != nil {
		return dto.SuccessResponse[dto.ListProjectTagsResponse]{}, projectTagHandlerError(err)
	}
	result := make([]dto.ProjectTagResponse, len(tags))
	for index := range tags {
		result[index] = projectTagResponse(tags[index])
	}
	return dto.NewTypedSuccessResponse(dto.ListProjectTagsResponse{Tags: result}), nil
}

func (h *Handler) GetProjectTag(
	ctx context.Context,
	request dto.ProjectTagDetailRequest,
) (dto.SuccessResponse[dto.ProjectTagDetailResponse], error) {
	tag, err := h.AssetManager.GetProjectTag(ctx, request.ProjectID, request.TagID)
	if err != nil {
		return dto.SuccessResponse[dto.ProjectTagDetailResponse]{}, projectTagHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.ProjectTagDetailResponse{Tag: projectTagResponse(tag)}), nil
}

func (h *Handler) UpdateProjectTag(
	ctx context.Context,
	request dto.UpdateProjectTagRequest,
) (dto.SuccessResponse[dto.UpdateProjectTagResponse], error) {
	tag, err := h.AssetManager.UpdateProjectTag(ctx, request.ProjectID, request.TagID, &domain.ProjectTagUpdate{
		Name:        request.Name,
		Description: request.Description,
		Color:       request.Color,
	})
	if err != nil {
		return dto.SuccessResponse[dto.UpdateProjectTagResponse]{}, projectTagHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.UpdateProjectTagResponse{Tag: projectTagResponse(tag)}), nil
}

func (h *Handler) DeleteProjectTag(
	ctx context.Context,
	request dto.DeleteProjectTagRequest,
) (dto.SuccessResponse[dto.DeleteProjectTagResponse], error) {
	if err := h.AssetManager.DeleteProjectTag(ctx, request.ProjectID, request.TagID); err != nil {
		return dto.SuccessResponse[dto.DeleteProjectTagResponse]{}, projectTagHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.DeleteProjectTagResponse{Success: true}), nil
}

func projectTagResponse(tag domain.ProjectTag) dto.ProjectTagResponse {
	return dto.ProjectTagResponse{
		TagID:       tag.ID,
		ProjectID:   tag.ProjectID,
		Name:        tag.Name,
		Description: tag.Description,
		Color:       tag.Color,
	}
}

func projectTagHandlerError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrInvalidProjectTag):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	case errors.Is(err, domain.ErrProjectTagNotFound),
		errors.Is(err, domain.ErrProjectTagProjectNotFound):
		return echo.NewHTTPError(http.StatusNotFound, err.Error()).SetInternal(err)
	case errors.Is(err, domain.ErrProjectTagConflict):
		return echo.NewHTTPError(http.StatusConflict, err.Error()).SetInternal(err)
	default:
		return err
	}
}
