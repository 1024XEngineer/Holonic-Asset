package generator

import (
	"encoding/json"
	"fmt"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type ExecutionResult struct {
	AssetID     uint `json:"asset_id"`
	AnimationID uint `json:"animation_id,omitempty"`
}

// CreateCharacterPrototypePayload is the complete input consumed by the
// character prototype task handler.
type CreateCharacterPrototypePayload struct {
	AssetName     string           `json:"asset_name"`
	CreativeBrief string           `json:"creative_brief"`
	Dimensions    assetdomain.Size `json:"dimensions"`
	Perspective   string           `json:"perspective"`
	Reference     string           `json:"reference"`
	ProjectID     uint             `json:"project_id"`
}

// CreateAnimationPayload is the common input consumed by character and object
// animation generation.
type CreateAnimationPayload struct {
	AssetName     string `json:"asset_name"`
	ProjectID     uint   `json:"project_id"`
	ParentID      uint   `json:"parent_id"`
	CreativeBrief string `json:"creative_brief"`
}

// CreateObjectPrototypePayload is the complete input consumed by the object
// prototype task handler.
type CreateObjectPrototypePayload struct {
	AssetName     string           `json:"asset_name"`
	CreativeBrief string           `json:"creative_brief"`
	Dimensions    assetdomain.Size `json:"dimensions"`
	Perspective   string           `json:"perspective"`
	Reference     string           `json:"reference"`
	ProjectID     uint             `json:"project_id"`
}

// SceneryLayerDefinition is produced by the pre-generation scenery planner.
type SceneryLayerDefinition struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	CreativeBrief string `json:"creative_brief"`
}

type SceneryProjectContext struct {
	Name           string `json:"name,omitempty"`
	GameType       string `json:"game_type,omitempty"`
	TargetPlatform string `json:"target_platform,omitempty"`
	Description    string `json:"description,omitempty"`
}

// CreateSceneryPayload is the queued, self-contained input for Scenery
// generation. The caller supplies one overall creative brief; Layers are
// selected later by the LLM planner and are deliberately absent here.
type CreateSceneryPayload struct {
	AssetName      string                `json:"asset_name"`
	CreativeBrief  string                `json:"creative_brief"`
	Style          string                `json:"style"`
	Dimensions     assetdomain.Size      `json:"dimensions"`
	Perspective    string                `json:"perspective"`
	ProjectContext SceneryProjectContext `json:"project_context"`
	Reference      string                `json:"reference"`
	ProjectID      uint                  `json:"project_id"`
}

func decodeExecutionPayload(taskType TaskType, payload json.RawMessage, target any) error {
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("generator: decode %s execution payload: %w", taskType, err)
	}
	return nil
}

func encodeExecutionResult(result ExecutionResult) (json.RawMessage, error) {
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("generator: encode execution result: %w", err)
	}
	return encoded, nil
}

// CreateTileSetPayload is the complete input consumed by the tileset task
// handler. TileNum is stored explicitly so a queued task is self-contained.
type CreateTileSetPayload struct {
	AssetName        string   `json:"asset_name"`
	ProjectID        uint     `json:"project_id"`
	CreativeBrief    string   `json:"creative_brief"`
	TileNum          uint     `json:"tile_num"`
	TileDescriptions []string `json:"tile_descriptions"`
	Reference        string   `json:"reference"`
}
