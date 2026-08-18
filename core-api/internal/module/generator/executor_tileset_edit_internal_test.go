package generator

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

func TestResolveTileSetTargetsRejectsInvalidPersistedContent(t *testing.T) {
	x, y := 1, 1
	target := TileSetEditTarget{Position: &TileSetEditPosition{X: &x, Y: &y}}
	dimensions := assetdomain.TileSetDimensions{
		TileSize: assetdomain.Size{Width: 16, Height: 16}, TileAmount: assetdomain.TileAmount{Columns: 2, Rows: 2},
	}
	url := "uploads/tile.png"
	tests := []struct {
		name    string
		content assetdomain.AssetContent
		want    string
	}{
		{
			name: "out of grid",
			content: assetdomain.AssetContent{Items: []assetdomain.TileSetItem{{Tiles: []assetdomain.Tile{
				{URL: &url, Position: assetdomain.TilePosition{X: 2, Y: 0}},
			}}}},
			want: "out of grid",
		},
		{
			name: "duplicate position",
			content: assetdomain.AssetContent{Items: []assetdomain.TileSetItem{{Tiles: []assetdomain.Tile{
				{URL: &url, Position: assetdomain.TilePosition{X: 0, Y: 0}},
				{URL: &url, Position: assetdomain.TilePosition{X: 0, Y: 0}},
			}}}},
			want: "duplicate Tile position",
		},
		{
			name: "missing resource",
			content: assetdomain.AssetContent{Items: []assetdomain.TileSetItem{{Tiles: []assetdomain.Tile{
				{Position: assetdomain.TilePosition{X: 0, Y: 0}},
			}}}},
			want: "has no resource",
		},
		{
			name: "unoccupied target",
			content: assetdomain.AssetContent{Items: []assetdomain.TileSetItem{{Tiles: []assetdomain.Tile{
				{URL: &url, Position: assetdomain.TilePosition{X: 0, Y: 0}},
			}}}},
			want: "is not occupied",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := resolveTileSetTargets([]TileSetEditTarget{target}, test.content, dimensions)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}

func TestLoadTileSetEditContextRejectsInvalidAssetState(t *testing.T) {
	valid := tileSetEditTestAsset(t)
	wantErr := errors.New("lookup unavailable")
	tests := []struct {
		name       string
		asset      assetdomain.Asset
		assetErr   error
		project    *projectdomain.Project
		projectErr error
		projectID  uint
		want       string
	}{
		{name: "asset lookup", assetErr: wantErr, projectID: 42, want: "lookup unavailable"},
		{name: "not found", projectID: 42, want: "not found"},
		{name: "wrong type", asset: func() assetdomain.Asset { value := valid; value.Type = assetdomain.AssetTypeObject; return value }(), projectID: 42, want: "must have type"},
		{name: "project mismatch", asset: valid, projectID: 9, want: "does not belong"},
		{name: "invalid dimensions", asset: func() assetdomain.Asset { value := valid; value.Dimensions = []byte(`{}`); return value }(), projectID: 42, want: "validate Tileset"},
		{name: "oversized dimensions", asset: func() assetdomain.Asset {
			value := valid
			value.Dimensions = []byte(`{"tileSize":{"width":1024,"height":1024},"tileAmount":{"columns":65,"rows":64}}`)
			return value
		}(), projectID: 42, want: "processing limits"},
		{name: "invalid content", asset: func() assetdomain.Asset { value := valid; value.Content = []byte(`{"items":`); return value }(), projectID: 42, want: "decode Tileset"},
		{name: "project lookup", asset: valid, projectErr: wantErr, projectID: 42, want: "lookup unavailable"},
		{name: "missing project", asset: valid, projectID: 42, want: "project 42 is required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assets := &tileSetEditAssetStub{tileSetWorkflowAssets: tileSetWorkflowAssets{asset: test.asset}, detailErr: test.assetErr}
			executor := &executor{assets: assets, projects: &tileSetEditProjectStub{project: test.project, err: test.projectErr}}
			_, err := executor.loadTileSetEditContext(context.Background(), 100, test.projectID)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}

func TestProcessTileSetEditImageUsesNextValidCandidate(t *testing.T) {
	processor := &tileSetEditProcessorStub{
		removeResults: []*imageprocessor.RemoveBackgroundResult{nil, {ImageBase64: "removed"}},
		resizeResults: []*imageprocessor.ResizeResult{{ImageBase64: "resized"}},
	}
	executor := &executor{processor: processor}
	result, err := executor.processTileSetEditImage(context.Background(), &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{
		{Base64: "first"}, {Base64: "second"},
	}}, 16, 16, []TileSetCoordinate{{0, 0}})
	if err != nil {
		t.Fatalf("process fallback candidate: %v", err)
	}
	if result.ImageBase64 != "resized" || processor.removeCalls != 2 || processor.resizeCalls != 1 {
		t.Fatalf("unexpected fallback result: result=%+v processor=%+v", result, processor)
	}
}

func TestProcessTileSetEditImageReportsCandidateFailures(t *testing.T) {
	wantErr := errors.New("processor unavailable")
	tests := []struct {
		name      string
		result    *imageclient.GenerateResult
		processor *tileSetEditProcessorStub
		want      string
	}{
		{name: "missing images", want: "expected at least one image"},
		{name: "empty candidate", result: &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{{}}}, processor: &tileSetEditProcessorStub{}, want: "candidate 0 is empty"},
		{name: "remove error", result: tileSetEditCandidates("image"), processor: &tileSetEditProcessorStub{removeErrs: []error{wantErr}}, want: "processor unavailable"},
		{name: "empty remove", result: tileSetEditCandidates("image"), processor: &tileSetEditProcessorStub{removeResults: []*imageprocessor.RemoveBackgroundResult{{}}}, want: "empty background-removal"},
		{name: "resize error", result: tileSetEditCandidates("image"), processor: &tileSetEditProcessorStub{removeResults: []*imageprocessor.RemoveBackgroundResult{{ImageBase64: "removed"}}, resizeErrs: []error{wantErr}}, want: "processor unavailable"},
		{name: "empty resize", result: tileSetEditCandidates("image"), processor: &tileSetEditProcessorStub{removeResults: []*imageprocessor.RemoveBackgroundResult{{ImageBase64: "removed"}}, resizeResults: []*imageprocessor.ResizeResult{{}}}, want: "empty resize result"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := &executor{processor: test.processor}
			_, err := executor.processTileSetEditImage(context.Background(), test.result, 16, 16, nil)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}

func TestTileSetEditPixelContaminatedRejectsGuideAndMatteColours(t *testing.T) {
	for _, pixel := range []color.RGBA{
		{R: 255, G: 255, B: 255, A: 255},
		{G: 255, A: 255},
		{R: 8, G: 8, B: 8, A: 255},
		{},
	} {
		if !tileSetEditPixelContaminated(pixel) {
			t.Fatalf("expected contaminated pixel: %+v", pixel)
		}
	}
	if tileSetEditPixelContaminated(color.RGBA{R: 70, G: 40, B: 100, A: 255}) {
		t.Fatal("valid dark-purple edit colour was rejected")
	}
}

func TestValidateTileSetTileEditStructureAllowsCompleteInteriorRedraw(t *testing.T) {
	original := image.NewRGBA(image.Rect(0, 0, 16, 16))
	redrawn := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			base := color.RGBA{R: 70, G: 40, B: 100, A: 255}
			original.SetRGBA(x, y, base)
			if tileSetTileEditInSeamBorder(x, y, 16, 16) {
				redrawn.SetRGBA(x, y, base)
			} else {
				redrawn.SetRGBA(x, y, color.RGBA{R: 190, G: 140, B: 30, A: 255})
			}
		}
	}
	encode := func(value image.Image) string {
		encoded, err := imageprocessor.EncodePNGBase64(value)
		if err != nil {
			t.Fatal(err)
		}
		return encoded
	}
	originalBase64 := encode(original)
	if err := validateTileSetTileEditStructure(originalBase64, encode(redrawn)); err != nil {
		t.Fatalf("complete interior redraw was rejected: %v", err)
	}
	if err := validateTileSetEditChanged(originalBase64, encode(redrawn)); err != nil {
		t.Fatalf("complete interior redraw was treated as a no-op: %v", err)
	}
	if err := validateTileSetEditChanged(originalBase64, originalBase64); err == nil || !strings.Contains(err.Error(), "every pixel unchanged") {
		t.Fatalf("expected exact no-op rejection, got %v", err)
	}
}

func TestValidateTileSetTileEditStructureRejectsGeometryChanges(t *testing.T) {
	encode := func(value image.Image) string {
		encoded, err := imageprocessor.EncodePNGBase64(value)
		if err != nil {
			t.Fatal(err)
		}
		return encoded
	}
	original := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			original.SetRGBA(x, y, color.RGBA{R: 70, G: 40, B: 100, A: 255})
		}
	}
	original.SetRGBA(2, 2, color.RGBA{})

	tests := []struct {
		name   string
		edited *image.RGBA
		want   string
	}{
		{name: "dimensions", edited: image.NewRGBA(image.Rect(0, 0, 5, 4)), want: "dimensions changed"},
		{name: "alpha silhouette", edited: func() *image.RGBA {
			value := image.NewRGBA(original.Bounds())
			draw.Draw(value, value.Bounds(), original, original.Bounds().Min, draw.Src)
			value.SetRGBA(2, 2, color.RGBA{R: 90, G: 80, B: 70, A: 255})
			return value
		}(), want: "alpha silhouette changed"},
		{name: "seam border", edited: func() *image.RGBA {
			value := image.NewRGBA(original.Bounds())
			draw.Draw(value, value.Bounds(), original, original.Bounds().Min, draw.Src)
			value.SetRGBA(0, 2, color.RGBA{R: 190, G: 140, B: 30, A: 255})
			return value
		}(), want: "seam border changed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateTileSetTileEditStructure(encode(original), encode(test.edited))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}

func TestGenerateTileSetTileEditRetriesProviderBatch(t *testing.T) {
	original := tileSetEditTransparentTestImage(t, 16, 16)
	edited := tileSetEditTransparentTestImageWithColor(t, 16, 16, color.RGBA{R: 110, G: 70, B: 180, A: 255})
	images := &tileSetEditImageStub{results: []*imageclient.GenerateResult{
		nil,
		tileSetEditCandidates("candidate"),
	}}
	processor := &tileSetEditProcessorStub{
		removeResults: []*imageprocessor.RemoveBackgroundResult{{ImageBase64: original}},
		resizeResults: []*imageprocessor.ResizeResult{{ImageBase64: edited}},
		verifyResult: &imageprocessor.VerificationReport{
			IsPNG: true, HasAlpha: true, Width: 16, Height: 16,
			NontransparentPixels: 196, TransparentPixels: 60, TransparentRGBScrubbed: true,
		},
	}
	url := "tile"
	executor := &executor{
		images: images, processor: processor,
		references: &tileSetEditReferenceStub{resolved: original},
	}
	region, err := executor.generateTileSetTileEdit(
		context.Background(), "add detail", &projectdomain.Project{}, "Plot",
		assetdomain.Tile{URL: &url},
		assetdomain.TileSetDimensions{TileSize: assetdomain.Size{Width: 16, Height: 16}}, nil,
	)
	if err != nil {
		t.Fatalf("retry Tile edit: %v", err)
	}
	if region.ImageBase64 == "" || images.calls != 2 || len(images.requests) != 2 || images.requests[1].N != 2 || images.requests[1].MaxAttempts != 2 {
		t.Fatalf("unexpected retry result: region=%+v images=%+v", region, images)
	}
}

func TestGenerateTileSetTileEditRejectsEmptyOriginalBeforeProvider(t *testing.T) {
	original := tileSetEditTestImageWithColor(t, 16, 16, color.RGBA{})
	images := &tileSetEditImageStub{}
	url := "tile"
	executor := &executor{
		images: images, references: &tileSetEditReferenceStub{resolved: original},
	}
	_, err := executor.generateTileSetTileEdit(
		context.Background(), "redraw", &projectdomain.Project{}, "Plot",
		assetdomain.Tile{URL: &url},
		assetdomain.TileSetDimensions{TileSize: assetdomain.Size{Width: 16, Height: 16}}, nil,
	)
	if err == nil || !strings.Contains(err.Error(), "has no visible pixels") || images.calls != 0 {
		t.Fatalf("expected preflight rejection without provider call: calls=%d err=%v", images.calls, err)
	}
}

func TestGenerateTileSetTileEditDoesNotUseQualityRetriesForProviderErrors(t *testing.T) {
	original := tileSetEditTransparentTestImage(t, 16, 16)
	providerErr := &imageclient.ProviderError{Kind: imageclient.ErrorKindUnavailable, Transient: true}
	images := &tileSetEditImageStub{errors: []error{providerErr}}
	url := "tile"
	executor := &executor{
		images: images, references: &tileSetEditReferenceStub{resolved: original},
	}
	_, err := executor.generateTileSetTileEdit(
		context.Background(), "redraw", &projectdomain.Project{}, "Plot",
		assetdomain.Tile{URL: &url},
		assetdomain.TileSetDimensions{TileSize: assetdomain.Size{Width: 16, Height: 16}}, nil,
	)
	if err == nil || !errors.Is(err, providerErr) || images.calls != 1 {
		t.Fatalf("provider failure consumed quality retries: calls=%d err=%v", images.calls, err)
	}
}

func TestGenerateTileSetItemEditRetriesProviderBatch(t *testing.T) {
	original := tileSetEditTransparentTestImage(t, 16, 16)
	edited := tileSetEditTransparentTestImageWithColor(t, 16, 16, color.RGBA{R: 110, G: 70, B: 180, A: 255})
	images := &tileSetEditImageStub{results: []*imageclient.GenerateResult{
		{Images: []imageclient.GeneratedImage{{}}},
		tileSetEditCandidates("candidate"),
	}}
	processor := &tileSetEditProcessorStub{
		removeResults: []*imageprocessor.RemoveBackgroundResult{{ImageBase64: original}},
		resizeResults: []*imageprocessor.ResizeResult{{ImageBase64: edited}},
		verifyResult: &imageprocessor.VerificationReport{
			IsPNG: true, HasAlpha: true, Width: 16, Height: 16,
			NontransparentPixels: 196, TransparentPixels: 60, TransparentRGBScrubbed: true,
		},
		splitResults: []*imageprocessor.SplitImageResult{{Regions: []imageprocessor.ImageRegion{{ImageBase64: original}}}},
	}
	url := "tile"
	executor := &executor{
		images: images, processor: processor,
		references: &tileSetEditReferenceStub{resolved: original},
	}
	regions, err := executor.generateTileSetItemEdit(
		context.Background(), "add detail", &projectdomain.Project{},
		assetdomain.TileSetItem{Name: "Plot", Tiles: []assetdomain.Tile{{URL: &url}}},
		assetdomain.TileSetDimensions{TileSize: assetdomain.Size{Width: 16, Height: 16}}, nil,
	)
	if err != nil {
		t.Fatalf("retry Item edit: %v", err)
	}
	if len(regions) != 1 || images.calls != 2 || processor.splitCalls != 1 {
		t.Fatalf("unexpected retry result: regions=%+v images=%+v processor=%+v", regions, images, processor)
	}
}

func TestLoadTileSetImageSupportsHTTPAndReportsInvalidResponses(t *testing.T) {
	pngBase64 := tileSetEditTestImage(t, 2, 2)
	pngBytes, err := base64.StdEncoding.DecodeString(pngBase64)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/image":
			_, _ = writer.Write(pngBytes)
		case "/bad-image":
			_, _ = writer.Write([]byte("not an image"))
		default:
			http.Error(writer, "missing", http.StatusNotFound)
		}
	}))
	defer server.Close()

	for _, reference := range []string{pngBase64, server.URL + "/image"} {
		decoded, loadErr := loadTileSetImage(context.Background(), reference)
		if loadErr != nil || decoded.Bounds().Dx() != 2 || decoded.Bounds().Dy() != 2 {
			t.Fatalf("load Tile image %q: image=%v err=%v", reference, decoded, loadErr)
		}
	}
	for _, test := range []struct{ path, want string }{{"/missing", "HTTP 404"}, {"/bad-image", "decode"}} {
		if _, loadErr := loadTileSetImage(context.Background(), server.URL+test.path); loadErr == nil || !strings.Contains(loadErr.Error(), test.want) {
			t.Fatalf("expected %q error, got %v", test.want, loadErr)
		}
	}
	if _, loadErr := loadTileSetImage(context.Background(), "http://[::1"); loadErr == nil {
		t.Fatal("expected invalid HTTP URL error")
	}
}

func TestVerifyTileSetImageRejectsInvalidReports(t *testing.T) {
	valid := &imageprocessor.VerificationReport{
		IsPNG: true, HasAlpha: true, Width: 16, Height: 16, NontransparentPixels: 64,
		TransparentPixels: 192, TransparentRGBScrubbed: true,
	}
	tests := []struct {
		name   string
		report *imageprocessor.VerificationReport
		err    error
		width  int
		height int
		want   string
	}{
		{name: "processor error", err: errors.New("verify unavailable"), width: 16, height: 16, want: "verify unavailable"},
		{name: "missing report", width: 16, height: 16, want: "missing verification report"},
		{name: "not png", report: cloneTileSetReport(valid, func(value *imageprocessor.VerificationReport) { value.IsPNG = false }), width: 16, height: 16, want: "must be a PNG"},
		{name: "invalid expected dimensions", report: valid, width: 0, height: 16, want: "must be positive"},
		{name: "missing alpha", report: cloneTileSetReport(valid, func(value *imageprocessor.VerificationReport) {
			value.HasAlpha = false
			value.NontransparentPixels = 64
		}), width: 16, height: 16, want: "with alpha"},
		{name: "wrong size", report: cloneTileSetReport(valid, func(value *imageprocessor.VerificationReport) { value.Width = 8 }), width: 16, height: 16, want: "want 16x16"},
		{name: "residual rgb", report: cloneTileSetReport(valid, func(value *imageprocessor.VerificationReport) { value.TransparentRGBScrubbed = false }), width: 16, height: 16, want: "residual RGB"},
		{name: "empty occupied cell", report: cloneTileSetReport(valid, func(value *imageprocessor.VerificationReport) { value.NontransparentPixels = 0 }), width: 16, height: 16, want: "occupied cell"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			processor := &tileSetEditProcessorStub{verifyResult: test.report, verifyErr: test.err}
			err := verifyTileSetImage(context.Background(), processor, "image", test.width, test.height, true, false)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}

func TestVerifyTileSetImageAllowsIntentionalEmptyOccupiedCell(t *testing.T) {
	report := &imageprocessor.VerificationReport{
		IsPNG: true, HasAlpha: true, Width: 16, Height: 16,
		TransparentPixels: 256, TransparentRGBScrubbed: true,
	}
	processor := &tileSetEditProcessorStub{verifyResult: report}
	if err := verifyTileSetImage(context.Background(), processor, "image", 16, 16, true, true); err != nil {
		t.Fatalf("intentional empty occupied cell was rejected: %v", err)
	}
}

func TestReconstructTileSetItemRejectsMalformedFootprints(t *testing.T) {
	dimensions := assetdomain.TileSetDimensions{TileSize: assetdomain.Size{Width: 2, Height: 2}}
	url := "tile"
	tests := []struct {
		name        string
		item        assetdomain.TileSetItem
		resolved    string
		resolvedErr error
		want        string
	}{
		{name: "empty", item: assetdomain.TileSetItem{Name: "empty"}, want: "has no Tiles"},
		{name: "missing resource", item: assetdomain.TileSetItem{Name: "missing", Tiles: []assetdomain.Tile{{}}}, want: "has no resource"},
		{name: "duplicate", item: assetdomain.TileSetItem{Name: "duplicate", Tiles: []assetdomain.Tile{{URL: &url}, {URL: &url}}}, want: "duplicate position"},
		{name: "oversized", item: assetdomain.TileSetItem{Name: "large", Tiles: []assetdomain.Tile{{URL: &url}, {URL: &url, Position: assetdomain.TilePosition{X: maxGeneratedItemImageEdge, Y: 0}}}}, want: "processing limits"},
		{name: "resolve failure", item: assetdomain.TileSetItem{Name: "resolve", Tiles: []assetdomain.Tile{{URL: &url}}}, resolvedErr: errors.New("resolve unavailable"), want: "resolve unavailable"},
		{name: "invalid image", item: assetdomain.TileSetItem{Name: "invalid", Tiles: []assetdomain.Tile{{URL: &url}}}, resolved: "invalid", want: "load Tileset Item"},
		{name: "wrong size", item: assetdomain.TileSetItem{Name: "wrong-size", Tiles: []assetdomain.Tile{{URL: &url}}}, resolved: tileSetEditTestImage(t, 1, 1), want: "want 2x2"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved := test.resolved
			if resolved == "" && test.resolvedErr == nil {
				resolved = tileSetEditTestImage(t, 2, 2)
			}
			executor := &executor{references: &tileSetEditReferenceStub{resolved: resolved, resolveErr: test.resolvedErr}}
			_, _, _, _, err := executor.reconstructTileSetItem(context.Background(), test.item, dimensions)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}

func TestAllocateTileSetEditUploadRejectsKeyFailures(t *testing.T) {
	wantErr := errors.New("key unavailable")
	position := assetdomain.TilePosition{X: 2, Y: 3}
	if _, err := allocateTileSetEditUpload(&tileSetEditReferenceStub{keyErr: wantErr}, tileSetResolvedTarget{}, position, imageprocessor.ImageRegion{}, map[string]struct{}{}); !errors.Is(err, wantErr) {
		t.Fatalf("expected allocation error, got %v", err)
	}
	allocated := map[string]struct{}{"uploads/repeated.png": {}}
	if _, err := allocateTileSetEditUpload(&tileSetEditReferenceStub{key: "uploads/repeated.png"}, tileSetResolvedTarget{}, position, imageprocessor.ImageRegion{}, allocated); err == nil || !strings.Contains(err.Error(), "duplicate object key") {
		t.Fatalf("expected duplicate key error, got %v", err)
	}
}

type tileSetEditAssetStub struct {
	tileSetWorkflowAssets
	detailErr error
}

func (s *tileSetEditAssetStub) GetDetail(context.Context, uint) (assetdomain.Asset, error) {
	if s.detailErr != nil {
		return assetdomain.Asset{}, s.detailErr
	}
	return s.asset, nil
}

type tileSetEditProjectStub struct {
	project *projectdomain.Project
	err     error
}

func (s *tileSetEditProjectStub) GetDetail(context.Context, uint) (*projectdomain.Project, error) {
	return s.project, s.err
}

type tileSetEditReferenceStub struct {
	resolved   string
	resolveErr error
	key        string
	keyErr     error
}

func (s *tileSetEditReferenceStub) ResolveReference(context.Context, string) (string, error) {
	return s.resolved, s.resolveErr
}

func (*tileSetEditReferenceStub) PersistReference(context.Context, string) (string, error) {
	return "", nil
}

func (s *tileSetEditReferenceStub) NewObjectKey(string) (string, error) {
	if s.keyErr != nil {
		return "", s.keyErr
	}
	return s.key, nil
}

func (*tileSetEditReferenceStub) PersistReferenceAt(context.Context, string, string) error {
	return nil
}
func (*tileSetEditReferenceStub) DeleteObjects(context.Context, []string) error { return nil }

type tileSetEditProcessorStub struct {
	removeResults []*imageprocessor.RemoveBackgroundResult
	removeErrs    []error
	resizeResults []*imageprocessor.ResizeResult
	resizeErrs    []error
	verifyResult  *imageprocessor.VerificationReport
	verifyErr     error
	splitResults  []*imageprocessor.SplitImageResult
	splitErrs     []error
	removeCalls   int
	resizeCalls   int
	splitCalls    int
}

func (s *tileSetEditProcessorStub) RemoveBackground(context.Context, *imageprocessor.RemoveBackgroundRequest) (*imageprocessor.RemoveBackgroundResult, error) {
	index := s.removeCalls
	s.removeCalls++
	if index < len(s.removeErrs) && s.removeErrs[index] != nil {
		return nil, s.removeErrs[index]
	}
	if index < len(s.removeResults) {
		return s.removeResults[index], nil
	}
	return &imageprocessor.RemoveBackgroundResult{}, nil
}

func (s *tileSetEditProcessorStub) Resize(context.Context, *imageprocessor.ResizeRequest) (*imageprocessor.ResizeResult, error) {
	index := s.resizeCalls
	s.resizeCalls++
	if index < len(s.resizeErrs) && s.resizeErrs[index] != nil {
		return nil, s.resizeErrs[index]
	}
	if index < len(s.resizeResults) {
		return s.resizeResults[index], nil
	}
	return &imageprocessor.ResizeResult{}, nil
}

func (s *tileSetEditProcessorStub) Verify(context.Context, *imageprocessor.VerifyRequest) (*imageprocessor.VerificationReport, error) {
	return s.verifyResult, s.verifyErr
}

func (s *tileSetEditProcessorStub) SplitImage(context.Context, *imageprocessor.SplitImageRequest) (*imageprocessor.SplitImageResult, error) {
	index := s.splitCalls
	s.splitCalls++
	if index < len(s.splitErrs) && s.splitErrs[index] != nil {
		return nil, s.splitErrs[index]
	}
	if index < len(s.splitResults) {
		return s.splitResults[index], nil
	}
	return nil, fmt.Errorf("unexpected split")
}

type tileSetEditImageStub struct {
	requests []*imageclient.GenerateRequest
	results  []*imageclient.GenerateResult
	errors   []error
	calls    int
}

func (s *tileSetEditImageStub) Generate(_ context.Context, request *imageclient.GenerateRequest) (*imageclient.GenerateResult, error) {
	index := s.calls
	s.calls++
	s.requests = append(s.requests, request)
	if index < len(s.errors) && s.errors[index] != nil {
		return nil, s.errors[index]
	}
	if index < len(s.results) {
		return s.results[index], nil
	}
	return nil, fmt.Errorf("missing image result")
}

func tileSetEditCandidates(values ...string) *imageclient.GenerateResult {
	images := make([]imageclient.GeneratedImage, len(values))
	for index, value := range values {
		images[index] = imageclient.GeneratedImage{Base64: value}
	}
	return &imageclient.GenerateResult{Images: images}
}

func tileSetEditTestAsset(t *testing.T) assetdomain.Asset {
	t.Helper()
	url := "uploads/tile.png"
	content, err := assetdomain.EncodeContent(assetdomain.AssetContent{Items: []assetdomain.TileSetItem{{
		Name: "Pot", Tiles: []assetdomain.Tile{{URL: &url, Position: assetdomain.TilePosition{X: 0, Y: 0}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	return assetdomain.Asset{
		ID: 100, ProjectID: 42, Type: assetdomain.AssetTypeTileSet, Version: 1,
		Perspective: assetdomain.PerspectiveTopDown,
		Dimensions:  []byte(`{"tileSize":{"width":16,"height":16},"tileAmount":{"columns":4,"rows":4}}`),
		Content:     content,
	}
}

func tileSetEditTestImage(t *testing.T, width, height int) string {
	t.Helper()
	return tileSetEditTestImageWithColor(t, width, height, color.RGBA{R: 20, G: 40, B: 60, A: 255})
}

func tileSetEditTestImageWithColor(t *testing.T, width, height int, pixel color.RGBA) string {
	t.Helper()
	value := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			value.SetRGBA(x, y, pixel)
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func tileSetEditTransparentTestImage(t *testing.T, width, height int) string {
	t.Helper()
	return tileSetEditTransparentTestImageWithColor(t, width, height, color.RGBA{R: 20, G: 40, B: 60, A: 255})
}

func tileSetEditTransparentTestImageWithColor(t *testing.T, width, height int, pixel color.RGBA) string {
	t.Helper()
	value := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			value.SetRGBA(x, y, pixel)
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func cloneTileSetReport(value *imageprocessor.VerificationReport, mutate func(*imageprocessor.VerificationReport)) *imageprocessor.VerificationReport {
	copy := *value
	mutate(&copy)
	return &copy
}
