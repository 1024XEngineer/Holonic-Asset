package asset

import (
	"encoding/json"
	"fmt"

	perspectivedomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/perspective"
)

type Perspective = perspectivedomain.Perspective

const (
	PerspectiveTopDown   = perspectivedomain.TopDown
	PerspectiveSideOn    = perspectivedomain.SideOn
	PerspectiveIsometric = perspectivedomain.Isometric
)

type AssetContent struct {
	Perspective    Perspective    `json:"perspective,omitempty"`
	DirectionCount uint           `json:"directionCount,omitempty"`
	Prototype      *Prototype     `json:"prototype,omitempty"`
	Animations     []Animation    `json:"animations,omitempty"`
	TileSize       *TileSize      `json:"tileSize,omitempty"`
	Items          []TileSetItem  `json:"items,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type Prototype []ImageResource

type Animation struct {
	ID     uint    `json:"id"`
	Name   string  `json:"name"`
	Frames []Frame `json:"frames"`
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

type TileSize struct {
	Width  uint `json:"width"`
	Height uint `json:"height"`
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
	if err := validateContentPerspective(content.Perspective); err != nil {
		return AssetContent{}, err
	}
	return content, nil
}

func EncodeContent(content AssetContent) (json.RawMessage, error) {
	if err := validateContentPerspective(content.Perspective); err != nil {
		return nil, err
	}
	value, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(value), nil
}

func validateContentPerspective(perspective Perspective) error {
	if perspective != "" && !perspective.Valid() {
		return fmt.Errorf("asset: invalid perspective %q", perspective)
	}
	return nil
}
