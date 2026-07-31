package dto

import (
	"encoding/json"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

type CreateGenerationRequest struct {
	ProjectID         uint               `param:"project_id" path:"project_id" json:"-" minimum:"1"`
	AssetID           *uint              `json:"assetId,omitempty" minimum:"1"`
	Kind              generator.TaskType `json:"kind" enum:"generate_character_prototype,generate_character_animation,regenerate_character_prototype,regenerate_character_animation,regenerate_character_frames,generate_object_prototype,generate_object_animation,regenerate_object_prototype,regenerate_object_animation,regenerate_object_frames,generate_tileset,regenerate_item,regenerate_tiles"`
	Prompt            string             `json:"prompt"`
	ReferenceMediaIDs []string           `json:"referenceMediaIds,omitempty"`
	TargetAssetPaths  []string           `json:"targetAssetPaths,omitempty"`
	Parameters        json.RawMessage    `json:"parameters,omitempty"`
}

type CreateGenerationResponse struct {
	GenerationRunID generator.RunID `json:"generationRunId" minimum:"1"`
}

type ListGenerationRunsRequest struct {
	ProjectID uint                    `param:"project_id" path:"project_id" json:"-" minimum:"1"`
	AssetID   *uint                   `query:"assetId" minimum:"1"`
	Status    generator.RunListStatus `query:"status" enum:"active"`
	Limit     int                     `query:"limit" minimum:"1" maximum:"100"`
	Cursor    string                  `query:"cursor"`
}

type GenerationRunListItemResponse struct {
	ID        generator.RunID    `json:"id" minimum:"1"`
	ProjectID uint               `json:"projectId" minimum:"1"`
	AssetID   *uint              `json:"assetId,omitempty" minimum:"1"`
	Kind      generator.TaskType `json:"kind" enum:"generate_character_prototype,generate_character_animation,regenerate_character_prototype,regenerate_character_animation,regenerate_character_frames,generate_object_prototype,generate_object_animation,regenerate_object_prototype,regenerate_object_animation,regenerate_object_frames,generate_tileset,regenerate_item,regenerate_tiles"`
	Status    taskdomain.Status  `json:"status" enum:"0,1,2,3,4"`
}

type ListGenerationRunsResponse struct {
	Items      []GenerationRunListItemResponse `json:"items" nullable:"false"`
	NextCursor string                          `json:"nextCursor,omitempty"`
}

type GetGenerationRequest struct {
	GenerationRunID generator.RunID `param:"run_id" path:"run_id" json:"-" minimum:"1"`
}

type GetGenerationResponse struct {
	ID        generator.RunID    `json:"id" minimum:"1"`
	ProjectID uint               `json:"projectId" minimum:"1"`
	AssetID   *uint              `json:"assetId,omitempty" minimum:"1"`
	Kind      generator.TaskType `json:"kind" enum:"generate_character_prototype,generate_character_animation,regenerate_character_prototype,regenerate_character_animation,regenerate_character_frames,generate_object_prototype,generate_object_animation,regenerate_object_prototype,regenerate_object_animation,regenerate_object_frames,generate_tileset,regenerate_item,regenerate_tiles"`
	Status    taskdomain.Status  `json:"status" enum:"0,1,2,3,4"`
	Result    json.RawMessage    `json:"result,omitempty"`
	Error     string             `json:"error,omitempty"`
}

type CancelGenerationRequest struct {
	GenerationRunID generator.RunID `param:"run_id" path:"run_id" json:"-" minimum:"1"`
}

type CancelGenerationResponse struct {
	Cancelled bool `json:"cancelled"`
}
