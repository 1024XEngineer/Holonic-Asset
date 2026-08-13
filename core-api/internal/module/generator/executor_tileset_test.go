package generator

import (
	"context"
	"image"
	"image/color"
	"strings"
	"sync"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

type tileSetGenerationImageStub struct {
	mu       sync.Mutex
	requests []*imageclient.GenerateRequest
}

func (s *tileSetGenerationImageStub) Generate(
	_ context.Context,
	request *imageclient.GenerateRequest,
) (*imageclient.GenerateResult, error) {
	copyRequest := *request
	copyRequest.ReferenceImages = append([]string(nil), request.ReferenceImages...)
	s.mu.Lock()
	s.requests = append(s.requests, &copyRequest)
	s.mu.Unlock()

	guide, err := imageprocessor.DecodeBase64Image(
		strings.TrimPrefix(request.ReferenceImages[0], "data:image/png;base64,"),
	)
	if err != nil {
		return nil, err
	}
	generated := image.NewRGBA(guide.Bounds())
	for y := guide.Bounds().Min.Y; y < guide.Bounds().Max.Y; y++ {
		for x := guide.Bounds().Min.X; x < guide.Bounds().Max.X; x++ {
			pixel := guide.RGBAAt(x, y)
			if pixel.R == 0 && pixel.G == 0 && pixel.B == 0 {
				generated.SetRGBA(x, y, color.RGBA{R: 128, G: 72, B: 40, A: 255})
			} else {
				generated.SetRGBA(x, y, color.RGBA{G: 255, A: 255})
			}
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(generated)
	if err != nil {
		return nil, err
	}
	return &imageclient.GenerateResult{
		Images: []imageclient.GeneratedImage{{Base64: encoded, MediaType: "image/png"}},
	}, nil
}

type tileSetGenerationProjectStub struct {
	project *projectdomain.Project
}

func (s *tileSetGenerationProjectStub) GetDetail(
	_ context.Context,
	_ uint,
) (*projectdomain.Project, error) {
	return s.project, nil
}

func TestProcessTileSetItemsBuildsGuideMaskAndOccupiedTiles(t *testing.T) {
	images := &tileSetGenerationImageStub{}
	executor := &executor{
		images:    images,
		processor: imageprocessor.NewProcessor(),
		projects: &tileSetGenerationProjectStub{project: &projectdomain.Project{
			ID:             9,
			Name:           "Forest Game",
			GameType:       "RPG",
			Description:    "woodland paths",
			Style:          "limited palette",
			TargetPlatform: projectdomain.PlatformTypePC,
			Perspective:    projectdomain.PerspectiveTopDown,
			Reference:      "projects/9/reference.png",
		}},
	}
	request := CreateTileSetPayload{
		ProjectID:     9,
		AssetName:     "Forest Terrain",
		CreativeBrief: "compact forest terrain",
		Dimensions: assetdomain.TileSetDimensions{
			TileSize:   assetdomain.Size{Width: 16, Height: 16},
			TileAmount: assetdomain.TileAmount{Columns: 8, Rows: 8},
		},
		Items: []TileSetItemDefinition{
			{Name: "Edge", Description: "grass edge", Shape: []TileSetCoordinate{{0, 0}, {1, 0}}},
			{Name: "Corner", Description: "inside corner", Shape: []TileSetCoordinate{{0, 0}, {1, 0}, {0, 1}}},
		},
	}

	processed, err := executor.processTileSetItems(context.Background(), request)
	if err != nil {
		t.Fatalf("process Tileset Items: %v", err)
	}
	if len(processed) != 2 || processed[0].Name != "Edge" || processed[1].Name != "Corner" {
		t.Fatalf("unexpected processed Items: %+v", processed)
	}
	if len(processed[0].Tiles) != 2 || len(processed[1].Tiles) != 3 {
		t.Fatalf("unexpected occupied Tile counts: %d, %d", len(processed[0].Tiles), len(processed[1].Tiles))
	}
	images.mu.Lock()
	requests := append([]*imageclient.GenerateRequest(nil), images.requests...)
	images.mu.Unlock()
	if len(requests) != 2 {
		t.Fatalf("expected one generation per Item, got %d", len(requests))
	}
	for _, imageRequest := range requests {
		if len(imageRequest.ReferenceImages) != 2 ||
			!strings.HasPrefix(imageRequest.ReferenceImages[0], "data:image/png;base64,") ||
			imageRequest.ReferenceImages[1] != "projects/9/reference.png" {
			t.Fatalf("unexpected reference order: %+v", imageRequest.ReferenceImages)
		}
		if !strings.HasPrefix(imageRequest.MaskImage, "data:image/png;base64,") {
			t.Fatalf("missing inline Shape mask: %q", imageRequest.MaskImage)
		}
		assertTileSetGuideMatchesMask(t, imageRequest.ReferenceImages[0], imageRequest.MaskImage)
		for _, expected := range []string{
			"NON-OVERRIDABLE STYLE RULES",
			"classic low-resolution 2D pixel art",
			"Forest Game",
			"compact forest terrain",
		} {
			if !strings.Contains(imageRequest.Prompt, expected) {
				t.Fatalf("prompt is missing %q: %s", expected, imageRequest.Prompt)
			}
		}
	}
}

func TestProcessTileSetItemsRejectsEmptyShape(t *testing.T) {
	executor := &executor{
		images:    &tileSetGenerationImageStub{},
		processor: imageprocessor.NewProcessor(),
		projects:  &tileSetGenerationProjectStub{},
	}
	request := CreateTileSetPayload{
		ProjectID:     9,
		AssetName:     "Forest Terrain",
		CreativeBrief: "compact forest terrain",
		Dimensions: assetdomain.TileSetDimensions{
			TileSize:   assetdomain.Size{Width: 16, Height: 16},
			TileAmount: assetdomain.TileAmount{Columns: 8, Rows: 8},
		},
		Items: []TileSetItemDefinition{{Name: "Edge", Description: "grass edge"}},
	}

	_, err := executor.processTileSetItems(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "shape must contain between") {
		t.Fatalf("expected empty Shape validation error, got %v", err)
	}
}

func assertTileSetGuideMatchesMask(t *testing.T, guideDataURL string, maskDataURL string) {
	t.Helper()
	guide, err := imageprocessor.DecodeBase64Image(
		strings.TrimPrefix(guideDataURL, "data:image/png;base64,"),
	)
	if err != nil {
		t.Fatalf("decode guide: %v", err)
	}
	mask, err := imageprocessor.DecodeBase64Image(
		strings.TrimPrefix(maskDataURL, "data:image/png;base64,"),
	)
	if err != nil {
		t.Fatalf("decode mask: %v", err)
	}
	if guide.Bounds() != mask.Bounds() {
		t.Fatalf("guide bounds %v do not match mask bounds %v", guide.Bounds(), mask.Bounds())
	}
	for y := guide.Bounds().Min.Y; y < guide.Bounds().Max.Y; y++ {
		for x := guide.Bounds().Min.X; x < guide.Bounds().Max.X; x++ {
			guidePixel := guide.RGBAAt(x, y)
			maskAlpha := mask.RGBAAt(x, y).A
			guideEditable := guidePixel.R == 0 && guidePixel.G == 0 && guidePixel.B == 0
			maskEditable := maskAlpha == 0
			if guideEditable != maskEditable {
				t.Fatalf("guide/mask mismatch at (%d,%d)", x, y)
			}
		}
	}
}

func TestAlignTileSetImageToShapeRecoversUShapeTopBar(t *testing.T) {
	const tileSize = 16
	original := image.NewRGBA(image.Rect(0, 0, 3*tileSize, 3*tileSize))
	paint := color.RGBA{R: 220, G: 140, B: 40, A: 255}
	for y := 8; y < 24; y++ {
		for x := 2; x < 46; x++ {
			original.SetRGBA(x, y, paint)
		}
	}
	for y := 16; y < 46; y++ {
		for x := 2; x < 13; x++ {
			original.SetRGBA(x, y, paint)
		}
		for x := 35; x < 46; x++ {
			original.SetRGBA(x, y, paint)
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(original)
	if err != nil {
		t.Fatal(err)
	}
	shape := []TileSetCoordinate{{0, 0}, {1, 0}, {2, 0}, {0, 1}, {2, 1}, {0, 2}, {2, 2}}
	alignedBase64, err := alignTileSetImageToShape(encoded, shape, 3, 3, tileSize, tileSize)
	if err != nil {
		t.Fatalf("align U-shaped Item: %v", err)
	}
	aligned, err := imageprocessor.DecodeBase64Image(alignedBase64)
	if err != nil {
		t.Fatal(err)
	}
	if aligned.RGBAAt(tileSize+4, 4).A == 0 {
		t.Fatal("top bar was not moved into the valid top-centre cell")
	}
	if aligned.RGBAAt(tileSize+4, tileSize+4).A != 0 {
		t.Fatal("U centre gap contains visible content")
	}
}

func TestAlignTileSetImageToShapeRejectsUnrecoverableClipping(t *testing.T) {
	const tileSize = 16
	original := image.NewRGBA(image.Rect(0, 0, 4*tileSize, 3*tileSize))
	paint := color.RGBA{R: 60, G: 130, B: 220, A: 255}
	shape := []TileSetCoordinate{{0, 0}, {1, 0}, {1, 1}, {2, 1}, {2, 2}, {3, 2}}
	for _, cell := range shape {
		for y := cell[1]*tileSize + 5; y < cell[1]*tileSize+11; y++ {
			for x := cell[0]*tileSize + 5; x < cell[0]*tileSize+11; x++ {
				original.SetRGBA(x, y, paint)
			}
		}
	}
	for y := tileSize + 1; y < 2*tileSize-1; y++ {
		for x := 1; x < tileSize-1; x++ {
			original.SetRGBA(x, y, paint)
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(original)
	if err != nil {
		t.Fatal(err)
	}
	_, err = alignTileSetImageToShape(encoded, shape, 4, 3, tileSize, tileSize)
	if err == nil || !strings.Contains(err.Error(), "cannot fit Shape without clipping") {
		t.Fatalf("expected protected-boundary error, got %v", err)
	}
}
