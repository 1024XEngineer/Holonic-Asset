package dto

import domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"

type ProjectResponse struct {
	UserID         uint                `json:"userID"`
	ID             uint                `json:"id"`
	Name           string              `json:"name"`
	GameType       domain.GameType     `json:"gameType"`
	ViewType       domain.ViewType     `json:"viewType"`
	TargetPlatform domain.PlatformType `json:"targetPlatform"`
	Description    string              `json:"description"`
	Reference      string              `json:"reference"`
	Style          string              `json:"style"`
}

type CreateProjectRequest struct {
	UserID         uint                `json:"userID"`
	Name           string              `json:"name"`
	GameType       domain.GameType     `json:"gameType"`
	ViewType       domain.ViewType     `json:"viewType"`
	TargetPlatform domain.PlatformType `json:"targetPlatform"`
	Description    string              `json:"description"`
	Reference      string              `json:"reference"`
	Style          string              `json:"style"`
}

type CreateProjectResponse struct {
	ID uint `json:"id"`
}

type ListProjectsRequest struct {
	UserID uint `query:"userID"`
}

type ListProjectsResponse struct {
	Projects []*ProjectResponse `json:"projects"`
}

type ProjectDetailRequest struct {
	ProjectID uint `query:"projectID"`
}

type ProjectDetailResponse struct {
	Project *ProjectResponse `json:"project"`
}

type UpdateProjectRequest struct {
	ProjectID      uint                 `json:"projectID"`
	Name           *string              `json:"name,omitempty"`
	GameType       *domain.GameType     `json:"gameType,omitempty"`
	ViewType       *domain.ViewType     `json:"viewType,omitempty"`
	TargetPlatform *domain.PlatformType `json:"targetPlatform,omitempty"`
	Description    *string              `json:"description,omitempty"`
	Reference      *string              `json:"reference,omitempty"`
	Style          *string              `json:"style,omitempty"`
}

type UpdateProjectResponse struct {
	Success bool `json:"success"`
}

type DeleteProjectRequest struct {
	ProjectID uint `json:"projectID"`
}

type DeleteProjectResponse struct {
	Success bool `json:"success"`
}
