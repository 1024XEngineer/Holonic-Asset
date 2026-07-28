package repository

import (
	"context"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type AssetRepository interface {
	GetAssetsByProjectID(ctx context.Context, projectID uint) ([]domain.Asset, error)
	GetAssetDetail(ctx context.Context, id uint) (*domain.Asset, error)
	UpdateTags(ctx context.Context, id uint, tags []string) ([]string, error)
	// CreateCharacterAsset creates a character asset and an empty protoType resource.
	CreateCharacterAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	CreateObjectAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	CreateTileSetAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	CreateUIAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	CreateSceneryAsset(ctx context.Context, asset *domain.Asset) (uint, error)
	// CreateImageResources creates image resources bound to a specific protoType.
	CreateImageResources(ctx context.Context, resource []domain.AssetResource) ([]domain.AssetResource, error)
	CreateAnimationResource(ctx context.Context, resource *domain.AssetResource) (uint, error)

	UpdateFrameResources(ctx context.Context, resource []domain.AssetResource) error
	DeleteFrameResourcesByAnimationID(ctx context.Context, id uint) error

	GetAnimations(ctx context.Context, assetID uint, version uint) ([]domain.AssetResource, error)
	GetProtoTypeResource(ctx context.Context, assetID uint, version uint) ([]domain.AssetResource, error)
	GetItemResources(ctx context.Context, assetID uint, version uint) ([]domain.AssetResource, error)

	// CreateRecord creates an AssetVersion, updates the Asset's version, and copies all resources under the asset to the new version.
	CreateRecord(ctx context.Context, version *domain.AssetVersion) (*domain.AssetVersion, error)
	// RollBackRecord deletes the AssetVersion, rolls back the Asset's version to the previous one, and deletes all resources of that version.
	RollBackRecord(ctx context.Context, assetID uint, version uint) (uint, error)
	// Copy performs a full copy of the version, asset, and all its resources to a new asset.
	Copy(ctx context.Context, assetID uint, version uint) (uint, error)
}

type AssetRepositoryImpl struct {
	AssetDao    dao.AssetDao
	ResourceDao dao.AssetResourceDao
	VersionDao  dao.AssetVersionDao
}

func NewAssetRepository(assetDao dao.AssetDao) AssetRepository {
	return &AssetRepositoryImpl{AssetDao: assetDao}
}

func (r *AssetRepositoryImpl) GetAssetsByProjectID(ctx context.Context, projectID uint) ([]domain.Asset, error) {
	assets, err := r.AssetDao.GetAssetsByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Asset, len(assets))
	for index := range assets {
		result[index] = convertAssetToDomain(assets[index])
	}
	return result, nil
}

func (r *AssetRepositoryImpl) GetAssetDetail(ctx context.Context, id uint) (*domain.Asset, error) {
	asset, err := r.AssetDao.GetAssetDetail(ctx, id)
	if err != nil {
		return nil, err
	}
	result := convertAssetToDomain(asset)
	return &result, nil
}

func convertAssetToDomain(asset dao.Asset) domain.Asset {
	return domain.Asset{
		ID:          asset.ID,
		Name:        asset.Name,
		ProjectID:   asset.ProjectID,
		Type:        domain.AssetType(asset.Type),
		Description: asset.Description,
		Tags:        append([]string(nil), asset.Tags...),
		Attributes:  append([]byte(nil), asset.Attributes...),
		Version:     asset.Version,
	}
}

func (r *AssetRepositoryImpl) UpdateTags(ctx context.Context, id uint, tags []string) ([]string, error) {
	_, err := r.AssetDao.UpdateTags(ctx, id, tags)
	return []string{}, err
}

func (r *AssetRepositoryImpl) CreateCharacterAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	re, err := r.AssetDao.CreateAsset(ctx, &dao.Asset{})
	if err != nil {
		return 0, err
	}

	_, err = r.VersionDao.CreateAssetVersion(ctx, &dao.AssetVersion{AssetID: re.ID, Version: re.Version})
	if err != nil {
		return 0, err
	}

	_, err = r.ResourceDao.CreateAssetResources(ctx, []dao.AssetResource{{AssetID: re.ID, AssetVersion: re.Version, Type: string(domain.AssetResourceTypeProtoType)}})
	return 0, err
}

func (r *AssetRepositoryImpl) CreateObjectAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	re, err := r.AssetDao.CreateAsset(ctx, &dao.Asset{})
	if err != nil {
		return 0, err
	}

	_, err = r.VersionDao.CreateAssetVersion(ctx, &dao.AssetVersion{AssetID: re.ID, Version: re.Version})
	if err != nil {
		return 0, err
	}

	_, err = r.ResourceDao.CreateAssetResources(ctx, []dao.AssetResource{{AssetID: re.ID, AssetVersion: re.Version, Type: string(domain.AssetResourceTypeProtoType)}})
	return 0, err
}

func (r *AssetRepositoryImpl) CreateTileSetAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	// This should run inside a transaction.
	re, err := r.AssetDao.CreateAsset(ctx, &dao.Asset{})
	if err != nil {
		return 0, err
	}

	_, err = r.VersionDao.CreateAssetVersion(ctx, &dao.AssetVersion{AssetID: re.ID, Version: re.Version})
	if err != nil {
		return 0, err
	}

	// The count is determined by the length of the items array in asset.Attributes; using a default of one for now.
	_, err = r.ResourceDao.CreateAssetResources(ctx, []dao.AssetResource{{AssetID: re.ID, AssetVersion: re.Version, Type: string(domain.AssetResourceTypeItem)}})
	return 0, err
}

func (r *AssetRepositoryImpl) CreateUIAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	// This should run inside a transaction.
	re, err := r.AssetDao.CreateAsset(ctx, &dao.Asset{})
	if err != nil {
		return 0, err
	}

	_, err = r.VersionDao.CreateAssetVersion(ctx, &dao.AssetVersion{AssetID: re.ID, Version: re.Version})
	return 0, err
}

func (r *AssetRepositoryImpl) CreateSceneryAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	// This should run inside a transaction.
	re, err := r.AssetDao.CreateAsset(ctx, &dao.Asset{})
	if err != nil {
		return 0, err
	}

	_, err = r.VersionDao.CreateAssetVersion(ctx, &dao.AssetVersion{AssetID: re.ID, Version: re.Version})
	if err != nil {
		return 0, err
	}
	return 0, err
}

func (r *AssetRepositoryImpl) GetProtoTypeResource(ctx context.Context, assetID uint, version uint) ([]domain.AssetResource, error) {
	// This should run inside a transaction.
	p, err := r.ResourceDao.GetProtoTypeResource(ctx, assetID, version)
	if err != nil {
		return []domain.AssetResource{}, err
	}
	_, err = r.ResourceDao.GetImageResources(ctx, p.ID)

	return []domain.AssetResource{}, err
}

func (r *AssetRepositoryImpl) GetAnimations(ctx context.Context, assetID uint, version uint) ([]domain.AssetResource, error) {
	_, err := r.ResourceDao.GetAnimations(ctx, assetID, version)
	return []domain.AssetResource{}, err
}

func (r *AssetRepositoryImpl) GetItemResources(ctx context.Context, assetID uint, version uint) ([]domain.AssetResource, error) {
	_, err := r.ResourceDao.GetItemResources(ctx, assetID, version)
	return []domain.AssetResource{}, err
}

func (r *AssetRepositoryImpl) CreateImageResources(ctx context.Context, resource []domain.AssetResource) ([]domain.AssetResource, error) {
	_, err := r.ResourceDao.CreateAssetResources(ctx, []dao.AssetResource{})
	return []domain.AssetResource{}, err
}

func (r *AssetRepositoryImpl) CreateAnimationResource(ctx context.Context, resource *domain.AssetResource) (uint, error) {
	_, err := r.ResourceDao.CreateAssetResources(ctx, []dao.AssetResource{})
	return 0, err
}

func (r *AssetRepositoryImpl) UpdateFrameResources(ctx context.Context, resource []domain.AssetResource) error {
	return r.ResourceDao.UpdateFrameResources(ctx, []dao.AssetResource{})
}

func (r *AssetRepositoryImpl) DeleteFrameResourcesByAnimationID(ctx context.Context, id uint) error {
	return r.ResourceDao.DeleteFrameResourcesByAnimationID(ctx, id)
}

func (r *AssetRepositoryImpl) CreateRecord(ctx context.Context, version *domain.AssetVersion) (*domain.AssetVersion, error) {
	// This should run inside a transaction.
	_, err := r.VersionDao.CreateAssetVersion(ctx, &dao.AssetVersion{})
	if err != nil {
		return &domain.AssetVersion{}, err
	}

	resources, err := r.ResourceDao.GetResourcesByAssetVersion(ctx, version.AssetID, version.Version)
	if err != nil {
		return &domain.AssetVersion{}, err
	}

	err = r.AssetDao.UpdateAssetVersion(ctx, version.AssetID, version.Version)
	if err != nil {
		return &domain.AssetVersion{}, err
	}

	err = r.ResourceDao.CopyResourcesToVersion(ctx, resources, version.Version)

	return &domain.AssetVersion{}, err
}

func (r *AssetRepositoryImpl) RollBackRecord(ctx context.Context, assetID uint, version uint) (uint, error) {
	err := r.AssetDao.UpdateAssetVersion(ctx, assetID, version)
	return 0, err
}

func (r *AssetRepositoryImpl) Copy(ctx context.Context, assetID uint, version uint) (uint, error) {
	// This should run inside a transaction.
	original, err := r.AssetDao.GetAssetDetail(ctx, assetID)
	if err != nil {
		return 0, err
	}

	newAsset, err := r.AssetDao.CreateAsset(ctx, &dao.Asset{
		Name:        original.Name,
		ProjectID:   original.ProjectID,
		Type:        original.Type,
		Description: original.Description,
		Tags:        original.Tags,
		Attributes:  original.Attributes,
		Version:     version,
	})
	if err != nil {
		return 0, err
	}

	resources, err := r.ResourceDao.GetResourcesByAssetVersion(ctx, assetID, version)
	if err != nil {
		return 0, err
	}

	versions, err := r.VersionDao.GetAssetVersionsByAssetID(ctx, assetID)
	if err != nil {
		return 0, err
	}

	if len(resources) > 0 {
		for i := range resources {
			resources[i].AssetID = newAsset.ID
		}
		if _, err := r.ResourceDao.CreateAssetResources(ctx, resources); err != nil {
			return 0, err
		}
	}

	if len(versions) > 0 {
		for i := range versions {
			versions[i].AssetID = newAsset.ID
		}
		if err := r.VersionDao.CreateAssetVersions(ctx, versions); err != nil {
			return 0, err
		}
	}

	return newAsset.ID, nil
}
