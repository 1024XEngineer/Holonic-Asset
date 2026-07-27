package service

import (
	"context"
)

// AssetDiscoveryService defines project-scoped tag lookup, filtering, and semantic discovery.
type AssetDiscoveryService interface {
	SearchAssets(
		ctx context.Context,
		request *SearchAssetsRequest,
	) (*AssetSearchResult, error)
	FindRelatedAssetsByTags(
		ctx context.Context,
		request *FindRelatedAssetsByTagsRequest,
	) (*AssetSearchResult, error)
	FilterAssets(
		ctx context.Context,
		request *FilterAssetsRequest,
	) (*AssetSearchResult, error)
	FindRelatedAssets(
		ctx context.Context,
		request *FindRelatedAssetsRequest,
	) (*AssetSearchResult, error)
}

// assetDiscoveryService is empty because Taxonomy currently has only placeholder
// behavior and, unlike implemented services, does not yet depend on a repository or provider.
type assetDiscoveryService struct{}

func NewAssetDiscoveryService() AssetDiscoveryService {
	return &assetDiscoveryService{}
}

func (*assetDiscoveryService) SearchAssets(
	context.Context,
	*SearchAssetsRequest,
) (*AssetSearchResult, error) {
	return emptyAssetSearchResult(), nil
}

func (*assetDiscoveryService) FindRelatedAssetsByTags(
	context.Context,
	*FindRelatedAssetsByTagsRequest,
) (*AssetSearchResult, error) {
	return emptyAssetSearchResult(), nil
}

func (*assetDiscoveryService) FilterAssets(
	context.Context,
	*FilterAssetsRequest,
) (*AssetSearchResult, error) {
	return emptyAssetSearchResult(), nil
}

func (*assetDiscoveryService) FindRelatedAssets(
	context.Context,
	*FindRelatedAssetsRequest,
) (*AssetSearchResult, error) {
	return emptyAssetSearchResult(), nil
}

func emptyAssetSearchResult() *AssetSearchResult {
	return &AssetSearchResult{Assets: []AssetSearchItem{}}
}

var _ AssetDiscoveryService = (*assetDiscoveryService)(nil)
