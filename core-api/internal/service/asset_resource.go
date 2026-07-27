package service

import (
	"context"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
)

// AssetResourceService manages resources under an asset.
type AssetResourceService interface {
	GetProtoTypeResource(ctx context.Context, assetID uint, version uint) ([]domain.AssetResource, error)

	// Animation resources.
	CreateAnimationResource(ctx context.Context, resource *domain.AssetResource) (uint, error)
	EditAnimationResource(ctx context.Context, id uint, resource []domain.AssetResource) error
	GetAnimations(ctx context.Context, assetID uint, version uint) ([]domain.AssetResource, error)

	// Frame resources (associated with an animation).
	CreateFrameResources(ctx context.Context, resource []domain.AssetResource) ([]domain.AssetResource, error)
	EditFrameResources(ctx context.Context, resource []domain.AssetResource) ([]domain.AssetResource, error)
	GetFrameResources(ctx context.Context, animationID uint) ([]domain.AssetResource, error)

	// Tile / Item resources.
	CreateTileResources(ctx context.Context, resource []domain.AssetResource) ([]domain.AssetResource, error)
	EditItemResources(ctx context.Context, id uint, resource []domain.AssetResource) ([]domain.AssetResource, error)
	GetItemResources(ctx context.Context, assetID uint, version uint) ([]domain.AssetResource, error)
	GetTilesResources(ctx context.Context, itemID uint) ([]domain.AssetResource, error)

	// Image resources.
	CreateImageResources(ctx context.Context, resource []domain.AssetResource) ([]domain.AssetResource, error)
	EditImageResources(ctx context.Context, resource []domain.AssetResource) ([]domain.AssetResource, error)
}

type AssetResourceServiceImpl struct {
	AssetRepository repository.AssetRepository
}

func (s *AssetResourceServiceImpl) GetProtoTypeResource(ctx context.Context, assetID uint, version uint) ([]domain.AssetResource, error) {
	_, err := s.AssetRepository.GetProtoTypeResource(ctx, assetID, version)
	return []domain.AssetResource{}, err
}

func (s *AssetResourceServiceImpl) GetAnimations(ctx context.Context, assetID uint, version uint) ([]domain.AssetResource, error) {
	_, err := s.AssetRepository.GetAnimations(ctx, assetID, version)
	return []domain.AssetResource{}, err
}

func (s *AssetResourceServiceImpl) GetItemResources(ctx context.Context, assetID uint, version uint) ([]domain.AssetResource, error) {
	_, err := s.AssetRepository.GetItemResources(ctx, assetID, version)
	return []domain.AssetResource{}, err
}
