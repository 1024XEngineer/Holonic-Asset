package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

type liveTileSetReferenceStore struct {
	mu      sync.Mutex
	counter int64
	objects map[string]string
	diskDir string
}

func (s *liveTileSetReferenceStore) ResolveReference(_ context.Context, reference string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if val, ok := s.objects[reference]; ok {
		return val, nil
	}
	return reference, nil
}

func (s *liveTileSetReferenceStore) PersistReference(_ context.Context, dataURI string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	key := fmt.Sprintf("tiles/tile_%d_%d.png", time.Now().UnixNano(), s.counter)
	s.objects[key] = dataURI
	return key, nil
}

func (s *liveTileSetReferenceStore) NewObjectKey(prefix string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	return fmt.Sprintf("%s/tile_%d_%d.png", prefix, time.Now().UnixNano(), s.counter), nil
}

func (s *liveTileSetReferenceStore) PersistReferenceAt(_ context.Context, key string, dataURI string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = dataURI

	// Also persist to disk for visual verification if directory is specified
	if s.diskDir != "" {
		if rawB64, ok := strings.CutPrefix(dataURI, "data:image/png;base64,"); ok {
			if decoded, err := base64.StdEncoding.DecodeString(rawB64); err == nil {
				fileName := strings.ReplaceAll(key, "/", "_")
				_ = os.WriteFile(filepath.Join(s.diskDir, fileName), decoded, 0o600)
			}
		}
	}
	return nil
}

func (s *liveTileSetReferenceStore) DeleteObjects(_ context.Context, keys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range keys {
		delete(s.objects, key)
	}
	return nil
}

func loadLiveImageConfig() (config.ImageClientConfig, error) {
	reader := viper.New()
	reader.SetConfigFile("../../config/config.yaml")
	if err := reader.ReadInConfig(); err != nil {
		return config.ImageClientConfig{}, err
	}
	section := reader.Sub("image")
	if section == nil {
		return config.ImageClientConfig{}, nil
	}
	var value config.ImageClientConfig
	if err := section.UnmarshalExact(&value); err != nil {
		return config.ImageClientConfig{}, err
	}
	return value, nil
}

func TestLiveAddTilesetItemEndToEndGeneration(t *testing.T) {
	if strings.TrimSpace(os.Getenv("HOLONIC_LLM_INTEGRATION")) != "1" {
		t.Skip("set HOLONIC_LLM_INTEGRATION=1 to run real tileset item generation test")
	}

	imageCfg, err := loadLiveImageConfig()
	if err != nil {
		t.Fatalf("load live image config: %v", err)
	}
	if imageCfg.BaseURL == "" || imageCfg.APIKey == "" {
		t.Skip("image config is missing BaseURL or APIKey")
	}

	provider := imageclient.NewQNAProvider(imageclient.QNAConfig{
		BaseURL:      imageCfg.BaseURL,
		APIKey:       imageCfg.APIKey,
		DefaultModel: imageCfg.DefaultModel,
	})
	imageService := imageclient.NewImageGenerationService(provider)
	processor := imageprocessor.NewProcessor()

	artifactDir := "/Users/lx/.gemini/antigravity/brain/03e53355-54e3-4379-9149-a8bb19de001d"
	_ = os.MkdirAll(artifactDir, 0o750)

	references := &liveTileSetReferenceStore{
		objects: make(map[string]string),
		diskDir: artifactDir,
	}

	// 1. Seed an existing TileSet with a 2x1 wooden bench at (0, 0) and (1, 0)
	benchTile1 := image.NewRGBA(image.Rect(0, 0, 16, 16))
	benchTile2 := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 4; y < 12; y++ {
		for x := 2; x < 14; x++ {
			benchTile1.SetRGBA(x, y, color.RGBA{R: 139, G: 69, B: 19, A: 255})
			benchTile2.SetRGBA(x, y, color.RGBA{R: 160, G: 82, B: 45, A: 255})
		}
	}
	bench1URI := "data:image/png;base64," + encodeTestPNG(t, benchTile1)
	bench2URI := "data:image/png;base64," + encodeTestPNG(t, benchTile2)
	bench1Key := "projects/42/tileset/100/bench_1.png"
	bench2Key := "projects/42/tileset/100/bench_2.png"
	_ = references.PersistReferenceAt(context.Background(), bench1Key, bench1URI)
	_ = references.PersistReferenceAt(context.Background(), bench2Key, bench2URI)

	existingContent := assetdomain.AssetContent{
		Items: []assetdomain.TileSetItem{
			{
				Name: "Wooden Bench",
				Tiles: []assetdomain.Tile{
					{URL: &bench1Key, Position: assetdomain.TilePosition{X: 0, Y: 0}},
					{URL: &bench2Key, Position: assetdomain.TilePosition{X: 1, Y: 0}},
				},
			},
		},
	}
	encodedContent, err := assetdomain.EncodeContent(existingContent)
	if err != nil {
		t.Fatalf("encode existing content: %v", err)
	}

	assets := &tileSetWorkflowAssets{
		asset: assetdomain.Asset{
			ID:          100,
			ProjectID:   42,
			Type:        assetdomain.AssetTypeTileSet,
			Version:     1,
			Perspective: assetdomain.PerspectiveTopDown,
			Dimensions:  json.RawMessage(`{"tileSize":{"width":32,"height":32},"tileAmount":{"columns":8,"rows":8}}`),
			Content:     encodedContent,
		},
		content: existingContent,
	}

	projectStub := &tileSetGenerationProjectStub{
		project: &projectdomain.Project{
			ID:             42,
			Name:           "Dungeon Odyssey",
			GameType:       "2D Top-Down Pixel RPG",
			Description:    "A dark fantasy pixel art dungeon crawler with stone crypts and treasures.",
			Style:          "16-bit low-res pixel art, rich palettes, clean hard edges",
			TargetPlatform: projectdomain.PlatformTypePC,
			Perspective:    projectdomain.PerspectiveTopDown,
		},
	}

	exec := &executor{
		images:     imageService,
		processor:  processor,
		assets:     assets,
		projects:   projectStub,
		references: references,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Log("Starting live AddTilesetItem generation for a 2x2 Ornate Treasure Chest...")
	startTime := time.Now()

	// 2. Request adding a 2x2 Golden Treasure Chest
	addPayload := AddTilesetItemPayload{
		ProjectID:     42,
		AssetID:       100,
		CreativeBrief: "A golden glowing treasure chest with ornate iron bands, studded lock, and glistening jewel inlays in classic pixel art style",
		Item: &AddTileSetItemDefinition{
			Name:        "Golden Treasure Chest",
			Description: "An ornate ancient golden treasure chest with glowing runes",
			Shape: []TileSetCoordinate{
				{0, 0}, {1, 0},
				{0, 1}, {1, 1},
			},
		},
	}
	payloadBytes, err := json.Marshal(addPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	rawResult, err := exec.Generate(ctx, AddTilesetItem, payloadBytes)
	if err != nil {
		t.Fatalf("execute AddTilesetItem failed: %v", err)
	}

	elapsed := time.Since(startTime)
	t.Logf("AddTilesetItem generation succeeded in %.2fs!", elapsed.Seconds())

	var execResult ExecutionResult
	if err := json.Unmarshal(rawResult, &execResult); err != nil {
		t.Fatalf("unmarshal execution result: %v", err)
	}

	if execResult.AssetID != 100 {
		t.Errorf("expected asset_id 100, got %d", execResult.AssetID)
	}
	if execResult.Version != 1 {
		t.Errorf("expected version 1, got %d", execResult.Version)
	}

	var addedContent assetdomain.AssetContent
	if err := json.Unmarshal(execResult.Content, &addedContent); err != nil {
		t.Fatalf("unmarshal added candidate content: %v", err)
	}

	if len(addedContent.Items) != 1 {
		t.Fatalf("expected 1 added item, got %d", len(addedContent.Items))
	}
	addedItem := addedContent.Items[0]
	t.Logf("Added Item: %s with %d tiles (generated resources: %v)", addedItem.Name, len(addedItem.Tiles), execResult.GeneratedResources)

	if len(addedItem.Tiles) != 4 {
		t.Fatalf("expected 4 tiles for 2x2 chest, got %d", len(addedItem.Tiles))
	}

	// 3. Compose a preview canvas showing the existing bench + the newly generated 2x2 chest on the 8x8 grid
	gridCols, gridRows := 8, 8
	tileW, tileH := 32, 32
	canvas := image.NewRGBA(image.Rect(0, 0, gridCols*tileW, gridRows*tileH))

	// Draw grid background
	for y := range gridRows * tileH {
		for x := range gridCols * tileW {
			if (x/tileW+y/tileH)%2 == 0 {
				canvas.SetRGBA(x, y, color.RGBA{R: 40, G: 44, B: 52, A: 255})
			} else {
				canvas.SetRGBA(x, y, color.RGBA{R: 50, G: 54, B: 62, A: 255})
			}
		}
	}

	// Draw existing tiles
	for _, it := range existingContent.Items {
		for _, tile := range it.Tiles {
			if tile.URL == nil {
				continue
			}
			dataURI, ok := references.objects[*tile.URL]
			if !ok {
				continue
			}
			img := decodeDataURIToImage(t, dataURI)
			destRect := image.Rect(tile.Position.X*tileW, tile.Position.Y*tileH, (tile.Position.X+1)*tileW, (tile.Position.Y+1)*tileH)
			draw.Draw(canvas, destRect, img, image.Point{}, draw.Over)
		}
	}

	// Draw newly added tiles
	for _, tile := range addedItem.Tiles {
		if tile.URL == nil {
			continue
		}
		dataURI, ok := references.objects[*tile.URL]
		if !ok {
			continue
		}
		img := decodeDataURIToImage(t, dataURI)
		destRect := image.Rect(tile.Position.X*tileW, tile.Position.Y*tileH, (tile.Position.X+1)*tileW, (tile.Position.Y+1)*tileH)
		draw.Draw(canvas, destRect, img, image.Point{}, draw.Over)
		t.Logf("  Tile at grid (%d, %d) -> %s", tile.Position.X, tile.Position.Y, *tile.URL)
	}

	// Save composite preview
	compositePath := filepath.Join(artifactDir, "generated_tileset_add_item_composite.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err == nil {
		_ = os.WriteFile(compositePath, buf.Bytes(), 0o600)
		t.Logf("Saved live tileset composite preview to: %s", compositePath)
	}
}

func encodeTestPNG(t *testing.T, img image.Image) string {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func decodeDataURIToImage(t *testing.T, dataURI string) image.Image {
	t.Helper()
	parts := strings.SplitN(dataURI, ",", 2)
	if len(parts) != 2 {
		t.Fatalf("invalid data URI: %s", dataURI)
	}
	rawBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode base64 data URI: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(rawBytes))
	if err != nil {
		t.Fatalf("decode PNG image: %v", err)
	}
	return img
}

type capturingImageService struct {
	inner         imageclient.ImageGenerationService
	mu            sync.Mutex
	lastGenerated string
}

func (c *capturingImageService) Generate(ctx context.Context, req *imageclient.GenerateRequest) (*imageclient.GenerateResult, error) {
	res, err := c.inner.Generate(ctx, req)
	if err == nil && res != nil && len(res.Images) > 0 {
		c.mu.Lock()
		c.lastGenerated = res.Images[0].Base64
		c.mu.Unlock()
	}
	return res, err
}

func (c *capturingImageService) GetLastGenerated() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastGenerated
}

func TestLiveSequentialTileSetItemGenerationWithUnprocessedReference(t *testing.T) {
	if strings.TrimSpace(os.Getenv("HOLONIC_LLM_INTEGRATION")) != "1" {
		t.Skip("set HOLONIC_LLM_INTEGRATION=1 to run sequential tileset item generation test")
	}

	imageCfg, err := loadLiveImageConfig()
	if err != nil {
		t.Fatalf("load live image config: %v", err)
	}
	if imageCfg.BaseURL == "" || imageCfg.APIKey == "" {
		t.Skip("image config is missing BaseURL or APIKey")
	}

	provider := imageclient.NewQNAProvider(imageclient.QNAConfig{
		BaseURL:      imageCfg.BaseURL,
		APIKey:       imageCfg.APIKey,
		DefaultModel: imageCfg.DefaultModel,
	})
	capturing := &capturingImageService{
		inner: imageclient.NewImageGenerationService(provider),
	}
	processor := imageprocessor.NewProcessor()

	artifactDir := "/Users/lx/.gemini/antigravity/brain/03e53355-54e3-4379-9149-a8bb19de001d"
	_ = os.MkdirAll(artifactDir, 0o750)

	references := &liveTileSetReferenceStore{
		objects: make(map[string]string),
		diskDir: artifactDir,
	}

	assets := &tileSetWorkflowAssets{
		asset: assetdomain.Asset{
			ID:          200,
			ProjectID:   88,
			Type:        assetdomain.AssetTypeTileSet,
			Version:     1,
			Perspective: assetdomain.PerspectiveTopDown,
			Dimensions:  json.RawMessage(`{"tileSize":{"width":32,"height":32},"tileAmount":{"columns":8,"rows":8}}`),
			Description: "16-bit SNES top-down pixel art alchemist workshop props with rich warm oak wood, glowing potion bottles, brass rivets and trims, clean dark outlines and stepped pixel shading",
		},
		content: assetdomain.AssetContent{Items: nil},
	}

	projectStub := &tileSetGenerationProjectStub{
		project: &projectdomain.Project{
			ID:             88,
			Name:           "Alchemist Laboratory",
			GameType:       "2D Top-Down Pixel RPG",
			Description:    "An ancient alchemist laboratory filled with magical artifacts, glowing potions, dark oak furniture, and mystical arcane tools.",
			Style:          "16-bit classic SNES pixel art, rich earthy and brass palette with vibrant potion glows, crisp 1-pixel outlines",
			TargetPlatform: projectdomain.PlatformTypePC,
			Perspective:    projectdomain.PerspectiveTopDown,
		},
	}

	exec := &executor{
		images:     capturing,
		processor:  processor,
		assets:     assets,
		projects:   projectStub,
		references: references,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	themeBrief := "16-bit classic top-down pixel art alchemist workshop furniture with warm oak wood, polished brass fixtures, glowing glass bottles, crisp hard edges"

	// --- Step 1: Generate initial TileSet with Item 1 (Alchemist Worktable) ---
	t.Log("=== [Step 1/4] Generating baseline Item 1: Alchemist Worktable ===")
	t1 := time.Now()
	createPayload := CreateTileSetPayload{
		AssetName:     "Alchemist Workshop Props",
		ProjectID:     88,
		CreativeBrief: themeBrief + ", featuring an oak worktable with potion vials and parchment scrolls",
		Dimensions: assetdomain.TileSetDimensions{
			TileSize:   assetdomain.Size{Width: 32, Height: 32},
			TileAmount: assetdomain.TileAmount{Columns: 8, Rows: 8},
		},
		Items: []TileSetItemDefinition{
			{
				Name:        "Alchemist Worktable",
				Description: "Heavy oak alchemy worktable with glowing potions and parchment",
				Shape:       []TileSetCoordinate{{0, 0}, {1, 0}},
			},
		},
	}
	createBytes, err := json.Marshal(createPayload)
	if err != nil {
		t.Fatalf("marshal create payload: %v", err)
	}

	rawRes1, err := exec.Generate(ctx, GenerateTileSet, createBytes)
	if err != nil {
		t.Fatalf("GenerateTileSet failed: %v", err)
	}
	t.Logf("Step 1 succeeded in %.2fs", time.Since(t1).Seconds())

	var execRes1 ExecutionResult
	if err := json.Unmarshal(rawRes1, &execRes1); err != nil {
		t.Fatalf("unmarshal result 1: %v", err)
	}
	assetID := execRes1.AssetID
	t.Logf("Created baseline TileSet Asset %d with %d items (total tiles: %d)", assetID, len(assets.content.Items), len(assets.content.Items[0].Tiles))

	// Store raw unprocessed image of Item 1 as the master style reference
	rawItem1B64 := capturing.GetLastGenerated()
	if rawItem1B64 == "" {
		t.Fatal("no raw image captured for Item 1")
	}
	item1UnprocessedKey := fmt.Sprintf("projects/88/tileset/%d/item1-unprocessed.png", assetID)
	_ = references.PersistReferenceAt(ctx, item1UnprocessedKey, "data:image/png;base64,"+rawItem1B64)
	t.Logf("Captured Item 1 unprocessed reference: %s", item1UnprocessedKey)

	// --- Step 2: Incremental Add Item 2 (Potion Bookshelf) with Item 1 Reference ---
	t.Log("=== [Step 2/4] Incremental Add Item 2: Potion Bookshelf (using Item 1 unprocessed reference) ===")
	t2 := time.Now()
	addItem2Payload := AddTilesetItemPayload{
		ProjectID:         88,
		AssetID:           assetID,
		CreativeBrief:     themeBrief + ", matching oak bookshelf with organized leather spellbooks and glowing glass potion flasks",
		CreatingReference: item1UnprocessedKey,
		Item: &AddTileSetItemDefinition{
			Name:        "Potion Bookshelf",
			Description: "Tall oak bookshelf filled with colorful potion vials and spellbooks matching the worktable style",
			Shape: []TileSetCoordinate{
				{0, 0}, {1, 0},
				{0, 1}, {1, 1},
			},
		},
	}
	add2Bytes, _ := json.Marshal(addItem2Payload)
	rawRes2, err := exec.Generate(ctx, AddTilesetItem, add2Bytes)
	if err != nil {
		t.Fatalf("AddTilesetItem (Item 2) failed: %v", err)
	}
	t.Logf("Step 2 succeeded in %.2fs", time.Since(t2).Seconds())

	var execRes2 ExecutionResult
	_ = json.Unmarshal(rawRes2, &execRes2)
	var content2 assetdomain.AssetContent
	_ = json.Unmarshal(execRes2.Content, &content2)
	assets.content.Items = append(assets.content.Items, content2.Items...)
	encoded2, _ := assetdomain.EncodeContent(assets.content)
	assets.asset.Content = encoded2

	rawItem2B64 := capturing.GetLastGenerated()
	item2UnprocessedKey := fmt.Sprintf("projects/88/tileset/%d/item2-unprocessed.png", assetID)
	_ = references.PersistReferenceAt(ctx, item2UnprocessedKey, "data:image/png;base64,"+rawItem2B64)

	// --- Step 3: Incremental Add Item 3 (Runed Brass Chest) with Item 1 Reference ---
	t.Log("=== [Step 3/4] Incremental Add Item 3: Runed Brass Chest (using Item 1 unprocessed reference) ===")
	t3 := time.Now()
	addItem3Payload := AddTilesetItemPayload{
		ProjectID:         88,
		AssetID:           assetID,
		CreativeBrief:     themeBrief + ", matching oak and brass reinforced treasure chest with glowing runic lock",
		CreatingReference: item1UnprocessedKey,
		Item: &AddTileSetItemDefinition{
			Name:        "Runed Brass Chest",
			Description: "Reinforced oak and brass chest with glowing runes matching the worktable materials",
			Shape: []TileSetCoordinate{
				{0, 0}, {1, 0},
				{0, 1}, {1, 1},
			},
		},
	}
	add3Bytes, _ := json.Marshal(addItem3Payload)
	rawRes3, err := exec.Generate(ctx, AddTilesetItem, add3Bytes)
	if err != nil {
		t.Fatalf("AddTilesetItem (Item 3) failed: %v", err)
	}
	t.Logf("Step 3 succeeded in %.2fs", time.Since(t3).Seconds())

	var execRes3 ExecutionResult
	_ = json.Unmarshal(rawRes3, &execRes3)
	var content3 assetdomain.AssetContent
	_ = json.Unmarshal(execRes3.Content, &content3)
	assets.content.Items = append(assets.content.Items, content3.Items...)
	encoded3, _ := assetdomain.EncodeContent(assets.content)
	assets.asset.Content = encoded3

	// --- Step 4: Incremental Add Item 4 (Alchemist Cauldron) with Item 1 Reference ---
	t.Log("=== [Step 4/4] Incremental Add Item 4: Alchemist Cauldron (using Item 1 unprocessed reference) ===")
	t4 := time.Now()
	addItem4Payload := AddTilesetItemPayload{
		ProjectID:         88,
		AssetID:           assetID,
		CreativeBrief:     themeBrief + ", dark iron bubbling potion cauldron with brass stand and glowing green magical vapor",
		CreatingReference: item1UnprocessedKey,
		Item: &AddTileSetItemDefinition{
			Name:        "Alchemist Cauldron",
			Description: "Iron and brass bubbling cauldron with glowing green brew matching the laboratory style",
			Shape: []TileSetCoordinate{
				{0, 0}, {1, 0},
				{0, 1}, {1, 1},
			},
		},
	}
	add4Bytes, _ := json.Marshal(addItem4Payload)
	rawRes4, err := exec.Generate(ctx, AddTilesetItem, add4Bytes)
	if err != nil {
		t.Fatalf("AddTilesetItem (Item 4) failed: %v", err)
	}
	t.Logf("Step 4 succeeded in %.2fs", time.Since(t4).Seconds())

	var execRes4 ExecutionResult
	_ = json.Unmarshal(rawRes4, &execRes4)
	var content4 assetdomain.AssetContent
	_ = json.Unmarshal(execRes4.Content, &content4)
	assets.content.Items = append(assets.content.Items, content4.Items...)

	// --- Stitch complete 8x8 TileSet grid composite ---
	gridCols, gridRows := 8, 8
	tileW, tileH := 32, 32
	canvas := image.NewRGBA(image.Rect(0, 0, gridCols*tileW, gridRows*tileH))

	// Checkerboard tile grid background
	for y := range gridRows * tileH {
		for x := range gridCols * tileW {
			if (x/tileW+y/tileH)%2 == 0 {
				canvas.SetRGBA(x, y, color.RGBA{R: 35, G: 38, B: 45, A: 255})
			} else {
				canvas.SetRGBA(x, y, color.RGBA{R: 45, G: 48, B: 55, A: 255})
			}
		}
	}

	t.Logf("Rendering final composite with %d generated items...", len(assets.content.Items))
	for itemIdx, it := range assets.content.Items {
		t.Logf("  Item %d: %s (%d tiles)", itemIdx+1, it.Name, len(it.Tiles))
		for _, tile := range it.Tiles {
			if tile.URL == nil {
				continue
			}
			dataURI, ok := references.objects[*tile.URL]
			if !ok {
				continue
			}
			img := decodeDataURIToImage(t, dataURI)
			destRect := image.Rect(tile.Position.X*tileW, tile.Position.Y*tileH, (tile.Position.X+1)*tileW, (tile.Position.Y+1)*tileH)
			draw.Draw(canvas, destRect, img, image.Point{}, draw.Over)
			t.Logf("    Tile (%d, %d) -> %s", tile.Position.X, tile.Position.Y, *tile.URL)
		}
	}

	compositePath := filepath.Join(artifactDir, "generated_tileset_multi_item_sequence.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err == nil {
		_ = os.WriteFile(compositePath, buf.Bytes(), 0o600)
		t.Logf("Saved multi-item sequence tileset composite to: %s", compositePath)
	}
}
