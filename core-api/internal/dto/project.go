package dto

import domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"

type ProjectResponse struct {
	UserID         uint                `json:"userID" minimum:"1"`
	ID             uint                `json:"id" minimum:"1"`
	Name           string              `json:"name"`
	GameType       string              `json:"gameType"`
	Perspective    domain.Perspective  `json:"perspective" enum:"Top-Down,Side-On,Isometric"`
	TargetPlatform domain.PlatformType `json:"targetPlatform" enum:",PC,Mobile,Web"`
	Description    string              `json:"description"`
	Reference      string              `json:"reference"`
	Style          string              `json:"style"`
}

type CreateProjectRequest struct {
	UserID         uint                `json:"userID" minimum:"1"`
	Name           string              `json:"name" minLength:"1"`
	GameType       string              `json:"gameType,omitempty"`
	Perspective    domain.Perspective  `json:"perspective,omitempty" enum:"Top-Down,Side-On,Isometric" default:"Top-Down"`
	TargetPlatform domain.PlatformType `json:"targetPlatform,omitempty" enum:",PC,Mobile,Web"`
	Description    string              `json:"description,omitempty"`
	Reference      string              `json:"reference,omitempty"`
	Style          string              `json:"style,omitempty"`
}

type CreateProjectResponse struct {
	ID uint `json:"id" minimum:"1"`
}

type GenerateReferenceRequest struct {
	Name           string              `json:"name" minLength:"1"`
	GameType       string              `json:"gameType,omitempty"`
	Perspective    domain.Perspective  `json:"perspective,omitempty" enum:"Top-Down,Side-On,Isometric" default:"Top-Down"`
	TargetPlatform domain.PlatformType `json:"targetPlatform,omitempty" enum:",PC,Mobile,Web"`
	Description    string              `json:"description,omitempty"`
	Reference      string              `json:"reference,omitempty"`
	Style          string              `json:"style,omitempty"`
}

type GenerateReferenceResponse struct {
	Reference string `json:"reference"`
}

type ListProjectsRequest struct {
	UserID uint `query:"userID" required:"true" minimum:"1"`
}

type ListProjectsResponse struct {
	Projects []*ProjectResponse `json:"projects" nullable:"false"`
}

type ProjectDetailRequest struct {
	ProjectID uint `query:"projectID" required:"true" minimum:"1"`
}

type ProjectDetailResponse struct {
	Project *ProjectResponse `json:"project"`
}

type UpdateProjectRequest struct {
	ProjectID      uint                 `json:"projectID" minimum:"1"`
	Name           *string              `json:"name,omitempty" minLength:"1"`
	GameType       *string              `json:"gameType,omitempty"`
	Perspective    *domain.Perspective  `json:"perspective,omitempty" enum:"Top-Down,Side-On,Isometric"`
	TargetPlatform *domain.PlatformType `json:"targetPlatform,omitempty" enum:",PC,Mobile,Web"`
	Description    *string              `json:"description,omitempty"`
	Reference      *string              `json:"reference,omitempty"`
	Style          *string              `json:"style,omitempty"`
}

type UpdateProjectResponse struct {
	Success bool `json:"success"`
}

type DeleteProjectRequest struct {
	ProjectID uint `json:"projectID" minimum:"1"`
}

type DeleteProjectResponse struct {
	Success bool `json:"success"`
}
