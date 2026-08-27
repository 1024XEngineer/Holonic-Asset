package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
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
		if imageRequest.N != 2 || imageRequest.MaxAttempts != 2 {
			t.Fatalf("expected two candidates and MaxAttempts 2 per Item, got N=%d MaxAttempts=%d", imageRequest.N, imageRequest.MaxAttempts)
		}
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

func TestProcessTileSetItemUsesSecondValidCandidate(t *testing.T) {
	images := &tileSetCandidateImageStub{}
	executor := &executor{
		images: images, processor: imageprocessor.NewProcessor(),
		projects: &tileSetGenerationProjectStub{project: &projectdomain.Project{
			ID: 9, Name: "Forest", Perspective: projectdomain.PerspectiveTopDown,
		}},
	}
	request := CreateTileSetPayload{
		ProjectID: 9, AssetName: "Forest", CreativeBrief: "terrain",
		Dimensions: assetdomain.TileSetDimensions{
			TileSize:   assetdomain.Size{Width: 16, Height: 16},
			TileAmount: assetdomain.TileAmount{Columns: 4, Rows: 4},
		},
		Items: []TileSetItemDefinition{{Name: "L", Description: "corner", Shape: []TileSetCoordinate{{0, 0}, {0, 1}, {1, 1}}}},
	}

	processed, err := executor.processTileSetItems(context.Background(), request)
	if err != nil {
		t.Fatalf("process second Tileset candidate: %v", err)
	}
	if len(processed) != 1 || len(processed[0].Tiles) != 3 || images.request == nil || images.request.N != 2 {
		t.Fatalf("unexpected second-candidate result: processed=%+v request=%+v", processed, images.request)
	}
}

type tileSetRetryImageStub struct {
	calls int
}

func (s *tileSetRetryImageStub) Generate(
	_ context.Context,
	request *imageclient.GenerateRequest,
) (*imageclient.GenerateResult, error) {
	s.calls++
	guide, err := imageprocessor.DecodeBase64Image(strings.TrimPrefix(request.ReferenceImages[0], "data:image/png;base64,"))
	if err != nil {
		return nil, err
	}
	empty, err := imageprocessor.EncodePNGBase64(image.NewRGBA(guide.Bounds()))
	if err != nil {
		return nil, err
	}
	if s.calls == 1 {
		return &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{{Base64: empty}, {Base64: empty}}}, nil
	}
	valid := image.NewRGBA(guide.Bounds())
	for y := guide.Bounds().Min.Y; y < guide.Bounds().Max.Y; y++ {
		for x := guide.Bounds().Min.X; x < guide.Bounds().Max.X; x++ {
			pixel := guide.RGBAAt(x, y)
			if pixel.R == 0 && pixel.G == 0 && pixel.B == 0 {
				valid.SetRGBA(x, y, color.RGBA{R: 160, G: 90, B: 30, A: 255})
			} else {
				valid.SetRGBA(x, y, color.RGBA{G: 255, A: 255})
			}
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(valid)
	if err != nil {
		return nil, err
	}
	return &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{{Base64: encoded}}}, nil
}

func TestProcessTileSetItemRegeneratesAfterInvalidCandidates(t *testing.T) {
	images := &tileSetRetryImageStub{}
	executor := &executor{
		images: images, processor: imageprocessor.NewProcessor(),
		projects: &tileSetGenerationProjectStub{project: &projectdomain.Project{
			ID: 9, Name: "Forest", Perspective: projectdomain.PerspectiveTopDown,
		}},
	}
	request := CreateTileSetPayload{
		ProjectID: 9, AssetName: "Forest", CreativeBrief: "terrain",
		Dimensions: assetdomain.TileSetDimensions{
			TileSize:   assetdomain.Size{Width: 16, Height: 16},
			TileAmount: assetdomain.TileAmount{Columns: 4, Rows: 4},
		},
		Items: []TileSetItemDefinition{{
			Name: "L", Description: "corner", Shape: []TileSetCoordinate{{0, 0}, {0, 1}, {1, 1}},
		}},
	}

	processed, err := executor.processTileSetItems(context.Background(), request)
	if err != nil {
		t.Fatalf("regenerate Tileset item: %v", err)
	}
	if images.calls != 2 || len(processed) != 1 || len(processed[0].Tiles) != 3 {
		t.Fatalf("expected one provider retry: calls=%d processed=%+v", images.calls, processed)
	}
}

func TestVerifyTileSetNoGuideLeakRejectsLargeNearBlackComponent(t *testing.T) {
	encode := func(value image.Image) string {
		encoded, err := imageprocessor.EncodePNGBase64(value)
		if err != nil {
			t.Fatal(err)
		}
		return encoded
	}
	for _, size := range []int{16, 32, 64} {
		outlined := image.NewRGBA(image.Rect(0, 0, size, size))
		for y := 1; y < size-1; y++ {
			for x := 1; x < size-1; x++ {
				pixel := color.RGBA{R: 90, G: 55, B: 130, A: 255}
				if x == 1 || y == 1 || x == size-2 || y == size-2 {
					pixel = color.RGBA{R: 4, G: 4, B: 4, A: 255}
				}
				outlined.SetRGBA(x, y, pixel)
			}
		}
		if err := verifyTileSetNoGuideLeak(encode(outlined)); err != nil {
			t.Fatalf("%dx%d black outline was rejected: %v", size, size, err)
		}
	}
	centralDarkSubject := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 2; y < 14; y++ {
		for x := 2; x < 14; x++ {
			centralDarkSubject.SetRGBA(x, y, color.RGBA{R: 4, G: 4, B: 4, A: 255})
		}
	}
	if err := verifyTileSetNoGuideLeak(encode(centralDarkSubject)); err != nil {
		t.Fatalf("central solid dark subject was rejected: %v", err)
	}

	leaked := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			leaked.SetRGBA(x, y, color.RGBA{R: 4, G: 4, B: 4, A: 255})
		}
	}
	if err := verifyTileSetNoGuideLeak(encode(leaked)); err == nil || !strings.Contains(err.Error(), "occupancy-guide") {
		t.Fatalf("expected guide residue rejection, got %v", err)
	}
}

func TestProcessTileSetItemCandidatePreservesWholeCanvasBeforeGridSplit(t *testing.T) {
	const tileSize = 16
	shape := []TileSetCoordinate{
		{0, 0}, {1, 0}, {2, 0},
		{0, 1}, {1, 1}, {2, 1},
		{0, 2}, {1, 2}, {2, 2},
	}
	source := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			source.SetRGBA(x, y, color.RGBA{G: 255, A: 255})
		}
	}
	for y := 8; y < 56; y++ {
		for x := 8; x < 56; x++ {
			source.SetRGBA(x, y, color.RGBA{R: 150, G: 86, B: 43, A: 255})
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(source)
	if err != nil {
		t.Fatal(err)
	}
	item := TileSetItemDefinition{Name: "Floor", Description: "continuous floor", Shape: shape}
	result, err := (&executor{processor: imageprocessor.NewProcessor()}).processTileSetItemCandidate(
		context.Background(),
		CreateTileSetPayload{
			Dimensions: assetdomain.TileSetDimensions{
				TileSize:   assetdomain.Size{Width: tileSize, Height: tileSize},
				TileAmount: assetdomain.TileAmount{Columns: 3, Rows: 3},
			},
		},
		item,
		0,
		0,
		3,
		3,
		shape,
		imageclient.GeneratedImage{Base64: encoded, MediaType: "image/png"},
		assetdomain.PerspectiveTopDown,
	)
	if err != nil {
		t.Fatalf("process Tileset Item: %v", err)
	}
	if len(result.Tiles) != len(shape) {
		t.Fatalf("got %d Tiles, want %d", len(result.Tiles), len(shape))
	}
	whole, err := imageprocessor.DecodeBase64Image(result.ImageBase64)
	if err != nil {
		t.Fatalf("decode whole Item: %v", err)
	}
	if whole.RGBAAt(0, 0).A != 0 || whole.RGBAAt(7, 7).A == 0 || whole.RGBAAt(47, 47).A != 0 {
		t.Fatal("whole-canvas normalization did not preserve the model's transparent margin")
	}
	reassembled := image.NewRGBA(whole.Bounds())
	for index, tile := range result.Tiles {
		decoded, decodeErr := imageprocessor.DecodeBase64Image(tile.ImageBase64)
		if decodeErr != nil {
			t.Fatalf("decode Tile %d: %v", index, decodeErr)
		}
		if decoded.Bounds().Size() != image.Pt(tileSize, tileSize) {
			t.Fatalf("Tile %d size = %v, want %dx%d", index, decoded.Bounds().Size(), tileSize, tileSize)
		}
		destination := image.Rect(
			(index%3)*tileSize,
			(index/3)*tileSize,
			(index%3+1)*tileSize,
			(index/3+1)*tileSize,
		)
		draw.Draw(reassembled, destination, decoded, decoded.Bounds().Min, draw.Src)
	}
	for y := range whole.Bounds().Dy() {
		for x := range whole.Bounds().Dx() {
			if got, want := reassembled.RGBAAt(x, y), whole.RGBAAt(x, y); got != want {
				t.Fatalf("reassembled pixel (%d,%d) = %+v, want %+v", x, y, got, want)
			}
		}
	}
}

type tileSetCandidateImageStub struct {
	request *imageclient.GenerateRequest
}

func (s *tileSetCandidateImageStub) Generate(
	_ context.Context,
	request *imageclient.GenerateRequest,
) (*imageclient.GenerateResult, error) {
	copyRequest := *request
	s.request = &copyRequest
	guide, err := imageprocessor.DecodeBase64Image(strings.TrimPrefix(request.ReferenceImages[0], "data:image/png;base64,"))
	if err != nil {
		return nil, err
	}
	valid := image.NewRGBA(guide.Bounds())
	for y := guide.Bounds().Min.Y; y < guide.Bounds().Max.Y; y++ {
		for x := guide.Bounds().Min.X; x < guide.Bounds().Max.X; x++ {
			pixel := guide.RGBAAt(x, y)
			if pixel.R == 0 && pixel.G == 0 && pixel.B == 0 {
				valid.SetRGBA(x, y, color.RGBA{R: 160, G: 90, B: 30, A: 255})
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

func TestAlignTileSetImageToShapeRejectsTransparentFullShape(t *testing.T) {
	const tileSize = 16
	original := image.NewRGBA(image.Rect(0, 0, 2*tileSize, tileSize))
	encoded, err := imageprocessor.EncodePNGBase64(original)
	if err != nil {
		t.Fatal(err)
	}

	_, err = alignTileSetImageToShape(
		encoded,
		[]TileSetCoordinate{{0, 0}, {1, 0}},
		2,
		1,
		tileSize,
		tileSize,
	)
	if err == nil || !strings.Contains(err.Error(), "no visible pixels") {
		t.Fatalf("expected transparent Item error, got %v", err)
	}
}

func TestAlignTileSetImageToShapeClipsModelSpillFromOmittedCells(t *testing.T) {
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
	alignedBase64, err := alignTileSetImageToShape(encoded, shape, 4, 3, tileSize, tileSize)
	if err != nil {
		t.Fatalf("align Shape with model spill: %v", err)
	}
	aligned, err := imageprocessor.DecodeBase64Image(alignedBase64)
	if err != nil {
		t.Fatal(err)
	}
	if aligned.RGBAAt(4, tileSize+4).A != 0 {
		t.Fatal("model spill remained visible in an omitted cell")
	}
}

func TestNormalizeTileSetCandidatePNG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			img.SetRGBA(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var jpegBuf bytes.Buffer
	if err := jpeg.Encode(&jpegBuf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode test jpeg: %v", err)
	}
	candidate := imageclient.GeneratedImage{
		Base64:    base64.StdEncoding.EncodeToString(jpegBuf.Bytes()),
		MediaType: "image/jpeg",
	}
	pngB64, err := normalizeTileSetCandidatePNG(candidate)
	if err != nil {
		t.Fatalf("normalize candidate PNG: %v", err)
	}
	pngBytes, err := base64.StdEncoding.DecodeString(pngB64)
	if err != nil {
		t.Fatalf("decode normalized PNG: %v", err)
	}
	if !imageprocessor.IsPNGBytes(pngBytes) {
		t.Fatal("normalized image is not a PNG")
	}
}

func TestProcessTileSetItemCandidateNormalizesRawImageToPNG(t *testing.T) {
	const tileSize = 16
	img := image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))
	// Fill with green background and an opaque red box in the middle
	for y := range tileSize {
		for x := range tileSize {
			if x >= 2 && x < tileSize-2 && y >= 2 && y < tileSize-2 {
				img.SetRGBA(x, y, color.RGBA{R: 180, G: 50, B: 30, A: 255})
			} else {
				img.SetRGBA(x, y, color.RGBA{G: 255, A: 255})
			}
		}
	}
	encodedPNG, err := imageprocessor.EncodePNGBase64(img)
	if err != nil {
		t.Fatal(err)
	}
	var jpegBuf bytes.Buffer
	if err := jpeg.Encode(&jpegBuf, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	candidate := imageclient.GeneratedImage{
		Base64:    encodedPNG,
		MediaType: "image/png",
	}

	exec := &executor{
		processor: imageprocessor.NewProcessor(),
	}
	request := CreateTileSetPayload{
		Dimensions: assetdomain.TileSetDimensions{
			TileSize: assetdomain.Size{Width: tileSize, Height: tileSize},
		},
	}
	item := TileSetItemDefinition{Name: "Block", Shape: []TileSetCoordinate{{0, 0}}}
	result, err := exec.processTileSetItemCandidate(
		context.Background(),
		request,
		item,
		0,
		0,
		1,
		1,
		[]TileSetCoordinate{{0, 0}},
		candidate,
		assetdomain.PerspectiveTopDown,
	)
	if err != nil {
		t.Fatalf("process candidate: %v", err)
	}
	if result.RawMediaType != "image/png" {
		t.Fatalf("raw media type = %q, want image/png", result.RawMediaType)
	}
	if result.MIMEType != "image/png" {
		t.Fatalf("mime type = %q, want image/png", result.MIMEType)
	}
	rawBytes, err := base64.StdEncoding.DecodeString(result.RawImageBase64)
	if err != nil || !imageprocessor.IsPNGBytes(rawBytes) {
		t.Fatalf("raw image is not valid PNG bytes")
	}
}
