package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type AssetRecord struct {
	ID        uint `gorm:"primaryKey"`
	AssetID   uint `gorm:"not null;uniqueIndex:idx_asset_record"`
	Version   uint `gorm:"not null;uniqueIndex:idx_asset_record"`
	ContentID uint `gorm:"not null;index"`
	CreatedAt time.Time
}

type AssetRecordDao interface {
	CreateAssetRecord(ctx context.Context, record *AssetRecord) (uint, error)
	CreateAssetRecords(ctx context.Context, records []AssetRecord) error
	DeleteAssetRecord(ctx context.Context, assetID uint, version uint) error
	DeleteAssetRecordsAfterVersion(ctx context.Context, assetID uint, version uint) error
	DeleteAssetRecordsByAssetID(ctx context.Context, assetID uint) error
	GetAssetRecord(ctx context.Context, assetID uint, version uint) (AssetRecord, error)
	GetAssetRecordsByAssetID(ctx context.Context, assetID uint) ([]AssetRecord, error)
}

type AssetRecordDaoImpl struct {
	DB *gorm.DB
}

func (a *AssetRecordDaoImpl) WithDB(db *gorm.DB) AssetRecordDao {
	return &AssetRecordDaoImpl{DB: db}
}

func (a *AssetRecordDaoImpl) CreateAssetRecord(ctx context.Context, record *AssetRecord) (uint, error) {
	if record == nil {
		return 0, fmt.Errorf("dao: asset record is nil")
	}
	if err := a.DB.WithContext(ctx).Create(record).Error; err != nil {
		return 0, fmt.Errorf("dao: create asset record: %w", err)
	}
	return record.ID, nil
}

func (a *AssetRecordDaoImpl) CreateAssetRecords(ctx context.Context, records []AssetRecord) error {
	if len(records) == 0 {
		return nil
	}
	if err := a.DB.WithContext(ctx).Create(&records).Error; err != nil {
		return fmt.Errorf("dao: create asset records: %w", err)
	}
	return nil
}

func (a *AssetRecordDaoImpl) DeleteAssetRecord(ctx context.Context, assetID uint, version uint) error {
	result := a.DB.WithContext(ctx).
		Where("asset_id = ? AND version = ?", assetID, version).
		Delete(&AssetRecord{})
	if result.Error != nil {
		return fmt.Errorf("dao: delete asset record %d/%d: %w", assetID, version, result.Error)
	}
	return nil
}

func (a *AssetRecordDaoImpl) DeleteAssetRecordsAfterVersion(ctx context.Context, assetID uint, version uint) error {
	result := a.DB.WithContext(ctx).
		Where("asset_id = ? AND version > ?", assetID, version).
		Delete(&AssetRecord{})
	if result.Error != nil {
		return fmt.Errorf("dao: delete asset records after %d/%d: %w", assetID, version, result.Error)
	}
	return nil
}

func (a *AssetRecordDaoImpl) DeleteAssetRecordsByAssetID(ctx context.Context, assetID uint) error {
	if err := a.DB.WithContext(ctx).Where("asset_id = ?", assetID).Delete(&AssetRecord{}).Error; err != nil {
		return fmt.Errorf("dao: delete asset records for asset %d: %w", assetID, err)
	}
	return nil
}

func (a *AssetRecordDaoImpl) GetAssetRecord(ctx context.Context, assetID uint, version uint) (AssetRecord, error) {
	var result AssetRecord
	err := a.DB.WithContext(ctx).
		Where("asset_id = ? AND version = ?", assetID, version).
		First(&result).Error
	if err != nil {
		return AssetRecord{}, fmt.Errorf("dao: get asset record %d/%d: %w", assetID, version, err)
	}
	return result, nil
}

func (a *AssetRecordDaoImpl) GetAssetRecordsByAssetID(ctx context.Context, assetID uint) ([]AssetRecord, error) {
	records := make([]AssetRecord, 0)
	err := a.DB.WithContext(ctx).
		Where("asset_id = ?", assetID).
		Order("version ASC").
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("dao: get asset records for asset %d: %w", assetID, err)
	}
	return records, nil
}
