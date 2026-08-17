package dao

import (
	"context"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Asset struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	ProjectID   uint `gorm:"index"`
	Type        string
	Description string
	Tags        []string `json:"tags" gorm:"serializer:json"`
	Perspective string
	Dimensions  datatypes.JSON `gorm:"type:jsonb"`
	ContentID   *uint          `gorm:"index"`
	Content     datatypes.JSON `json:"content" gorm:"-"`
	Version     uint
}

type AssetUpdate struct {
	Name        *string
	Description *string
	Tags        *[]string
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
	UpdateAssetCurrentContent(ctx context.Context, id uint, version uint, contentID uint) error
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
	type assetListRow struct {
		ID          uint
		Name        string
		ProjectID   uint
		Type        string
		Description string
		Tags        []string `gorm:"serializer:json"`
		Perspective string
		Dimensions  datatypes.JSON
		Content     datatypes.JSON
		Version     uint
	}

	rows := make([]assetListRow, 0)
	err := a.DB.WithContext(ctx).
		Table("assets AS a").
		Select("a.id, a.name, a.project_id, a.type, a.description, a.tags, a.perspective, a.dimensions, c.content, a.version").
		Joins("LEFT JOIN asset_contents AS c ON c.id = a.content_id").
		Where("a.project_id = ?", projectID).
		Order("a.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	assets := make([]Asset, len(rows))
	for index, row := range rows {
		assets[index] = Asset{
			ID:          row.ID,
			Name:        row.Name,
			ProjectID:   row.ProjectID,
			Type:        row.Type,
			Description: row.Description,
			Tags:        row.Tags,
			Perspective: row.Perspective,
			Dimensions:  row.Dimensions,
			Content:     row.Content,
			Version:     row.Version,
		}
	}
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
		values["tags"] = *update.Tags
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
) error {
	result := a.DB.WithContext(ctx).
		Model(&Asset{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"version":    version,
			"content_id": contentID,
		})
	if result.Error != nil {
		return fmt.Errorf("dao: update current content for asset %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("dao: asset %d not found", id)
	}
	return nil
}
