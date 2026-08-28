package asset

import "encoding/json"

type AssetContent struct {
	DirectionCount uint           `json:"directionCount,omitempty"`
	Prototype      *Prototype     `json:"prototype,omitempty"`
	Animations     []Animation    `json:"animations,omitempty"`
	Items          []TileSetItem  `json:"items,omitempty"`
	Components     []UIComponent  `json:"components,omitempty"`
	Layers         []SceneryLayer `json:"layers,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type Size struct {
	Width  uint `json:"width"`
	Height uint `json:"height"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Prototype []ImageResource

type Animation struct {
	ID         uint                       `json:"id"`
	GroupID    uint                       `json:"groupId,omitempty"`
	Name       string                     `json:"name"`
	Frames     []Frame                    `json:"frames"`
	Generation *AnimationGenerationConfig `json:"generation,omitempty"`
}

// AnimationGenerationConfig stores the effective parameters used to generate
// an animation. It is persisted with asset content and exposed through asset
// responses so clients can round-trip the complete content.
type AnimationGenerationConfig struct {
	Direction string `json:"direction"`
	Style     string `json:"style,omitempty"`
	Action    string `json:"action,omitempty"`

	FrameCount  int `json:"frameCount"`
	Columns     int `json:"columns"`
	FrameWidth  int `json:"frameWidth"`
	FrameHeight int `json:"frameHeight"`
	FPS         int `json:"fps"`

	Resolution  string `json:"resolution"`
	Duration    int    `json:"duration"`
	AspectRatio string `json:"aspectRatio"`
}

type ImageResource struct {
	ID       uint            `json:"id"`
	URL      *string         `json:"url,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type Frame struct {
	ID       uint            `json:"id"`
	URL      *string         `json:"url,omitempty"`
	Duration uint            `json:"duration,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type TileSetItem struct {
	Name  string `json:"name"`
	Tiles []Tile `json:"tiles,omitempty"`
}

type Tile struct {
	URL      *string      `json:"url,omitempty"`
	Position TilePosition `json:"position"`
}

type TilePosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type UIComponent struct {
	ID       uint            `json:"id"`
	Name     string          `json:"name"`
	Size     Size            `json:"size"`
	Position Position        `json:"position"`
	Anchor   *Position       `json:"anchor,omitempty"`
	Pivot    *Position       `json:"pivot,omitempty"`
	Texture  json.RawMessage `json:"texture,omitempty"`
	Color    json.RawMessage `json:"color,omitempty"`
	Opacity  *float64        `json:"opacity,omitempty"`
	State    json.RawMessage `json:"state,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
}

type SceneryLayer struct {
	ID        uint            `json:"id"`
	Name      string          `json:"name"`
	Resource  string          `json:"resource"`
	Position  Position        `json:"position"`
	Transform json.RawMessage `json:"transform,omitempty"`
	Visible   *bool           `json:"visible,omitempty"`
	Opacity   *float64        `json:"opacity,omitempty"`
	ZIndex    *int            `json:"zIndex,omitempty"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
}

func NewAssetContent(assetType AssetType) AssetContent {
	content := AssetContent{}
	if assetType == AssetTypeCharacter || assetType == AssetTypeObject {
		prototype := Prototype{}
		content.Prototype = &prototype
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
	value, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(value), nil
}
