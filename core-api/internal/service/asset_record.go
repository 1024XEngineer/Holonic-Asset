package service

import (
	"context"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
)

// AssetVersionService manages asset versioning.
type AssetVersionService interface {
	CreateRecord(ctx context.Context, version *domain.AssetVersion) (uint, error)
	GetVersionHistory(ctx context.Context, assetID uint) ([]domain.AssetVersion, error)
	RollBackVersion(ctx context.Context, assetID uint, version uint) (uint, error)
	Copy(ctx context.Context, assetID uint, version uint) (uint, error)
}

type AssetVersionServiceImpl struct {
	AssetRepository repository.AssetRepository
}

func (s *AssetVersionServiceImpl) CreateRecord(ctx context.Context, version *domain.AssetVersion) (uint, error) {
	_, err := s.AssetRepository.CreateRecord(ctx, version)
	return 0, err
}

func (s *AssetVersionServiceImpl) GetVersionHistory(ctx context.Context, assetID uint) ([]domain.AssetVersion, error) {
	return []domain.AssetVersion{}, nil
}

func (s *AssetVersionServiceImpl) RollBackVersion(ctx context.Context, assetID uint, version uint) (uint, error) {
	_, err := s.AssetRepository.RollBackRecord(ctx, assetID, version)
	return 0, err
}

func (s *AssetVersionServiceImpl) Copy(ctx context.Context, assetID uint, version uint) (uint, error) {
	_, err := s.AssetRepository.Copy(ctx, assetID, version)
	return 0, err
}
