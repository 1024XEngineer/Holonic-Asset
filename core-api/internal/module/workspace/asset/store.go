package asset

import "context"

// Store persists assets, their content, and immutable version records.
type Store interface {
	CreateProjectTag(ctx context.Context, tag *ProjectTag) error
	ListProjectTags(ctx context.Context, projectID uint) ([]ProjectTag, error)
	GetProjectTag(ctx context.Context, projectID, tagID uint) (*ProjectTag, error)
	UpdateProjectTag(ctx context.Context, projectID, tagID uint, update *ProjectTagUpdate) (*ProjectTag, error)
	DeleteProjectTag(ctx context.Context, projectID, tagID uint) error
	GetAssetsByProjectID(ctx context.Context, projectID uint, filter AssetListFilter) ([]Asset, error)
	GetAssetDetail(ctx context.Context, id uint) (*Asset, error)
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
