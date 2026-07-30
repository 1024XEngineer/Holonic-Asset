package project

import "context"

// Manager exposes the project lifecycle to transports and other modules.
type Manager interface {
	Create(ctx context.Context, project *Project) error
	ListByUID(ctx context.Context, userID uint) ([]*Project, error)
	GetDetail(ctx context.Context, projectID uint) (*Project, error)
	Update(ctx context.Context, update *ProjectUpdate) error
	Delete(ctx context.Context, projectID uint) error
}

type manager struct {
	store Store
}

func NewManager(store Store) Manager {
	return &manager{store: store}
}

func (m *manager) Create(ctx context.Context, project *Project) error {
	return m.store.Insert(ctx, project)
}

func (m *manager) ListByUID(ctx context.Context, userID uint) ([]*Project, error) {
	return m.store.FindByUserID(ctx, userID)
}

func (m *manager) GetDetail(ctx context.Context, projectID uint) (*Project, error) {
	return m.store.FindByID(ctx, projectID)
}

// Update applies only fields supplied by the caller.
// Reference validation and Claims ownership checks are deferred to the storage/auth integration.
func (m *manager) Update(ctx context.Context, update *ProjectUpdate) error {
	return m.store.Update(ctx, update)
}

func (m *manager) Delete(ctx context.Context, projectID uint) error {
	return m.store.Remove(ctx, projectID)
}

var _ Manager = (*manager)(nil)
