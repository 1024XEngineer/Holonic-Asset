package export

import (
	"context"
	"encoding/json"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const TaskType = "export_asset"

const FormatVersion = 1

type CreateRequest struct {
	AssetID uint `json:"assetId"`
	Version uint `json:"version,omitempty"`
}

type CreateResponse struct {
	ExportID uint   `json:"exportId" minimum:"1"`
	TaskID   uint   `json:"taskId" minimum:"1"`
	Status   string `json:"status"`
}

type ExportResponse struct {
	ExportID    uint   `json:"exportId" minimum:"1"`
	AssetID     uint   `json:"assetId" minimum:"1"`
	RecordID    uint   `json:"recordId,omitempty"`
	Version     uint   `json:"version" minimum:"1"`
	Status      string `json:"status"`
	FileName    string `json:"fileName,omitempty"`
	FileSize    int64  `json:"fileSize,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	DownloadURL string `json:"downloadUrl,omitempty"`
	Error       string `json:"error,omitempty"`
}

type Snapshot struct {
	AssetID     uint
	ProjectID   uint
	RecordID    uint
	Version     uint
	Name        string
	Description string
	Type        assetdomain.AssetType
	Perspective assetdomain.Perspective
	Dimensions  json.RawMessage
	Content     json.RawMessage
}

type taskPayload struct {
	Snapshot  Snapshot `json:"snapshot"`
	ObjectKey string   `json:"objectKey"`
}

type taskResult struct {
	AssetID     uint   `json:"assetId"`
	RecordID    uint   `json:"recordId,omitempty"`
	Version     uint   `json:"version"`
	FileName    string `json:"fileName"`
	FileSize    int64  `json:"fileSize"`
	SHA256      string `json:"sha256"`
	ObjectKey   string `json:"objectKey"`
	DownloadURL string `json:"downloadUrl"`
}

type AssetReader interface {
	GetDetail(context.Context, uint) (assetdomain.Asset, error)
	GetRecordHistory(context.Context, uint) ([]assetdomain.AssetRecord, error)
}

type ReferenceResolver interface {
	ResolveReference(context.Context, string) (string, error)
}

type ArtifactStore interface {
	PutArtifact(context.Context, string, string, []byte) error
	ResolveReference(context.Context, string) (string, error)
}

type TaskManager interface {
	Publish(context.Context, *taskdomain.Task) (uint, error)
	GetDetail(context.Context, uint) (*taskdomain.Task, error)
}
