package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

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

type ProcessedSceneryLayer struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	ImageBase64 string `json:"image_base64"`
	MediaType   string `json:"media_type"`
}

const (
	sceneryLayerPlanSchemaName   = "scenery_layer_plan"
	sceneryLayerLayoutSchemaName = "scenery_layer_layout"
	sceneryBatchIDBytes          = 16
	sceneryCleanupTTL            = 15 * time.Second
)

var sceneryLayerPlanJSONSchema = json.RawMessage(`{
  "type": "object",
  "additionalProperties": false,
  "required": ["layers"],
  "properties": {
    "layers": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["name", "creative_brief"],
        "properties": {
          "name": {"type": "string", "minLength": 1},
          "creative_brief": {"type": "string", "minLength": 1}
        }
      }
    }
  }
}`)

var sceneryLayerLayoutJSONSchema = json.RawMessage(`{
  "type":"object","additionalProperties":false,"required":["layers"],"properties":{"layers":{
    "type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,
      "required":["id","position","scale","rotation","opacity","zIndex"],"properties":{
        "id":{"type":"integer","minimum":1},
        "position":{"type":"object","additionalProperties":false,"required":["x","y"],"properties":{"x":{"type":"number"},"y":{"type":"number"}}},
        "scale":{"type":"object","additionalProperties":false,"required":["x","y"],"properties":{"x":{"type":"number","exclusiveMinimum":0},"y":{"type":"number","exclusiveMinimum":0}}},
        "rotation":{"type":"number"},"opacity":{"type":"number","minimum":0,"maximum":1},"zIndex":{"type":"integer"}
      }
    }
  }}
}`)

type sceneryLayerPlanResponse struct {
	Layers *[]sceneryLayerPlanCandidate `json:"layers"`
}

type sceneryLayerPlanCandidate struct {
	Name          *string `json:"name"`
	CreativeBrief *string `json:"creative_brief"`
}

type SceneryLayoutVector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type SceneryLayerLayout struct {
	Position SceneryLayoutVector `json:"position"`
	Scale    SceneryLayoutVector `json:"scale"`
	Rotation float64             `json:"rotation"`
	Opacity  float64             `json:"opacity"`
	ZIndex   int                 `json:"zIndex"`
}

type LaidOutSceneryLayer struct {
	ID          uint               `json:"id"`
	Name        string             `json:"name"`
	ImageBase64 string             `json:"image_base64"`
	MediaType   string             `json:"media_type"`
	Layout      SceneryLayerLayout `json:"layout"`
}

type sceneryLayoutResponse struct {
	Layers *[]sceneryLayoutCandidate `json:"layers"`
}

type sceneryLayoutCandidate struct {
	ID       *uint                         `json:"id"`
	Position *sceneryLayoutVectorCandidate `json:"position"`
	Scale    *sceneryLayoutVectorCandidate `json:"scale"`
	Rotation *float64                      `json:"rotation"`
	Opacity  *float64                      `json:"opacity"`
	ZIndex   *int                          `json:"zIndex"`
}

type sceneryLayoutVectorCandidate struct {
	X *float64 `json:"x"`
	Y *float64 `json:"y"`
}

type sceneryTransform struct {
	Scale    SceneryLayoutVector `json:"scale"`
	Rotation float64             `json:"rotation"`
}

func decodeSceneryLayerPlan(raw []byte) ([]SceneryLayerDefinition, error) {
	invalid := func(reason string) error {
		return fmt.Errorf("%w: %s", ErrInvalidSceneryPlan, reason)
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var response sceneryLayerPlanResponse
	if err := decoder.Decode(&response); err != nil {
		return nil, invalid(err.Error())
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, invalid("trailing data")
		}
		return nil, invalid(err.Error())
	}
	if response.Layers == nil {
		return nil, invalid("layers is required")
	}
	if len(*response.Layers) == 0 {
		return nil, invalid("at least one layer is required")
	}

	layers := make([]SceneryLayerDefinition, len(*response.Layers))
	names := make(map[string]struct{}, len(layers))
	for index, candidate := range *response.Layers {
		if candidate.Name == nil {
			return nil, invalid(fmt.Sprintf("layer %d name is required", index+1))
		}
		name := strings.TrimSpace(*candidate.Name)
		if name == "" {
			return nil, invalid(fmt.Sprintf("layer %d name is required", index+1))
		}
		normalizedName := strings.ToLower(name)
		if _, duplicate := names[normalizedName]; duplicate {
			return nil, invalid(fmt.Sprintf("layer name %q is duplicated", name))
		}
		names[normalizedName] = struct{}{}

		if candidate.CreativeBrief == nil {
			return nil, invalid(fmt.Sprintf("layer %d creative brief is required", index+1))
		}
		creativeBrief := strings.TrimSpace(*candidate.CreativeBrief)
		if creativeBrief == "" {
			return nil, invalid(fmt.Sprintf("layer %d creative brief is required", index+1))
		}
		layers[index] = SceneryLayerDefinition{
			ID:            uint(index + 1),
			Name:          name,
			CreativeBrief: creativeBrief,
		}
	}
	return layers, nil
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
