package dto

import (
	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
)

type GetAssetsResponse struct {
	AssetID     uint
	Name        string
	ProjectID   uint
	Type        domain.AssetType
	Description string
	Tags        []string
	Version     uint
}

type AssetDetailResponse struct {
}

type GetAssetResourcesRequest struct {
	AssetID uint
	Version uint
}

type GetAssetResourcesResponse struct {
	Resources []domain.AssetResource
}

type CreateCharacterAssetRequest struct {
	Asset *domain.Asset
}

type CreateCharacterAssetResponse struct {
	ID uint
}

type CreateObjectAssetRequest struct {
	Asset *domain.Asset
}

type CreateObjectAssetResponse struct {
	ID uint
}

type CreateTileSetAssetRequest struct {
	Asset *domain.Asset
}

type CreateTileSetAssetResponse struct {
	ID uint
}

type CreateAnimationRequest struct {
	Name    string
	AssetID uint
	Type    domain.AssetType
}

type CreateAnimationResponse struct {
	ID uint
}

type RecordAssetRequest struct {
	AssetID uint
}

type RecordAssetResponse struct {
}

type CopyAssetRequest struct {
	AssetID uint
}

type CopyAssetResponse struct {
	NewAssetID uint
}

type AddTagsRequest struct {
	AssetID uint
	Tags    []string
}

type AddTagsResponse struct {
	Tags []string
}

type RollBackAssetRequest struct {
	AssetID uint
	Version uint
}

type RollBackAssetResponse struct {
	Asset *domain.Asset
}
