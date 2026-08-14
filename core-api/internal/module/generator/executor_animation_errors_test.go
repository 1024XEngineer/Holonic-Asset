package generator

import (
	"context"
	"errors"
	"image"
	"reflect"
	"strings"
	"testing"

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
		{name: "video service", service: &animationGenerationService{}, want: ErrVideoServiceRequired},
		{name: "image processor", service: &animationGenerationService{videos: &animationVideoServiceStub{}}, want: ErrImageProcessorRequired},
		{
			name: "video processor",
			service: &animationGenerationService{
				videos:    &animationVideoServiceStub{},
				processor: &animationProcessorStub{},
			},
			want: ErrVideoFrameExtractorRequired,
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

func TestNormalizeAnimationGenerationRequestValidatesTargetFrameIndices(t *testing.T) {
	base := AnimationGenerationRequest{
		ReferenceImage: "reference", ReferenceImageContext: true, FrameCount: 4,
		Columns: 2, FrameWidth: 64, FrameHeight: 64, FPS: 10, Duration: 5,
	}
	tests := []struct {
		name    string
		targets []int
		context bool
		want    string
	}{
		{name: "outside context", targets: []int{4}, context: true, want: "outside the 4-frame context"},
		{name: "unordered", targets: []int{2, 1}, context: true, want: "unique and ordered"},
		{name: "duplicate", targets: []int{1, 1}, context: true, want: "unique and ordered"},
		{name: "without context", targets: []int{1}, context: false, want: "require an animation context reference"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := base
			request.ReferenceImageContext = test.context
			request.TargetFrameIndices = test.targets
			_, err := normalizeAnimationGenerationRequest(&request)
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

func TestSelectEditFrameContextIndicesSamplesOrderedFullSegment(t *testing.T) {
	analysis := videoprocessor.FrameSequenceAnalysis{
		FPS: 12, ForegroundRatio: .25,
		Frames: make([]videoprocessor.FrameObservation, 10),
	}
	for index := range analysis.Frames {
		analysis.Frames[index].Safe = true
	}
	indices, err := selectEditFrameContextIndices(analysis, 4)
	if err != nil {
		t.Fatalf("select context indices: %v", err)
	}
	want := []int{0, 3, 6, 9}
	if !reflect.DeepEqual(indices, want) {
		t.Fatalf("indices = %v, want %v", indices, want)
	}
}

func TestSelectEditFrameContextIndicesSkipsUnsafeBoundaryFrames(t *testing.T) {
	analysis := videoprocessor.FrameSequenceAnalysis{
		FPS: 12, ForegroundRatio: .25,
		Frames: []videoprocessor.FrameObservation{
			{Safe: false}, {Safe: true}, {Safe: true}, {Safe: true}, {Safe: true},
		},
	}
	indices, err := selectEditFrameContextIndices(analysis, 3)
	if err != nil {
		t.Fatalf("select context indices: %v", err)
	}
	want := []int{1, 3, 4}
	if !reflect.DeepEqual(indices, want) {
		t.Fatalf("indices = %v, want %v", indices, want)
	}
}

func TestSelectEditFrameContextIndicesRejectsUnsafeFrame(t *testing.T) {
	analysis := videoprocessor.FrameSequenceAnalysis{
		FPS: 12, ForegroundRatio: .25,
		Frames: []videoprocessor.FrameObservation{{Safe: true}, {Safe: false}, {Safe: true}},
	}
	_, err := selectEditFrameContextIndices(analysis, 3)
	var qualityError *videoprocessor.QualityError
	if !errors.As(err, &qualityError) || qualityError.Kind != "framing" {
		t.Fatalf("expected framing quality error, got %v", err)
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

func TestProcessAnimationVideoUsesOrderedContextSelection(t *testing.T) {
	frames := animationTestVideoFrames(5)
	processor := &animationProcessorStub{splitResult: &imageprocessor.SplitImageResult{
		ImageBase64: "sheet", MIMEType: "image/png",
		Regions: []imageprocessor.ImageRegion{
			{Index: 0, ImageBase64: "frame-1"},
			{Index: 1, ImageBase64: "frame-2"},
			{Index: 2, ImageBase64: "frame-3"},
		},
	}}
	videoProcessor := &animationVideoProcessorStub{}
	videoProcessor.results = []*videoprocessor.Result{{Frames: frames[1:4]}}
	service := &animationGenerationService{processor: processor, videoProcessor: videoProcessor}

	// The real processor invokes Select before returning the selected images. The
	// stub records the callback, so invoke it with a sequence containing unsafe
	// boundary samples to exercise the edit-context selector and loop metadata.
	videoProcessor.results = nil
	videoProcessor.errors = []error{errors.New("capture options")}
	_, _ = service.processVideo(context.Background(), []byte("video"), AnimationGenerationRequest{
		ReferenceImageContext: true, FrameCount: 3, Columns: 3, FrameWidth: 32, FrameHeight: 32,
	})
	if len(videoProcessor.options) != 1 || videoProcessor.options[0].Select == nil {
		t.Fatalf("context selection callback was not configured: %+v", videoProcessor.options)
	}
	indices, err := videoProcessor.options[0].Select(videoprocessor.FrameSequenceAnalysis{
		FPS: 12, ForegroundRatio: .25,
		Frames: []videoprocessor.FrameObservation{{Safe: false}, {Safe: true}, {Safe: true}, {Safe: true}, {Safe: false}},
	})
	if err != nil {
		t.Fatalf("select ordered context: %v", err)
	}
	if !reflect.DeepEqual(indices, []int{1, 2, 3}) {
		t.Fatalf("context indices = %v, want [1 2 3]", indices)
	}
}

func TestSelectEditFrameContextIndicesValidatesSequence(t *testing.T) {
	tests := []struct {
		name       string
		analysis   videoprocessor.FrameSequenceAnalysis
		frameCount int
		want       string
		kind       string
	}{
		{name: "non-positive count", frameCount: 0, want: "must be positive"},
		{name: "foreground", frameCount: 1, analysis: videoprocessor.FrameSequenceAnalysis{ForegroundRatio: 0.001, Frames: []videoprocessor.FrameObservation{{Safe: true}}}, want: "chroma-key separation failed", kind: "foreground"},
		{name: "too few candidates", frameCount: 2, analysis: videoprocessor.FrameSequenceAnalysis{ForegroundRatio: .25, Frames: []videoprocessor.FrameObservation{{Safe: true}}}, want: "need at least 2"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := selectEditFrameContextIndices(test.analysis, test.frameCount)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
			if test.kind != "" {
				var qualityError *videoprocessor.QualityError
				if !errors.As(err, &qualityError) || qualityError.Kind != test.kind {
					t.Fatalf("expected %s quality error, got %v", test.kind, err)
				}
			}
		})
	}
}

func TestSelectEditFrameContextIndicesUsesMiddleSafeFrameForSingleTarget(t *testing.T) {
	indices, err := selectEditFrameContextIndices(videoprocessor.FrameSequenceAnalysis{
		ForegroundRatio: .25,
		Frames:          []videoprocessor.FrameObservation{{Safe: false}, {Safe: true}, {Safe: true}, {Safe: true}, {Safe: false}},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indices, []int{2}) {
		t.Fatalf("single target indices = %v, want [2]", indices)
	}
}
