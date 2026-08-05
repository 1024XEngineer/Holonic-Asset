package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

type ProjectHandler struct {
	manager domain.Manager
}

func NewProjectHandler(manager domain.Manager) *ProjectHandler {
	return &ProjectHandler{manager: manager}
}

func (h *ProjectHandler) Create(
	c context.Context,
	request dto.CreateProjectRequest,
) (dto.SuccessResponse[dto.CreateProjectResponse], error) {
	project := &domain.Project{
		UserID:         request.UserID,
		Name:           request.Name,
		GameType:       request.GameType,
		Perspective:    request.Perspective,
		TargetPlatform: request.TargetPlatform,
		Description:    request.Description,
		Reference:      request.Reference,
		Style:          request.Style,
	}
	if err := h.manager.Create(c, project); err != nil {
		return dto.SuccessResponse[dto.CreateProjectResponse]{}, projectHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.CreateProjectResponse{ID: project.ID}), nil
}

func (h *ProjectHandler) GenerateReference(
	c context.Context,
	request dto.GenerateProjectReferenceRequest,
) (dto.SuccessResponse[dto.GenerateProjectReferenceResponse], error) {
	project := &domain.Project{
		Name:           request.Name,
		GameType:       request.GameType,
		Perspective:    request.Perspective,
		TargetPlatform: request.TargetPlatform,
		Description:    request.Description,
		Reference:      request.Reference,
		Style:          request.Style,
	}

	reference, err := h.manager.GenerateReference(c, project)
	if err != nil {
		return dto.SuccessResponse[dto.GenerateProjectReferenceResponse]{}, projectHandlerError(err)
	}

	return dto.NewTypedSuccessResponse(dto.GenerateProjectReferenceResponse{Reference: reference}), nil
}

func (h *ProjectHandler) ListByUID(
	c context.Context,
	request dto.ListProjectsRequest,
) (dto.SuccessResponse[dto.ListProjectsResponse], error) {
	projects, err := h.manager.ListByUID(c, request.UserID)
	if err != nil {
		return dto.SuccessResponse[dto.ListProjectsResponse]{}, projectHandlerError(err)
	}

	response := make([]*dto.ProjectResponse, len(projects))
	for i, project := range projects {
		response[i] = projectResponse(project)
	}
	return dto.NewTypedSuccessResponse(dto.ListProjectsResponse{Projects: response}), nil
}

func (h *ProjectHandler) GetDetail(
	c context.Context,
	request dto.ProjectDetailRequest,
) (dto.SuccessResponse[dto.ProjectDetailResponse], error) {
	project, err := h.manager.GetDetail(c, request.ProjectID)
	if err != nil {
		return dto.SuccessResponse[dto.ProjectDetailResponse]{}, projectHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.ProjectDetailResponse{Project: projectResponse(project)}), nil
}

func (h *ProjectHandler) Update(
	c context.Context,
	request dto.UpdateProjectRequest,
) (dto.SuccessResponse[dto.UpdateProjectResponse], error) {
	update := &domain.ProjectUpdate{
		ID:             request.ProjectID,
		Name:           request.Name,
		GameType:       request.GameType,
		Perspective:    request.Perspective,
		TargetPlatform: request.TargetPlatform,
		Description:    request.Description,
		Reference:      request.Reference,
		Style:          request.Style,
	}
	if err := h.manager.Update(c, update); err != nil {
		return dto.SuccessResponse[dto.UpdateProjectResponse]{}, projectHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.UpdateProjectResponse{Success: true}), nil
}

func (h *ProjectHandler) Delete(
	c context.Context,
	request dto.DeleteProjectRequest,
) (dto.SuccessResponse[dto.DeleteProjectResponse], error) {
	if err := h.manager.Delete(c, request.ProjectID); err != nil {
		return dto.SuccessResponse[dto.DeleteProjectResponse]{}, projectHandlerError(err)
	}
	return dto.NewTypedSuccessResponse(dto.DeleteProjectResponse{Success: true}), nil
}

func projectResponse(project *domain.Project) *dto.ProjectResponse {
	if project == nil {
		return nil
	}
	return &dto.ProjectResponse{
		UserID:         project.UserID,
		ID:             project.ID,
		Name:           project.Name,
		GameType:       project.GameType,
		Perspective:    project.Perspective,
		TargetPlatform: project.TargetPlatform,
		Description:    project.Description,
		Reference:      project.Reference,
		Style:          project.Style,
	}
}

func projectHandlerError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrInvalidProject):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	case errors.Is(err, domain.ErrProjectNotFound):
		return echo.NewHTTPError(http.StatusNotFound, domain.ErrProjectNotFound.Error()).SetInternal(err)
	default:
		return err
	}
}
