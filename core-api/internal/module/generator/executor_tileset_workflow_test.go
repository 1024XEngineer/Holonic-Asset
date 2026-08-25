package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

func TestTileSetWorkflowGeneratesEditsTileAndEditsCompleteItem(t *testing.T) {
	assets := &tileSetWorkflowAssets{}
	references := &tileSetWorkflowReferences{objects: make(map[string]string)}
	executor := &executor{
		images: &tileSetWorkflowImages{}, processor: imageprocessor.NewProcessor(), assets: assets,
		projects: &tileSetGenerationProjectStub{project: &projectdomain.Project{
			ID: 42, Name: "Ruined Observatory", GameType: "RPG", Description: "moonlit ruins",
			Style: "limited indigo, brass and moss palette", TargetPlatform: projectdomain.PlatformTypePC,
			Perspective: projectdomain.PerspectiveTopDown,
		}},
		references: references,
	}
	create := CreateTileSetPayload{
		AssetName: "Observatory Props", ProjectID: 42,
		CreativeBrief: "weathered arcane furniture with readable silhouettes and connected cross-tile seams",
		Dimensions: assetdomain.TileSetDimensions{
			TileSize:   assetdomain.Size{Width: 16, Height: 16},
			TileAmount: assetdomain.TileAmount{Columns: 8, Rows: 8},
		},
		Items: []TileSetItemDefinition{
			{Name: "U desk", Description: "horseshoe scholar desk", Shape: []TileSetCoordinate{{0, 0}, {1, 0}, {2, 0}, {0, 1}, {2, 1}}},
			{Name: "Z shelf", Description: "zigzag alchemy shelf", Shape: []TileSetCoordinate{{0, 0}, {1, 0}, {1, 1}, {2, 1}}},
		},
	}
	createPayload, err := json.Marshal(create)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := executor.Generate(context.Background(), GenerateTileSet, createPayload)
	if err != nil {
		t.Fatalf("generate Tileset workflow: %v", err)
	}
	assertTileSetWorkflowResult(t, generated, 100, 1)
	if assets.asset.Version != 1 || len(assets.content.Items) != 2 || len(references.objects) != 11 {
		t.Fatalf("unexpected generated state: asset=%+v content=%+v objects=%d", assets.asset, assets.content, len(references.objects))
	}
	beforeTileEdit := tileSetWorkflowURLs(assets.content)
	tilePosition := assets.content.Items[0].Tiles[1].Position
	x, y := tilePosition.X, tilePosition.Y
	tileEdit := EditTilesPayload{
		AssetID: 100, ProjectID: 42, CreativeBrief: "inlay one tiny gold moon rune",
		Targets: []TileSetEditTarget{{Position: &TileSetEditPosition{X: &x, Y: &y}}},
	}
	tilePayload, err := json.Marshal(tileEdit)
	if err != nil {
		t.Fatal(err)
	}
	editedTile, err := executor.Generate(context.Background(), EditTiles, tilePayload)
	if err != nil {
		t.Fatalf("edit one Tile workflow: %v", err)
	}
	assertTileSetWorkflowResult(t, editedTile, 100, 1)
	applyTileSetWorkflowCandidate(t, assets, editedTile)
	afterTileEdit := tileSetWorkflowURLs(assets.content)
	if changed := changedTileSetWorkflowURLs(beforeTileEdit, afterTileEdit); len(changed) != 1 || changed[0] != "0.1" {
		t.Fatalf("single-Tile edit changed unexpected URLs: %v", changed)
	}
	beforeItemEdit := afterTileEdit
	itemPosition := assets.content.Items[1].Tiles[0].Position
	x, y = itemPosition.X, itemPosition.Y
	itemEdit := EditTilesetItemPayload{
		AssetID: 100, ProjectID: 42, CreativeBrief: "replace moss with luminous violet crystal veins",
		Target: &TileSetEditTarget{Position: &TileSetEditPosition{X: &x, Y: &y}},
	}
	itemPayload, err := json.Marshal(itemEdit)
	if err != nil {
		t.Fatal(err)
	}
	editedItem, err := executor.Generate(context.Background(), EditTilesetItem, itemPayload)
	if err != nil {
		t.Fatalf("edit complete Item workflow: %v", err)
	}
	assertTileSetWorkflowResult(t, editedItem, 100, 2)
	applyTileSetWorkflowCandidate(t, assets, editedItem)
	afterItemEdit := tileSetWorkflowURLs(assets.content)
	changed := changedTileSetWorkflowURLs(beforeItemEdit, afterItemEdit)
	if len(changed) != 4 {
		t.Fatalf("complete Item edit changed %d URLs, want 4: %v", len(changed), changed)
	}
	for _, path := range changed {
		if !strings.HasPrefix(path, "1.") {
			t.Fatalf("complete Item edit changed non-target Item URL %q", path)
		}
	}
}

func TestTileSetEditReturnsCandidateWithoutMutatingPersistedState(t *testing.T) {
	assets := &tileSetWorkflowAssets{}
	existing := "uploads/existing.png"
	content := assetdomain.AssetContent{Items: []assetdomain.TileSetItem{{
		Name:  "Pot",
		Tiles: []assetdomain.Tile{{URL: &existing, Position: assetdomain.TilePosition{X: 0, Y: 0}}},
	}}}
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	assets.asset = assetdomain.Asset{
		ID: 100, ProjectID: 42, Type: assetdomain.AssetTypeTileSet, Version: 1,
		Perspective: assetdomain.PerspectiveTopDown,
		Dimensions:  json.RawMessage(`{"tileSize":{"width":16,"height":16},"tileAmount":{"columns":4,"rows":4}}`),
		Content:     encoded,
	}
	assets.content = content
	tileImage := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 2; y < 14; y++ {
		for x := 2; x < 14; x++ {
			tileImage.SetRGBA(x, y, color.RGBA{R: 120, G: 70, B: 40, A: 255})
		}
	}
	tileBase64, err := imageprocessor.EncodePNGBase64(tileImage)
	if err != nil {
		t.Fatal(err)
	}
	references := &tileSetWorkflowReferences{objects: map[string]string{
		existing: "data:image/png;base64," + tileBase64,
	}}
	executor := &executor{
		images: &tileSetWorkflowImages{}, processor: imageprocessor.NewProcessor(), assets: assets,
		projects: &tileSetGenerationProjectStub{project: &projectdomain.Project{
			ID: 42, Name: "Ruins", Perspective: projectdomain.PerspectiveTopDown,
		}},
		references: references,
	}
	x, y := 0, 0
	payload, err := json.Marshal(EditTilesPayload{
		AssetID: 100, ProjectID: 42, CreativeBrief: "add a crack",
		Targets: []TileSetEditTarget{{Position: &TileSetEditPosition{X: &x, Y: &y}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := executor.Generate(context.Background(), EditTiles, payload)
	if err != nil {
		t.Fatalf("generate Tileset edit candidate: %v", err)
	}
	if _, retained := references.objects[existing]; !retained {
		t.Fatal("revision cleanup deleted the historical Tile object")
	}
	if len(references.deleted) != 0 {
		t.Fatalf("candidate resources must remain until apply or discard: %v", references.deleted)
	}
	if assets.asset.Version != 1 || *assets.content.Items[0].Tiles[0].URL != existing {
		t.Fatalf("candidate generation mutated persisted state: asset=%+v content=%+v", assets.asset, assets.content)
	}
	var application ExecutionResult
	if err := json.Unmarshal(result, &application); err != nil || application.Version != 1 || len(application.GeneratedResources) != 1 {
		t.Fatalf("unexpected Tileset application candidate: result=%+v err=%v", application, err)
	}
}

func TestAddTileSetItemReturnsCandidateWithoutMutatingPersistedState(t *testing.T) {
	existing := "uploads/existing.png"
	content := assetdomain.AssetContent{Items: []assetdomain.TileSetItem{{
		Name:  "Pot",
		Tiles: []assetdomain.Tile{{URL: &existing, Position: assetdomain.TilePosition{X: 0, Y: 0}}},
	}}}
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	assets := &tileSetWorkflowAssets{
		asset: assetdomain.Asset{
			ID: 100, ProjectID: 42, Type: assetdomain.AssetTypeTileSet, Version: 3,
			Perspective: assetdomain.PerspectiveTopDown,
			Dimensions:  json.RawMessage(`{"tileSize":{"width":16,"height":16},"tileAmount":{"columns":4,"rows":2}}`),
			Content:     encoded,
		},
		content: content,
	}
	references := &tileSetWorkflowReferences{objects: map[string]string{existing: "existing"}}
	executor := &executor{
		images: &tileSetWorkflowImages{}, processor: imageprocessor.NewProcessor(), assets: assets,
		projects: &tileSetGenerationProjectStub{project: &projectdomain.Project{
			ID: 42, Name: "Ruins", Perspective: projectdomain.PerspectiveTopDown,
		}},
		references: references,
	}
	payload, err := json.Marshal(AddTilesetItemPayload{
		AssetID: 100, ProjectID: 42, CreativeBrief: "add observatory furniture",
		Item: &AddTileSetItemDefinition{
			Name: "Bench", Description: "two-cell brass bench", Shape: []TileSetCoordinate{{0, 0}, {1, 0}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	resultRaw, err := executor.Generate(context.Background(), AddTilesetItem, payload)
	if err != nil {
		t.Fatalf("add Tileset Item candidate: %v", err)
	}
	var result ExecutionResult
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		t.Fatal(err)
	}
	if result.AssetID != 100 || result.Version != 3 || len(result.GeneratedResources) != 3 {
		t.Fatalf("unexpected Item addition result: %+v", result)
	}
	candidate, err := (assetdomain.Asset{Type: assetdomain.AssetTypeTileSet, Content: result.Content}).DecodeContent()
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Items) != 1 || candidate.Items[0].Name != "Bench" || len(candidate.Items[0].Tiles) != 2 {
		t.Fatalf("unexpected Item candidate: %+v", candidate.Items)
	}
	wantPositions := []assetdomain.TilePosition{{X: 1, Y: 0}, {X: 2, Y: 0}}
	for index, tile := range candidate.Items[0].Tiles {
		if tile.Position != wantPositions[index] || tile.URL == nil || *tile.URL == existing {
			t.Fatalf("unexpected candidate Tile %d: %+v", index, tile)
		}
	}
	if assets.asset.Version != 3 || len(assets.content.Items) != 1 || assets.content.Items[0].Name != "Pot" {
		t.Fatalf("Item candidate mutated persisted state: asset=%+v content=%+v", assets.asset, assets.content)
	}
}

func TestResolveTileSetUnprocessedReference(t *testing.T) {
	if got := resolveTileSetUnprocessedReference(assetdomain.AssetContent{}); got != "" {
		t.Fatalf("expected empty string for empty content, got %q", got)
	}

	emptyTileURL := ""
	contentNoURL := assetdomain.AssetContent{
		Items: []assetdomain.TileSetItem{
			{Name: "Item1", Tiles: []assetdomain.Tile{{URL: nil}, {URL: &emptyTileURL}}},
		},
	}
	if got := resolveTileSetUnprocessedReference(contentNoURL); got != "" {
		t.Fatalf("expected empty string for nil/blank tile URLs, got %q", got)
	}

	validURL := "uploads/tileset/tile-001.png"
	contentWithURL := assetdomain.AssetContent{
		Items: []assetdomain.TileSetItem{
			{Name: "Item1", Tiles: []assetdomain.Tile{{URL: &validURL}}},
		},
	}
	want := "uploads/tileset/tile-001-unprocessed.png"
	if got := resolveTileSetUnprocessedReference(contentWithURL); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestStabilizeTileSetTileEditPreservesAlphaAndSeamEdge(t *testing.T) {
	original := image.NewRGBA(image.Rect(0, 0, 4, 4))
	generated := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			original.SetRGBA(x, y, color.RGBA{R: 20, G: 40, B: 60, A: 255})
			generated.SetRGBA(x, y, color.RGBA{R: 220, G: 180, B: 30, A: 255})
		}
	}
	original.SetRGBA(1, 1, color.RGBA{})
	generated.SetRGBA(2, 1, color.RGBA{R: 250, G: 250, B: 250, A: 255})
	originalBase64, err := imageprocessor.EncodePNGBase64(original)
	if err != nil {
		t.Fatal(err)
	}
	generatedBase64, err := imageprocessor.EncodePNGBase64(generated)
	if err != nil {
		t.Fatal(err)
	}

	stabilizedBase64, err := stabilizeTileSetTileEdit(originalBase64, generatedBase64, 4, 4)
	if err != nil {
		t.Fatalf("stabilize Tile edit: %v", err)
	}
	stabilized, err := imageprocessor.DecodeBase64Image(stabilizedBase64)
	if err != nil {
		t.Fatal(err)
	}
	if stabilized.RGBAAt(1, 1).A != 0 {
		t.Fatal("stabilized edit changed the original alpha silhouette")
	}
	if stabilized.RGBAAt(0, 2) != original.RGBAAt(0, 2) {
		t.Fatal("stabilized edit changed a protected seam-edge pixel")
	}
	if stabilized.RGBAAt(2, 2).R != 220 {
		t.Fatal("stabilized edit did not preserve generated interior detail")
	}
	if stabilized.RGBAAt(2, 1) != original.RGBAAt(2, 1) {
		t.Fatal("stabilized edit retained white matte contamination")
	}
}

type tileSetWorkflowImages struct{}

func (*tileSetWorkflowImages) Generate(
	_ context.Context,
	request *imageclient.GenerateRequest,
) (*imageclient.GenerateResult, error) {
	if request.N == 2 && request.MaskImage != "" {
		guide, err := imageprocessor.DecodeBase64Image(strings.TrimPrefix(request.ReferenceImages[0], "data:image/png;base64,"))
		if err != nil {
			return nil, err
		}
		valid := image.NewRGBA(guide.Bounds())
		artwork := color.RGBA{R: 110, G: 70, B: 190, A: 255}
		if len(request.ReferenceImages) > 1 {
			artwork = color.RGBA{R: 210, G: 140, B: 45, A: 255}
		}
		for y := guide.Bounds().Min.Y; y < guide.Bounds().Max.Y; y++ {
			for x := guide.Bounds().Min.X; x < guide.Bounds().Max.X; x++ {
				pixel := guide.RGBAAt(x, y)
				if pixel.R == 0 && pixel.G == 0 && pixel.B == 0 {
					valid.SetRGBA(x, y, artwork)
				} else {
					valid.SetRGBA(x, y, color.RGBA{G: 255, A: 255})
				}
			}
		}
		validBase64, err := imageprocessor.EncodePNGBase64(valid)
		if err != nil {
			return nil, err
		}
		emptyBase64, err := imageprocessor.EncodePNGBase64(image.NewRGBA(guide.Bounds()))
		if err != nil {
			return nil, err
		}
		return &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{
			{Base64: emptyBase64, MediaType: "image/png"},
			{Base64: validBase64, MediaType: "image/png"},
		}}, nil
	}
	original, err := imageprocessor.DecodeBase64Image(strings.TrimPrefix(request.ReferenceImages[0], "data:image/png;base64,"))
	if err != nil {
		return nil, err
	}
	edited := image.NewRGBA(original.Bounds())
	for y := original.Bounds().Min.Y; y < original.Bounds().Max.Y; y++ {
		for x := original.Bounds().Min.X; x < original.Bounds().Max.X; x++ {
			pixel := original.RGBAAt(x, y)
			if pixel.A != 0 {
				edited.SetRGBA(x, y, color.RGBA{R: 230, G: 180, B: 30, A: pixel.A})
			}
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(edited)
	if err != nil {
		return nil, err
	}
	return &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{{Base64: encoded, MediaType: "image/png"}}}, nil
}

type tileSetWorkflowReferences struct {
	mu          sync.Mutex
	next        int
	objects     map[string]string
	deleted     []string
	uploadFail  error
	metadataErr error
}

func (s *tileSetWorkflowReferences) GetObjectMetadata(
	_ context.Context,
	reference string,
) (*upload.ObjectMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.metadataErr != nil {
		return nil, s.metadataErr
	}
	if _, ok := s.objects[reference]; !ok {
		return nil, upload.ErrObjectNotFound
	}
	return &upload.ObjectMetadata{ObjectKey: reference}, nil
}

func (s *tileSetWorkflowReferences) ResolveReference(_ context.Context, reference string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if stored, ok := s.objects[reference]; ok {
		return stored, nil
	}
	return reference, nil
}

func (*tileSetWorkflowReferences) PersistReference(_ context.Context, reference string) (string, error) {
	return reference, nil
}

func (s *tileSetWorkflowReferences) NewObjectKey(string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	return fmt.Sprintf("uploads/workflow-%03d.png", s.next), nil
}

func (s *tileSetWorkflowReferences) PersistReferenceAt(_ context.Context, key string, reference string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.uploadFail != nil {
		return s.uploadFail
	}
	s.objects[key] = reference
	return nil
}

func (s *tileSetWorkflowReferences) DeleteObjects(_ context.Context, keys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range keys {
		delete(s.objects, key)
	}
	s.deleted = append(s.deleted, keys...)
	return nil
}

type tileSetWorkflowAssets struct {
	asset   assetdomain.Asset
	content assetdomain.AssetContent
}

func (s *tileSetWorkflowAssets) GetDetail(_ context.Context, id uint) (assetdomain.Asset, error) {
	if s.asset.ID != id {
		return assetdomain.Asset{}, nil
	}
	return s.asset, nil
}

func (*tileSetWorkflowAssets) CreateCharacterAsset(context.Context, *assetdomain.Asset) (*assetdomain.Asset, error) {
	return nil, fmt.Errorf("unexpected character creation")
}

func (*tileSetWorkflowAssets) CreateObjectAsset(context.Context, *assetdomain.Asset) (uint, error) {
	return 0, fmt.Errorf("unexpected object creation")
}

func (*tileSetWorkflowAssets) CreateSceneryAsset(context.Context, *assetdomain.Asset) (uint, error) {
	return 0, fmt.Errorf("unexpected scenery creation")
}

func (s *tileSetWorkflowAssets) CreateTileSetAsset(_ context.Context, value *assetdomain.Asset) (uint, error) {
	s.asset = *value
	s.asset.ID = 100
	s.asset.Version = 1
	content, err := s.asset.DecodeContent()
	if err != nil {
		return 0, err
	}
	s.content = content
	return s.asset.ID, nil
}

func assertTileSetWorkflowResult(t *testing.T, raw json.RawMessage, assetID uint, version uint) {
	t.Helper()
	var result ExecutionResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode Tileset workflow result: %v", err)
	}
	if result.AssetID != assetID || result.Version != version {
		t.Fatalf("unexpected Tileset workflow result: %+v", result)
	}
}

func applyTileSetWorkflowCandidate(t *testing.T, assets *tileSetWorkflowAssets, raw json.RawMessage) {
	t.Helper()
	var result ExecutionResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode Tileset application candidate: %v", err)
	}
	content, err := (assetdomain.Asset{Type: assetdomain.AssetTypeTileSet, Content: result.Content}).DecodeContent()
	if err != nil {
		t.Fatalf("decode Tileset candidate content: %v", err)
	}
	for _, candidate := range content.Items {
		index := -1
		for itemIndex := range assets.content.Items {
			if assets.content.Items[itemIndex].Name == candidate.Name {
				index = itemIndex
				break
			}
		}
		if index >= 0 {
			for _, candidateTile := range candidate.Tiles {
				tileIndex := -1
				for currentIndex, currentTile := range assets.content.Items[index].Tiles {
					if currentTile.Position == candidateTile.Position {
						tileIndex = currentIndex
						break
					}
				}
				if tileIndex >= 0 {
					assets.content.Items[index].Tiles[tileIndex] = candidateTile
				} else {
					assets.content.Items[index].Tiles = append(assets.content.Items[index].Tiles, candidateTile)
				}
			}
		} else {
			assets.content.Items = append(assets.content.Items, candidate)
		}
	}
	encoded, err := assetdomain.EncodeContent(assets.content)
	if err != nil {
		t.Fatalf("encode merged Tileset candidate: %v", err)
	}
	assets.asset.Content = encoded
	assets.asset.Version++
}

func tileSetWorkflowURLs(content assetdomain.AssetContent) map[string]string {
	result := make(map[string]string)
	for itemIndex, item := range content.Items {
		for tileIndex, tile := range item.Tiles {
			if tile.URL != nil {
				result[fmt.Sprintf("%d.%d", itemIndex, tileIndex)] = *tile.URL
			}
		}
	}
	return result
}

func changedTileSetWorkflowURLs(before map[string]string, after map[string]string) []string {
	changed := make([]string, 0)
	for path, beforeURL := range before {
		if after[path] != beforeURL {
			changed = append(changed, path)
		}
	}
	return changed
}

var _ AssetWriter = (*tileSetWorkflowAssets)(nil)
var _ ReferenceStore = (*tileSetWorkflowReferences)(nil)
var _ imageclient.ImageGenerationService = (*tileSetWorkflowImages)(nil)
