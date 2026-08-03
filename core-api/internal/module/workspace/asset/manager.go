package asset

import "context"

// Manager exposes asset lifecycle, content, and version operations.
type Manager interface {
	GetAssets(ctx context.Context, projectID uint, filter AssetListFilter) ([]Asset, error)
	GetDetail(ctx context.Context, id uint) (Asset, error)
	Delete(ctx context.Context, id uint) error
	UpdateAsset(ctx context.Context, id uint, update *AssetUpdate) (*Asset, error)
	UpdateContent(ctx context.Context, id uint, content AssetContent) error
	UpdateAnimationFrames(ctx context.Context, assetID uint, animationID uint, frames []Frame) error
	UpdatePrototypeImages(ctx context.Context, assetID uint, images []ImageResource) error
	CreateCharacterAsset(ctx context.Context, asset *Asset) (*Asset, error)
	CreateObjectAsset(ctx context.Context, asset *Asset) (uint, error)
	CreateTileSetAsset(ctx context.Context, asset *Asset) (uint, error)
	CreateUIAsset(ctx context.Context, asset *Asset) (uint, error)
	CreateSceneryAsset(ctx context.Context, asset *Asset) (uint, error)
	CreateAnimation(ctx context.Context, assetID uint, name string) (uint, error)
	CreateRecord(ctx context.Context, record *AssetRecord) (*AssetRecord, error)
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

func (m *manager) UpdateContent(ctx context.Context, id uint, content AssetContent) error {
	return m.store.UpdateContent(ctx, id, content)
}

func (m *manager) UpdateAnimationFrames(ctx context.Context, assetID uint, animationID uint, frames []Frame) error {
	return m.store.UpdateAnimationFrames(ctx, assetID, animationID, frames)
}

func (m *manager) UpdatePrototypeImages(ctx context.Context, assetID uint, images []ImageResource) error {
	return m.store.UpdatePrototypeImages(ctx, assetID, images)
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

func (m *manager) CreateUIAsset(ctx context.Context, asset *Asset) (uint, error) {
	return m.store.CreateUIAsset(ctx, asset)
}

func (m *manager) CreateSceneryAsset(ctx context.Context, asset *Asset) (uint, error) {
	return m.store.CreateSceneryAsset(ctx, asset)
}

func (m *manager) CreateAnimation(ctx context.Context, assetID uint, name string) (uint, error) {
	return m.store.CreateAnimation(ctx, assetID, name)
}

func (m *manager) CreateRecord(ctx context.Context, record *AssetRecord) (*AssetRecord, error) {
	return m.store.CreateRecord(ctx, record)
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
