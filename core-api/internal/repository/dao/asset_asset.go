package dao

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type Asset struct {
	ID           uint `gorm:"primaryKey"`
	Name         string
	ProjectID    uint `gorm:"index"`
	Type         string
	Description  string
	Tags         []assetdomain.Tag `json:"tags" gorm:"type:jsonb;serializer:asset_tags"`
	Perspective  string
	Dimensions   datatypes.JSON `gorm:"type:jsonb"`
	ThumbnailURL string
	ContentID    *uint          `gorm:"index"`
	Content      datatypes.JSON `json:"content" gorm:"-"`
	Version      uint
}

type AssetUpdate struct {
	Name        *string
	Description *string
	Tags        *[]assetdomain.Tag
	Perspective *string
	Dimensions  *datatypes.JSON
}

type AssetDao interface {
	CreateAsset(ctx context.Context, asset *Asset) (Asset, error)
	GetAssetsByProjectID(ctx context.Context, projectID uint) ([]Asset, error)
	GetAsset(ctx context.Context, id uint) (Asset, error)
	GetAssetForUpdate(ctx context.Context, id uint) (Asset, error)
	UpdateAsset(ctx context.Context, id uint, update *AssetUpdate) (Asset, error)
	DeleteAsset(ctx context.Context, id uint) error
	UpdateAssetCurrentContent(ctx context.Context, id uint, version uint, contentID uint, thumbnailURL string) error
}

type AssetDaoImpl struct {
	DB *gorm.DB
}

func (a *AssetDaoImpl) WithDB(db *gorm.DB) AssetDao {
	return &AssetDaoImpl{DB: db}
}

func (a *AssetDaoImpl) DBHandle() *gorm.DB {
	return a.DB
}

func (a *AssetDaoImpl) GetAssetsByProjectID(ctx context.Context, projectID uint) ([]Asset, error) {
	assets := make([]Asset, 0)
	err := a.DB.WithContext(ctx).
		Where("project_id = ?", projectID).
		Select("id, name, project_id, type, description, tags, perspective, dimensions, thumbnail_url, version").
		Order("id ASC").
		Find(&assets).Error
	return assets, err
}

func (a *AssetDaoImpl) CreateAsset(ctx context.Context, asset *Asset) (Asset, error) {
	if asset == nil {
		return Asset{}, fmt.Errorf("dao: asset is nil")
	}
	if asset.Version == 0 {
		asset.Version = 1
	}
	if err := a.DB.WithContext(ctx).Create(asset).Error; err != nil {
		return Asset{}, fmt.Errorf("dao: create asset: %w", err)
	}
	return *asset, nil
}

func (a *AssetDaoImpl) GetAsset(ctx context.Context, id uint) (Asset, error) {
	var asset Asset
	err := a.DB.WithContext(ctx).First(&asset, id).Error
	return asset, err
}

func (a *AssetDaoImpl) GetAssetForUpdate(ctx context.Context, id uint) (Asset, error) {
	var asset Asset
	err := a.DB.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&asset, id).Error
	return asset, err
}

func (a *AssetDaoImpl) UpdateAsset(ctx context.Context, id uint, update *AssetUpdate) (Asset, error) {
	if update == nil {
		return Asset{}, fmt.Errorf("dao: asset update is nil")
	}

	values := make(map[string]any)
	if update.Name != nil {
		values["name"] = *update.Name
	}
	if update.Description != nil {
		values["description"] = *update.Description
	}
	if update.Tags != nil {
		encoded, err := json.Marshal(*update.Tags)
		if err != nil {
			return Asset{}, fmt.Errorf("dao: encode asset tags: %w", err)
		}
		// Asset.Tags is a serialized text column. Map updates bypass GORM's
		// field serializer, so pass the JSON representation explicitly.
		values["tags"] = string(encoded)
	}
	if update.Perspective != nil {
		values["perspective"] = *update.Perspective
	}
	if update.Dimensions != nil {
		values["dimensions"] = *update.Dimensions
	}
	query := a.DB.WithContext(ctx).Model(&Asset{}).Where("id = ?", id)
	if len(values) > 0 {
		if result := query.Updates(values); result.Error != nil {
			return Asset{}, fmt.Errorf("dao: update asset %d: %w", id, result.Error)
		} else if result.RowsAffected == 0 {
			return Asset{}, fmt.Errorf("dao: asset %d not found", id)
		}
	}

	var asset Asset
	if err := a.DB.WithContext(ctx).First(&asset, id).Error; err != nil {
		return Asset{}, fmt.Errorf("dao: get updated asset %d: %w", id, err)
	}
	return asset, nil
}

func (a *AssetDaoImpl) DeleteAsset(ctx context.Context, id uint) error {
	result := a.DB.WithContext(ctx).Where("id = ?", id).Delete(&Asset{})
	if result.Error != nil {
		return fmt.Errorf("dao: delete asset %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("dao: asset %d not found", id)
	}
	return nil
}

func (a *AssetDaoImpl) UpdateAssetCurrentContent(
	ctx context.Context,
	id uint,
	version uint,
	contentID uint,
	thumbnailURL string,
) error {
	result := a.DB.WithContext(ctx).
		Model(&Asset{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"version":       version,
			"content_id":    contentID,
			"thumbnail_url": thumbnailURL,
		})
	if result.Error != nil {
		return fmt.Errorf("dao: update current content for asset %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("dao: asset %d not found", id)
	}
	return nil
}
