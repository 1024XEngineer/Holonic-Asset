package dto

import (
	"encoding/json"
	"time"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
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

type GetAssetsRequest struct {
	Query string             `query:"query"`
	Tags  []string           `query:"tags"`
	Types []domain.AssetType `query:"types"`
}

type AssetDetailResponse struct {
	AssetID     uint             `json:"assetId"`
	Name        string           `json:"name"`
	ProjectID   uint             `json:"projectId"`
	Type        domain.AssetType `json:"type"`
	Description string           `json:"description"`
	Tags        []string         `json:"tags"`
	Attributes  json.RawMessage  `json:"attributes"`
	Content     json.RawMessage  `json:"content,omitempty"`
	Version     uint             `json:"version"`
}

type RecordAssetRequest struct {
	AssetID uint `json:"assetId"`
}

type RecordAssetResponse struct {
	RecordID  uint            `json:"recordId"`
	AssetID   uint            `json:"assetId"`
	Version   uint            `json:"version"`
	ContentID uint            `json:"contentId"`
	CreatedAt time.Time       `json:"createdAt"`
	Content   json.RawMessage `json:"content,omitempty"`
}

type AssetRecordResponse = RecordAssetResponse

type GetAssetRecordsResponse struct {
	Records []AssetRecordResponse `json:"records"`
}

type CopyAssetRequest struct {
	AssetID uint `json:"assetId"`
}

type CopyAssetResponse struct {
	NewAssetID uint `json:"newAssetId"`
}

type UpdateAssetRequest struct {
	AssetID     uint              `json:"assetId"`
	Name        *string           `json:"name,omitempty"`
	ProjectID   *uint             `json:"projectId,omitempty"`
	Type        *domain.AssetType `json:"type,omitempty"`
	Description *string           `json:"description,omitempty"`
	Tags        *[]string         `json:"tags,omitempty"`
	Attributes  *json.RawMessage  `json:"attributes,omitempty"`
}

type UpdateAssetResponse struct {
	AssetID     uint             `json:"assetId"`
	Name        string           `json:"name"`
	ProjectID   uint             `json:"projectId"`
	Type        domain.AssetType `json:"type"`
	Description string           `json:"description"`
	Tags        []string         `json:"tags"`
	Attributes  json.RawMessage  `json:"attributes"`
	Version     uint             `json:"version"`
}

type RollBackAssetRequest struct {
	AssetID uint `json:"assetId"`
	Version uint `json:"version"`
}

type RollBackAssetResponse struct {
	AssetID   uint `json:"assetId"`
	Version   uint `json:"version"`
	ContentID uint `json:"contentId"`
}
