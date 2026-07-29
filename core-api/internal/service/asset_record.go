package service

import (
	"context"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
)

// AssetRecordService manages asset records.
type AssetRecordService interface {
	CreateRecord(ctx context.Context, record *domain.AssetRecord) (*domain.AssetRecord, error)
	GetRecordHistory(ctx context.Context, assetID uint) ([]domain.AssetRecord, error)
	RollBackRecord(ctx context.Context, assetID uint, version uint) (*domain.AssetRecord, error)
	Copy(ctx context.Context, assetID uint, version uint) (uint, error)
}

type AssetRecordServiceImpl struct {
	AssetRepository repository.AssetRepository
}

func (s *AssetRecordServiceImpl) CreateRecord(ctx context.Context, record *domain.AssetRecord) (*domain.AssetRecord, error) {
	return s.AssetRepository.CreateRecord(ctx, record)
}

func (s *AssetRecordServiceImpl) GetRecordHistory(ctx context.Context, assetID uint) ([]domain.AssetRecord, error) {
	return s.AssetRepository.GetRecordHistory(ctx, assetID)
}

func (s *AssetRecordServiceImpl) RollBackRecord(ctx context.Context, assetID uint, version uint) (*domain.AssetRecord, error) {
	return s.AssetRepository.RollBackRecord(ctx, assetID, version)
}

func (s *AssetRecordServiceImpl) Copy(ctx context.Context, assetID uint, version uint) (uint, error) {
	return s.AssetRepository.Copy(ctx, assetID, version)
}
