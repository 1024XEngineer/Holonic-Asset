package generator

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

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
	Columns       int    `json:"columns,omitempty"`
	FrameWidth    int    `json:"frame_width,omitempty"`
	FrameHeight   int    `json:"frame_height,omitempty"`
	FPS           int    `json:"fps,omitempty"`
	Resolution    string `json:"resolution,omitempty"`
	Duration      int    `json:"duration,omitempty"`
	AspectRatio   string `json:"aspect_ratio,omitempty"`
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

// EditTilesetItemPayload is the complete input consumed by an Item edit task.
type EditTilesetItemPayload struct {
	AssetID          uint     `json:"asset_id"`
	ProjectID        uint     `json:"project_id"`
	CreativeBrief    string   `json:"creative_brief"`
	TargetAssetPaths []string `json:"target_asset_paths"`
}

// EditTilesPayload is the complete input consumed by a Tile edit task.
type EditTilesPayload struct {
	AssetID          uint     `json:"asset_id"`
	ProjectID        uint     `json:"project_id"`
	CreativeBrief    string   `json:"creative_brief"`
	TargetAssetPaths []string `json:"target_asset_paths"`
}

const (
	maxTileSetItems           = 64
	maxTilesPerItem           = 256
	maxTileSetGridTiles       = 4096
	maxTileEdge               = 1024
	maxGeneratedItemImageEdge = 4096
	maxTileEditTargets        = 256
	maxAssetNameLength        = 200
	maxCreativeBriefLength    = 4000
	maxItemNameLength         = 200
	maxItemDescriptionLength  = 2000

	// MaxTileSetItemConcurrency bounds the per-task fan-out used by the item
	// generation workflow implemented in the next phase.
	MaxTileSetItemConcurrency = 4
)

type TileSetItemTarget struct {
	ItemIndex int
}

type TileSetTileTarget struct {
	ItemIndex int
	TileIndex int
}

func ParseTilesetItemTargetPath(path string) (TileSetItemTarget, error) {
	parts := strings.Split(path, ".")
	if len(parts) != 2 || parts[0] != "items" {
		return TileSetItemTarget{}, invalidTaskPayload("invalid Tileset Item target path %q", path)
	}
	itemIndex, err := parseTargetIndex(parts[1])
	if err != nil {
		return TileSetItemTarget{}, invalidTaskPayload("invalid Tileset Item target path %q", path)
	}
	return TileSetItemTarget{ItemIndex: itemIndex}, nil
}

func ParseTilesetTileTargetPath(path string) (TileSetTileTarget, error) {
	parts := strings.Split(path, ".")
	if len(parts) != 4 || parts[0] != "items" || parts[2] != "tiles" {
		return TileSetTileTarget{}, invalidTaskPayload("invalid Tileset Tile target path %q", path)
	}
	itemIndex, err := parseTargetIndex(parts[1])
	if err != nil {
		return TileSetTileTarget{}, invalidTaskPayload("invalid Tileset Tile target path %q", path)
	}
	tileIndex, err := parseTargetIndex(parts[3])
	if err != nil {
		return TileSetTileTarget{}, invalidTaskPayload("invalid Tileset Tile target path %q", path)
	}
	return TileSetTileTarget{ItemIndex: itemIndex, TileIndex: tileIndex}, nil
}

func parseTargetIndex(value string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("empty index")
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("non-decimal index")
		}
	}
	index, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return index, nil
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
		if err := validateRequiredText(prefix+".name", item.Name, maxItemNameLength); err != nil {
			return err
		}
		if err := validateRequiredText(prefix+".description", item.Description, maxItemDescriptionLength); err != nil {
			return err
		}
		if len(item.Shape) == 0 || len(item.Shape) > maxTilesPerItem {
			return invalidTaskPayload("%s.shape must contain between 1 and %d coordinates", prefix, maxTilesPerItem)
		}
		if err := validateItemShape(prefix, item.Shape, payload); err != nil {
			return err
		}
		totalTiles += len(item.Shape)
		if totalTiles > maxTileSetGridTiles || uint64(totalTiles) > tileSetGridCapacity(payload) {
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

func validateItemShape(prefix string, shape []TileSetCoordinate, payload *CreateTileSetPayload) error {
	seen := make(map[TileSetCoordinate]struct{}, len(shape))
	minX, minY := shape[0][0], shape[0][1]
	maxX, maxY := minX, minY
	for _, coordinate := range shape {
		x, y := coordinate[0], coordinate[1]
		if x < 0 || y < 0 {
			return invalidTaskPayload("%s.shape contains a negative coordinate", prefix)
		}
		if uint64(x) >= uint64(payload.Dimensions.TileAmount.Columns) ||
			uint64(y) >= uint64(payload.Dimensions.TileAmount.Rows) {
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
	boundingWidth := uint64(maxX-minX+1) * uint64(payload.Dimensions.TileSize.Width)   //nolint:gosec // Values are nonnegative and bounded.
	boundingHeight := uint64(maxY-minY+1) * uint64(payload.Dimensions.TileSize.Height) //nolint:gosec // Values are nonnegative and bounded.
	if boundingWidth > maxGeneratedItemImageEdge || boundingHeight > maxGeneratedItemImageEdge {
		return invalidTaskPayload("%s.shape produces an image larger than %d pixels per edge", prefix, maxGeneratedItemImageEdge)
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
	if len(payload.TargetAssetPaths) != 1 {
		return invalidTaskPayload("edit_tileset_item requires exactly one target path")
	}
	_, err := ParseTilesetItemTargetPath(payload.TargetAssetPaths[0])
	return err
}

func validateEditTilesPayload(payload *EditTilesPayload) error {
	if payload == nil {
		return invalidTaskPayload("Tile edit payload is required")
	}
	if err := validateEditPayloadBase(payload.ProjectID, payload.AssetID, payload.CreativeBrief); err != nil {
		return err
	}
	if len(payload.TargetAssetPaths) == 0 || len(payload.TargetAssetPaths) > maxTileEditTargets {
		return invalidTaskPayload("edit_tiles requires between 1 and %d target paths", maxTileEditTargets)
	}
	seen := make(map[TileSetTileTarget]struct{}, len(payload.TargetAssetPaths))
	for _, path := range payload.TargetAssetPaths {
		target, err := ParseTilesetTileTargetPath(path)
		if err != nil {
			return err
		}
		if _, duplicate := seen[target]; duplicate {
			return invalidTaskPayload("edit_tiles contains duplicate target %q", path)
		}
		seen[target] = struct{}{}
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
