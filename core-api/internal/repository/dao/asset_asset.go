package dao

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"
)

type Asset struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	ProjectID   uint `gorm:"index"`
	Type        string
	Description string
	Tags        []string        `json:"tags" gorm:"serializer:json"`
	Attributes  json.RawMessage `json:"attributes" gorm:"serializer:json"`
	Version     uint
}

type AssetDao interface {
	CreateAsset(ctx context.Context, asset *Asset) (Asset, error)
	GetAssetsByProjectID(ctx context.Context, projectID uint) ([]Asset, error)
	GetAssetDetail(ctx context.Context, id uint) (Asset, error)
	UpdateTags(ctx context.Context, id uint, tags []string) ([]string, error)
	UpdateAssetVersion(ctx context.Context, id uint, version uint) error
}

type AssetDaoImpl struct {
	DB *gorm.DB
}

func (a *AssetDaoImpl) GetAssetsByProjectID(ctx context.Context, projectID uint) ([]Asset, error) {
	assets := make([]Asset, 0)
	err := a.DB.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("id ASC").
		Find(&assets).Error
	return assets, err
}

func (a *AssetDaoImpl) CreateAsset(ctx context.Context, asset *Asset) (Asset, error) {
	return Asset{}, nil
}

func (a *AssetDaoImpl) GetAssetDetail(ctx context.Context, id uint) (Asset, error) {
	var asset Asset
	err := a.DB.WithContext(ctx).First(&asset, id).Error
	return asset, err
}

func (a *AssetDaoImpl) UpdateTags(ctx context.Context, id uint, tags []string) ([]string, error) {
	return []string{}, nil
}

func (a *AssetDaoImpl) UpdateAssetVersion(ctx context.Context, id uint, version uint) error {
	return nil
}
