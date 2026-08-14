package generator

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
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
			err := verifyTileSetImage(context.Background(), processor, "image", test.width, test.height, true)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}

func TestReconstructTileSetItemRejectsMalformedFootprints(t *testing.T) {
	dimensions := assetdomain.TileSetDimensions{TileSize: assetdomain.Size{Width: 2, Height: 2}}
	url := "tile"
	tests := []struct {
		name string
		item assetdomain.TileSetItem
		want string
	}{
		{name: "empty", item: assetdomain.TileSetItem{Name: "empty"}, want: "has no Tiles"},
		{name: "missing resource", item: assetdomain.TileSetItem{Name: "missing", Tiles: []assetdomain.Tile{{}}}, want: "has no resource"},
		{name: "duplicate", item: assetdomain.TileSetItem{Name: "duplicate", Tiles: []assetdomain.Tile{{URL: &url}, {URL: &url}}}, want: "duplicate position"},
		{name: "oversized", item: assetdomain.TileSetItem{Name: "large", Tiles: []assetdomain.Tile{{URL: &url}, {URL: &url, Position: assetdomain.TilePosition{X: maxGeneratedItemImageEdge, Y: 0}}}}, want: "processing limits"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := &executor{references: &tileSetEditReferenceStub{resolved: tileSetEditTestImage(t, 2, 2)}}
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
	resolved string
	key      string
	keyErr   error
}

func (s *tileSetEditReferenceStub) ResolveReference(context.Context, string) (string, error) {
	return s.resolved, nil
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
	removeCalls   int
	resizeCalls   int
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

func (*tileSetEditProcessorStub) SplitImage(context.Context, *imageprocessor.SplitImageRequest) (*imageprocessor.SplitImageResult, error) {
	return nil, fmt.Errorf("unexpected split")
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
	value := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			value.SetRGBA(x, y, color.RGBA{R: 20, G: 40, B: 60, A: 255})
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
