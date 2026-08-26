package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const maxPrototypeReferenceImages = 5

// CreateCharacterPrototypePayload is the complete input consumed by the
// character prototype task handler.
type CreateCharacterPrototypePayload struct {
	AssetName         string            `json:"asset_name"`
	CreativeBrief     string            `json:"creative_brief"`
	Dimensions        assetdomain.Size  `json:"dimensions"`
	Perspective       string            `json:"perspective"`
	CreatingReference string            `json:"creating_reference"` // User-supplied subject or concept reference.
	ProjectReference  string            `json:"project_reference"`  // Backend-supplied project style reference.
	Tags              []assetdomain.Tag `json:"tags,omitempty"`
	NexusReferences   []string          `json:"nexus_references,omitempty"`
	ProjectID         uint              `json:"project_id"`
}

// EditCharacterPrototypePayload is the self-contained input consumed by the
// character prototype edit task handler.
type EditCharacterPrototypePayload struct {
	AssetID          uint   `json:"asset_id"`
	ProjectID        uint   `json:"project_id"`
	EditInstructions string `json:"edit_instructions"`
}

// EditObjectPrototypePayload is the self-contained input consumed by the
// object prototype edit task handler.
type EditObjectPrototypePayload struct {
	AssetID          uint   `json:"asset_id"`
	ProjectID        uint   `json:"project_id"`
	EditInstructions string `json:"edit_instructions"`
}

// CreateAnimationPayload is the common input consumed by character and object
// animation generation.
type CreateAnimationPayload struct {
	AnimationName string `json:"animation_name"`
	ProjectID     uint   `json:"project_id"`
	AssetID       uint   `json:"asset_id"`
	// Direction is an English name shared by character and object assets, such as
	// "front", "left", or "back_right". The name is resolved against the
	// asset's direction_count and prototype ordering.
	Direction     string `json:"direction"`
	CreativeBrief string `json:"creative_brief"`
	Style         string `json:"style,omitempty"`
	FrameCount    int    `json:"frame_count,omitempty"`
	FrameWidth    int    `json:"frame_width,omitempty"`
	FrameHeight   int    `json:"frame_height,omitempty"`
	FPS           int    `json:"fps,omitempty"`
	Resolution    string `json:"resolution,omitempty"`
	Duration      int    `json:"duration,omitempty"`
}

// EditAnimationPayload is the self-contained input consumed by the animation
// regeneration task handler. The generation parameters are loaded from the
// persisted animation content; only the latest creative brief is supplied by
// the caller.
type EditAnimationPayload struct {
	AssetID       uint   `json:"asset_id"`
	AnimationID   uint   `json:"animation_id"`
	ProjectID     uint   `json:"project_id"`
	CreativeBrief string `json:"creative_brief"`
}

// CreateObjectPrototypePayload is the complete input consumed by the object
// prototype task handler.
type CreateObjectPrototypePayload struct {
	AssetName         string            `json:"asset_name"`
	CreativeBrief     string            `json:"creative_brief"`
	Dimensions        assetdomain.Size  `json:"dimensions"`
	Perspective       string            `json:"perspective"`
	CreatingReference string            `json:"creating_reference"` // User-supplied subject or concept reference.
	ProjectReference  string            `json:"project_reference"`  // Backend-supplied project style reference.
	Tags              []assetdomain.Tag `json:"tags,omitempty"`
	NexusReferences   []string          `json:"nexus_references,omitempty"`
	ProjectID         uint              `json:"project_id"`
}

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

type CreateSceneryPayload struct {
	AssetName         string                `json:"asset_name"`
	CreativeBrief     string                `json:"creative_brief"`
	Dimensions        assetdomain.Size      `json:"dimensions"`
	Perspective       string                `json:"perspective"`
	ProjectContext    SceneryProjectContext `json:"project_context"`
	CreatingReference string                `json:"creating_reference"`
	ProjectReference  string                `json:"project_reference"`
	ProjectID         uint                  `json:"project_id"`
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

var sceneryLayerPlanJSONSchema = json.RawMessage(`{"type":"object","additionalProperties":false,"required":["layers"],"properties":{"layers":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"required":["name","creative_brief"],"properties":{"name":{"type":"string","minLength":1},"creative_brief":{"type":"string","minLength":1}}}}}}`)

var sceneryLayerLayoutJSONSchema = json.RawMessage(`{"type":"object","additionalProperties":false,"required":["layers"],"properties":{"layers":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"required":["id","position","scale","rotation","opacity","zIndex"],"properties":{"id":{"type":"integer","minimum":1},"position":{"type":"object","additionalProperties":false,"required":["x","y"],"properties":{"x":{"type":"number"},"y":{"type":"number"}}},"scale":{"type":"object","additionalProperties":false,"required":["x","y"],"properties":{"x":{"type":"number","exclusiveMinimum":0},"y":{"type":"number","exclusiveMinimum":0}}},"rotation":{"type":"number"},"opacity":{"type":"number","minimum":0,"maximum":1},"zIndex":{"type":"integer"}}}}}}`)

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
	invalid := func(reason string) error { return fmt.Errorf("%w: %s", ErrInvalidSceneryPlan, reason) }
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
	if response.Layers == nil || len(*response.Layers) == 0 {
		return nil, invalid("at least one layer is required")
	}
	layers := make([]SceneryLayerDefinition, len(*response.Layers))
	names := make(map[string]struct{}, len(layers))
	for index, candidate := range *response.Layers {
		if candidate.Name == nil || strings.TrimSpace(*candidate.Name) == "" {
			return nil, invalid(fmt.Sprintf("layer %d name is required", index+1))
		}
		name := strings.TrimSpace(*candidate.Name)
		key := strings.ToLower(name)
		if _, duplicate := names[key]; duplicate {
			return nil, invalid(fmt.Sprintf("layer name %q is duplicated", name))
		}
		names[key] = struct{}{}
		if candidate.CreativeBrief == nil || strings.TrimSpace(*candidate.CreativeBrief) == "" {
			return nil, invalid(fmt.Sprintf("layer %d creative brief is required", index+1))
		}
		layers[index] = SceneryLayerDefinition{ID: uint(index + 1), Name: name, CreativeBrief: strings.TrimSpace(*candidate.CreativeBrief)}
	}
	return layers, nil
}

// TileSetCoordinate is one occupied cell in an item's local grid.
type TileSetCoordinate [2]int

// TileSetItemDefinition describes one complete image generated for a Tileset
// item. Shape remains task input and is not persisted in Asset Content.
type TileSetItemDefinition struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Shape       []TileSetCoordinate `json:"shape"`
}

// CreateTileSetPayload is the complete input consumed by the Tileset task
// handler.
type CreateTileSetPayload struct {
	AssetName     string                        `json:"asset_name"`
	ProjectID     uint                          `json:"project_id"`
	CreativeBrief string                        `json:"creative_brief"`
	Dimensions    assetdomain.TileSetDimensions `json:"dimensions"`
	Items         []TileSetItemDefinition       `json:"items"`
}

// AddTileSetItemDefinition describes one Item to append to an existing
// Tileset. Origin is a global grid coordinate; when omitted, execution chooses
// the first available row-major placement.
type AddTileSetItemDefinition struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Shape       []TileSetCoordinate `json:"shape"`
	Origin      *TileSetOrigin      `json:"origin,omitempty"`
}

type TileSetOrigin struct {
	X *int `json:"x"`
	Y *int `json:"y"`
}

// AddTilesetItemPayload is the complete input consumed by an Item addition
// task. Dimensions and occupied positions are always loaded from the Asset.
type AddTilesetItemPayload struct {
	AssetID           uint                      `json:"asset_id"`
	ProjectID         uint                      `json:"project_id"`
	CreativeBrief     string                    `json:"creative_brief"`
	CreatingReference string                    `json:"creating_reference,omitempty"`
	Item              *AddTileSetItemDefinition `json:"item"`
}

// EditTilesetItemPayload is the complete input consumed by an Item edit task.
type EditTilesetItemPayload struct {
	AssetID           uint               `json:"asset_id"`
	ProjectID         uint               `json:"project_id"`
	CreativeBrief     string             `json:"creative_brief"`
	Target            *TileSetEditTarget `json:"target"`
	CreatingReference string             `json:"creating_reference,omitempty"`
}

// EditTilesPayload is the complete input consumed by a Tile edit task.
type EditTilesPayload struct {
	AssetID           uint                `json:"asset_id"`
	ProjectID         uint                `json:"project_id"`
	CreativeBrief     string              `json:"creative_brief"`
	Targets           []TileSetEditTarget `json:"targets"`
	CreatingReference string              `json:"creating_reference,omitempty"`
}

const (
	maxTileSetItems            = 64
	maxTilesPerItem            = 256
	maxTileSetGridTiles        = 4096
	maxTileEdge                = 1024
	maxGeneratedItemImageEdge  = 4096
	maxTileEditTargets         = 256
	maxAssetNameLength         = 200
	maxCreativeBriefLength     = 4000
	maxItemNameLength          = 200
	maxItemDescriptionLength   = 2000
	maxCreatingReferenceLength = 8 << 20
)

// TileSetEditTarget identifies an occupied global Tileset cell. Execution
// resolves the matching Tile and its owning Item after loading the Asset.
type TileSetEditTarget struct {
	Position *TileSetEditPosition `json:"position"`
}

type TileSetEditPosition struct {
	X *int `json:"x"`
	Y *int `json:"y"`
}

func validateCreateTileSetPayload(payload *CreateTileSetPayload) error {
	if payload == nil {
		return invalidTaskPayload("Tileset payload is required")
	}
	if payload.ProjectID == 0 {
		return invalidTaskPayload("project_id must be positive")
	}
	if err := validateRequiredText("asset_name", payload.AssetName, maxAssetNameLength); err != nil {
		return err
	}
	if err := validateRequiredText("creative_brief", payload.CreativeBrief, maxCreativeBriefLength); err != nil {
		return err
	}
	if err := validateTileSetDimensions(payload); err != nil {
		return err
	}
	if len(payload.Items) == 0 || len(payload.Items) > maxTileSetItems {
		return invalidTaskPayload("items must contain between 1 and %d definitions", maxTileSetItems)
	}

	totalTiles := 0
	for itemIndex := range payload.Items {
		item := &payload.Items[itemIndex]
		prefix := fmt.Sprintf("items[%d]", itemIndex)
		if err := validateTileSetItemDefinition(prefix, *item, payload.Dimensions); err != nil {
			return err
		}
		totalTiles += len(item.Shape)
		if uint64(totalTiles) > tileSetGridCapacity(payload) {
			return invalidTaskPayload("items contain more occupied Tiles than the Tileset grid supports")
		}
	}
	return nil
}

func validateTileSetDimensions(payload *CreateTileSetPayload) error {
	dimensions := payload.Dimensions
	if dimensions.TileSize.Width == 0 || dimensions.TileSize.Height == 0 ||
		dimensions.TileAmount.Columns == 0 || dimensions.TileAmount.Rows == 0 {
		return invalidTaskPayload("dimensions must contain positive tileSize and tileAmount values")
	}
	if dimensions.TileSize.Width > maxTileEdge || dimensions.TileSize.Height > maxTileEdge {
		return invalidTaskPayload("tileSize width and height must not exceed %d pixels", maxTileEdge)
	}
	capacity := tileSetGridCapacity(payload)
	if capacity > maxTileSetGridTiles {
		return invalidTaskPayload("tileAmount must not exceed %d total Tiles", maxTileSetGridTiles)
	}
	return nil
}

func tileSetGridCapacity(payload *CreateTileSetPayload) uint64 {
	return uint64(payload.Dimensions.TileAmount.Columns) * uint64(payload.Dimensions.TileAmount.Rows)
}

func validateTileSetItemDefinition(
	prefix string,
	item TileSetItemDefinition,
	dimensions assetdomain.TileSetDimensions,
) error {
	if err := validateRequiredText(prefix+".name", item.Name, maxItemNameLength); err != nil {
		return err
	}
	if err := validateRequiredText(prefix+".description", item.Description, maxItemDescriptionLength); err != nil {
		return err
	}
	if len(item.Shape) == 0 || len(item.Shape) > maxTilesPerItem {
		return invalidTaskPayload("%s.shape must contain between 1 and %d coordinates", prefix, maxTilesPerItem)
	}
	return validateItemShape(prefix, item.Shape, dimensions)
}

func validateItemShape(
	prefix string,
	shape []TileSetCoordinate,
	dimensions assetdomain.TileSetDimensions,
) error {
	seen := make(map[TileSetCoordinate]struct{}, len(shape))
	minX, minY := shape[0][0], shape[0][1]
	maxX, maxY := minX, minY
	for _, coordinate := range shape {
		x, y := coordinate[0], coordinate[1]
		if x < 0 || y < 0 {
			return invalidTaskPayload("%s.shape contains a negative coordinate", prefix)
		}
		if uint64(x) >= uint64(dimensions.TileAmount.Columns) ||
			uint64(y) >= uint64(dimensions.TileAmount.Rows) {
			return invalidTaskPayload("%s.shape cannot fit inside tileAmount", prefix)
		}
		if _, duplicate := seen[coordinate]; duplicate {
			return invalidTaskPayload("%s.shape contains duplicate coordinate [%d,%d]", prefix, x, y)
		}
		seen[coordinate] = struct{}{}
		minX, minY = min(minX, x), min(minY, y)
		maxX, maxY = max(maxX, x), max(maxY, y)
	}

	// Coordinates and Tile edges have already been bounded above, so these
	// conversions cannot overflow uint64.
	boundingWidth := uint64(maxX-minX+1) * uint64(dimensions.TileSize.Width)   //nolint:gosec // Values are nonnegative and bounded.
	boundingHeight := uint64(maxY-minY+1) * uint64(dimensions.TileSize.Height) //nolint:gosec // Values are nonnegative and bounded.
	if boundingWidth > maxGeneratedItemImageEdge || boundingHeight > maxGeneratedItemImageEdge {
		return invalidTaskPayload("%s.shape produces an image larger than %d pixels per edge", prefix, maxGeneratedItemImageEdge)
	}
	return nil
}

func validateAddTilesetItemPayload(payload *AddTilesetItemPayload) error {
	if payload == nil {
		return invalidTaskPayload("Tileset Item addition payload is required")
	}
	if err := validateEditPayloadBase(payload.ProjectID, payload.AssetID, payload.CreativeBrief); err != nil {
		return err
	}
	if err := validateOptionalCreatingReference(payload.CreatingReference); err != nil {
		return err
	}
	if payload.Item == nil {
		return invalidTaskPayload("item is required")
	}
	if err := validateRequiredText("item.name", payload.Item.Name, maxItemNameLength); err != nil {
		return err
	}
	if err := validateRequiredText("item.description", payload.Item.Description, maxItemDescriptionLength); err != nil {
		return err
	}
	if len(payload.Item.Shape) == 0 || len(payload.Item.Shape) > maxTilesPerItem {
		return invalidTaskPayload("item.shape must contain between 1 and %d coordinates", maxTilesPerItem)
	}
	if _, err := normalizeTileSetShape(payload.Item.Shape); err != nil {
		return invalidTaskPayload("item.shape is invalid: %v", err)
	}
	if payload.Item.Origin != nil {
		if payload.Item.Origin.X == nil || payload.Item.Origin.Y == nil {
			return invalidTaskPayload("item.origin must contain x and y")
		}
		if *payload.Item.Origin.X < 0 || *payload.Item.Origin.Y < 0 {
			return invalidTaskPayload("item.origin must contain nonnegative coordinates")
		}
	}
	return nil
}

func validateEditTilesetItemPayload(payload *EditTilesetItemPayload) error {
	if payload == nil {
		return invalidTaskPayload("Tileset Item edit payload is required")
	}
	if err := validateEditPayloadBase(payload.ProjectID, payload.AssetID, payload.CreativeBrief); err != nil {
		return err
	}
	if err := validateOptionalCreatingReference(payload.CreatingReference); err != nil {
		return err
	}
	if err := validateTileSetEditTarget("target", payload.Target); err != nil {
		return err
	}
	return nil
}

func validateEditTilesPayload(payload *EditTilesPayload) error {
	if payload == nil {
		return invalidTaskPayload("Tile edit payload is required")
	}
	if err := validateEditPayloadBase(payload.ProjectID, payload.AssetID, payload.CreativeBrief); err != nil {
		return err
	}
	if err := validateOptionalCreatingReference(payload.CreatingReference); err != nil {
		return err
	}
	if len(payload.Targets) == 0 || len(payload.Targets) > maxTileEditTargets {
		return invalidTaskPayload("edit_tiles requires between 1 and %d targets", maxTileEditTargets)
	}
	seen := make(map[assetdomain.TilePosition]struct{}, len(payload.Targets))
	for targetIndex := range payload.Targets {
		target := &payload.Targets[targetIndex]
		if err := validateTileSetEditTarget(fmt.Sprintf("targets[%d]", targetIndex), target); err != nil {
			return err
		}
		position := assetdomain.TilePosition{X: *target.Position.X, Y: *target.Position.Y}
		if _, duplicate := seen[position]; duplicate {
			return invalidTaskPayload("edit_tiles contains duplicate target position (%d,%d)", position.X, position.Y)
		}
		seen[position] = struct{}{}
	}
	return nil
}

func validateTileSetEditTarget(field string, target *TileSetEditTarget) error {
	if target == nil || target.Position == nil {
		return invalidTaskPayload("%s.position is required", field)
	}
	if target.Position.X == nil || target.Position.Y == nil {
		return invalidTaskPayload("%s.position must contain x and y", field)
	}
	if *target.Position.X < 0 || *target.Position.Y < 0 {
		return invalidTaskPayload("%s.position must contain nonnegative coordinates", field)
	}
	return nil
}

func validateEditPayloadBase(projectID, assetID uint, creativeBrief string) error {
	if projectID == 0 {
		return invalidTaskPayload("project_id must be positive")
	}
	if assetID == 0 {
		return invalidTaskPayload("asset_id must be positive")
	}
	if err := validateRequiredText("creative_brief", creativeBrief, maxCreativeBriefLength); err != nil {
		return err
	}
	return nil
}

func validateOptionalCreatingReference(creatingReference string) error {
	if creatingReference == "" {
		return nil
	}
	if len(creatingReference) > maxCreatingReferenceLength {
		return invalidTaskPayload("creating reference exceeds maximum length of %d bytes", maxCreatingReferenceLength)
	}
	if strings.TrimSpace(creatingReference) == "" {
		return invalidTaskPayload("creating reference must not be blank")
	}
	for _, r := range creatingReference {
		if unicode.IsControl(r) {
			return invalidTaskPayload("creating reference contains invalid control characters")
		}
	}
	return nil
}

func validateRequiredText(field, value string, maximum int) error {
	if strings.TrimSpace(value) == "" {
		return invalidTaskPayload("%s is required", field)
	}
	if utf8.RuneCountInString(value) > maximum {
		return invalidTaskPayload("%s exceeds maximum length of %d characters", field, maximum)
	}
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return invalidTaskPayload("%s contains invalid control characters", field)
		}
	}
	return nil
}

func invalidTaskPayload(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidTaskPayload, fmt.Sprintf(format, args...))
}

// EditFramesParameters contains the edit_frames-specific options carried by
// the generic generation request parameters object.
type EditFramesParameters struct {
	AnimationID uint   `json:"animationId"`
	FrameIDs    []uint `json:"frameIds"`
}

// EditFramesPayload is the self-contained input consumed by the unified
// character/object animation frame edit task. FrameIDs are persisted frame
// identifiers, not zero-based array offsets.
type EditFramesPayload struct {
	AssetID     uint   `json:"asset_id"`
	ProjectID   uint   `json:"project_id"`
	AnimationID uint   `json:"animation_id"`
	FrameIDs    []uint `json:"frame_ids"`
	Prompt      string `json:"prompt"`
}
