package executor

import (
	"context"
	"errors"
	"image"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	videoprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/video"
)

func TestNewAnimationGenerationServiceConfiguresDefaultVideoProcessor(t *testing.T) {
	service, ok := NewAnimationGenerationService(
		&animationVideoServiceStub{},
		&animationProcessorStub{},
	).(*animationGenerationService)
	if !ok || service.videoProcessor == nil || service.referenceHTTPClient == nil {
		t.Fatalf("unexpected animation generation service: %#v", service)
	}
}

func TestAnimationGenerationRequiresDependencies(t *testing.T) {
	request := &AnimationGenerationRequest{ReferenceImage: "reference"}
	tests := []struct {
		name    string
		service *animationGenerationService
		want    error
	}{
		{name: "video service", service: &animationGenerationService{}, want: generator.ErrVideoServiceRequired},
		{name: "image processor", service: &animationGenerationService{videos: &animationVideoServiceStub{}}, want: generator.ErrImageProcessorRequired},
		{
			name: "video processor",
			service: &animationGenerationService{
				videos:    &animationVideoServiceStub{},
				processor: &animationProcessorStub{},
			},
			want: generator.ErrVideoFrameExtractorRequired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.service.Generate(context.Background(), request)
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestNormalizeAnimationGenerationRequestRejectsInvalidOptions(t *testing.T) {
	valid := AnimationGenerationRequest{
		ReferenceImage: "reference",
		FrameCount:     4,
		Columns:        2,
		FrameWidth:     64,
		FrameHeight:    64,
		FPS:            10,
		Duration:       5,
	}
	tests := []struct {
		name   string
		mutate func(*AnimationGenerationRequest)
		want   string
	}{
		{name: "reference", mutate: func(value *AnimationGenerationRequest) { value.ReferenceImage = " " }, want: "reference image is required"},
		{name: "frame count", mutate: func(value *AnimationGenerationRequest) { value.FrameCount = 33 }, want: "frame count"},
		{name: "columns", mutate: func(value *AnimationGenerationRequest) { value.Columns = 9 }, want: "columns"},
		{name: "rows", mutate: func(value *AnimationGenerationRequest) { value.FrameCount, value.Columns = 32, 1 }, want: "8 rows"},
		{name: "dimensions", mutate: func(value *AnimationGenerationRequest) { value.FrameWidth = 1025 }, want: "dimensions"},
		{name: "fps", mutate: func(value *AnimationGenerationRequest) { value.FPS = 61 }, want: "FPS"},
		{name: "duration", mutate: func(value *AnimationGenerationRequest) { value.Duration = 16 }, want: "duration"},
	}

	if _, err := normalizeAnimationGenerationRequest(nil); err == nil {
		t.Fatal("expected nil request to be rejected")
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := valid
			test.mutate(&value)
			_, err := normalizeAnimationGenerationRequest(&value)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestLoadAnimationReferenceRejectsResolverFailures(t *testing.T) {
	wantErr := errors.New("resolve failed")
	service := &animationGenerationService{referenceResolver: &animationReferenceResolverStub{err: wantErr}}
	if _, err := service.loadAnimationReference(context.Background(), "key"); !errors.Is(err, wantErr) {
		t.Fatalf("expected resolver error, got %v", err)
	}
	service.referenceResolver = &animationReferenceResolverStub{}
	if _, err := service.loadAnimationReference(context.Background(), "key"); err == nil || !strings.Contains(err.Error(), "empty result") {
		t.Fatalf("expected empty resolver result, got %v", err)
	}
	if _, err := (&animationGenerationService{}).loadAnimationReference(context.Background(), " "); err == nil {
		t.Fatal("expected empty reference error")
	}
}

func TestProcessAnimationVideoRejectsInvalidProcessorResults(t *testing.T) {
	request := AnimationGenerationRequest{FrameCount: 2, Columns: 2, FrameWidth: 32, FrameHeight: 32}
	tests := []struct {
		name      string
		processed *videoprocessor.Result
		split     *imageprocessor.SplitImageResult
		want      string
	}{
		{name: "no frames", processed: &videoprocessor.Result{}, want: "frames are required"},
		{
			name: "different dimensions",
			processed: &videoprocessor.Result{Frames: []image.Image{
				image.NewNRGBA(image.Rect(0, 0, 8, 8)),
				image.NewNRGBA(image.Rect(0, 0, 9, 8)),
			}},
			want: "dimensions differ",
		},
		{
			name: "incomplete normalization",
			processed: &videoprocessor.Result{Frames: []image.Image{
				image.NewNRGBA(image.Rect(0, 0, 8, 8)),
				image.NewNRGBA(image.Rect(0, 0, 8, 8)),
			}},
			split: &imageprocessor.SplitImageResult{},
			want:  "empty or incomplete result",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &animationGenerationService{
				videoProcessor: &animationVideoProcessorStub{results: []*videoprocessor.Result{test.processed}},
				processor:      &animationProcessorStub{splitResult: test.split},
			}
			_, err := service.processVideo(context.Background(), []byte("video"), request)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestPackAnimationVideoFramesValidatesGridInputs(t *testing.T) {
	frame := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	if _, err := packAnimationVideoFrames([]image.Image{frame}, 0); err == nil {
		t.Fatal("expected non-positive columns to be rejected")
	}
	if _, err := packAnimationVideoFrames([]image.Image{image.NewNRGBA(image.Rectangle{})}, 1); err == nil {
		t.Fatal("expected empty frame dimensions to be rejected")
	}
}
