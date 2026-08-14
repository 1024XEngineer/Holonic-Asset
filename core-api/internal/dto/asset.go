package dto

import (
	"encoding/json"
	"time"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type AssetListItemResponse struct {
	AssetID     uint               `json:"assetId" minimum:"1"`
	Name        string             `json:"name"`
	ProjectID   uint               `json:"projectId" minimum:"1"`
	Type        domain.AssetType   `json:"type" enum:"character,object,tileSet,audio,uiset,scenery"`
	Description string             `json:"description"`
	Perspective domain.Perspective `json:"perspective" enum:"Top-Down,Side-On,Isometric"`
	Dimensions  json.RawMessage    `json:"dimensions"`
	Tags        []string           `json:"tags"`
	Version     uint               `json:"version"`
}

type GetAssetsResponse struct {
	Assets []AssetListItemResponse `json:"assets" nullable:"false"`
}

type GetAssetsRequest struct {
	ProjectID uint               `param:"project_id" path:"project_id" json:"-" minimum:"1"`
	Query     string             `query:"query"`
	Tags      []string           `query:"tags,explode"`
	Types     []domain.AssetType `query:"types,explode" enum:"character,object,tileSet,audio,uiset,scenery"`
}

type AssetDetailRequest struct {
	AssetID uint `param:"asset_id" path:"asset_id" json:"-" minimum:"1"`
}

type AssetDetailResponse struct {
	AssetID     uint               `json:"assetId" minimum:"1"`
	Name        string             `json:"name"`
	ProjectID   uint               `json:"projectId" minimum:"1"`
	Type        domain.AssetType   `json:"type" enum:"character,object,tileSet,audio,uiset,scenery"`
	Description string             `json:"description"`
	Perspective domain.Perspective `json:"perspective" enum:"Top-Down,Side-On,Isometric"`
	Dimensions  json.RawMessage    `json:"dimensions"`
	Tags        []string           `json:"tags"`
	Content     json.RawMessage    `json:"content,omitempty"`
	Version     uint               `json:"version"`
}

type RecordAssetRequest struct {
	AssetID uint            `json:"assetId" minimum:"1"`
	Content json.RawMessage `json:"content"`
}

type RecordAssetResponse struct {
	RecordID    uint               `json:"recordId" minimum:"1"`
	AssetID     uint               `json:"assetId" minimum:"1"`
	Version     uint               `json:"version"`
	ContentID   uint               `json:"contentId" minimum:"1"`
	CreatedAt   time.Time          `json:"createdAt"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Perspective domain.Perspective `json:"perspective" enum:"Top-Down,Side-On,Isometric"`
	Dimensions  json.RawMessage    `json:"dimensions"`
	Content     json.RawMessage    `json:"content,omitempty"`
}

type AssetRecordResponse = RecordAssetResponse

type GetAssetRecordsResponse struct {
	Records []AssetRecordResponse `json:"records" nullable:"false"`
}

type GetAssetRecordsRequest struct {
	AssetID uint `param:"asset_id" path:"asset_id" json:"-" minimum:"1"`
}

type CopyAssetRequest struct {
	AssetID uint `json:"assetId" minimum:"1"`
}

type CopyAssetResponse struct {
	NewAssetID uint `json:"newAssetId" minimum:"1"`
}

type UpdateAssetRequest struct {
	AssetID     uint                `json:"assetId" minimum:"1"`
	Name        *string             `json:"name,omitempty"`
	Description *string             `json:"description,omitempty"`
	Tags        *[]string           `json:"tags,omitempty"`
	Perspective *domain.Perspective `json:"perspective,omitempty" enum:"Top-Down,Side-On,Isometric"`
	Dimensions  *json.RawMessage    `json:"dimensions,omitempty"`
}

type UpdateAssetResponse struct {
	AssetID     uint               `json:"assetId" minimum:"1"`
	Name        string             `json:"name"`
	ProjectID   uint               `json:"projectId" minimum:"1"`
	Type        domain.AssetType   `json:"type" enum:"character,object,tileSet,audio,uiset,scenery"`
	Description string             `json:"description"`
	Perspective domain.Perspective `json:"perspective" enum:"Top-Down,Side-On,Isometric"`
	Dimensions  json.RawMessage    `json:"dimensions"`
	Tags        []string           `json:"tags"`
	Version     uint               `json:"version"`
}

type RollBackAssetRequest struct {
	AssetID uint `json:"assetId" minimum:"1"`
	Version uint `json:"version" minimum:"1"`
}

type RollBackAssetResponse struct {
	AssetID   uint `json:"assetId" minimum:"1"`
	Version   uint `json:"version"`
	ContentID uint `json:"contentId" minimum:"1"`
}

type DeleteAssetRequest struct {
	AssetID uint `json:"assetId" minimum:"1"`
}

type DeleteAssetResponse struct {
	Success bool `json:"success"`
}
