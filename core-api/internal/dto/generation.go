package dto

import (
	"encoding/json"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
)

type GenerationStatus string

type CreateGenerationRequest struct {
	ProjectID        uint               `param:"project_id" path:"project_id" json:"-" minimum:"1"`
	AssetID          *uint              `json:"assetId,omitempty" minimum:"1"`
	Kind             generator.TaskType `json:"kind" enum:"generate_character_prototype,edit_character_prototype,edit_frames,generate_object_prototype,edit_object_prototype,generate_animation,edit_animation,generate_scenery,generate_tileset,edit_tileset_item,edit_tiles"`
	CreativeBrief    string             `json:"creative_brief" minLength:"1"`
	TargetAssetPaths []string           `json:"targetAssetPaths,omitempty"`
	Parameters       json.RawMessage    `json:"parameters,omitempty"`
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
	Kind      generator.TaskType `json:"kind" enum:"generate_character_prototype,edit_character_prototype,edit_frames,generate_object_prototype,edit_object_prototype,generate_animation,edit_animation,generate_scenery,generate_tileset,edit_tileset_item,edit_tiles"`
	Status    GenerationStatus   `json:"status" enum:"pending,processing,awaiting_application,completed,failed,cancelled"`
}

type ListGenerationRunsResponse struct {
	Items      []GenerationRunListItemResponse `json:"items" nullable:"false"`
	NextCursor string                          `json:"nextCursor,omitempty"`
}

type GetGenerationRequest struct {
	GenerationRunID generator.RunID `param:"run_id" path:"run_id" json:"-" minimum:"1"`
}

type GenerationResult struct {
	AssetID     uint            `json:"asset_id,omitempty" minimum:"1"`
	AnimationID uint            `json:"animation_id,omitempty" minimum:"1"`
	Version     uint            `json:"version,omitempty" minimum:"1"`
	Content     json.RawMessage `json:"content,omitempty"`
}

type GetGenerationResponse struct {
	ID        generator.RunID    `json:"id" minimum:"1"`
	ProjectID uint               `json:"projectId" minimum:"1"`
	AssetID   *uint              `json:"assetId,omitempty" minimum:"1"`
	Kind      generator.TaskType `json:"kind" enum:"generate_character_prototype,edit_character_prototype,edit_frames,generate_object_prototype,edit_object_prototype,generate_animation,edit_animation,generate_scenery,generate_tileset,edit_tileset_item,edit_tiles"`
	Status    GenerationStatus   `json:"status" enum:"pending,processing,awaiting_application,completed,failed,cancelled"`
	Result    *GenerationResult  `json:"result,omitempty"`
	Error     string             `json:"error,omitempty"`
}

type CancelGenerationRequest struct {
	GenerationRunID generator.RunID `param:"run_id" path:"run_id" json:"-" minimum:"1"`
}

type CancelGenerationResponse struct {
	Cancelled bool `json:"cancelled"`
}

type ResolveGenerationApplicationRequest struct {
	GenerationRunID generator.RunID `param:"run_id" path:"run_id" json:"-" minimum:"1"`
	Applied         bool            `json:"applied"`
}
