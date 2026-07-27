package service

import assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"

// SearchAssetsRequest contains project-scoped text and taxonomy criteria.
type SearchAssetsRequest struct {
	ProjectID uint
	Query     string
	Tags      []string
	Types     []assetdomain.AssetType
}

// FindRelatedAssetsByTagsRequest identifies the asset whose tags seed discovery.
type FindRelatedAssetsByTagsRequest struct {
	ProjectID uint
	AssetID   uint
}

// FilterAssetsRequest contains project-scoped structured taxonomy criteria.
type FilterAssetsRequest struct {
	ProjectID uint
	Tags      []string
	Types     []assetdomain.AssetType
}

// FindRelatedAssetsRequest identifies the asset that seeds semantic discovery.
type FindRelatedAssetsRequest struct {
	ProjectID uint
	AssetID   uint
}

// AssetSearchItem is the public asset summary returned by discovery endpoints.
type AssetSearchItem struct {
	ID          uint
	Name        string
	ProjectID   uint
	Type        assetdomain.AssetType
	Description string
	Tags        []string
	Version     uint
}

// AssetSearchResult is the shared result contract for asset discovery.
type AssetSearchResult struct {
	Assets []AssetSearchItem
}
