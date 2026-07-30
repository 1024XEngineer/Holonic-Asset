package handler

import (
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

func (h *ProjectHandler) Create(c *echox.Context, request dto.CreateProjectRequest) (dto.CreateProjectResponse, error) {
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
	err := h.manager.Create(c, project)
	return dto.CreateProjectResponse{}, err
}

func (h *ProjectHandler) ListByUID(c *echox.Context, request dto.ListProjectsRequest) (dto.ListProjectsResponse, error) {
	_, err := h.manager.ListByUID(c, request.UserID)
	return dto.ListProjectsResponse{}, err
}

func (h *ProjectHandler) GetDetail(c *echox.Context, request dto.ProjectDetailRequest) (dto.ProjectDetailResponse, error) {
	_, err := h.manager.GetDetail(c, request.ProjectID)
	return dto.ProjectDetailResponse{}, err
}

func (h *ProjectHandler) Update(c *echox.Context, request dto.UpdateProjectRequest) (dto.UpdateProjectResponse, error) {
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
	err := h.manager.Update(c, update)
	return dto.UpdateProjectResponse{Success: err == nil}, err
}

func (h *ProjectHandler) Delete(c *echox.Context, request dto.DeleteProjectRequest) (dto.DeleteProjectResponse, error) {
	err := h.manager.Delete(c, request.ProjectID)
	return dto.DeleteProjectResponse{}, err
}
