package asset_test

import (
	"encoding/json"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestAssetContentPreservesPrototypeAndAnimationResourceArrays(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	content.DirectionCount = 4
	prototype := domain.Prototype{
		{ID: 2101, URL: new("https://cdn.example/prototype-01.png")},
		{ID: 2102, URL: new("https://cdn.example/prototype-02.png")},
	}
	content.Prototype = &prototype
	content.Animations = []domain.Animation{{
		ID:   3001,
		Name: "walk",
		Frames: []domain.Frame{{
			ID:       2201,
			URL:      new("https://cdn.example/walk-01.png"),
			Duration: 100,
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
	if decoded.Prototype == nil {
		t.Fatalf("expected prototype: %+v", decoded.Prototype)
	}
	if len(*decoded.Prototype) != 2 || (*decoded.Prototype)[0].ID != 2101 {
		t.Fatalf("prototype resources were not preserved: %+v", decoded.Prototype)
	}
	if len(decoded.Animations) != 1 || len(decoded.Animations[0].Frames) != 1 || decoded.Animations[0].Frames[0].ID != 2201 {
		t.Fatalf("animation frames were not preserved: %+v", decoded.Animations)
	}
	if string(payload) == "" || json.Valid(payload) == false {
		t.Fatalf("invalid encoded content: %s", payload)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("decode raw asset content: %v", err)
	}
	if _, exists := raw["directions"]; exists {
		t.Fatalf("directions must not be encoded: %s", payload)
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

func TestAssetContentRejectsUnsupportedPerspective(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	content.Perspective = "side_on"
	if _, err := domain.EncodeContent(content); err == nil {
		t.Fatal("expected legacy perspective to be rejected when encoding content")
	}

	asset := domain.Asset{
		Type:    domain.AssetTypeCharacter,
		Content: json.RawMessage(`{"perspective":"side_on"}`),
	}
	if _, err := asset.DecodeContent(); err == nil {
		t.Fatal("expected legacy perspective to be rejected when decoding content")
	}
}

func TestAssetContentKeepsDirectionCountIndependentFromPrototypeImages(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	content.Perspective = domain.PerspectiveSideOn
	content.DirectionCount = 2
	prototype := domain.Prototype{{ID: 1}, {ID: 2}, {ID: 3}}
	content.Prototype = &prototype

	payload, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode asset content: %v", err)
	}
	var decoded domain.AssetContent
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode asset content: %v", err)
	}
	if decoded.DirectionCount != 2 {
		t.Fatalf("unexpected direction count: %d", decoded.DirectionCount)
	}
	if decoded.Perspective != domain.PerspectiveSideOn {
		t.Fatalf("unexpected perspective: %q", decoded.Perspective)
	}
	if decoded.Prototype == nil || len(*decoded.Prototype) != 3 {
		t.Fatalf("prototype images should be preserved independently: %+v", decoded.Prototype)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("decode raw asset content: %v", err)
	}
	if raw["perspective"] != "Side-On" {
		t.Fatalf("expected perspective field: %s", payload)
	}
	if _, exists := raw["viewMode"]; exists {
		t.Fatalf("legacy viewMode field must not be encoded: %s", payload)
	}
}

func TestAssetContentPreservesTileGridPositionAndFixedSize(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeTileSet)
	content.TileSize = &domain.TileSize{Width: 32, Height: 32}
	content.Items = []domain.TileSetItem{{
		Name: "grass",
		Tiles: []domain.Tile{{
			URL:      new("https://cdn.example.com/tileset/grass/center.png"),
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
	if decoded.Items[0].Tiles[0].URL == nil || *decoded.Items[0].Tiles[0].URL != "https://cdn.example.com/tileset/grass/center.png" {
		t.Fatalf("unexpected tile URL: %+v", decoded.Items[0].Tiles[0].URL)
	}
}
