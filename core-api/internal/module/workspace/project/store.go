package project

import "context"

// Store persists projects without exposing infrastructure details to the module.
type Store interface {
	Insert(ctx context.Context, project *Project) error
	FindByID(ctx context.Context, projectID uint) (*Project, error)
	FindByUserID(ctx context.Context, userID uint) ([]*Project, error)
	Update(ctx context.Context, update *ProjectUpdate) error
	Remove(ctx context.Context, projectID uint) error
}
