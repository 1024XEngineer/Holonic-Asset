package asset

import "context"

// Manager exposes asset lifecycle, content, and version operations.
type Manager interface {
	CreateProjectTag(ctx context.Context, tag ProjectTag) (ProjectTag, error)
	ListProjectTags(ctx context.Context, projectID uint) ([]ProjectTag, error)
	GetProjectTag(ctx context.Context, projectID, tagID uint) (ProjectTag, error)
	UpdateProjectTag(ctx context.Context, projectID, tagID uint, update *ProjectTagUpdate) (ProjectTag, error)
	DeleteProjectTag(ctx context.Context, projectID, tagID uint) error
	GetAssets(ctx context.Context, projectID uint, filter AssetListFilter) ([]Asset, error)
	GetDetail(ctx context.Context, id uint) (Asset, error)
	Delete(ctx context.Context, id uint) error
	UpdateAsset(ctx context.Context, id uint, update *AssetUpdate) (*Asset, error)
	CreateCharacterAsset(ctx context.Context, asset *Asset) (*Asset, error)
	CreateObjectAsset(ctx context.Context, asset *Asset) (uint, error)
	CreateTileSetAsset(ctx context.Context, asset *Asset) (uint, error)
	CreateUISetAsset(ctx context.Context, asset *Asset) (uint, error)
	CreateSceneryAsset(ctx context.Context, asset *Asset) (uint, error)
	CreateRecord(ctx context.Context, record *AssetRecord, expectedVersion uint) (*AssetRecord, error)
	GetRecordHistory(ctx context.Context, assetID uint) ([]AssetRecord, error)
	RollBackRecord(ctx context.Context, assetID uint, version uint) (*AssetRecord, error)
	Copy(ctx context.Context, assetID uint, version uint) (uint, error)
}

type manager struct {
	store Store
}

func NewManager(store Store) Manager {
	return &manager{store: store}
}

func (m *manager) CreateProjectTag(ctx context.Context, tag ProjectTag) (ProjectTag, error) {
	if err := tag.validateCreate(); err != nil {
		return ProjectTag{}, err
	}
	if err := m.store.CreateProjectTag(ctx, &tag); err != nil {
		return ProjectTag{}, err
	}
	return tag, nil
}

func (m *manager) ListProjectTags(ctx context.Context, projectID uint) ([]ProjectTag, error) {
	if projectID == 0 {
		return nil, invalidProjectTag("projectID is required")
	}
	return m.store.ListProjectTags(ctx, projectID)
}

func (m *manager) GetProjectTag(ctx context.Context, projectID, tagID uint) (ProjectTag, error) {
	if err := validateProjectTagScope(projectID, tagID); err != nil {
		return ProjectTag{}, err
	}
	tag, err := m.store.GetProjectTag(ctx, projectID, tagID)
	if err != nil {
		return ProjectTag{}, err
	}
	if tag == nil {
		return ProjectTag{}, ErrProjectTagNotFound
	}
	return *tag, nil
}

func (m *manager) UpdateProjectTag(
	ctx context.Context,
	projectID uint,
	tagID uint,
	update *ProjectTagUpdate,
) (ProjectTag, error) {
	if err := validateProjectTagScope(projectID, tagID); err != nil {
		return ProjectTag{}, err
	}
	if err := update.validate(); err != nil {
		return ProjectTag{}, err
	}
	tag, err := m.store.UpdateProjectTag(ctx, projectID, tagID, update)
	if err != nil {
		return ProjectTag{}, err
	}
	if tag == nil {
		return ProjectTag{}, ErrProjectTagNotFound
	}
	return *tag, nil
}

func (m *manager) DeleteProjectTag(ctx context.Context, projectID, tagID uint) error {
	if err := validateProjectTagScope(projectID, tagID); err != nil {
		return err
	}
	return m.store.DeleteProjectTag(ctx, projectID, tagID)
}

func (m *manager) GetAssets(ctx context.Context, projectID uint, filter AssetListFilter) ([]Asset, error) {
	return m.store.GetAssetsByProjectID(ctx, projectID, filter)
}

func (m *manager) GetDetail(ctx context.Context, id uint) (Asset, error) {
	value, err := m.store.GetAssetDetail(ctx, id)
	if err != nil {
		return Asset{}, err
	}
	if value == nil {
		return Asset{}, nil
	}
	return *value, nil
}

func (m *manager) Delete(ctx context.Context, id uint) error {
	return m.store.Delete(ctx, id)
}

func (m *manager) UpdateAsset(ctx context.Context, id uint, update *AssetUpdate) (*Asset, error) {
	return m.store.UpdateAsset(ctx, id, update)
}

func (m *manager) CreateCharacterAsset(ctx context.Context, asset *Asset) (*Asset, error) {
	return m.store.CreateCharacterAsset(ctx, asset)
}

func (m *manager) CreateObjectAsset(ctx context.Context, asset *Asset) (uint, error) {
	return m.store.CreateObjectAsset(ctx, asset)
}

func (m *manager) CreateTileSetAsset(ctx context.Context, asset *Asset) (uint, error) {
	return m.store.CreateTileSetAsset(ctx, asset)
}

func (m *manager) CreateUISetAsset(ctx context.Context, asset *Asset) (uint, error) {
	return m.store.CreateUISetAsset(ctx, asset)
}

func (m *manager) CreateSceneryAsset(ctx context.Context, asset *Asset) (uint, error) {
	return m.store.CreateSceneryAsset(ctx, asset)
}

func (m *manager) CreateRecord(ctx context.Context, record *AssetRecord, expectedVersion uint) (*AssetRecord, error) {
	return m.store.CreateRecord(ctx, record, expectedVersion)
}

func (m *manager) GetRecordHistory(ctx context.Context, assetID uint) ([]AssetRecord, error) {
	return m.store.GetRecordHistory(ctx, assetID)
}

func (m *manager) RollBackRecord(ctx context.Context, assetID uint, version uint) (*AssetRecord, error) {
	return m.store.RollBackRecord(ctx, assetID, version)
}

func (m *manager) Copy(ctx context.Context, assetID uint, version uint) (uint, error) {
	return m.store.Copy(ctx, assetID, version)
}

var _ Manager = (*manager)(nil)
