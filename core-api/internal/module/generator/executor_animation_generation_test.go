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

	videoclient "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/video_client"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	videoprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/video"
)

type animationVideoServiceStub struct {
	requests  []*videoclient.GenerateRequest
	generated int
	err       error
}

func (s *animationVideoServiceStub) Generate(
	_ context.Context,
	request *videoclient.GenerateRequest,
) (*videoclient.GenerateResult, error) {
	copy := *request
	s.requests = append(s.requests, &copy)
	if s.err != nil {
		return nil, s.err
	}
	s.generated++
	return &videoclient.GenerateResult{
		RequestID: fmt.Sprintf("request-%d", s.generated),
		VideoURL:  fmt.Sprintf("https://video.example/%d.mp4", s.generated),
	}, nil
}

func (s *animationVideoServiceStub) Download(context.Context, string) ([]byte, error) {
	return []byte("video"), nil
}

type animationProcessorStub struct {
	foregroundBase64 string
	removeRequests   []*imageprocessor.RemoveBackgroundRequest
	resizeRequests   []*imageprocessor.ResizeRequest
	splitRequest     *imageprocessor.SplitImageRequest
	splitResult      *imageprocessor.SplitImageResult
	splitErr         error
}

func (s *animationProcessorStub) RemoveBackground(
	_ context.Context,
	request *imageprocessor.RemoveBackgroundRequest,
) (*imageprocessor.RemoveBackgroundResult, error) {
	copy := *request
	s.removeRequests = append(s.removeRequests, &copy)
	return &imageprocessor.RemoveBackgroundResult{
		ImageBase64: s.foregroundBase64,
		MIMEType:    "image/png",
	}, nil
}

func (s *animationProcessorStub) Resize(
	_ context.Context,
	request *imageprocessor.ResizeRequest,
) (*imageprocessor.ResizeResult, error) {
	copy := *request
	s.resizeRequests = append(s.resizeRequests, &copy)
	return &imageprocessor.ResizeResult{
		ImageBase64: s.foregroundBase64,
		MIMEType:    "image/png",
	}, nil
}

func (s *animationProcessorStub) Verify(
	context.Context,
	*imageprocessor.VerifyRequest,
) (*imageprocessor.VerificationReport, error) {
	return &imageprocessor.VerificationReport{Passed: true}, nil
}

func (s *animationProcessorStub) SplitImage(
	_ context.Context,
	request *imageprocessor.SplitImageRequest,
) (*imageprocessor.SplitImageResult, error) {
	copy := *request
	s.splitRequest = &copy
	return s.splitResult, s.splitErr
}

type animationReferenceResolverStub struct {
	resolved string
	err      error
	requests []string
}

func (s *animationReferenceResolverStub) ResolveReference(_ context.Context, reference string) (string, error) {
	s.requests = append(s.requests, reference)
	if s.err != nil {
		return "", s.err
	}
	return s.resolved, nil
}

type animationVideoProcessorStub struct {
	results []*videoprocessor.Result
	errors  []error
	options []videoprocessor.ProcessOptions
	calls   int
}

func (s *animationVideoProcessorStub) Process(_ context.Context, _ []byte, options videoprocessor.ProcessOptions) (*videoprocessor.Result, error) {
	index := s.calls
	s.calls++
	s.options = append(s.options, options)
	if index < len(s.errors) && s.errors[index] != nil {
		return nil, s.errors[index]
	}
	if index >= len(s.results) {
		return nil, errors.New("unexpected video processor call")
	}
	return s.results[index], nil
}

func TestPrepareAnimationReferenceResolvesAndDownloadsUnprocessedImage(t *testing.T) {
	foreground := animationTestForeground(t)
	raw, err := base64.StdEncoding.DecodeString(foreground)
	if err != nil {
		t.Fatalf("decode test foreground: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hero-unprocessed.png" {
			t.Errorf("download path = %q, want /hero-unprocessed.png", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(raw)
	}))
	defer server.Close()

	resolver := &animationReferenceResolverStub{resolved: server.URL + "/hero-unprocessed.png"}
	processor := &animationProcessorStub{foregroundBase64: foreground}
	service := newAnimationGenerationServiceWithResolver(
		&animationVideoServiceStub{},
		processor,
		&animationVideoProcessorStub{},
		resolver,
	).(*animationGenerationService)

	result, err := service.prepareAnimationReference(context.Background(), "uploads/hero-unprocessed.png", false)
	if err != nil {
		t.Fatalf("prepare downloaded reference: %v", err)
	}
	if len(resolver.requests) != 1 || resolver.requests[0] != "uploads/hero-unprocessed.png" {
		t.Fatalf("unexpected resolver requests: %v", resolver.requests)
	}
	if len(processor.resizeRequests) != 1 {
		t.Fatalf("expected one resize request, got %d", len(processor.resizeRequests))
	}
	if _, err := imageprocessor.DecodeBase64Image(result); err != nil {
		t.Fatalf("prepared result is not an image: %v", err)
	}
}

func TestPrepareAnimationReferenceRejectsDownloadFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing unprocessed image", http.StatusNotFound)
	}))
	defer server.Close()

	service := newAnimationGenerationServiceWithResolver(
		nil,
		nil,
		&animationVideoProcessorStub{},
		&animationReferenceResolverStub{resolved: server.URL + "/hero-unprocessed.png"},
	).(*animationGenerationService)
	_, err := service.prepareAnimationReference(context.Background(), "uploads/hero-unprocessed.png", false)
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("expected HTTP 404 error, got %v", err)
	}
}

func TestPrepareAnimationReferenceRejectsInvalidDownloadedImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not an image"))
	}))
	defer server.Close()

	service := newAnimationGenerationServiceWithResolver(
		nil,
		nil,
		&animationVideoProcessorStub{},
		&animationReferenceResolverStub{resolved: server.URL + "/hero-unprocessed.png"},
	).(*animationGenerationService)
	_, err := service.prepareAnimationReference(context.Background(), "uploads/hero-unprocessed.png", false)
	if err == nil || !strings.Contains(err.Error(), "decode downloaded animation reference") {
		t.Fatalf("expected invalid-image error, got %v", err)
	}
}

func TestAnimationGridColumnsPrefersSquareLayout(t *testing.T) {
	tests := []struct {
		frameCount int
		want       int
	}{
		{frameCount: 1, want: 1},
		{frameCount: 5, want: 3},
		{frameCount: 8, want: 3},
		{frameCount: 9, want: 3},
		{frameCount: 10, want: 4},
		{frameCount: 16, want: 4},
		{frameCount: 17, want: 5},
		{frameCount: 32, want: 6},
	}
	for _, test := range tests {
		if got := animationGridColumns(test.frameCount); got != test.want {
			t.Errorf("animationGridColumns(%d) = %d, want %d", test.frameCount, got, test.want)
		}
	}
}

func TestNormalizeAnimationGenerationRequestAppliesDefaults(t *testing.T) {
	result, err := normalizeAnimationGenerationRequest(&AnimationGenerationRequest{
		ReferenceImage: " data:image/png;base64,parent ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.FrameCount != defaultAnimationFrameCount ||
		result.Columns != animationGridColumns(defaultAnimationFrameCount) ||
		result.FrameWidth != defaultAnimationFrameWidth ||
		result.FrameHeight != defaultAnimationFrameHeight ||
		result.FPS != defaultAnimationFPS ||
		result.Resolution != defaultAnimationResolution ||
		result.Duration != defaultAnimationDuration ||
		result.AspectRatio != defaultAnimationAspectRatio {
		t.Fatalf("unexpected defaults: %+v", result)
	}
	if result.ReferenceImage != "data:image/png;base64,parent" || result.Action != "idle" {
		t.Fatalf("unexpected normalized request: %+v", result)
	}
}

func TestNormalizeAnimationGenerationRequestSupportsGameFrameSizes(t *testing.T) {
	for _, size := range []int{32, 64} {
		result, err := normalizeAnimationGenerationRequest(&AnimationGenerationRequest{
			ReferenceImage: "prepared",
			FrameWidth:     size,
			FrameHeight:    size,
		})
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		if result.FrameWidth != size || result.FrameHeight != size {
			t.Fatalf("size %d normalized to %dx%d", size, result.FrameWidth, result.FrameHeight)
		}
	}
	_, err := normalizeAnimationGenerationRequest(&AnimationGenerationRequest{
		ReferenceImage: "prepared",
		FrameWidth:     31,
		FrameHeight:    32,
	})
	if err == nil {
		t.Fatal("31px frame should be rejected")
	}
}

func TestAnimationGenerationKeepsPreparedGreenReference(t *testing.T) {
	prepared := animationTestPreparedGreenReference(t)
	videos := &animationVideoServiceStub{}
	processor := &animationProcessorStub{}
	wantErr := errors.New("stop after provider call")
	videoProcessor := &animationVideoProcessorStub{errors: []error{wantErr}}
	service := newAnimationGenerationService(videos, processor, videoProcessor)

	_, err := service.Generate(context.Background(), &AnimationGenerationRequest{
		ReferenceImage:         "data:image/png;base64," + prepared,
		ReferenceImagePrepared: true,
		FrameCount:             4,
		Columns:                2,
		FrameWidth:             32,
		FrameHeight:            32,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("generate animation: %v", err)
	}
	if len(processor.removeRequests) != 0 || len(processor.resizeRequests) != 0 {
		t.Fatalf("prepared reference was modified: remove=%d resize=%d", len(processor.removeRequests), len(processor.resizeRequests))
	}
	if len(videos.requests) != 1 {
		t.Fatalf("video requests = %d, want 1", len(videos.requests))
	}
	got, decodeErr := imageprocessor.DecodeBase64Image(videos.requests[0].ReferenceImageBase64)
	if decodeErr != nil {
		t.Fatalf("decode provider reference: %v", decodeErr)
	}
	if got.Bounds().Dx() != 256 || got.Bounds().Dy() != 512 {
		t.Fatalf("prepared reference resized to %v", got.Bounds().Size())
	}
	if corner := color.NRGBAModel.Convert(got.At(0, 0)).(color.NRGBA); corner != (color.NRGBA{G: 255, A: 255}) {
		t.Fatalf("prepared reference corner = %#v", corner)
	}
}

func TestAnimationGenerationUsesParentPrototypeAndRetriesQualityError(t *testing.T) {
	foreground := animationTestForeground(t)
	parent := animationTestOpaquePrototype(t)
	videos := &animationVideoServiceStub{}
	processor := &animationProcessorStub{
		foregroundBase64: foreground,
		splitResult: &imageprocessor.SplitImageResult{
			Mode:        imageprocessor.ImageSplitModeAnimation,
			ImageBase64: "spritesheet",
			MIMEType:    "image/png",
			Regions: []imageprocessor.ImageRegion{
				{Index: 0, ImageBase64: "frame-1", MIMEType: "image/png"},
				{Index: 1, ImageBase64: "frame-2", MIMEType: "image/png"},
				{Index: 2, ImageBase64: "frame-3", MIMEType: "image/png"},
				{Index: 3, ImageBase64: "frame-4", MIMEType: "image/png"},
			},
		},
	}
	qualityErr := &videoprocessor.QualityError{Kind: "framing", Message: "unsafe framing"}
	videoProcessor := &animationVideoProcessorStub{
		errors:  []error{fmt.Errorf("wrapped quality error: %w", qualityErr), nil},
		results: []*videoprocessor.Result{nil, {Frames: animationTestVideoFrames(4)}},
	}
	service := newAnimationGenerationService(videos, processor, videoProcessor)
	action := "以左脚为轴完成不规则仪式动作，然后把容器放回腰间"

	result, err := service.Generate(context.Background(), &AnimationGenerationRequest{
		Description:    "knight",
		Action:         action,
		ReferenceImage: parent,
		FrameCount:     4,
		Columns:        2,
		FrameWidth:     64,
		FrameHeight:    64,
		FPS:            10,
	})
	if err != nil {
		t.Fatalf("generate animation: %v", err)
	}
	if result.VideoAttempts != 2 || result.VideoRequestID != "request-2" ||
		result.FrameDurationMS != 100 || len(result.Frames) != 4 {
		t.Fatalf("unexpected generation result: %+v", result)
	}
	if len(videos.requests) != 2 || !strings.Contains(videos.requests[1].Prompt, "QUALITY RETRY OVERRIDE") {
		t.Fatalf("quality retry was not issued: %+v", videos.requests)
	}
	if len(videoProcessor.options) != 2 || videoProcessor.options[1].AnalysisFPS != animationAnalysisFPS ||
		videoProcessor.options[1].Select == nil || videoProcessor.options[1].ChromaKey.HueMin != animationChromaHueMin {
		t.Fatalf("executor did not supply media selection policy: %+v", videoProcessor.options)
	}
	if !strings.Contains(videos.requests[0].Prompt, action) ||
		!strings.Contains(videos.requests[0].Prompt, "interpret the requested action by its actual meaning") {
		t.Fatalf("semantic action was not preserved in prompt: %s", videos.requests[0].Prompt)
	}
	if len(processor.removeRequests) != 1 ||
		processor.removeRequests[0].ImageBase64 != parent ||
		processor.removeRequests[0].MatteColor != "auto" {
		t.Fatalf("parent prototype was not passed directly to background removal: %+v", processor.removeRequests)
	}
	if len(processor.resizeRequests) != 1 ||
		processor.resizeRequests[0].ImageBase64 != foreground ||
		processor.resizeRequests[0].Options.Width != animationReferenceSize ||
		processor.resizeRequests[0].Options.Height != animationReferenceSize ||
		processor.resizeRequests[0].Options.Margin != imageprocessor.AnimationFrameMargin(animationReferenceSize, animationReferenceSize) {
		t.Fatalf("unexpected parent prototype resize request: %+v", processor.resizeRequests)
	}
	greenReference, decodeErr := imageprocessor.DecodeBase64Image(videos.requests[0].ReferenceImageBase64)
	if decodeErr != nil {
		t.Fatalf("decode video reference: %v", decodeErr)
	}
	if got := color.NRGBAModel.Convert(greenReference.At(0, 0)).(color.NRGBA); got != (color.NRGBA{G: 255, A: 255}) {
		t.Fatalf("video reference corner = %#v, want pure green", got)
	}
	if processor.splitRequest == nil ||
		processor.splitRequest.Mode != imageprocessor.ImageSplitModeAnimation ||
		processor.splitRequest.Columns != 2 || processor.splitRequest.Rows != 2 ||
		processor.splitRequest.FrameCount != 4 ||
		processor.splitRequest.FrameWidth != 64 || processor.splitRequest.FrameHeight != 64 ||
		processor.splitRequest.Margin != imageprocessor.AnimationFrameMargin(64, 64) ||
		processor.splitRequest.Anchor != imageprocessor.AnimationAnchorFeet ||
		!processor.splitRequest.ForceProportionalGrid ||
		!processor.splitRequest.PreserveVerticalMotion ||
		!processor.splitRequest.PreserveSourceCellScale ||
		processor.splitRequest.Background == nil ||
		processor.splitRequest.Background.MatteColor != "auto" {
		t.Fatalf("unexpected split request: %+v", processor.splitRequest)
	}
}

func TestPrepareGreenReferenceKeepsExistingTransparentPrototype(t *testing.T) {
	prototype := animationTestForeground(t)
	processor := &animationProcessorStub{foregroundBase64: prototype}
	service := &animationGenerationService{processor: processor}

	_, err := service.prepareGreenReference(context.Background(), "data:image/png;base64,"+prototype)
	if err != nil {
		t.Fatalf("prepare transparent reference: %v", err)
	}
	if len(processor.removeRequests) != 0 {
		t.Fatalf("transparent prototype should not be background-removed again: %+v", processor.removeRequests)
	}
	if len(processor.resizeRequests) != 1 || processor.resizeRequests[0].ImageBase64 != "data:image/png;base64,"+prototype {
		t.Fatalf("transparent prototype was not resized directly: %+v", processor.resizeRequests)
	}
}

func TestAnimationGenerationDoesNotRetryNonQualityError(t *testing.T) {
	foreground := animationTestForeground(t)
	videos := &animationVideoServiceStub{}
	processor := &animationProcessorStub{foregroundBase64: foreground}
	wantErr := errors.New("ffmpeg failed")
	videoProcessor := &animationVideoProcessorStub{errors: []error{wantErr}}
	service := newAnimationGenerationService(videos, processor, videoProcessor)

	_, err := service.Generate(context.Background(), &AnimationGenerationRequest{
		ReferenceImage: animationTestOpaquePrototype(t),
		FrameCount:     4,
		Columns:        2,
		FrameWidth:     64,
		FrameHeight:    64,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected extractor error, got %v", err)
	}
	if len(videos.requests) != 1 || videoProcessor.calls != 1 {
		t.Fatalf("non-quality error retried: video=%d processor=%d", len(videos.requests), videoProcessor.calls)
	}
}

func TestProcessAnimationVideoUsesRealAnimationNormalizer(t *testing.T) {
	videoProcessor := &animationVideoProcessorStub{results: []*videoprocessor.Result{{
		Frames: animationTestVideoFrames(4),
	}}}
	service := &animationGenerationService{
		processor:      imageprocessor.NewProcessor(),
		videoProcessor: videoProcessor,
	}
	result, err := service.processVideo(context.Background(), []byte("video"), AnimationGenerationRequest{
		Action:      "idle breathing",
		FrameCount:  4,
		Columns:     2,
		FrameWidth:  64,
		FrameHeight: 64,
	})
	if err != nil {
		t.Fatalf("process video: %v", err)
	}
	if len(result.Frames) != 4 || result.Normalization == nil || result.Spritesheet == "" {
		t.Fatalf("unexpected normalized result: %+v", result)
	}
	for index, frame := range result.Frames {
		decoded, decodeErr := imageprocessor.DecodeBase64Image(frame.ImageBase64)
		if decodeErr != nil {
			t.Fatalf("decode frame %d: %v", index, decodeErr)
		}
		if decoded.Bounds().Dx() != 64 || decoded.Bounds().Dy() != 64 {
			t.Fatalf("frame %d has size %v", index, decoded.Bounds().Size())
		}
	}
}

func TestProcessAnimationVideoAutoDetectsNonGreenMatte(t *testing.T) {
	videoProcessor := &animationVideoProcessorStub{results: []*videoprocessor.Result{{
		Frames: animationTestVideoFramesWithMatte(4, color.NRGBA{R: 235, G: 235, B: 235, A: 255}),
	}}}
	service := &animationGenerationService{
		processor:      imageprocessor.NewProcessor(),
		videoProcessor: videoProcessor,
	}

	result, err := service.processVideo(context.Background(), []byte("video"), AnimationGenerationRequest{
		Action: "idle breathing", FrameCount: 4, Columns: 2, FrameWidth: 64, FrameHeight: 64,
	})
	if err != nil {
		t.Fatalf("process video with non-green matte: %v", err)
	}
	if result.Normalization == nil || result.Normalization.BackgroundRemovalReport == nil {
		t.Fatal("expected automatic background removal report")
	}
	if result.Normalization.BackgroundRemovalReport.MatteColorSource != "auto-sampled" {
		t.Fatalf("matte source = %q, want auto-sampled", result.Normalization.BackgroundRemovalReport.MatteColorSource)
	}
}

func animationTestForeground(t *testing.T) string {
	t.Helper()
	frame := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	draw.Draw(frame, image.Rect(30, 16, 66, 88), &image.Uniform{C: color.NRGBA{R: 140, G: 50, B: 35, A: 255}}, image.Point{}, draw.Src)
	encoded, err := imageprocessor.EncodePNGBase64(frame)
	if err != nil {
		t.Fatalf("encode foreground: %v", err)
	}
	return encoded
}

func animationTestPreparedGreenReference(t *testing.T) string {
	t.Helper()
	frame := image.NewNRGBA(image.Rect(0, 0, 256, 512))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(frame, image.Rect(96, 80, 160, 448), &image.Uniform{C: color.NRGBA{R: 140, G: 50, B: 35, A: 255}}, image.Point{}, draw.Src)
	encoded, err := imageprocessor.EncodePNGBase64(frame)
	if err != nil {
		t.Fatalf("encode prepared green reference: %v", err)
	}
	return encoded
}

func animationTestOpaquePrototype(t *testing.T) string {
	t.Helper()
	frame := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{R: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(frame, image.Rect(30, 16, 66, 88), &image.Uniform{C: color.NRGBA{R: 140, G: 50, B: 35, A: 255}}, image.Point{}, draw.Src)
	encoded, err := imageprocessor.EncodePNGBase64(frame)
	if err != nil {
		t.Fatalf("encode opaque prototype: %v", err)
	}
	return "data:image/png;base64," + encoded
}

func animationTestVideoFrames(count int) []image.Image {
	return animationTestVideoFramesWithMatte(count, color.NRGBA{G: 255, A: 255})
}

func animationTestVideoFramesWithMatte(count int, matte color.NRGBA) []image.Image {
	frames := make([]image.Image, count)
	for index := range frames {
		frame := image.NewNRGBA(image.Rect(0, 0, 96, 96))
		draw.Draw(frame, frame.Bounds(), &image.Uniform{C: matte}, image.Point{}, draw.Src)
		offset := index % 3
		draw.Draw(frame, image.Rect(30+offset, 18, 66+offset, 88), &image.Uniform{C: color.NRGBA{R: 140, G: 50, B: 35, A: 255}}, image.Point{}, draw.Src)
		frames[index] = frame
	}
	return frames
}

var _ videoclient.VideoGenerationService = (*animationVideoServiceStub)(nil)
var _ imageprocessor.Processor = (*animationProcessorStub)(nil)
var _ videoprocessor.Processor = (*animationVideoProcessorStub)(nil)
