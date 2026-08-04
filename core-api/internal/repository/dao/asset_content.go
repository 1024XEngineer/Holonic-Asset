package dao

import (
	"context"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AssetContent struct {
	ID      uint           `gorm:"primaryKey"`
	AssetID uint           `gorm:"not null;index"`
	Content datatypes.JSON `gorm:"type:jsonb;not null"`
}

type AssetContentDao interface {
	CreateAssetContent(ctx context.Context, content *AssetContent) (AssetContent, error)
	CreateAssetContents(ctx context.Context, contents []AssetContent) error
	GetAssetContent(ctx context.Context, id uint) (AssetContent, error)
	GetAssetContentsByAssetID(ctx context.Context, assetID uint) ([]AssetContent, error)
	DeleteAssetContents(ctx context.Context, ids []uint) error
	DeleteAssetContentsByAssetID(ctx context.Context, assetID uint) error
}

type AssetContentDaoImpl struct {
	DB *gorm.DB
}

func (a *AssetContentDaoImpl) WithDB(db *gorm.DB) AssetContentDao {
	return &AssetContentDaoImpl{DB: db}
}

func (a *AssetContentDaoImpl) CreateAssetContent(ctx context.Context, content *AssetContent) (AssetContent, error) {
	if content == nil {
		return AssetContent{}, fmt.Errorf("dao: asset content is nil")
	}
	if err := a.DB.WithContext(ctx).Create(content).Error; err != nil {
		return AssetContent{}, fmt.Errorf("dao: create asset content: %w", err)
	}
	return *content, nil
}

func (a *AssetContentDaoImpl) CreateAssetContents(ctx context.Context, contents []AssetContent) error {
	if len(contents) == 0 {
		return nil
	}
	if err := a.DB.WithContext(ctx).Create(&contents).Error; err != nil {
		return fmt.Errorf("dao: create asset contents: %w", err)
	}
	return nil
}

func (a *AssetContentDaoImpl) GetAssetContent(ctx context.Context, id uint) (AssetContent, error) {
	var content AssetContent
	if err := a.DB.WithContext(ctx).First(&content, id).Error; err != nil {
		return AssetContent{}, fmt.Errorf("dao: get asset content %d: %w", id, err)
	}
	return content, nil
}

func (a *AssetContentDaoImpl) GetAssetContentsByAssetID(ctx context.Context, assetID uint) ([]AssetContent, error) {
	contents := make([]AssetContent, 0)
	if err := a.DB.WithContext(ctx).
		Where("asset_id = ?", assetID).
		Order("id ASC").
		Find(&contents).Error; err != nil {
		return nil, fmt.Errorf("dao: get asset contents for asset %d: %w", assetID, err)
	}
	return contents, nil
}

func (a *AssetContentDaoImpl) DeleteAssetContentsByAssetID(ctx context.Context, assetID uint) error {
	if err := a.DB.WithContext(ctx).Where("asset_id = ?", assetID).Delete(&AssetContent{}).Error; err != nil {
		return fmt.Errorf("dao: delete asset contents for asset %d: %w", assetID, err)
	}
	return nil
}

func (a *AssetContentDaoImpl) DeleteAssetContents(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if err := a.DB.WithContext(ctx).Where("id IN ?", ids).Delete(&AssetContent{}).Error; err != nil {
		return fmt.Errorf("dao: delete asset contents: %w", err)
	}
	return nil
}
