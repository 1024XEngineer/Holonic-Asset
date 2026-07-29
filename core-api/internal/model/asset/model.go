package asset

import (
	"encoding/json"
	"time"
)

type Asset struct {
	ID          uint
	Name        string
	ProjectID   uint
	Type        AssetType
	Description string
	Tags        []string        `json:"tags"`
	Attributes  json.RawMessage `json:"attributes,omitempty"`
	Content     json.RawMessage `json:"content,omitempty"`
	Version     uint
}

type AssetListFilter struct {
	Query string
	Tags  []string
	Types []AssetType
}

type AssetUpdate struct {
	Name        *string
	ProjectID   *uint
	Type        *AssetType
	Description *string
	Tags        *[]string
	Attributes  *json.RawMessage
}

type ViewMode string

const (
	ViewModeSideOn  ViewMode = "side_on"
	ViewModeTopDown ViewMode = "top_down"
)

type AssetContent struct {
	ViewMode       ViewMode       `json:"viewMode,omitempty"`
	DirectionCount uint           `json:"directionCount,omitempty"`
	Prototype      *Prototype     `json:"prototype,omitempty"`
	Animations     []Animation    `json:"animations,omitempty"`
	TileSize       *TileSize      `json:"tileSize,omitempty"`
	Items          []TileSetItem  `json:"items,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type Prototype struct {
	Directions map[string]PrototypeDirection `json:"directions,omitempty"`
}

type PrototypeDirection struct {
	Image *ImageResource `json:"image,omitempty"`
}

type Animation struct {
	ID         uint                          `json:"id"`
	Name       string                        `json:"name"`
	Directions map[string]AnimationDirection `json:"directions,omitempty"`
}

type AnimationDirection struct {
	Frames []Frame `json:"frames,omitempty"`
}

type ImageResource struct {
	URL      *string         `json:"url,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type Frame struct {
	URL      *string         `json:"url,omitempty"`
	Duration uint            `json:"duration,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type TileSize struct {
	Width  uint `json:"width"`
	Height uint `json:"height"`
}

type TileSetItem struct {
	Name  string `json:"name"`
	Tiles []Tile `json:"tiles,omitempty"`
}

type Tile struct {
	Name     string          `json:"name"`
	URL      *string         `json:"url,omitempty"`
	Position TilePosition    `json:"position"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type TilePosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func NewAssetContent(assetType AssetType) AssetContent {
	content := AssetContent{}
	if assetType == AssetTypeCharacter || assetType == AssetTypeObject {
		content.Prototype = &Prototype{}
	}
	return content
}

func (a Asset) DecodeContent() (AssetContent, error) {
	if len(a.Content) == 0 {
		return NewAssetContent(a.Type), nil
	}

	var content AssetContent
	if err := json.Unmarshal(a.Content, &content); err != nil {
		return AssetContent{}, err
	}
	return content, nil
}

func EncodeContent(content AssetContent) (json.RawMessage, error) {
	content.normalizePrototypeDirections()
	value, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(value), nil
}

func DirectionsForCount(count uint) []string {
	switch count {
	case 1:
		return []string{"front"}
	case 2:
		return []string{"left", "right"}
	case 4:
		return []string{"up", "down", "left", "right"}
	case 8:
		return []string{
			"up",
			"down",
			"left",
			"right",
			"up_left",
			"up_right",
			"down_left",
			"down_right",
		}
	default:
		return nil
	}
}

func (content *AssetContent) normalizePrototypeDirections() {
	directions := DirectionsForCount(content.DirectionCount)
	if content.Prototype == nil || len(directions) == 0 {
		return
	}
	if content.Prototype.Directions == nil {
		content.Prototype.Directions = make(map[string]PrototypeDirection, len(directions))
	}
	supported := make(map[string]struct{}, len(directions))
	for _, direction := range directions {
		supported[direction] = struct{}{}
		if _, ok := content.Prototype.Directions[direction]; !ok {
			content.Prototype.Directions[direction] = PrototypeDirection{}
		}
	}
	for direction := range content.Prototype.Directions {
		if _, ok := supported[direction]; !ok {
			delete(content.Prototype.Directions, direction)
		}
	}
}

type AssetRecord struct {
	ID        uint
	AssetID   uint
	Version   uint
	ContentID uint
	CreatedAt time.Time
	Content   json.RawMessage
}
