package dto

import (
	"encoding/json"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
)

type AssetListItemResponse struct {
	AssetID     uint             `json:"assetId"`
	Name        string           `json:"name"`
	ProjectID   uint             `json:"projectId"`
	Type        domain.AssetType `json:"type"`
	Description string           `json:"description"`
	Tags        []string         `json:"tags"`
	Version     uint             `json:"version"`
}

type GetAssetsResponse struct {
	Assets []AssetListItemResponse `json:"assets"`
}

type AssetDetailResponse struct {
	AssetID     uint             `json:"assetId"`
	Name        string           `json:"name"`
	ProjectID   uint             `json:"projectId"`
	Type        domain.AssetType `json:"type"`
	Description string           `json:"description"`
	Tags        []string         `json:"tags"`
	Attributes  json.RawMessage  `json:"attributes"`
	Version     uint             `json:"version"`
}

type GetAssetResourcesRequest struct {
	AssetID uint
	Version uint
}

type GetAssetResourcesResponse struct {
	Resources []domain.AssetResource `json:"resources"`
}

type CreateCharacterAssetRequest struct {
	Asset *domain.Asset
}

type CreateCharacterAssetResponse struct {
	ID uint `json:"id"`
}

type CreateObjectAssetRequest struct {
	Asset *domain.Asset
}

type CreateObjectAssetResponse struct {
	ID uint `json:"id"`
}

type CreateTileSetAssetRequest struct {
	Asset *domain.Asset
}

type CreateTileSetAssetResponse struct {
	ID uint `json:"id"`
}

type CreateAnimationRequest struct {
	Name    string
	AssetID uint
	Type    domain.AssetType
}

type CreateAnimationResponse struct {
	ID uint `json:"id"`
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
	NewAssetID uint `json:"newAssetId"`
}

type AddTagsRequest struct {
	AssetID uint
	Tags    []string
}

type AddTagsResponse struct {
	Tags []string `json:"tags"`
}

type RollBackAssetRequest struct {
	AssetID uint
	Version uint
}

type RollBackAssetResponse struct {
	Asset *domain.Asset `json:"asset"`
}
