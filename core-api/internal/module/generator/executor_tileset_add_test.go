package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

func TestValidateAddTilesetItemAsset(t *testing.T) {
	tileURL := "uploads/tile1.png"
	validDim := json.RawMessage(`{"tileSize":{"width":16,"height":16},"tileAmount":{"columns":8,"rows":8}}`)
	validContent, _ := assetdomain.EncodeContent(assetdomain.AssetContent{
		Items: []assetdomain.TileSetItem{
			{Name: "Item1", Tiles: []assetdomain.Tile{{URL: &tileURL, Position: assetdomain.TilePosition{X: 0, Y: 0}}}},
		},
	})

	baseReq := AddTilesetItemPayload{
		AssetID:   10,
		ProjectID: 5,
		Item: &AddTileSetItemDefinition{
			Name:        "Chair",
			Description: "wooden chair",
			Shape:       []TileSetCoordinate{{0, 0}},
		},
	}

	t.Run("asset ID 0", func(t *testing.T) {
		err := validateAddTilesetItemAsset(assetdomain.Asset{ID: 0}, baseReq)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not found error, got %v", err)
		}
	})

	t.Run("asset type mismatch", func(t *testing.T) {
		err := validateAddTilesetItemAsset(assetdomain.Asset{
			ID: 10, ProjectID: 5, Type: assetdomain.AssetTypeCharacter, Dimensions: validDim,
		}, baseReq)
		if err == nil || !strings.Contains(err.Error(), "must have type") {
			t.Fatalf("expected type error, got %v", err)
		}
	})

	t.Run("project mismatch", func(t *testing.T) {
		err := validateAddTilesetItemAsset(assetdomain.Asset{
			ID: 10, ProjectID: 99, Type: assetdomain.AssetTypeTileSet, Dimensions: validDim,
		}, baseReq)
		if err == nil || !strings.Contains(err.Error(), "does not belong to project") {
			t.Fatalf("expected project mismatch error, got %v", err)
		}
	})

	t.Run("invalid dimensions structure", func(t *testing.T) {
		err := validateAddTilesetItemAsset(assetdomain.Asset{
			ID: 10, ProjectID: 5, Type: assetdomain.AssetTypeTileSet,
			Dimensions: json.RawMessage(`{}`),
		}, baseReq)
		if err == nil {
			t.Fatal("expected dimensions validation error")
		}
	})

	t.Run("unmarshal dimensions failure", func(t *testing.T) {
		err := validateAddTilesetItemAsset(assetdomain.Asset{
			ID: 10, ProjectID: 5, Type: assetdomain.AssetTypeTileSet,
			Dimensions: json.RawMessage(`"not-json-object"`),
		}, baseReq)
		if err == nil {
			t.Fatal("expected dimensions unmarshal error")
		}
	})

	t.Run("dimensions exceed max limits", func(t *testing.T) {
		hugeDim := json.RawMessage(`{"tileSize":{"width":2048,"height":2048},"tileAmount":{"columns":8,"rows":8}}`)
		err := validateAddTilesetItemAsset(assetdomain.Asset{
			ID: 10, ProjectID: 5, Type: assetdomain.AssetTypeTileSet, Dimensions: hugeDim,
		}, baseReq)
		if err == nil || !strings.Contains(err.Error(), "exceed processing limits") {
			t.Fatalf("expected limits error, got %v", err)
		}
	})

	t.Run("invalid content json", func(t *testing.T) {
		err := validateAddTilesetItemAsset(assetdomain.Asset{
			ID: 10, ProjectID: 5, Type: assetdomain.AssetTypeTileSet, Dimensions: validDim,
			Content: json.RawMessage(`invalid-json`),
		}, baseReq)
		if err == nil || !strings.Contains(err.Error(), "decode Tileset asset") {
			t.Fatalf("expected decode content error, got %v", err)
		}
	})

	t.Run("placement failure due to full grid", func(t *testing.T) {
		fullItems := make([]assetdomain.TileSetItem, 0, 64)
		for y := range 8 {
			for x := range 8 {
				fullItems = append(fullItems, assetdomain.TileSetItem{
					Name:  "tile",
					Tiles: []assetdomain.Tile{{URL: &tileURL, Position: assetdomain.TilePosition{X: x, Y: y}}},
				})
			}
		}
		fullContent, _ := assetdomain.EncodeContent(assetdomain.AssetContent{Items: fullItems})
		err := validateAddTilesetItemAsset(assetdomain.Asset{
			ID: 10, ProjectID: 5, Type: assetdomain.AssetTypeTileSet, Dimensions: validDim,
			Content: fullContent,
		}, baseReq)
		if err == nil {
			t.Fatal("expected placement error on full grid")
		}
	})

	t.Run("valid asset passes", func(t *testing.T) {
		err := validateAddTilesetItemAsset(assetdomain.Asset{
			ID: 10, ProjectID: 5, Type: assetdomain.AssetTypeTileSet, Dimensions: validDim,
			Content: validContent,
		}, baseReq)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})
}

type mockReferenceStore struct {
	objects   map[string]string
	deleteErr error
	uploadErr error
	keyErr    error
	keySeq    int
}

func (m *mockReferenceStore) ResolveReference(_ context.Context, key string) (string, error) {
	if val, ok := m.objects[key]; ok {
		return val, nil
	}
	return "data:image/png;base64,mock", nil
}

func (m *mockReferenceStore) PersistReference(ctx context.Context, dataURL string) (string, error) {
	key, err := m.NewObjectKey("uploads")
	if err != nil {
		return "", err
	}
	return key, m.PersistReferenceAt(ctx, key, dataURL)
}

func (m *mockReferenceStore) PersistReferenceAt(_ context.Context, key string, dataURL string) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}
	if m.objects == nil {
		m.objects = make(map[string]string)
	}
	m.objects[key] = dataURL
	return nil
}

func (m *mockReferenceStore) DeleteObjects(_ context.Context, keys []string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for _, k := range keys {
		delete(m.objects, k)
	}
	return nil
}

func (m *mockReferenceStore) NewObjectKey(prefix string) (string, error) {
	if m.keyErr != nil {
		return "", m.keyErr
	}
	m.keySeq++
	return fmt.Sprintf("%s_%d", strings.ReplaceAll(prefix, "/", "_"), m.keySeq), nil
}

func TestBuildTileSetUploadsValidation(t *testing.T) {
	mockRef := &mockReferenceStore{}
	items := []processedTileSetItem{
		{
			Index: 0,
			Name:  "Item0",
			Tiles: []imageprocessor.ImageRegion{
				{ImageBase64: "b64", MIMEType: "image/png"},
			},
		},
	}

	t.Run("count mismatch", func(t *testing.T) {
		_, err := buildTileSetUploads(mockRef, items, nil)
		if err == nil || !strings.Contains(err.Error(), "does not match Item count") {
			t.Fatalf("expected count mismatch error, got %v", err)
		}
	})

	t.Run("placement index mismatch", func(t *testing.T) {
		layout := []tileSetPlacement{
			{ItemIndex: 99, Positions: []TileSetCoordinate{{0, 0}}},
		}
		_, err := buildTileSetUploads(mockRef, items, layout)
		if err == nil || !strings.Contains(err.Error(), "does not match processed Tiles") {
			t.Fatalf("expected layout mismatch error, got %v", err)
		}
	})

	t.Run("positions count mismatch", func(t *testing.T) {
		layout := []tileSetPlacement{
			{ItemIndex: 0, Positions: []TileSetCoordinate{{0, 0}, {1, 0}}},
		}
		_, err := buildTileSetUploads(mockRef, items, layout)
		if err == nil || !strings.Contains(err.Error(), "does not match processed Tiles") {
			t.Fatalf("expected layout mismatch error, got %v", err)
		}
	})

	t.Run("key allocation error", func(t *testing.T) {
		errStore := &mockReferenceStore{keyErr: errors.New("key failed")}
		layout := []tileSetPlacement{
			{ItemIndex: 0, Positions: []TileSetCoordinate{{0, 0}}},
		}
		_, err := buildTileSetUploads(errStore, items, layout)
		if err == nil || !strings.Contains(err.Error(), "key failed") {
			t.Fatalf("expected key failure, got %v", err)
		}
	})
}

func TestPersistTileSetUploadsErrorHandling(t *testing.T) {
	t.Run("upload failure triggers cleanup and returns error", func(t *testing.T) {
		errStore := &mockReferenceStore{uploadErr: errors.New("s3 failure")}
		exec := &executor{references: errStore}
		uploads := []tileSetTileUpload{
			{
				itemIndex: 0,
				tileIndex: 0,
				position:  TileSetCoordinate{0, 0},
				region:    imageprocessor.ImageRegion{ImageBase64: "abc", MIMEType: "image/png"},
				objectKey: "uploads/tile1.png",
			},
		}

		_, err := exec.persistTileSetUploads(context.Background(), uploads)
		if err == nil || !strings.Contains(err.Error(), "s3 failure") {
			t.Fatalf("expected upload failure, got %v", err)
		}
	})

	t.Run("context cancellation during upload", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		exec := &executor{references: &mockReferenceStore{}}
		uploads := []tileSetTileUpload{
			{
				itemIndex: 0,
				tileIndex: 0,
				position:  TileSetCoordinate{0, 0},
				region:    imageprocessor.ImageRegion{ImageBase64: "abc", MIMEType: "image/png"},
				objectKey: "uploads/tile1.png",
			},
		}

		_, err := exec.persistTileSetUploads(ctx, uploads)
		if err == nil {
			t.Fatal("expected context canceled error")
		}
	})
}

func TestPublishAddedTileSetItemValidation(t *testing.T) {
	mockRef := &mockReferenceStore{}
	exec := &executor{references: mockRef}

	validItem := processedTileSetItem{
		Index:          0,
		Name:           "Bench",
		RawImageBase64: "rawb64",
		RawMediaType:   "image/png",
		Tiles: []imageprocessor.ImageRegion{
			{ImageBase64: "tileb64", MIMEType: "image/png"},
		},
	}
	validPlacement := tileSetPlacement{
		ItemIndex: 0,
		Positions: []TileSetCoordinate{{1, 1}},
	}

	t.Run("successful publish returns result and generated resources", func(t *testing.T) {
		rawRes, err := exec.publishAddedTileSetItem(context.Background(), 10, 2, validItem, validPlacement)
		if err != nil {
			t.Fatalf("publish added item: %v", err)
		}
		var result ExecutionResult
		if err := json.Unmarshal(rawRes, &result); err != nil {
			t.Fatal(err)
		}
		if result.AssetID != 10 || result.Version != 2 {
			t.Fatalf("unexpected asset ID / version: %+v", result)
		}
		if len(result.GeneratedResources) != 2 {
			t.Fatalf("expected 2 generated resources (tile + unprocessed), got %d", len(result.GeneratedResources))
		}
	})

	t.Run("empty item name triggers cleanup error", func(t *testing.T) {
		emptyNameItem := validItem
		emptyNameItem.Name = "   "
		_, err := exec.publishAddedTileSetItem(context.Background(), 10, 2, emptyNameItem, validPlacement)
		if err == nil || !strings.Contains(err.Error(), "candidate is empty") {
			t.Fatalf("expected empty candidate error, got %v", err)
		}
	})

	t.Run("empty tiles triggers cleanup error", func(t *testing.T) {
		emptyTilesItem := validItem
		emptyTilesItem.Tiles = nil
		emptyPlacement := tileSetPlacement{ItemIndex: 0, Positions: nil}
		_, err := exec.publishAddedTileSetItem(context.Background(), 10, 2, emptyTilesItem, emptyPlacement)
		if err == nil || !strings.Contains(err.Error(), "candidate is empty") {
			t.Fatalf("expected empty candidate error, got %v", err)
		}
	})
}

func TestAddTileSetItemEndToEndVariations(t *testing.T) {
	const tileSize = 16
	tileImg := image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))
	for y := range tileSize {
		for x := range tileSize {
			tileImg.SetRGBA(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	tileB64, _ := imageprocessor.EncodePNGBase64(tileImg)

	existingURL := "uploads/existing-tile.png"
	dimJSON := json.RawMessage(`{"tileSize":{"width":16,"height":16},"tileAmount":{"columns":8,"rows":8}}`)
	initialContent, _ := assetdomain.EncodeContent(assetdomain.AssetContent{
		Items: []assetdomain.TileSetItem{
			{Name: "Existing", Tiles: []assetdomain.Tile{{URL: &existingURL, Position: assetdomain.TilePosition{X: 0, Y: 0}}}},
		},
	})

	t.Run("payload validation failure", func(t *testing.T) {
		exec := &executor{}
		_, err := exec.addTileSetItem(context.Background(), AddTilesetItemPayload{})
		if err == nil {
			t.Fatal("expected payload validation error")
		}
	})

	t.Run("asset not found in edit context", func(t *testing.T) {
		exec := &executor{
			assets: &tileSetWorkflowAssets{asset: assetdomain.Asset{ID: 0}},
		}
		_, err := exec.addTileSetItem(context.Background(), AddTilesetItemPayload{
			AssetID: 100, ProjectID: 42,
			Item: &AddTileSetItemDefinition{Name: "Item", Shape: []TileSetCoordinate{{0, 0}}},
		})
		if err == nil {
			t.Fatal("expected asset not found error")
		}
	})

	t.Run("add item with custom creating reference and merged brief", func(t *testing.T) {
		assets := &tileSetWorkflowAssets{
			asset: assetdomain.Asset{
				ID: 100, ProjectID: 42, Type: assetdomain.AssetTypeTileSet, Version: 1,
				Dimensions:  dimJSON,
				Content:     initialContent,
				Description: "base rustic style",
			},
			content: assetdomain.AssetContent{
				Items: []assetdomain.TileSetItem{
					{Name: "Existing", Tiles: []assetdomain.Tile{{Position: assetdomain.TilePosition{X: 0, Y: 0}}}},
				},
			},
		}
		references := &tileSetWorkflowReferences{
			objects: map[string]string{
				"custom/ref.png": "data:image/png;base64," + tileB64,
			},
		}
		exec := &executor{
			images:     &tileSetWorkflowImages{},
			processor:  imageprocessor.NewProcessor(),
			assets:     assets,
			projects:   &tileSetGenerationProjectStub{project: &projectdomain.Project{ID: 42, Perspective: projectdomain.PerspectiveTopDown}},
			references: references,
		}

		payload := AddTilesetItemPayload{
			AssetID:           100,
			ProjectID:         42,
			CreativeBrief:     "iron rivets",
			CreatingReference: "custom/ref.png",
			Item: &AddTileSetItemDefinition{
				Name:        "Chest",
				Description: "reinforced iron chest",
				Shape:       []TileSetCoordinate{{0, 0}},
			},
		}

		resRaw, err := exec.addTileSetItem(context.Background(), payload)
		if err != nil {
			t.Fatalf("addTileSetItem: %v", err)
		}
		var result ExecutionResult
		if err := json.Unmarshal(resRaw, &result); err != nil {
			t.Fatal(err)
		}
		if result.AssetID != 100 || len(result.GeneratedResources) != 2 {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("add item with auto-resolved reference and empty request brief", func(t *testing.T) {
		unprocessedURL := "uploads/existing-tile-unprocessed.png"
		assets := &tileSetWorkflowAssets{
			asset: assetdomain.Asset{
				ID: 100, ProjectID: 42, Type: assetdomain.AssetTypeTileSet, Version: 1,
				Dimensions:  dimJSON,
				Content:     initialContent,
				Description: "base rustic style",
			},
			content: assetdomain.AssetContent{
				Items: []assetdomain.TileSetItem{
					{Name: "Existing", Tiles: []assetdomain.Tile{{URL: &existingURL, Position: assetdomain.TilePosition{X: 0, Y: 0}}}},
				},
			},
		}
		references := &tileSetWorkflowReferences{
			objects: map[string]string{
				unprocessedURL: "data:image/png;base64," + tileB64,
			},
		}
		exec := &executor{
			images:     &tileSetWorkflowImages{},
			processor:  imageprocessor.NewProcessor(),
			assets:     assets,
			projects:   &tileSetGenerationProjectStub{project: &projectdomain.Project{ID: 42, Perspective: projectdomain.PerspectiveTopDown}},
			references: references,
		}

		payload := AddTilesetItemPayload{
			AssetID:       100,
			ProjectID:     42,
			CreativeBrief: "vibrant potions",
			Item: &AddTileSetItemDefinition{
				Name:        "Chest",
				Description: "reinforced iron chest",
				Shape:       []TileSetCoordinate{{0, 0}},
			},
		}

		resRaw, err := exec.addTileSetItem(context.Background(), payload)
		if err != nil {
			t.Fatalf("addTileSetItem: %v", err)
		}
		var result ExecutionResult
		if err := json.Unmarshal(resRaw, &result); err != nil {
			t.Fatal(err)
		}
		if result.AssetID != 100 {
			t.Fatalf("unexpected asset ID: %d", result.AssetID)
		}
	})

	t.Run("add item when asset description is empty", func(t *testing.T) {
		assets := &tileSetWorkflowAssets{
			asset: assetdomain.Asset{
				ID: 100, ProjectID: 42, Type: assetdomain.AssetTypeTileSet, Version: 1,
				Dimensions:  dimJSON,
				Content:     initialContent,
				Description: "",
			},
			content: assetdomain.AssetContent{
				Items: []assetdomain.TileSetItem{
					{Name: "Existing", Tiles: []assetdomain.Tile{{URL: &existingURL, Position: assetdomain.TilePosition{X: 0, Y: 0}}}},
				},
			},
		}
		references := &tileSetWorkflowReferences{
			objects: map[string]string{
				"uploads/existing-tile-unprocessed.png": "data:image/png;base64," + tileB64,
			},
		}
		exec := &executor{
			images:     &tileSetWorkflowImages{},
			processor:  imageprocessor.NewProcessor(),
			assets:     assets,
			projects:   &tileSetGenerationProjectStub{project: &projectdomain.Project{ID: 42, Perspective: projectdomain.PerspectiveTopDown}},
			references: references,
		}

		payload := AddTilesetItemPayload{
			AssetID:       100,
			ProjectID:     42,
			CreativeBrief: "custom standalone brief",
			Item: &AddTileSetItemDefinition{
				Name:        "Chest",
				Description: "standalone chest",
				Shape:       []TileSetCoordinate{{0, 0}},
			},
		}

		resRaw, err := exec.addTileSetItem(context.Background(), payload)
		if err != nil {
			t.Fatalf("addTileSetItem: %v", err)
		}
		var result ExecutionResult
		if err := json.Unmarshal(resRaw, &result); err != nil {
			t.Fatal(err)
		}
		if result.AssetID != 100 {
			t.Fatalf("unexpected asset ID: %d", result.AssetID)
		}
	})

	t.Run("add item placement failure", func(t *testing.T) {
		x, y := 7, 7
		payload := AddTilesetItemPayload{
			AssetID:       100,
			ProjectID:     42,
			CreativeBrief: "out of bounds placement",
			Item: &AddTileSetItemDefinition{
				Name:        "Huge",
				Description: "out of bounds",
				Shape:       []TileSetCoordinate{{0, 0}, {1, 0}},
				Origin:      &TileSetOrigin{X: &x, Y: &y},
			},
		}
		assets := &tileSetWorkflowAssets{
			asset: assetdomain.Asset{
				ID: 100, ProjectID: 42, Type: assetdomain.AssetTypeTileSet, Version: 1,
				Dimensions: dimJSON, Content: initialContent,
			},
		}
		exec := &executor{
			assets:   assets,
			projects: &tileSetGenerationProjectStub{project: &projectdomain.Project{ID: 42, Perspective: projectdomain.PerspectiveTopDown}},
		}
		_, err := exec.addTileSetItem(context.Background(), payload)
		if err == nil || !strings.Contains(err.Error(), "outside the") {
			t.Fatalf("expected outside grid error, got %v", err)
		}
	})

	t.Run("publish added item with cleanup error on delete", func(t *testing.T) {
		failStore := &mockReferenceStore{
			deleteErr: errors.New("delete failed"),
		}
		exec := &executor{references: failStore}
		emptyNameItem := processedTileSetItem{
			Index: 0,
			Name:  "",
			Tiles: []imageprocessor.ImageRegion{{ImageBase64: "b64", MIMEType: "image/png"}},
		}
		placement := tileSetPlacement{ItemIndex: 0, Positions: []TileSetCoordinate{{0, 0}}}
		_, err := exec.publishAddedTileSetItem(context.Background(), 100, 1, emptyNameItem, placement)
		if err == nil || !strings.Contains(err.Error(), "delete failed") {
			t.Fatalf("expected combined cleanup error, got %v", err)
		}
	})
}
