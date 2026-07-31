package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

type ProjectHandler struct {
	manager domain.Manager
}

func NewProjectHandler(manager domain.Manager) *ProjectHandler {
	return &ProjectHandler{manager: manager}
}

func (h *ProjectHandler) Create(c *echox.Context, request dto.CreateProjectRequest) (dto.Response, error) {
	project := &domain.Project{
		UserID:         request.UserID,
		Name:           request.Name,
		GameType:       request.GameType,
		ViewType:       request.ViewType,
		TargetPlatform: request.TargetPlatform,
		Description:    request.Description,
		Reference:      request.Reference,
		Style:          request.Style,
	}
	if err := h.manager.Create(c, project); err != nil {
		return dto.Response{}, projectHandlerError(err)
	}
	return dto.NewSuccessResponse(dto.CreateProjectResponse{ID: project.ID}), nil
}

func (h *ProjectHandler) ListByUID(c *echox.Context, request dto.ListProjectsRequest) (dto.Response, error) {
	projects, err := h.manager.ListByUID(c, request.UserID)
	if err != nil {
		return dto.Response{}, projectHandlerError(err)
	}

	response := make([]*dto.ProjectResponse, len(projects))
	for i, project := range projects {
		response[i] = projectResponse(project)
	}
	return dto.NewSuccessResponse(dto.ListProjectsResponse{Projects: response}), nil
}

func (h *ProjectHandler) GetDetail(c *echox.Context, request dto.ProjectDetailRequest) (dto.Response, error) {
	project, err := h.manager.GetDetail(c, request.ProjectID)
	if err != nil {
		return dto.Response{}, projectHandlerError(err)
	}
	return dto.NewSuccessResponse(dto.ProjectDetailResponse{Project: projectResponse(project)}), nil
}

func (h *ProjectHandler) Update(c *echox.Context, request dto.UpdateProjectRequest) (dto.Response, error) {
	update := &domain.ProjectUpdate{
		ID:             request.ProjectID,
		Name:           request.Name,
		GameType:       request.GameType,
		ViewType:       request.ViewType,
		TargetPlatform: request.TargetPlatform,
		Description:    request.Description,
		Reference:      request.Reference,
		Style:          request.Style,
	}
	if err := h.manager.Update(c, update); err != nil {
		return dto.Response{}, projectHandlerError(err)
	}
	return dto.NewSuccessResponse(dto.UpdateProjectResponse{Success: true}), nil
}

func (h *ProjectHandler) Delete(c *echox.Context, request dto.DeleteProjectRequest) (dto.Response, error) {
	if err := h.manager.Delete(c, request.ProjectID); err != nil {
		return dto.Response{}, projectHandlerError(err)
	}
	return dto.NewSuccessResponse(dto.DeleteProjectResponse{Success: true}), nil
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
		ViewType:       project.ViewType,
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
