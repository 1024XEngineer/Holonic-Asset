package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestComposeSceneryReferencePreviewUsesBackToFrontCompositingOrder(t *testing.T) {
	dimensions := assetdomain.Size{Width: 2, Height: 1}
	plan := []SceneryLayerDefinition{{ID: 1}, {ID: 2}, {ID: 3}}
	processed := map[uint]ProcessedSceneryLayer{
		2: sceneryPreviewLayer(t, 2, []color.RGBA{{R: 255, A: 255}, {R: 255, A: 255}}),
		3: sceneryPreviewLayer(t, 3, []color.RGBA{{B: 255, A: 255}, {}}),
	}

	preview, err := composeSceneryReferencePreview(dimensions, plan, processed)
	if err != nil {
		t.Fatalf("compose scenery reference preview: %v", err)
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(preview, "data:image/png;base64,"))
	if err != nil {
		t.Fatalf("decode scenery reference preview: %v", err)
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode scenery reference PNG: %v", err)
	}
	frontPixel := color.RGBAModel.Convert(decoded.At(0, 0)).(color.RGBA)
	backPixel := color.RGBAModel.Convert(decoded.At(1, 0)).(color.RGBA)
	if frontPixel.B != 255 || frontPixel.R != 0 || backPixel.R != 255 || backPixel.B != 0 {
		t.Fatalf("unexpected composite pixels: front=%+v back=%+v", frontPixel, backPixel)
	}
}

func TestComposeSceneryReferencePreviewRejectsInvalidProcessedLayer(t *testing.T) {
	_, err := composeSceneryReferencePreview(
		assetdomain.Size{Width: 2, Height: 1},
		[]SceneryLayerDefinition{{ID: 1}, {ID: 2}},
		map[uint]ProcessedSceneryLayer{2: {ID: 2, ImageBase64: "not-base64"}},
	)
	if err == nil || !strings.Contains(err.Error(), "decode layer 2 base64") {
		t.Fatalf("expected contextual preview error, got %v", err)
	}
}

func sceneryPreviewLayer(t *testing.T, id uint, pixels []color.RGBA) ProcessedSceneryLayer {
	t.Helper()
	fixture := image.NewRGBA(image.Rect(0, 0, len(pixels), 1))
	for index, pixel := range pixels {
		fixture.SetRGBA(index, 0, pixel)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, fixture); err != nil {
		t.Fatalf("encode preview layer: %v", err)
	}
	return ProcessedSceneryLayer{
		ID: id, ImageBase64: base64.StdEncoding.EncodeToString(encoded.Bytes()), MediaType: "image/png",
	}
}

func TestDecodeSceneryLayoutsAssociatesUnorderedResponseByStableID(t *testing.T) {
	approved, notes, layouts, err := decodeSceneryLayouts([]byte(`{
		"approved": true,
		"review_notes": "Well grounded, correct perspective",
		"layers":[
			{"id":2,"position":{"x":100,"y":40},"scale":{"x":0.8,"y":0.8},"rotation":15,"opacity":0.75,"zIndex":20},
			{"id":1,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":-10}
		]
	}`), sceneryLayoutTestLayers(), sceneryLayoutTestDimensions())
	if err != nil {
		t.Fatalf("decode valid scenery layout: %v", err)
	}
	if !approved || notes != "Well grounded, correct perspective" {
		t.Fatalf("unexpected review decision: approved=%v, notes=%q", approved, notes)
	}
	if len(layouts) != 2 || layouts[1].ZIndex != -10 || layouts[2].Position.X != 100 ||
		layouts[2].Scale.X != 0.8 || layouts[2].Rotation != 15 || layouts[2].Opacity != 0.75 {
		t.Fatalf("unexpected layouts: %+v", layouts)
	}
}

func TestDecodeSceneryLayoutsNormalizesOpaqueBackdrop(t *testing.T) {
	layers := []ProcessedSceneryLayer{{ID: 1, Name: "Sky"}, {ID: 2, Name: "Trees"}, {ID: 3, Name: "Mountains"}}
	approved, notes, layouts, err := decodeSceneryLayouts([]byte(`{
		"approved": false,
		"review_notes": "Trees need ground adjustment",
		"layers":[
			{"id":1,"position":{"x":20,"y":10},"scale":{"x":0.5,"y":0.5},"rotation":5,"opacity":0.5,"zIndex":10},
			{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":10},
			{"id":3,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":-5}
		]
	}`), layers, sceneryLayoutTestDimensions())
	if err != nil {
		t.Fatalf("decode scenery layout: %v", err)
	}
	if approved || notes != "Trees need ground adjustment" {
		t.Fatalf("unexpected review decision: approved=%v, notes=%q", approved, notes)
	}
	backdrop := layouts[1]
	if backdrop.Position != (SceneryLayoutVector{}) || backdrop.Scale != (SceneryLayoutVector{X: 1, Y: 1}) ||
		backdrop.Rotation != 0 || backdrop.Opacity != 1 || backdrop.ZIndex != 0 {
		t.Fatalf("backdrop was not normalized: %+v", backdrop)
	}
	if layouts[3].ZIndex != 1 || layouts[2].ZIndex != 2 {
		t.Fatalf("overlay order was not normalized deterministically: %+v", layouts)
	}
}

func TestDecodeSceneryLayoutsRejectsInvalidModelOutput(t *testing.T) {
	first := `{"id":1,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":0}`
	second := `{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}`
	validReview := `"approved":true,"review_notes":"ok"`
	tests := []struct {
		name string
		json string
	}{
		{name: "malformed", json: `{"layers":[`},
		{name: "missing layers", json: `{` + validReview + `}`},
		{name: "missing approved", json: `{"review_notes":"ok","layers":[` + first + `,` + second + `]}`},
		{name: "missing review_notes", json: `{"approved":true,"layers":[` + first + `,` + second + `]}`},
		{name: "blank review_notes", json: `{"approved":true,"review_notes":"   ","layers":[` + first + `,` + second + `]}`},
		{name: "missing layer", json: `{` + validReview + `,"layers":[` + first + `]}`},
		{name: "duplicate ID", json: `{` + validReview + `,"layers":[` + first + `,` + first + `]}`},
		{name: "unknown ID", json: `{` + validReview + `,"layers":[` + first + `,{"id":3,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "unknown root field", json: `{` + validReview + `,"layers":[` + first + `,` + second + `],"explanation":"ok"}`},
		{name: "unknown layer field", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1,"visible":true}]}`},
		{name: "unknown position field", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0,"anchor":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "unknown scale field", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1,"uniform":true},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "missing ID", json: `{` + validReview + `,"layers":[` + first + `,{"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "missing position", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "missing position coordinate", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "missing scale", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "missing scale coordinate", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "missing rotation", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"opacity":1,"zIndex":1}]}`},
		{name: "missing opacity", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"zIndex":1}]}`},
		{name: "missing zIndex", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1}]}`},
		{name: "non-integer zIndex", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1.5}]}`},
		{name: "number overflow", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":1e1000,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "zero scale", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":0,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "negative scale", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":-1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "opacity below range", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":-0.1,"zIndex":1}]}`},
		{name: "opacity above range", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1.1,"zIndex":1}]}`},
		{name: "outside canvas", json: `{` + validReview + `,"layers":[` + first + `,{"id":2,"position":{"x":640,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`},
		{name: "trailing data", json: `{` + validReview + `,"layers":[` + first + `,` + second + `]} {}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, _, err := decodeSceneryLayouts([]byte(test.json), sceneryLayoutTestLayers(), sceneryLayoutTestDimensions())
			if !errors.Is(err, ErrInvalidSceneryLayout) {
				t.Fatalf("expected invalid scenery layout, got %v", err)
			}
		})
	}
}

func TestValidateSceneryLayoutRejectsInvalidCandidates(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*sceneryLayoutCandidate)
	}{
		{name: "zero ID", mutate: func(candidate *sceneryLayoutCandidate) { value := uint(0); candidate.ID = &value }},
		{name: "non-finite position", mutate: func(candidate *sceneryLayoutCandidate) { value := math.Inf(1); candidate.Position.X = &value }},
		{name: "non-finite scale", mutate: func(candidate *sceneryLayoutCandidate) { value := math.NaN(); candidate.Scale.Y = &value }},
		{name: "non-uniform scale", mutate: func(candidate *sceneryLayoutCandidate) { value := 0.75; candidate.Scale.Y = &value }},
		{name: "non-finite rotation", mutate: func(candidate *sceneryLayoutCandidate) { value := math.Inf(-1); candidate.Rotation = &value }},
		{name: "non-finite opacity", mutate: func(candidate *sceneryLayoutCandidate) { value := math.NaN(); candidate.Opacity = &value }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := validSceneryLayoutCandidate()
			test.mutate(&candidate)
			if _, _, err := validateSceneryLayoutCandidate(candidate, sceneryLayoutTestDimensions()); err == nil {
				t.Fatal("expected invalid candidate to be rejected")
			}
		})
	}
}

func TestDecodeSceneryLayoutsRejectsInvalidProcessedLayerIDs(t *testing.T) {
	valid := []byte(`{"approved":true,"review_notes":"ok","layers":[{"id":1,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":0},{"id":2,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":1}]}`)
	for name, layers := range map[string][]ProcessedSceneryLayer{
		"zero":      {{ID: 0}, {ID: 2}},
		"duplicate": {{ID: 1}, {ID: 1}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, _, err := decodeSceneryLayouts(valid, layers, sceneryLayoutTestDimensions()); !errors.Is(err, ErrInvalidSceneryLayout) {
				t.Fatalf("expected invalid processed layers, got %v", err)
			}
		})
	}
}

func validSceneryLayoutCandidate() sceneryLayoutCandidate {
	id := uint(1)
	zero, one := float64(0), float64(1)
	zIndex := 0
	return sceneryLayoutCandidate{
		ID: &id, Position: &sceneryLayoutVectorCandidate{X: &zero, Y: &zero},
		Scale: &sceneryLayoutVectorCandidate{X: &one, Y: &one}, Rotation: &zero, Opacity: &one, ZIndex: &zIndex,
	}
}

func sceneryLayoutTestLayers() []ProcessedSceneryLayer {
	return []ProcessedSceneryLayer{{ID: 1, Name: "Sky"}, {ID: 2, Name: "Mountains"}}
}

func sceneryLayoutTestDimensions() assetdomain.Size {
	return assetdomain.Size{Width: 640, Height: 360}
}

func TestComposeSceneryReferencePreviewInvalidCases(t *testing.T) {
	// Zero/negative dimensions
	_, err := composeSceneryReferencePreview(
		assetdomain.Size{Width: 0, Height: 100},
		[]SceneryLayerDefinition{{ID: 1}},
		map[uint]ProcessedSceneryLayer{1: {ID: 1, ImageBase64: "dGVzdA=="}},
	)
	if err == nil || !strings.Contains(err.Error(), "invalid canvas dimensions") {
		t.Fatalf("expected invalid canvas dimensions error, got %v", err)
	}

	// Non-image valid base64
	_, err = composeSceneryReferencePreview(
		assetdomain.Size{Width: 10, Height: 10},
		[]SceneryLayerDefinition{{ID: 1}},
		map[uint]ProcessedSceneryLayer{1: {ID: 1, ImageBase64: base64.StdEncoding.EncodeToString([]byte("plain text not image"))}},
	)
	if err == nil || !strings.Contains(err.Error(), "decode layer 1 image") {
		t.Fatalf("expected decode layer image error, got %v", err)
	}

	// Dimension mismatch (layer is 2x1, canvas is 10x10)
	fixture := image.NewRGBA(image.Rect(0, 0, 2, 1))
	var buf bytes.Buffer
	_ = png.Encode(&buf, fixture)
	_, err = composeSceneryReferencePreview(
		assetdomain.Size{Width: 10, Height: 10},
		[]SceneryLayerDefinition{{ID: 1}},
		map[uint]ProcessedSceneryLayer{1: {ID: 1, ImageBase64: base64.StdEncoding.EncodeToString(buf.Bytes())}},
	)
	if err == nil || !strings.Contains(err.Error(), "expected 10x10") {
		t.Fatalf("expected dimension mismatch error, got %v", err)
	}
}

type testSceneryLogger struct {
	infos  []string
	errors []string
}

func (l *testSceneryLogger) Info(msg string, fields ...logger.Field) {
	l.infos = append(l.infos, msg)
}

func (l *testSceneryLogger) Error(msg string, fields ...logger.Field) {
	l.errors = append(l.errors, msg)
}

func (l *testSceneryLogger) Warn(msg string, fields ...logger.Field)  {}
func (l *testSceneryLogger) Debug(msg string, fields ...logger.Field) {}
func (l *testSceneryLogger) Sync() error                              { return nil }

func TestSceneryLoggingCoverage(t *testing.T) {
	payload := CreateSceneryPayload{
		ProjectID: 1, AssetName: "Forest", Dimensions: assetdomain.Size{Width: 640, Height: 360},
	}
	started := time.Now().Add(-10 * time.Millisecond)

	// nil logger shouldn't panic
	execNil := &executor{}
	execNil.logSceneryStage("start", payload, "plan", started)
	execNil.logSceneryFailure(payload, "plan", started, errors.New("fail"))

	// active logger
	log := &testSceneryLogger{}
	exec := &executor{logger: log}
	exec.logSceneryStage("stage completed", payload, "plan", started, logger.Int("layers", 3))
	if len(log.infos) != 1 {
		t.Fatalf("expected 1 info log, got %d", len(log.infos))
	}

	exec.logSceneryFailure(payload, "plan", started, errors.New("regular error"))
	exec.logSceneryFailure(payload, "generate", started, &llmclient.ProviderError{
		Provider: "gemini", Kind: "rate_limit", StatusCode: 429, Message: "quota exceeded",
	})
	exec.logSceneryFailure(payload, "layout", started, &llmclient.ProviderError{
		Provider: "gemini", Kind: "bad_request", StatusCode: 400, Message: "invalid", Cause: errors.New("root cause"),
	})
	if len(log.errors) != 3 {
		t.Fatalf("expected 3 error logs, got %d", len(log.errors))
	}
}

type stubReferenceStore struct {
	resolved string
	err      error
}

func (s *stubReferenceStore) ResolveReference(_ context.Context, key string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.resolved != "" {
		return s.resolved, nil
	}
	return "resolved:" + key, nil
}

func (s *stubReferenceStore) PersistReference(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (s *stubReferenceStore) NewObjectKey(_ string) (string, error) {
	return "", nil
}

func (s *stubReferenceStore) PersistReferenceAt(_ context.Context, _, _ string) error {
	return nil
}

func (s *stubReferenceStore) DeleteObjects(_ context.Context, _ []string) error {
	return nil
}

type stubResourceStore struct {
	deleted []string
	err     error
}

func (s *stubResourceStore) PutObject(_ context.Context, _, _ string, _ []byte) error {
	return nil
}

func (s *stubResourceStore) DeleteObject(_ context.Context, key string) error {
	if s.err != nil {
		return s.err
	}
	s.deleted = append(s.deleted, key)
	return nil
}

func TestGenerateSceneryLayersEdgeCases(t *testing.T) {
	ctx := context.Background()

	// 1. Reference resolution error
	execRefErr := &executor{
		references: &stubReferenceStore{err: errors.New("resolver failure")},
	}
	_, err := execRefErr.generateSceneryLayers(ctx, CreateSceneryPayload{
		CreatingReference: "ref1",
	}, []SceneryLayerDefinition{{ID: 1}})
	if err == nil || !strings.Contains(err.Error(), "resolve scenery reference") {
		t.Fatalf("expected reference error, got %v", err)
	}

	// 2. ProjectReference fallback when CreatingReference is empty and context canceled
	refStore := &stubReferenceStore{}
	execRefProj := &executor{
		references: refStore,
	}
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = execRefProj.generateSceneryLayers(canceledCtx, CreateSceneryPayload{
		ProjectReference: "proj-ref",
	}, []SceneryLayerDefinition{{ID: 1}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestNewSceneryAssetAndBatchID(t *testing.T) {
	batchID, err := newSceneryBatchID()
	if err != nil || len(batchID) == 0 {
		t.Fatalf("expected non-empty batch ID without error, got %v", err)
	}

	asset, err := newSceneryAsset(CreateSceneryPayload{
		ProjectID: 10, AssetName: "Castle", Perspective: "Side-On",
		Dimensions: assetdomain.Size{Width: 800, Height: 600},
	}, []assetdomain.SceneryLayer{{
		ID: 1, Name: "Backdrop",
	}})
	if err != nil {
		t.Fatalf("newSceneryAsset failed: %v", err)
	}

	if asset.ProjectID != 10 || asset.Name != "Castle" || asset.Perspective != assetdomain.PerspectiveSideOn ||
		asset.Type != assetdomain.AssetTypeScenery {
		t.Fatalf("unexpected asset: %+v", asset)
	}

	// Delete resources test
	resStore := &stubResourceStore{}
	exec := &executor{resources: resStore}
	if err := exec.deleteSceneryResources(context.Background(), []string{"k1", "k2"}); err != nil {
		t.Fatalf("deleteSceneryResources unexpected error: %v", err)
	}
	if len(resStore.deleted) != 2 {
		t.Fatalf("expected 2 deleted resources, got %d", len(resStore.deleted))
	}

	resErrStore := &stubResourceStore{err: errors.New("storage down")}
	execErr := &executor{resources: resErrStore}
	if err := execErr.deleteSceneryResources(context.Background(), []string{"k1"}); err == nil {
		t.Fatal("expected error from deleteSceneryResources")
	}
}

type sceneryLayerImageStub struct {
	requests []*imageclient.GenerateRequest
	result   *imageclient.GenerateResult
	err      error
}

func (s *sceneryLayerImageStub) Generate(_ context.Context, request *imageclient.GenerateRequest) (*imageclient.GenerateResult, error) {
	copy := *request
	copy.ReferenceImages = append([]string(nil), request.ReferenceImages...)
	s.requests = append(s.requests, &copy)
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

type sceneryReferenceRoundTripFunc func(*http.Request) (*http.Response, error)

func (function sceneryReferenceRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestGenerateSceneryLayersPassesDownloadedReferenceToImageGeneration(t *testing.T) {
	referenceImage := image.NewNRGBA(image.Rect(0, 0, imageprocessor.DefaultReferenceMaxEdge, 1))
	for x := range imageprocessor.DefaultReferenceMaxEdge {
		referenceImage.SetNRGBA(x, 0, color.NRGBA{R: 20, G: 40, B: 60, A: 255})
	}
	var referenceBuffer bytes.Buffer
	if err := png.Encode(&referenceBuffer, referenceImage); err != nil {
		t.Fatalf("encode reference fixture: %v", err)
	}
	referenceBytes := referenceBuffer.Bytes()
	candidate := sceneryPreviewLayer(t, 1, []color.RGBA{{R: 20, G: 40, B: 60, A: 255}, {R: 20, G: 40, B: 60, A: 255}})
	images := &sceneryLayerImageStub{result: &imageclient.GenerateResult{
		Images: []imageclient.GeneratedImage{{Base64: candidate.ImageBase64, MediaType: "image/png"}},
	}}
	exec := &executor{
		images:     images,
		processor:  imageprocessor.NewProcessor(),
		references: &stubReferenceStore{resolved: "https://example.com/reference.png"},
		referenceHTTPClient: &http.Client{Transport: sceneryReferenceRoundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.Method != http.MethodGet || request.URL.String() != "https://example.com/reference.png" {
				t.Fatalf("unexpected reference download request: %s %s", request.Method, request.URL)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"image/png"}},
				Body:       io.NopCloser(bytes.NewReader(referenceBytes)),
			}, nil
		})},
	}

	layers, err := exec.generateSceneryLayers(context.Background(), CreateSceneryPayload{
		Dimensions:        assetdomain.Size{Width: 2, Height: 1},
		CreatingReference: "uploads/reference.png",
	}, []SceneryLayerDefinition{{ID: 1, Name: "Backdrop"}})
	if err != nil {
		t.Fatalf("generate scenery layers with downloaded reference: %v", err)
	}
	if len(layers) != 1 || layers[0].ID != 1 {
		t.Fatalf("unexpected generated layers: %+v", layers)
	}
	if len(images.requests) != 1 {
		t.Fatalf("image generation calls = %d, want 1", len(images.requests))
	}
	wantReference := "data:image/png;base64," + base64.StdEncoding.EncodeToString(referenceBytes)
	if got := images.requests[0].ReferenceImages; len(got) != 1 || got[0] != wantReference {
		t.Fatalf("image references = %q, want downloaded data URL %q", got, wantReference)
	}
}

func TestGenerateSceneryLayersWrapsReferenceDownloadError(t *testing.T) {
	downloadErr := errors.New("download failed")
	exec := &executor{
		references: &stubReferenceStore{resolved: "https://example.com/reference.png"},
		referenceHTTPClient: &http.Client{Transport: sceneryReferenceRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, downloadErr
		})},
	}

	_, err := exec.generateSceneryLayers(context.Background(), CreateSceneryPayload{
		CreatingReference: "uploads/reference.png",
	}, []SceneryLayerDefinition{{ID: 1}})
	if !errors.Is(err, downloadErr) || !strings.Contains(err.Error(), "resolve scenery reference") {
		t.Fatalf("expected wrapped download error, got: %v", err)
	}
}
