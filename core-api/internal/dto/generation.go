package dto

import (
	"encoding/json"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/generation"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

type CreateGenerationRequest struct {
	ProjectID         uint            `param:"project_id" json:"-"`
	AssetID           *uint           `json:"assetId,omitempty"`
	Kind              domain.TaskType `json:"kind"`
	Prompt            string          `json:"prompt"`
	ReferenceMediaIDs []string        `json:"referenceMediaIds,omitempty"`
	TargetAssetPaths  []string        `json:"targetAssetPaths,omitempty"`
	Parameters        json.RawMessage `json:"parameters,omitempty"`
}

type CreateGenerationResponse struct {
	GenerationRunID domain.RunID `json:"generationRunId"`
}

type ListGenerationRunsRequest struct {
	ProjectID uint                 `param:"project_id" json:"-"`
	AssetID   *uint                `query:"assetId"`
	Status    domain.RunListStatus `query:"status"`
	Limit     int                  `query:"limit"`
	Cursor    string               `query:"cursor"`
}

type GenerationRunListItemResponse struct {
	ID        domain.RunID      `json:"id"`
	ProjectID uint              `json:"projectId"`
	AssetID   *uint             `json:"assetId,omitempty"`
	Kind      domain.TaskType   `json:"kind"`
	Status    taskdomain.Status `json:"status"`
}

type ListGenerationRunsResponse struct {
	Items      []GenerationRunListItemResponse `json:"items"`
	NextCursor string                          `json:"nextCursor,omitempty"`
}

type GetGenerationRequest struct {
	GenerationRunID domain.RunID `param:"run_id" json:"-"`
}

type GetGenerationResponse struct {
	ID        domain.RunID      `json:"id"`
	ProjectID uint              `json:"projectId"`
	AssetID   *uint             `json:"assetId,omitempty"`
	Kind      domain.TaskType   `json:"kind"`
	Status    taskdomain.Status `json:"status"`
	Result    json.RawMessage   `json:"result,omitempty"`
	Error     string            `json:"error,omitempty"`
}

type CancelGenerationRequest struct {
	GenerationRunID domain.RunID `param:"run_id" json:"-"`
}

type CancelGenerationResponse struct {
	Cancelled bool `json:"cancelled"`
}
