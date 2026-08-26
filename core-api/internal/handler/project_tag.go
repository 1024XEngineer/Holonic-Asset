package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	tagdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/tag"
)

type ProjectTagHandler struct {
	manager tagdomain.Manager
}

func NewProjectTagHandler(manager tagdomain.Manager) *ProjectTagHandler {
	return &ProjectTagHandler{manager: manager}
}

func (h *ProjectTagHandler) CreateProjectTag(
	ctx context.Context,
	request dto.CreateProjectTagRequest,
) (dto.SuccessResponse[dto.CreateProjectTagResponse], error) {
	if h == nil || h.manager == nil {
		return dto.SuccessResponse[dto.CreateProjectTagResponse]{}, echo.NewHTTPError(
			http.StatusInternalServerError,
			"tag manager is not initialized",
		)
	}
	tag, err := h.manager.CreateProjectTag(ctx, tagdomain.Tag{
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

func (h *ProjectTagHandler) ListProjectTags(
	ctx context.Context,
	request dto.ListProjectTagsRequest,
) (dto.SuccessResponse[dto.ListProjectTagsResponse], error) {
	if h == nil || h.manager == nil {
		return dto.SuccessResponse[dto.ListProjectTagsResponse]{}, echo.NewHTTPError(
			http.StatusInternalServerError,
			"tag manager is not initialized",
		)
	}
	tags, err := h.manager.ListProjectTags(ctx, request.ProjectID)
	if err != nil {
		return dto.SuccessResponse[dto.ListProjectTagsResponse]{}, projectTagHandlerError(err)
	}
	result := make([]dto.ProjectTagResponse, len(tags))
	for index := range tags {
		result[index] = projectTagResponse(tags[index])
	}
	return dto.NewTypedSuccessResponse(dto.ListProjectTagsResponse{Tags: result}), nil
}

func (h *ProjectTagHandler) GetProjectTag(
	ctx context.Context,
	request dto.ProjectTagDetailRequest,
) (dto.SuccessResponse[dto.ProjectTagDetailResponse], error) {
	if h == nil || h.manager == nil {
		return dto.SuccessResponse[dto.ProjectTagDetailResponse]{}, echo.NewHTTPError(
			http.StatusInternalServerError,
			"tag manager is not initialized",
		)
	}
	tag, err := h.manager.GetProjectTag(ctx, request.ProjectID, request.TagID)
	if err != nil {
		return dto.SuccessResponse[dto.ProjectTagDetailResponse]{}, projectTagHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.ProjectTagDetailResponse{Tag: projectTagResponse(tag)}), nil
}

func (h *ProjectTagHandler) UpdateProjectTag(
	ctx context.Context,
	request dto.UpdateProjectTagRequest,
) (dto.SuccessResponse[dto.UpdateProjectTagResponse], error) {
	if h == nil || h.manager == nil {
		return dto.SuccessResponse[dto.UpdateProjectTagResponse]{}, echo.NewHTTPError(
			http.StatusInternalServerError,
			"tag manager is not initialized",
		)
	}
	tag, err := h.manager.UpdateProjectTag(ctx, request.ProjectID, request.TagID, &tagdomain.TagUpdate{
		Name:        request.Name,
		Description: request.Description,
		Color:       request.Color,
	})
	if err != nil {
		return dto.SuccessResponse[dto.UpdateProjectTagResponse]{}, projectTagHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.UpdateProjectTagResponse{Tag: projectTagResponse(tag)}), nil
}

func (h *ProjectTagHandler) DeleteProjectTag(
	ctx context.Context,
	request dto.DeleteProjectTagRequest,
) (dto.SuccessResponse[dto.DeleteProjectTagResponse], error) {
	if h == nil || h.manager == nil {
		return dto.SuccessResponse[dto.DeleteProjectTagResponse]{}, echo.NewHTTPError(
			http.StatusInternalServerError,
			"tag manager is not initialized",
		)
	}
	if err := h.manager.DeleteProjectTag(ctx, request.ProjectID, request.TagID); err != nil {
		return dto.SuccessResponse[dto.DeleteProjectTagResponse]{}, projectTagHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.DeleteProjectTagResponse{Success: true}), nil
}

func projectTagResponse(tag tagdomain.Tag) dto.ProjectTagResponse {
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
	case errors.Is(err, tagdomain.ErrInvalidTag):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	case errors.Is(err, tagdomain.ErrTagNotFound),
		errors.Is(err, tagdomain.ErrTagProjectNotFound):
		return echo.NewHTTPError(http.StatusNotFound, err.Error()).SetInternal(err)
	case errors.Is(err, tagdomain.ErrTagConflict):
		return echo.NewHTTPError(http.StatusConflict, err.Error()).SetInternal(err)
	default:
		return err
	}
}
