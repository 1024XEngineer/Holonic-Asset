package tag

import "context"

// Manager exposes tag lifecycle operations scoped to projects.
type Manager interface {
	CreateProjectTag(ctx context.Context, tag Tag) (Tag, error)
	ListProjectTags(ctx context.Context, projectID uint) ([]Tag, error)
	GetProjectTag(ctx context.Context, projectID, tagID uint) (Tag, error)
	UpdateProjectTag(ctx context.Context, projectID, tagID uint, update *TagUpdate) (Tag, error)
	DeleteProjectTag(ctx context.Context, projectID, tagID uint) error
}

type manager struct {
	store Store
}

func NewManager(store Store) Manager {
	return &manager{store: store}
}

func (m *manager) CreateProjectTag(ctx context.Context, tag Tag) (Tag, error) {
	if err := tag.validateCreate(); err != nil {
		return Tag{}, err
	}
	if err := m.store.CreateProjectTag(ctx, &tag); err != nil {
		return Tag{}, err
	}
	return tag, nil
}

func (m *manager) ListProjectTags(ctx context.Context, projectID uint) ([]Tag, error) {
	if projectID == 0 {
		return nil, invalidTag("projectID is required")
	}
	return m.store.ListProjectTags(ctx, projectID)
}

func (m *manager) GetProjectTag(ctx context.Context, projectID, tagID uint) (Tag, error) {
	if err := validateTagScope(projectID, tagID); err != nil {
		return Tag{}, err
	}
	tag, err := m.store.GetProjectTag(ctx, projectID, tagID)
	if err != nil {
		return Tag{}, err
	}
	if tag == nil {
		return Tag{}, ErrTagNotFound
	}
	return *tag, nil
}

func (m *manager) UpdateProjectTag(
	ctx context.Context,
	projectID uint,
	tagID uint,
	update *TagUpdate,
) (Tag, error) {
	if err := validateTagScope(projectID, tagID); err != nil {
		return Tag{}, err
	}
	if err := update.validate(); err != nil {
		return Tag{}, err
	}
	tag, err := m.store.UpdateProjectTag(ctx, projectID, tagID, update)
	if err != nil {
		return Tag{}, err
	}
	if tag == nil {
		return Tag{}, ErrTagNotFound
	}
	return *tag, nil
}

func (m *manager) DeleteProjectTag(ctx context.Context, projectID, tagID uint) error {
	if err := validateTagScope(projectID, tagID); err != nil {
		return err
	}
	return m.store.DeleteProjectTag(ctx, projectID, tagID)
}

var _ Manager = (*manager)(nil)
