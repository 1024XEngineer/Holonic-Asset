package service

import (
	"context"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
)

// AssetService manages CRUD operations for assets.
type AssetService interface {
	GetAssets(ctx context.Context, projectID uint, filter domain.AssetListFilter) ([]domain.Asset, error)
	GetDetail(ctx context.Context, id uint) (domain.Asset, error)
	UpdateAsset(ctx context.Context, id uint, update *domain.AssetUpdate) (*domain.Asset, error)
	UpdateContent(ctx context.Context, id uint, content domain.AssetContent) error
	UpdateAnimationDirection(
		ctx context.Context,
		assetID uint,
		animationID uint,
		direction string,
		frames []domain.Frame,
	) error
	UpdatePrototypeImages(ctx context.Context, assetID uint, images map[string]domain.ImageResource) error

	// Creates a Character asset and initializes prototype content.
	CreateCharacterAsset(ctx context.Context, asset *domain.Asset) (*domain.Asset, error)
	CreateObjectAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	CreateTileSetAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	CreateUIAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	CreateSceneryAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	CreateAnimation(ctx context.Context, assetID uint, name string) (uint, error)
}

type AssetServiceImpl struct {
	AssetRepository repository.AssetRepository
}

func NewAssetService(assetRepository repository.AssetRepository) AssetService {
	return &AssetServiceImpl{AssetRepository: assetRepository}
}

func (a *AssetServiceImpl) GetAssets(ctx context.Context, projectID uint, filter domain.AssetListFilter) ([]domain.Asset, error) {
	return a.AssetRepository.GetAssetsByProjectID(ctx, projectID, filter)
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

func (a *AssetServiceImpl) UpdateAsset(
	ctx context.Context,
	id uint,
	update *domain.AssetUpdate,
) (*domain.Asset, error) {
	asset, err := a.AssetRepository.UpdateAsset(ctx, id, update)
	if err != nil {
		return nil, err
	}
	return asset, nil
}

func (a *AssetServiceImpl) UpdateContent(
	ctx context.Context,
	id uint,
	content domain.AssetContent,
) error {
	return a.AssetRepository.UpdateContent(ctx, id, content)
}

func (a *AssetServiceImpl) UpdateAnimationDirection(
	ctx context.Context,
	assetID uint,
	animationID uint,
	direction string,
	frames []domain.Frame,
) error {
	return a.AssetRepository.UpdateAnimationDirection(
		ctx,
		assetID,
		animationID,
		direction,
		frames,
	)
}

func (a *AssetServiceImpl) UpdatePrototypeImages(
	ctx context.Context,
	assetID uint,
	images map[string]domain.ImageResource,
) error {
	return a.AssetRepository.UpdatePrototypeImages(ctx, assetID, images)
}

func (a *AssetServiceImpl) CreateCharacterAsset(ctx context.Context, asset *domain.Asset) (*domain.Asset, error) {
	return a.AssetRepository.CreateCharacterAsset(ctx, asset)
}

func (a *AssetServiceImpl) CreateObjectAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	return a.AssetRepository.CreateObjectAsset(ctx, asset)
}

func (a *AssetServiceImpl) CreateTileSetAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	return a.AssetRepository.CreateTileSetAsset(ctx, asset)
}

func (a *AssetServiceImpl) CreateUIAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	return a.AssetRepository.CreateUIAsset(ctx, asset)
}

func (a *AssetServiceImpl) CreateSceneryAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	return a.AssetRepository.CreateSceneryAsset(ctx, asset)
}

func (a *AssetServiceImpl) CreateAnimation(ctx context.Context, assetID uint, name string) (uint, error) {
	return a.AssetRepository.CreateAnimation(ctx, assetID, name)
}
