package generator

import (
	"strings"
	"testing"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestDecodeSceneryLayerPlan(t *testing.T) {
	t.Run("valid scenery layer plan", func(t *testing.T) {
		raw := []byte(`{
			"layers": [
				{"name": "Background Mountains", "creative_brief": "Distant snowy mountains under twilight"},
				{"name": "Foreground Trees", "creative_brief": "Silhouette pine trees in the foreground"}
			]
		}`)
		layers, err := decodeSceneryLayerPlan(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(layers) != 2 {
			t.Fatalf("expected 2 layers, got %d", len(layers))
		}
		if layers[0].ID != 1 || layers[0].Name != "Background Mountains" {
			t.Fatalf("unexpected layer 0: %+v", layers[0])
		}
		if layers[1].ID != 2 || layers[1].Name != "Foreground Trees" {
			t.Fatalf("unexpected layer 1: %+v", layers[1])
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		_, err := decodeSceneryLayerPlan([]byte(`{not json`))
		if err == nil {
			t.Fatal("expected error for malformed json")
		}
	})

	t.Run("trailing data", func(t *testing.T) {
		raw := []byte(`{"layers":[{"name":"L1","creative_brief":"brief"}]} extra`)
		_, err := decodeSceneryLayerPlan(raw)
		if err == nil {
			t.Fatal("expected error for trailing data")
		}
	})

	t.Run("empty layers", func(t *testing.T) {
		raw := []byte(`{"layers":[]}`)
		_, err := decodeSceneryLayerPlan(raw)
		if err == nil {
			t.Fatal("expected error for empty layers")
		}
	})

	t.Run("missing layer name", func(t *testing.T) {
		raw := []byte(`{"layers":[{"name":"","creative_brief":"brief"}]}`)
		_, err := decodeSceneryLayerPlan(raw)
		if err == nil {
			t.Fatal("expected error for empty layer name")
		}
	})

	t.Run("duplicate layer name case insensitive", func(t *testing.T) {
		raw := []byte(`{
			"layers": [
				{"name": "Sky", "creative_brief": "Sky backdrop"},
				{"name": "sky", "creative_brief": "Clouds"}
			]
		}`)
		_, err := decodeSceneryLayerPlan(raw)
		if err == nil {
			t.Fatal("expected error for duplicate layer name")
		}
	})

	t.Run("missing creative brief", func(t *testing.T) {
		raw := []byte(`{"layers":[{"name":"Sky","creative_brief":"  "}]}`)
		_, err := decodeSceneryLayerPlan(raw)
		if err == nil {
			t.Fatal("expected error for empty creative brief")
		}
	})
}

func TestValidateCreateTileSetPayload(t *testing.T) {
	validDimensions := assetdomain.TileSetDimensions{
		TileSize:   assetdomain.Size{Width: 32, Height: 32},
		TileAmount: assetdomain.TileAmount{Columns: 4, Rows: 4},
	}
	validItems := []TileSetItemDefinition{
		{
			Name:        "Wall",
			Description: "Stone wall",
			Shape:       []TileSetCoordinate{{0, 0}, {0, 1}},
		},
	}

	t.Run("nil payload", func(t *testing.T) {
		if err := validateCreateTileSetPayload(nil); err == nil {
			t.Fatal("expected error for nil payload")
		}
	})

	t.Run("project_id zero", func(t *testing.T) {
		p := &CreateTileSetPayload{
			ProjectID:     0,
			AssetName:     "Tileset 1",
			CreativeBrief: "brief",
			Dimensions:    validDimensions,
			Items:         validItems,
		}
		if err := validateCreateTileSetPayload(p); err == nil {
			t.Fatal("expected error for project_id = 0")
		}
	})

	t.Run("invalid asset name", func(t *testing.T) {
		p := &CreateTileSetPayload{
			ProjectID:     1,
			AssetName:     "   ",
			CreativeBrief: "brief",
			Dimensions:    validDimensions,
			Items:         validItems,
		}
		if err := validateCreateTileSetPayload(p); err == nil {
			t.Fatal("expected error for blank asset name")
		}
	})

	t.Run("invalid dimensions", func(t *testing.T) {
		p := &CreateTileSetPayload{
			ProjectID:     1,
			AssetName:     "Tileset",
			CreativeBrief: "brief",
			Dimensions: assetdomain.TileSetDimensions{
				TileSize:   assetdomain.Size{Width: 2048, Height: 32},
				TileAmount: assetdomain.TileAmount{Columns: 4, Rows: 4},
			},
			Items: validItems,
		}
		if err := validateCreateTileSetPayload(p); err == nil {
			t.Fatal("expected error for tileSize > 1024")
		}

		p.Dimensions = assetdomain.TileSetDimensions{
			TileSize:   assetdomain.Size{Width: 32, Height: 32},
			TileAmount: assetdomain.TileAmount{Columns: 100, Rows: 100}, // 10000 > 4096
		}
		if err := validateCreateTileSetPayload(p); err == nil {
			t.Fatal("expected error for capacity > 4096")
		}
	})

	t.Run("no items", func(t *testing.T) {
		p := &CreateTileSetPayload{
			ProjectID:     1,
			AssetName:     "Tileset",
			CreativeBrief: "brief",
			Dimensions:    validDimensions,
			Items:         nil,
		}
		if err := validateCreateTileSetPayload(p); err == nil {
			t.Fatal("expected error for empty items")
		}
	})

	t.Run("item shape negative coordinate", func(t *testing.T) {
		p := &CreateTileSetPayload{
			ProjectID:     1,
			AssetName:     "Tileset",
			CreativeBrief: "brief",
			Dimensions:    validDimensions,
			Items: []TileSetItemDefinition{
				{
					Name:        "Item",
					Description: "Desc",
					Shape:       []TileSetCoordinate{{-1, 0}},
				},
			},
		}
		if err := validateCreateTileSetPayload(p); err == nil {
			t.Fatal("expected error for negative coordinate")
		}
	})

	t.Run("item shape exceeds tileAmount", func(t *testing.T) {
		p := &CreateTileSetPayload{
			ProjectID:     1,
			AssetName:     "Tileset",
			CreativeBrief: "brief",
			Dimensions:    validDimensions,
			Items: []TileSetItemDefinition{
				{
					Name:        "Item",
					Description: "Desc",
					Shape:       []TileSetCoordinate{{5, 0}},
				},
			},
		}
		if err := validateCreateTileSetPayload(p); err == nil {
			t.Fatal("expected error for coordinate exceeding columns")
		}
	})

	t.Run("item shape duplicate coordinate", func(t *testing.T) {
		p := &CreateTileSetPayload{
			ProjectID:     1,
			AssetName:     "Tileset",
			CreativeBrief: "brief",
			Dimensions:    validDimensions,
			Items: []TileSetItemDefinition{
				{
					Name:        "Item",
					Description: "Desc",
					Shape:       []TileSetCoordinate{{1, 1}, {1, 1}},
				},
			},
		}
		if err := validateCreateTileSetPayload(p); err == nil {
			t.Fatal("expected error for duplicate coordinate")
		}
	})

	t.Run("total tiles exceed capacity", func(t *testing.T) {
		p := &CreateTileSetPayload{
			ProjectID:     1,
			AssetName:     "Tileset",
			CreativeBrief: "brief",
			Dimensions: assetdomain.TileSetDimensions{
				TileSize:   assetdomain.Size{Width: 32, Height: 32},
				TileAmount: assetdomain.TileAmount{Columns: 1, Rows: 1},
			},
			Items: []TileSetItemDefinition{
				{
					Name:        "Item 1",
					Description: "Desc 1",
					Shape:       []TileSetCoordinate{{0, 0}},
				},
				{
					Name:        "Item 2",
					Description: "Desc 2",
					Shape:       []TileSetCoordinate{{0, 0}},
				},
			},
		}
		if err := validateCreateTileSetPayload(p); err == nil {
			t.Fatal("expected error for total tiles exceeding grid capacity")
		}
	})
}

func TestValidateAddTilesetItemPayload(t *testing.T) {
	t.Run("nil payload", func(t *testing.T) {
		if err := validateAddTilesetItemPayload(nil); err == nil {
			t.Fatal("expected error for nil payload")
		}
	})

	t.Run("missing item", func(t *testing.T) {
		p := &AddTilesetItemPayload{
			ProjectID:     1,
			AssetID:       2,
			CreativeBrief: "brief",
			Item:          nil,
		}
		if err := validateAddTilesetItemPayload(p); err == nil {
			t.Fatal("expected error for nil item")
		}
	})

	t.Run("invalid item origin", func(t *testing.T) {
		neg := -1
		p := &AddTilesetItemPayload{
			ProjectID:     1,
			AssetID:       2,
			CreativeBrief: "brief",
			Item: &AddTileSetItemDefinition{
				Name:        "Item",
				Description: "Desc",
				Shape:       []TileSetCoordinate{{0, 0}},
				Origin:      &TileSetOrigin{X: &neg, Y: &neg},
			},
		}
		if err := validateAddTilesetItemPayload(p); err == nil {
			t.Fatal("expected error for negative origin")
		}
	})
}

func TestValidateEditTilesetItemPayload(t *testing.T) {
	t.Run("nil payload", func(t *testing.T) {
		if err := validateEditTilesetItemPayload(nil); err == nil {
			t.Fatal("expected error for nil payload")
		}
	})

	t.Run("nil target", func(t *testing.T) {
		p := &EditTilesetItemPayload{
			ProjectID:     1,
			AssetID:       2,
			CreativeBrief: "brief",
			Target:        nil,
		}
		if err := validateEditTilesetItemPayload(p); err == nil {
			t.Fatal("expected error for nil target")
		}
	})
}

func TestValidateEditTilesPayload(t *testing.T) {
	t.Run("nil payload", func(t *testing.T) {
		if err := validateEditTilesPayload(nil); err == nil {
			t.Fatal("expected error for nil payload")
		}
	})

	t.Run("no targets", func(t *testing.T) {
		p := &EditTilesPayload{
			ProjectID:     1,
			AssetID:       2,
			CreativeBrief: "brief",
			Targets:       nil,
		}
		if err := validateEditTilesPayload(p); err == nil {
			t.Fatal("expected error for empty targets")
		}
	})

	t.Run("duplicate targets", func(t *testing.T) {
		x, y := 1, 2
		p := &EditTilesPayload{
			ProjectID:     1,
			AssetID:       2,
			CreativeBrief: "brief",
			Targets: []TileSetEditTarget{
				{Position: &TileSetEditPosition{X: &x, Y: &y}},
				{Position: &TileSetEditPosition{X: &x, Y: &y}},
			},
		}
		if err := validateEditTilesPayload(p); err == nil {
			t.Fatal("expected error for duplicate targets")
		}
	})
}

func TestValidateOptionalCreatingReference(t *testing.T) {
	t.Run("empty string is valid", func(t *testing.T) {
		if err := validateOptionalCreatingReference(""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("blank string is invalid", func(t *testing.T) {
		if err := validateOptionalCreatingReference("   "); err == nil {
			t.Fatal("expected error for whitespace-only reference")
		}
	})

	t.Run("control character is invalid", func(t *testing.T) {
		if err := validateOptionalCreatingReference("ref\x00data"); err == nil {
			t.Fatal("expected error for control character")
		}
	})

	t.Run("too long is invalid", func(t *testing.T) {
		longRef := strings.Repeat("a", (8<<20)+1)
		if err := validateOptionalCreatingReference(longRef); err == nil {
			t.Fatal("expected error for oversized reference")
		}
	})
}

func TestValidateRequiredText(t *testing.T) {
	t.Run("valid text with newlines and tabs", func(t *testing.T) {
		if err := validateRequiredText("field", "line 1\nline 2\ttabbed", 100); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty or whitespace only", func(t *testing.T) {
		if err := validateRequiredText("field", "   ", 100); err == nil {
			t.Fatal("expected error for blank text")
		}
	})

	t.Run("exceeds maximum length", func(t *testing.T) {
		if err := validateRequiredText("field", "abcdef", 5); err == nil {
			t.Fatal("expected error for exceeded length")
		}
	})

	t.Run("contains forbidden control character", func(t *testing.T) {
		if err := validateRequiredText("field", "bad\x07char", 100); err == nil {
			t.Fatal("expected error for control char")
		}
	})
}

func TestValidateAddAndEditTilesetPayloadEdgeCases(t *testing.T) {
	t.Run("add item with bad creating reference", func(t *testing.T) {
		p := &AddTilesetItemPayload{
			ProjectID:         1,
			AssetID:           2,
			CreativeBrief:     "brief",
			CreatingReference: "bad\x00ref",
			Item: &AddTileSetItemDefinition{
				Name:        "Item",
				Description: "Desc",
				Shape:       []TileSetCoordinate{{0, 0}},
			},
		}
		if err := validateAddTilesetItemPayload(p); err == nil {
			t.Fatal("expected error for bad creating reference in add item")
		}
	})

	t.Run("add item with invalid shape", func(t *testing.T) {
		p := &AddTilesetItemPayload{
			ProjectID:     1,
			AssetID:       2,
			CreativeBrief: "brief",
			Item: &AddTileSetItemDefinition{
				Name:        "Item",
				Description: "Desc",
				Shape:       nil,
			},
		}
		if err := validateAddTilesetItemPayload(p); err == nil {
			t.Fatal("expected error for empty shape in add item")
		}
	})

	t.Run("edit item with bad creating reference", func(t *testing.T) {
		x, y := 0, 0
		p := &EditTilesetItemPayload{
			ProjectID:         1,
			AssetID:           2,
			CreativeBrief:     "brief",
			CreatingReference: "bad\x00ref",
			Target:            &TileSetEditTarget{Position: &TileSetEditPosition{X: &x, Y: &y}},
		}
		if err := validateEditTilesetItemPayload(p); err == nil {
			t.Fatal("expected error for bad creating reference in edit item")
		}
	})

	t.Run("edit tiles with bad creating reference", func(t *testing.T) {
		x, y := 0, 0
		p := &EditTilesPayload{
			ProjectID:         1,
			AssetID:           2,
			CreativeBrief:     "brief",
			CreatingReference: "bad\x00ref",
			Targets:           []TileSetEditTarget{{Position: &TileSetEditPosition{X: &x, Y: &y}}},
		}
		if err := validateEditTilesPayload(p); err == nil {
			t.Fatal("expected error for bad creating reference in edit tiles")
		}
	})

	t.Run("add item with blank or overly long name and description", func(t *testing.T) {
		p := &AddTilesetItemPayload{
			ProjectID:     1,
			AssetID:       2,
			CreativeBrief: "brief",
			Item: &AddTileSetItemDefinition{
				Name:        "   ",
				Description: "Desc",
				Shape:       []TileSetCoordinate{{0, 0}},
			},
		}
		if err := validateAddTilesetItemPayload(p); err == nil {
			t.Fatal("expected error for blank item name")
		}

		p.Item.Name = "Valid Name"
		p.Item.Description = "   "
		if err := validateAddTilesetItemPayload(p); err == nil {
			t.Fatal("expected error for blank item description")
		}
	})

	t.Run("scenery plan trailing malformed json", func(t *testing.T) {
		raw := []byte(`{"layers":[{"name":"L1","creative_brief":"b"}]} {invalid`)
		if _, err := decodeSceneryLayerPlan(raw); err == nil {
			t.Fatal("expected error for trailing malformed json")
		}
	})
}
