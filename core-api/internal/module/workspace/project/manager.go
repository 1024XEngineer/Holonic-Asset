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
	if err := project.ValidateCreate(); err != nil {
		return err
	}
	return m.store.Insert(ctx, project)
}

func (m *manager) ListByUID(ctx context.Context, userID uint) ([]*Project, error) {
	if err := ValidateUserID(userID); err != nil {
		return nil, err
	}
	return m.store.FindByUserID(ctx, userID)
}

func (m *manager) GetDetail(ctx context.Context, projectID uint) (*Project, error) {
	if err := ValidateProjectID(projectID); err != nil {
		return nil, err
	}
	return m.store.FindByID(ctx, projectID)
}

func (m *manager) Update(ctx context.Context, update *ProjectUpdate) error {
	if err := update.Validate(); err != nil {
		return err
	}
	return m.store.Update(ctx, update)
}

func (m *manager) Delete(ctx context.Context, projectID uint) error {
	if err := ValidateProjectID(projectID); err != nil {
		return err
	}
	return m.store.Remove(ctx, projectID)
}

var _ Manager = (*manager)(nil)
