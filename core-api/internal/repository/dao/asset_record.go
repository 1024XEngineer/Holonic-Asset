package dao

import (
	"context"

	"gorm.io/gorm"
)

type AssetVersion struct {
	ID      uint
	AssetID uint
	Version uint
}

type AssetVersionDao interface {
	CreateAssetVersion(ctx context.Context, version *AssetVersion) (uint, error)
	CreateAssetVersions(ctx context.Context, version []AssetVersion) error
	DeleteAssetVersion(ctx context.Context, assetID uint, version uint) error
	GetAssetVersionsByAssetID(ctx context.Context, assetID uint) ([]AssetVersion, error)
}

type AssetVersionDaoImpl struct {
	DB *gorm.DB
}

func (a *AssetVersionDaoImpl) CreateAssetVersion(ctx context.Context, version *AssetVersion) (uint, error) {
	return 0, nil
}

func (a *AssetVersionDaoImpl) DeleteAssetVersion(ctx context.Context, assetID uint, version uint) error {
	return nil
}

func (a *AssetVersionDaoImpl) GetAssetVersionsByAssetID(ctx context.Context, assetID uint) ([]AssetVersion, error) {
	return []AssetVersion{}, nil
}
