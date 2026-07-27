package dao

import (
	"context"

	"gorm.io/gorm"
)

type AssetResourceDao interface {
	CreateAssetResources(ctx context.Context, resource []AssetResource) (uint, error)
	GetProtoTypeResource(ctx context.Context, assetID uint, version uint) (AssetResource, error)
	GetImageResources(ctx context.Context, protoTypeID uint) ([]AssetResource, error)
	GetAnimations(ctx context.Context, assetID uint, version uint) ([]AssetResource, error)
	GetItemResources(ctx context.Context, assetID uint, version uint) ([]AssetResource, error)
	UpdateFrameResources(ctx context.Context, resource []AssetResource) error
	DeleteFrameResourcesByAnimationID(ctx context.Context, id uint) error
	GetResourcesByAssetVersion(ctx context.Context, assetID uint, version uint) ([]AssetResource, error)
	CopyResourcesToVersion(ctx context.Context, resources []AssetResource, version uint) error
	DeleteResourcesByVersion(ctx context.Context, assetID uint, version uint) error
}

type AssetResource struct {
	ID           uint
	Name         string
	ParentID     *uint
	AssetID      uint
	AssetVersion uint
	Type         string
	Url          *string
	Status       uint
}

type AssetResourceDaoImpl struct {
	DB *gorm.DB
}

func (a *AssetResourceDaoImpl) CreateAssetResources(ctx context.Context, resource []AssetResource) (uint, error) {
	// Upsert: update if exists, insert if not.
	return 0, nil
}

func (a *AssetResourceDaoImpl) GetProtoTypeResource(ctx context.Context, assetID uint, version uint) (AssetResource, error) {
	return AssetResource{}, nil
}

func (a *AssetResourceDaoImpl) GetImageResources(ctx context.Context, protoTypeID uint) ([]AssetResource, error) {
	return []AssetResource{}, nil
}

func (a *AssetResourceDaoImpl) GetAnimations(ctx context.Context, assetID uint, version uint) ([]AssetResource, error) {
	return []AssetResource{}, nil
}

func (a *AssetResourceDaoImpl) GetItemResources(ctx context.Context, assetID uint, version uint) ([]AssetResource, error) {
	return []AssetResource{}, nil
}

func (a *AssetResourceDaoImpl) UpdateFrameResources(ctx context.Context, resource []AssetResource) error {
	return nil
}

func (a *AssetResourceDaoImpl) DeleteFrameResourcesByAnimationID(ctx context.Context, id uint) error {
	return nil
}

func (a *AssetResourceDaoImpl) UpdateProtoTypeResources(ctx context.Context, resource []AssetResource) error {
	return nil
}

func (a *AssetResourceDaoImpl) GetResourcesByAssetVersion(ctx context.Context, assetID uint, version uint) ([]AssetResource, error) {
	return []AssetResource{}, nil
}

func (a *AssetResourceDaoImpl) CopyResourcesToVersion(ctx context.Context, resources []AssetResource, version uint) error {
	return nil
}

func (a *AssetResourceDaoImpl) DeleteResourcesByVersion(ctx context.Context, version uint) error {
	return nil
}
