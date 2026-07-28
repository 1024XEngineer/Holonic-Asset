package service

import (
	"context"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
)

// AssetService manages CRUD operations for assets.
type AssetService interface {
	GetAssets(ctx context.Context, projectID uint) ([]domain.Asset, error)
	GetDetail(ctx context.Context, id uint) (domain.Asset, error)
	UpdateTags(ctx context.Context, id uint, tags []string) ([]string, error)

	// Creates a Character asset and initializes an empty prototype resource.
	CreateCharacterAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	CreateObjectAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	CreateTileSetAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	CreateUIAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	CreateSceneryAsset(ctx context.Context, asset *domain.Asset) (uint, error)
}

type AssetServiceImpl struct {
	AssetRepository repository.AssetRepository
}

func NewAssetService(assetRepository repository.AssetRepository) AssetService {
	return &AssetServiceImpl{AssetRepository: assetRepository}
}

func (a *AssetServiceImpl) GetAssets(ctx context.Context, projectID uint) ([]domain.Asset, error) {
	return a.AssetRepository.GetAssetsByProjectID(ctx, projectID)
}

func (a *AssetServiceImpl) GetDetail(ctx context.Context, id uint) (domain.Asset, error) {
	asset, err := a.AssetRepository.GetAssetDetail(ctx, id)
	if err != nil {
		return domain.Asset{}, err
	}
	if asset == nil {
		return domain.Asset{}, nil
	}
	return *asset, nil
}

func (a *AssetServiceImpl) UpdateTags(ctx context.Context, id uint, tags []string) ([]string, error) {
	_, err := a.AssetRepository.UpdateTags(ctx, id, tags)
	return []string{}, err
}

func (a *AssetServiceImpl) CreateCharacterAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	_, err := a.AssetRepository.CreateCharacterAsset(ctx, asset)
	// waiting for task
	return 0, err
}

func (a *AssetServiceImpl) CreateObjectAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	_, err := a.AssetRepository.CreateObjectAsset(ctx, asset)
	// waiting for task
	return 0, err
}

func (a *AssetServiceImpl) CreateTileSetAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	_, err := a.AssetRepository.CreateTileSetAsset(ctx, asset)
	// waiting for task
	return 0, err
}

func (a *AssetServiceImpl) CreateUIAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	_, err := a.AssetRepository.CreateUIAsset(ctx, asset)
	return 0, err
}

func (a *AssetServiceImpl) CreateSceneryAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	_, err := a.AssetRepository.CreateSceneryAsset(ctx, asset)
	return 0, err
}
