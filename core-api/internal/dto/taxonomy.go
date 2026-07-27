package dto

import assetmodel "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"

type SearchAssetsRequest struct {
	ProjectID uint                   `param:"projectId"`
	Query     string                 `query:"q"`
	Tags      []string               `query:"tags"`
	Types     []assetmodel.AssetType `query:"types"`
}

type FindRelatedAssetsByTagsRequest struct {
	ProjectID uint `query:"projectId"`
	AssetID   uint `query:"assetId"`
}

type FilterAssetsRequest struct {
	ProjectID uint                   `query:"projectId"`
	Tags      []string               `query:"tags"`
	Types     []assetmodel.AssetType `query:"types"`
}

type FindRelatedAssetsRequest struct {
	ProjectID uint `param:"projectId"`
	AssetID   uint `param:"assetId"`
}

type AssetSearchItem struct {
	ID          uint                 `json:"id"`
	Name        string               `json:"name"`
	ProjectID   uint                 `json:"projectId"`
	Type        assetmodel.AssetType `json:"type"`
	Description string               `json:"description"`
	Tags        []string             `json:"tags"`
	Version     uint                 `json:"version"`
}

type AssetSearchResult struct {
	Assets []AssetSearchItem `json:"assets"`
}
