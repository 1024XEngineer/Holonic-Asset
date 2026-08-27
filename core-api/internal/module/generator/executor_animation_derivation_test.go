package generator_test

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestExecutorDerivesMissingTopDownDirectionsConcurrentlyFromInitialSnapshot(t *testing.T) {
	frameDataURL := derivationTestFrameDataURL(t, color.NRGBA{R: 180, G: 60, B: 40, A: 255})
	asset := derivationTestAsset(t, 4, "back", frameDataURL)
	events := []string{}
	animations := &concurrentDerivationAnimationService{
		wantConcurrent: 2,
		result: &generator.AnimationGenerationResult{
			Frames: []imageprocessor.ImageRegion{
				{ImageBase64: strings.TrimPrefix(frameDataURL, "data:image/png;base64,"), MIMEType: "image/png"},
				{ImageBase64: strings.TrimPrefix(frameDataURL, "data:image/png;base64,"), MIMEType: "image/png"},
			},
			FrameDurationMS: 100,
		},
	}
	images := &imageGenerationServiceStub{
		events: &events,
		result: &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{{
			Base64: strings.TrimPrefix(frameDataURL, "data:image/png;base64,"), MediaType: "image/png",
		}}},
	}
	processor := imageprocessor.NewProcessor()
	references := &executorReferenceStoreStub{resolveValues: map[string]string{
		"uploads/generated-1.png": frameDataURL,
		"uploads/generated-2.png": frameDataURL,
	}}
	executor := generator.NewExecutorWithDependencies(
		images,
		processor,
		&generationAssetWriterStub{events: &events, parentAsset: asset},
		generator.ExecutorDependencies{Animations: animations, References: references},
	)

	result, err := executor.Generate(context.Background(), generator.DeriveAnimation, json.RawMessage(`{
		"asset_id":7,
		"project_id":11,
		"source_animation_id":3,
		"target_directions":["left","right"]
	}`))
	if err != nil {
		t.Fatalf("derive animation: %v", err)
	}
	animations.mu.Lock()
	requests := append([]*generator.AnimationGenerationRequest(nil), animations.requests...)
	animations.mu.Unlock()
	if len(requests) != 2 {
		t.Fatalf("both missing side directions must use video from the initial snapshot: %+v", requests)
	}
	orientations := make(map[string]bool, len(requests))
	for _, request := range requests {
		orientations[request.TargetOrientation] = true
		if !strings.HasPrefix(request.DerivationSourceImage, "data:image/png;base64,") {
			t.Fatal("video derivation did not receive a PNG data URL source animation sheet")
		}
	}
	if !orientations["Left / West / screen-left view"] || !orientations["Right / East / screen-right view"] {
		t.Fatalf("unexpected concurrent target orientations: %+v", requests)
	}
	if len(images.requests) != 0 {
		t.Fatalf("same-batch output must not enable image derivation: %+v", images.requests)
	}

	application, content := decodeExecutionContent(t, result, assetdomain.AssetTypeCharacter)
	if application.AnimationID != 3 || len(content.Animations) != 2 {
		t.Fatalf("unexpected derivation candidate: application=%+v content=%+v", application, content)
	}
	for index, direction := range []string{"left", "right"} {
		animation := content.Animations[index]
		if animation.GroupID != 3 || animation.Generation == nil || animation.Generation.Direction != direction || len(animation.Frames) != 2 {
			t.Fatalf("unexpected derived animation %d: %+v", index, animation)
		}
	}
}

type concurrentDerivationAnimationService struct {
	mu             sync.Mutex
	wantConcurrent int
	requests       []*generator.AnimationGenerationRequest
	result         *generator.AnimationGenerationResult
	allStarted     chan struct{}
	once           sync.Once
}

func (s *concurrentDerivationAnimationService) Generate(
	ctx context.Context,
	request *generator.AnimationGenerationRequest,
) (*generator.AnimationGenerationResult, error) {
	s.mu.Lock()
	if s.allStarted == nil {
		s.allStarted = make(chan struct{})
	}
	copy := *request
	s.requests = append(s.requests, &copy)
	allStarted := s.allStarted
	if len(s.requests) == s.wantConcurrent {
		s.once.Do(func() { close(allStarted) })
	}
	s.mu.Unlock()
	select {
	case <-allStarted:
		return s.result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

var _ generator.AnimationGenerationService = (*concurrentDerivationAnimationService)(nil)

func TestExecutorUsesExistingMirrorAnimationForImageDerivation(t *testing.T) {
	frameDataURL := derivationTestFrameDataURL(t, color.NRGBA{R: 40, G: 80, B: 180, A: 255})
	asset := derivationTestAsset(t, 2, "left", frameDataURL)
	largePrototype := derivationTestFrameDataURLSize(t, 128, color.NRGBA{R: 40, G: 80, B: 180, A: 255})
	content, err := asset.DecodeContent()
	if err != nil {
		t.Fatalf("decode derivation asset: %v", err)
	}
	for index := range *content.Prototype {
		(*content.Prototype)[index].URL = &largePrototype
	}
	asset.Content, err = assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode derivation asset with high-resolution prototypes: %v", err)
	}
	events := []string{}
	images := &imageGenerationServiceStub{
		events: &events,
		result: &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{{
			Base64: strings.TrimPrefix(frameDataURL, "data:image/png;base64,"), MediaType: "image/png",
		}}},
	}
	animations := &animationGenerationServiceStub{events: &events}
	executor := generator.NewExecutorWithDependencies(
		images,
		imageprocessor.NewProcessor(),
		&generationAssetWriterStub{events: &events, parentAsset: asset},
		generator.ExecutorDependencies{Animations: animations, References: &executorReferenceStoreStub{}},
	)

	_, err = executor.Generate(context.Background(), generator.DeriveAnimation, json.RawMessage(`{
		"asset_id":7,"project_id":11,"source_animation_id":3,"target_directions":["right"]
	}`))
	if err != nil {
		t.Fatalf("derive mirror animation: %v", err)
	}
	if len(animations.requests) != 0 || len(images.requests) != 1 {
		t.Fatalf("existing left direction should derive right through image editing: video=%d image=%d", len(animations.requests), len(images.requests))
	}
}

func TestExecutorRemovesGreenMatteAndGeneratedGridGuttersFromImageDerivation(t *testing.T) {
	frameDataURL := derivationTestFrameDataURL(t, color.NRGBA{R: 40, G: 80, B: 180, A: 255})
	asset := derivationTestAsset(t, 2, "left", frameDataURL)
	generatedSheet := derivationTestOpaqueGridSheetDataURL(t)
	events := []string{}
	images := &imageGenerationServiceStub{
		events: &events,
		result: &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{{
			Base64: strings.TrimPrefix(generatedSheet, "data:image/png;base64,"), MediaType: "image/png",
		}}},
	}
	references := &executorReferenceStoreStub{}
	executor := generator.NewExecutorWithDependencies(
		images,
		imageprocessor.NewProcessor(),
		&generationAssetWriterStub{events: &events, parentAsset: asset},
		generator.ExecutorDependencies{
			Animations: &animationGenerationServiceStub{events: &events},
			References: references,
		},
	)

	_, err := executor.Generate(context.Background(), generator.DeriveAnimation, json.RawMessage(`{
		"asset_id":7,"project_id":11,"source_animation_id":3,"target_directions":["right"]
	}`))
	if err != nil {
		t.Fatalf("derive mirror animation: %v", err)
	}
	if len(references.persisted) != 2 {
		t.Fatalf("persisted frame count = %d, want 2", len(references.persisted))
	}
	for index, reference := range references.persisted {
		frame, decodeErr := imageprocessor.DecodeBase64Image(reference)
		if decodeErr != nil {
			t.Fatalf("decode persisted frame %d: %v", index+1, decodeErr)
		}
		var transparent bool
		for y := frame.Bounds().Min.Y; y < frame.Bounds().Max.Y; y++ {
			for x := frame.Bounds().Min.X; x < frame.Bounds().Max.X; x++ {
				r, g, b, alpha := frame.At(x, y).RGBA()
				if alpha <= uint32(imageprocessor.TransparentAlphaMax)*0x101 {
					transparent = true
					continue
				}
				red, green, blue := int(r>>8), int(g>>8), int(b>>8)
				if green > red+50 && green > blue+50 {
					t.Fatalf("persisted frame %d retains an opaque green matte at (%d,%d)", index+1, x, y)
				}
			}
		}
		if !transparent {
			t.Fatalf("persisted frame %d has no transparent background", index+1)
		}
	}
}

func TestExecutorRejectsExistingAndUnsupportedDerivationDirections(t *testing.T) {
	frameDataURL := derivationTestFrameDataURL(t, color.NRGBA{R: 80, G: 150, B: 60, A: 255})
	asset := derivationTestAsset(t, 4, "back", frameDataURL)
	events := []string{}
	executor := generator.NewExecutorWithDependencies(
		&imageGenerationServiceStub{events: &events},
		imageprocessor.NewProcessor(),
		&generationAssetWriterStub{events: &events, parentAsset: asset},
		generator.ExecutorDependencies{Animations: &animationGenerationServiceStub{events: &events}, References: &executorReferenceStoreStub{}},
	)

	for _, test := range []struct {
		directions string
		want       string
	}{
		{directions: `["back"]`, want: "already contains direction"},
		{directions: `["front_right"]`, want: "unavailable for an asset with 4 directions"},
		{directions: `["left","left"]`, want: "is duplicated"},
	} {
		payload := fmt.Sprintf(`{"asset_id":7,"project_id":11,"source_animation_id":3,"target_directions":%s}`, test.directions)
		_, err := executor.Generate(context.Background(), generator.DeriveAnimation, json.RawMessage(payload))
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("directions %s: got %v, want %q", test.directions, err, test.want)
		}
	}
}

func derivationTestAsset(t *testing.T, directionCount uint, direction, frameDataURL string) assetdomain.Asset {
	t.Helper()
	content := assetdomain.NewAssetContent(assetdomain.AssetTypeCharacter)
	content.DirectionCount = directionCount
	prototype := make(assetdomain.Prototype, directionCount)
	for index := range prototype {
		url := frameDataURL
		prototype[index] = assetdomain.ImageResource{ID: uint(index + 1), URL: &url}
	}
	content.Prototype = &prototype
	frames := []assetdomain.Frame{
		{ID: 1, URL: &frameDataURL, Duration: 100},
		{ID: 2, URL: &frameDataURL, Duration: 100},
	}
	content.Animations = []assetdomain.Animation{{
		ID:     3,
		Name:   "spray",
		Frames: frames,
		Generation: &assetdomain.AnimationGenerationConfig{
			Direction: direction, Style: "pixel art", Action: "spray cleaning liquid",
			FrameCount: 2, Columns: 2, FrameWidth: 32, FrameHeight: 32,
			FPS: 10, Resolution: "720p", Duration: 5, AspectRatio: "1:1",
		},
	}}
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode derivation test asset: %v", err)
	}
	return assetdomain.Asset{
		ID: 7, ProjectID: 11, Version: 5, Type: assetdomain.AssetTypeCharacter,
		Name: "maintenance robot", Description: "blue maintenance robot",
		Dimensions: json.RawMessage(`{"width":32,"height":32}`), Content: encoded,
	}
}

func derivationTestFrameDataURL(t *testing.T, subject color.NRGBA) string {
	return derivationTestFrameDataURLSize(t, 32, subject)
}

func derivationTestFrameDataURLSize(t *testing.T, size int, subject color.NRGBA) string {
	t.Helper()
	frame := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := size / 4; y < size-size/8; y++ {
		for x := size * 5 / 16; x < size*11/16; x++ {
			frame.SetNRGBA(x, y, subject)
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(frame)
	if err != nil {
		t.Fatalf("encode derivation test frame: %v", err)
	}
	return "data:image/png;base64," + encoded
}

func derivationTestOpaqueGridSheetDataURL(t *testing.T) string {
	t.Helper()
	sheet := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	for y := range 32 {
		for x := range 64 {
			sheet.SetNRGBA(x, y, color.NRGBA{R: 248, G: 250, B: 248, A: 255})
		}
	}
	for y := 2; y < 30; y++ {
		for x := 2; x < 30; x++ {
			sheet.SetNRGBA(x, y, color.NRGBA{G: 255, A: 255})
		}
		for x := 34; x < 62; x++ {
			sheet.SetNRGBA(x, y, color.NRGBA{G: 255, A: 255})
		}
	}
	for y := 8; y < 26; y++ {
		for x := 12; x < 20; x++ {
			sheet.SetNRGBA(x, y, color.NRGBA{R: 40, G: 80, B: 180, A: 255})
		}
		for x := 44; x < 52; x++ {
			sheet.SetNRGBA(x, y, color.NRGBA{R: 180, G: 80, B: 40, A: 255})
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(sheet)
	if err != nil {
		t.Fatalf("encode opaque derivation grid sheet: %v", err)
	}
	return "data:image/png;base64," + encoded
}
