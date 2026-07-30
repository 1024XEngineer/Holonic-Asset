package asset

import "context"

// Store persists assets, their content, and immutable version records.
type Store interface {
	GetAssetsByProjectID(ctx context.Context, projectID uint, filter AssetListFilter) ([]Asset, error)
	GetAssetDetail(ctx context.Context, id uint) (*Asset, error)
	UpdateAsset(ctx context.Context, id uint, update *AssetUpdate) (*Asset, error)
	UpdateContent(ctx context.Context, assetID uint, content AssetContent) error
	UpdateAnimationFrames(ctx context.Context, assetID uint, animationID uint, frames []Frame) error
	CreateCharacterAsset(ctx context.Context, asset *Asset) (*Asset, error)
	CreateObjectAsset(ctx context.Context, asset *Asset) (uint, error)
	CreateTileSetAsset(ctx context.Context, asset *Asset) (uint, error)
	CreateUIAsset(ctx context.Context, asset *Asset) (uint, error)
	CreateSceneryAsset(ctx context.Context, asset *Asset) (uint, error)
	CreateAnimation(ctx context.Context, assetID uint, name string) (uint, error)
	UpdatePrototypeImages(ctx context.Context, assetID uint, images []ImageResource) error
	CreateRecord(ctx context.Context, record *AssetRecord) (*AssetRecord, error)
	GetRecordHistory(ctx context.Context, assetID uint) ([]AssetRecord, error)
	RollBackRecord(ctx context.Context, assetID uint, version uint) (*AssetRecord, error)
	Copy(ctx context.Context, assetID uint, version uint) (uint, error)
}
