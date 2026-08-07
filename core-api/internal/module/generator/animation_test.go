package generator

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	videoclient "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/video_client"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
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

type animationExtractorStub struct {
	results [][]image.Image
	errors  []error
	calls   int
}

func (s *animationExtractorStub) Extract(context.Context, []byte, int) ([]image.Image, error) {
	index := s.calls
	s.calls++
	if index < len(s.errors) && s.errors[index] != nil {
		return nil, s.errors[index]
	}
	if index >= len(s.results) {
		return nil, errors.New("unexpected extractor call")
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
		&animationExtractorStub{},
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
		&animationExtractorStub{},
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
		&animationExtractorStub{},
		&animationReferenceResolverStub{resolved: server.URL + "/hero-unprocessed.png"},
	).(*animationGenerationService)
	_, err := service.prepareAnimationReference(context.Background(), "uploads/hero-unprocessed.png", false)
	if err == nil || !strings.Contains(err.Error(), "decode downloaded animation reference") {
		t.Fatalf("expected invalid-image error, got %v", err)
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
		result.Columns != defaultAnimationColumns ||
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
	extractor := &animationExtractorStub{errors: []error{wantErr}}
	service := newAnimationGenerationService(videos, processor, extractor)

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
	qualityErr := &AnimationVideoQualityError{Kind: "framing", Message: "unsafe framing"}
	extractor := &animationExtractorStub{
		errors:  []error{fmt.Errorf("wrapped quality error: %w", qualityErr), nil},
		results: [][]image.Image{nil, animationTestVideoFrames(12)},
	}
	service := newAnimationGenerationService(videos, processor, extractor)
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
		!processor.splitRequest.PreserveSourceCellScale {
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
	extractor := &animationExtractorStub{errors: []error{wantErr}}
	service := newAnimationGenerationService(videos, processor, extractor)

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
	if len(videos.requests) != 1 || extractor.calls != 1 {
		t.Fatalf("non-quality error retried: video=%d extractor=%d", len(videos.requests), extractor.calls)
	}
}

func TestProcessAnimationVideoUsesRealAnimationNormalizer(t *testing.T) {
	extractor := &animationExtractorStub{results: [][]image.Image{animationTestVideoFrames(12)}}
	service := &animationGenerationService{
		processor: imageprocessor.NewProcessor(),
		extractor: extractor,
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

func TestSelectAnimationLoopFramesUsesCompleteLongCycle(t *testing.T) {
	frames := make([]image.Image, 48)
	for index := range frames {
		frame := image.NewNRGBA(image.Rect(0, 0, 120, 120))
		draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
		draw.Draw(frame, image.Rect(49, 24, 69, 104), &image.Uniform{C: color.NRGBA{R: 105, G: 50, B: 32, A: 255}}, image.Point{}, draw.Src)
		var prop image.Rectangle
		switch {
		case index < 16:
			x := 26 + animationMinInt(index%8, 8-index%8)*3
			prop = image.Rect(x, 42, x+42, 49)
		case index < 30:
			y := 42 + (index-16)*3
			prop = image.Rect(35, y, 88, y+7)
		default:
			y := 84 - (index-30)*2
			prop = image.Rect(35, y, 88, y+7)
		}
		draw.Draw(frame, prop, &image.Uniform{C: color.NRGBA{R: 105, G: 50, B: 32, A: 255}}, image.Point{}, draw.Src)
		frames[index] = frame
	}

	indices, loop, err := selectAnimationLoopFrames(frames, 16, 12)
	if err != nil {
		t.Fatal(err)
	}
	if loop.SpanRatio < animationMinLoopSpanRatio || indices[len(indices)-1] != loop.EndFrame || loop.StartFrame > int(math.Ceil(float64(len(frames))*animationInitialWindowRatio)) {
		t.Fatalf("selector omitted part of the complete cycle: loop=%+v indices=%v", loop, indices)
	}
	if loop.Method != "subject_mse_full_cycle" {
		t.Fatalf("loop method = %q", loop.Method)
	}
	for index := 1; index < len(indices); index++ {
		if indices[index] <= indices[index-1] {
			t.Fatalf("sampled indices are not strictly ordered: %v", indices)
		}
	}
}

func TestSelectAnimationLoopFramesSkipsTransientUnsafeSourceFrames(t *testing.T) {
	frames := animationTestVideoFrames(48)
	unsafe := map[int]bool{10: true, 30: true}
	for index := range unsafe {
		frame, ok := frames[index].(*image.NRGBA)
		if !ok {
			t.Fatalf("frame %d has type %T", index, frames[index])
		}
		draw.Draw(frame, image.Rect(0, 40, 12, 48), &image.Uniform{C: color.NRGBA{R: 105, G: 50, B: 32, A: 255}}, image.Point{}, draw.Src)
	}

	indices, loop, err := selectAnimationLoopFrames(frames, 8, 12)
	if err != nil {
		t.Fatalf("select loop with transient unsafe source frames: %v", err)
	}
	if loop.SpanRatio < animationMinLoopSpanRatio {
		t.Fatalf("selected interval is too short: %+v", loop)
	}
	for _, index := range indices {
		if unsafe[index] {
			t.Fatalf("unsafe source frame %d was selected: %v", index, indices)
		}
	}
	if err := validateAnimationMotionSafeAreaAtIndices(frames, indices); err != nil {
		t.Fatalf("selected frames should remain inside the safety band: %v", err)
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
	frames := make([]image.Image, count)
	for index := range frames {
		frame := image.NewNRGBA(image.Rect(0, 0, 96, 96))
		draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
		offset := index % 3
		draw.Draw(frame, image.Rect(30+offset, 18, 66+offset, 88), &image.Uniform{C: color.NRGBA{R: 140, G: 50, B: 35, A: 255}}, image.Point{}, draw.Src)
		frames[index] = frame
	}
	return frames
}

var _ videoclient.VideoGenerationService = (*animationVideoServiceStub)(nil)
var _ imageprocessor.Processor = (*animationProcessorStub)(nil)
var _ animationFrameExtractor = (*animationExtractorStub)(nil)

func TestSelectAnimationLoopFramesDoesNotSampleIdleTail(t *testing.T) {
	const total = 60
	frames := make([]image.Image, total)
	for index := range frames {
		frame := image.NewNRGBA(image.Rect(0, 0, 120, 120))
		draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
		draw.Draw(frame, image.Rect(49, 24, 69, 104), &image.Uniform{C: color.NRGBA{R: 105, G: 50, B: 32, A: 255}}, image.Point{}, draw.Src)

		// The action completes by frame 42 and the provider then holds the
		// recovered pose until the end of the five-second video.
		propX, propY := 35, 42
		switch {
		case index < 10:
			propY = 42 + index*2
		case index < 26:
			propX = 35 + (index-10)*2
			propY = 60 + (index-10)*2
		case index < 35:
			propX = 65 - (index-26)*3
			propY = 90 - (index-26)*4
		case index < 43:
			propX = 38 - (index-35)*1
			propY = 54 - (index-35)*1
		}
		if index >= 43 {
			propX, propY = 35, 42
		}
		draw.Draw(frame, image.Rect(propX, propY, propX+42, propY+7), &image.Uniform{C: color.NRGBA{R: 105, G: 50, B: 32, A: 255}}, image.Point{}, draw.Src)
		frames[index] = frame
	}

	indices, loop, err := selectAnimationLoopFrames(frames, 8, 12)
	if err != nil {
		t.Fatalf("select loop: %v", err)
	}
	if loop.StartFrame != 0 {
		t.Fatalf("action should begin at the source initial pose: %+v", loop)
	}
	if loop.EndFrame < 40 || loop.EndFrame > 45 {
		t.Fatalf("selector should include recovery but exclude idle tail: %+v indices=%v", loop, indices)
	}
	if indices[len(indices)-1] != loop.EndFrame {
		t.Fatalf("last sampled frame must be recovery endpoint: %+v indices=%v", loop, indices)
	}
}
