package tag

import "context"

// Store persists reusable project-scoped tags.
type Store interface {
	CreateProjectTag(ctx context.Context, tag *Tag) error
	ListProjectTags(ctx context.Context, projectID uint) ([]Tag, error)
	GetProjectTag(ctx context.Context, projectID, tagID uint) (*Tag, error)
	UpdateProjectTag(ctx context.Context, projectID, tagID uint, update *TagUpdate) (*Tag, error)
	DeleteProjectTag(ctx context.Context, projectID, tagID uint) error
}
