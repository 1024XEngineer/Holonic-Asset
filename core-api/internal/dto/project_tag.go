package dto

type ProjectTagResponse struct {
	TagID       uint   `json:"tagId" minimum:"1"`
	ProjectID   uint   `json:"projectId" minimum:"1"`
	Name        string `json:"name" minLength:"1" maxLength:"100"`
	Description string `json:"description,omitempty" maxLength:"255"`
	Color       string `json:"color" pattern:"^#[0-9A-Fa-f]{6}$"`
}

type CreateProjectTagRequest struct {
	ProjectID   uint   `json:"-"`
	Name        string `json:"name" minLength:"1" maxLength:"100"`
	Description string `json:"description,omitempty" maxLength:"255"`
	Color       string `json:"color,omitempty" pattern:"^#[0-9A-Fa-f]{6}$"`
}

type CreateProjectTagResponse struct {
	Tag ProjectTagResponse `json:"tag"`
}

type ListProjectTagsRequest struct {
	ProjectID uint `param:"project_id" path:"project_id" json:"-" minimum:"1"`
}

type ListProjectTagsResponse struct {
	Tags []ProjectTagResponse `json:"tags" nullable:"false"`
}

type ProjectTagDetailRequest struct {
	ProjectID uint `param:"project_id" path:"project_id" json:"-" minimum:"1"`
	TagID     uint `param:"tag_id" path:"tag_id" json:"-" minimum:"1"`
}

type ProjectTagDetailResponse struct {
	Tag ProjectTagResponse `json:"tag"`
}

type UpdateProjectTagRequest struct {
	ProjectID   uint    `json:"-"`
	TagID       uint    `json:"-"`
	Name        *string `json:"name,omitempty" minLength:"1" maxLength:"100"`
	Description *string `json:"description,omitempty" maxLength:"255"`
	Color       *string `json:"color,omitempty" pattern:"^#[0-9A-Fa-f]{6}$"`
}

type UpdateProjectTagResponse struct {
	Tag ProjectTagResponse `json:"tag"`
}

type DeleteProjectTagRequest struct {
	ProjectID uint `param:"project_id" path:"project_id" json:"-" minimum:"1"`
	TagID     uint `param:"tag_id" path:"tag_id" json:"-" minimum:"1"`
}

type DeleteProjectTagResponse struct {
	Success bool `json:"success"`
}
