package asset_test

import (
	"encoding/json"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
)

func TestAssetContentSupportsDynamicAnimationDirections(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	content.DirectionCount = 2
	content.Animations = []domain.Animation{{
		ID:   7,
		Name: "walk",
		Directions: map[string]domain.AnimationDirection{
			"left": {
				Frames: []domain.Frame{{}},
			},
			"right": {},
		},
	}}

	payload, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode asset content: %v", err)
	}

	var decoded domain.AssetContent
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode asset content: %v", err)
	}
	if decoded.Prototype == nil {
		t.Fatalf("expected prototype: %+v", decoded.Prototype)
	}
	if len(decoded.Animations) != 1 || len(decoded.Animations[0].Directions) != 2 {
		t.Fatalf("dynamic direction data was not preserved: %+v", decoded.Animations)
	}
}

func TestAssetDecodeContentInitializesMissingContent(t *testing.T) {
	content, err := (domain.Asset{Type: domain.AssetTypeObject}).DecodeContent()
	if err != nil {
		t.Fatalf("decode missing content: %v", err)
	}
	if content.Prototype == nil {
		t.Fatalf("unexpected initialized content: %+v", content)
	}
}

func TestAssetContentMatchesPrototypeDirectionsToDirectionCount(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	content.ViewMode = domain.ViewModeSideOn
	content.DirectionCount = 2
	content.Prototype.Directions = map[string]domain.PrototypeDirection{
		"left":   {Image: &domain.ImageResource{}},
		"unused": {Image: &domain.ImageResource{}},
	}

	payload, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode asset content: %v", err)
	}
	var decoded domain.AssetContent
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode asset content: %v", err)
	}
	if len(decoded.Prototype.Directions) != 2 {
		t.Fatalf("expected two side-on directions: %+v", decoded.Prototype.Directions)
	}
	if _, ok := decoded.Prototype.Directions["unused"]; ok {
		t.Fatal("unsupported prototype direction should be removed")
	}
	if _, ok := decoded.Prototype.Directions["right"]; !ok {
		t.Fatalf("missing direction should be initialized")
	}
}

func TestAssetContentPreservesTileGridPositionAndFixedSize(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeTileSet)
	content.TileSize = &domain.TileSize{Width: 32, Height: 32}
	content.Items = []domain.TileSetItem{{
		Name: "grass",
		Tiles: []domain.Tile{{
			Name:     "grass-center",
			Position: domain.TilePosition{X: 0, Y: 1},
		}},
	}}

	payload, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode asset content: %v", err)
	}

	var decoded domain.AssetContent
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode asset content: %v", err)
	}
	if decoded.TileSize == nil || decoded.TileSize.Width != 32 || decoded.TileSize.Height != 32 {
		t.Fatalf("unexpected fixed tile size: %+v", decoded.TileSize)
	}
	if len(decoded.Items) != 1 || len(decoded.Items[0].Tiles) != 1 {
		t.Fatalf("unexpected tileset items: %+v", decoded.Items)
	}
	if position := decoded.Items[0].Tiles[0].Position; position.X != 0 || position.Y != 1 {
		t.Fatalf("unexpected tile position: %+v", position)
	}
}
