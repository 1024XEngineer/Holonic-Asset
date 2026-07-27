package dao

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"
)

type Asset struct {
	ID          uint
	Name        string
	ProjectID   uint
	Type        string
	Description string
	Tags        []string        `json:"tags"`
	Attributes  json.RawMessage `json:"attributes"`
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
	return nil, nil
}

func (a *AssetDaoImpl) CreateAsset(ctx context.Context, asset *Asset) (Asset, error) {
	return Asset{}, nil
}

func (a *AssetDaoImpl) GetAssetDetail(ctx context.Context, id uint) (Asset, error) {
	return Asset{}, nil
}

func (a *AssetDaoImpl) UpdateTags(ctx context.Context, id uint, tags []string) ([]string, error) {
	return []string{}, nil
}

func (a *AssetDaoImpl) UpdateAssetVersion(ctx context.Context, id uint, version uint) error {
	return nil
}
