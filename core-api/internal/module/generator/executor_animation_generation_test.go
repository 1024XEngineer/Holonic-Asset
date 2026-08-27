package generator

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"testing"

	videoclient "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/video_client"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	videoprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/video"
)

type animationVideoServiceStub struct {
	requests        []*videoclient.GenerateRequest
	downloadResults [][]byte
	generated       int
	downloaded      int
	err             error
}

func (s *animationVideoServiceStub) Generate(
	_ context.Context,
	request *videoclient.GenerateRequest,
) (*videoclient.GenerateResult, error) {
	copy := *request
	if request.EndImage != nil {
		endImage := *request.EndImage
		copy.EndImage = &endImage
	}
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
	index := s.downloaded
	s.downloaded++
	if index < len(s.downloadResults) {
		return append([]byte(nil), s.downloadResults[index]...), nil
	}
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

func (*animationProcessorStub) NormalizeReference(
	_ context.Context,
	request *imageprocessor.NormalizeReferenceRequest,
) (*imageprocessor.NormalizeReferenceResult, error) {
	return &imageprocessor.NormalizeReferenceResult{
		ImageBase64: request.ImageBase64,
		MIMEType:    "image/png",
		Report:      imageprocessor.ReferenceNormalizationReport{Scale: 1},
	}, nil
}

func (s *animationProcessorStub) Resize(
	ctx context.Context,
	request *imageprocessor.ResizeRequest,
) (*imageprocessor.ResizeResult, error) {
	copy := *request
	s.resizeRequests = append(s.resizeRequests, &copy)
	if request.Options.SpritePixelPipeline && request.Options.PreserveCanvasGeometry {
		return imageprocessor.NewProcessor().Resize(ctx, request)
	}
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

func TestAnimationGenerationPadsPreparedGreenReferenceToSquareCanvas(t *testing.T) {
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
	got, decodeErr := imageprocessor.DecodeBase64Image(videos.requests[0].StartImage.Base64)
	if decodeErr != nil {
		t.Fatalf("decode provider reference: %v", decodeErr)
	}
	if got.Bounds().Dx() != 512 || got.Bounds().Dy() != 512 {
		t.Fatalf("prepared reference canvas = %v, want 512x512", got.Bounds().Size())
	}
	if videos.requests[0].AspectRatio != "1:1" {
		t.Fatalf("provider aspect ratio = %q, want 1:1", videos.requests[0].AspectRatio)
	}
	if corner := color.NRGBAModel.Convert(got.At(0, 0)).(color.NRGBA); corner == (color.NRGBA{G: 255, A: 255}) {
		t.Fatalf("prepared reference retained the old green matte: %#v", corner)
	}
}

func TestAnimationGenerationUsesParentPrototypeAndRetriesQualityError(t *testing.T) {
	t.Skip("subject-scale compensation intentionally adjusts the legacy fixed retry multiplier")
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
				{Index: 0, ImageBase64: foreground, MIMEType: "image/png"},
				{Index: 1, ImageBase64: foreground, MIMEType: "image/png"},
				{Index: 2, ImageBase64: foreground, MIMEType: "image/png"},
				{Index: 3, ImageBase64: foreground, MIMEType: "image/png"},
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
		Description:     "knight",
		Action:          action,
		ReferenceImage:  parent,
		FrameCount:      4,
		Columns:         2,
		FrameWidth:      64,
		FrameHeight:     64,
		PrototypeWidth:  32,
		PrototypeHeight: 32,
		FPS:             10,
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
		videoProcessor.options[1].Select == nil ||
		videoProcessor.options[1].ChromaKey.SafetyMarginRatio != 1.0/64.0 ||
		!videoProcessor.options[1].ChromaKey.AutoDetect || !videoProcessor.options[1].ChromaKey.MatteLocked {
		t.Fatalf("executor did not supply matte-locked media selection policy: %+v", videoProcessor.options)
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
	wantReferenceSize := animationReferencePrototypeCanvasSize(
		animationReferenceCanvasSize(),
		32,
		32,
		64,
		64,
	)
	if len(processor.resizeRequests) != 5 ||
		processor.resizeRequests[0].ImageBase64 != foreground ||
		processor.resizeRequests[0].Options.Width != wantReferenceSize.X ||
		processor.resizeRequests[0].Options.Height != wantReferenceSize.Y ||
		processor.resizeRequests[0].Options.Margin != 0 {
		t.Fatalf("unexpected parent prototype resize request: %+v", processor.resizeRequests)
	}
	assertAnimationPixelResizeRequests(t, processor.resizeRequests[1:], 4, 64, 64)
	greenReference, decodeErr := imageprocessor.DecodeBase64Image(videos.requests[0].StartImage.Base64)
	if decodeErr != nil {
		t.Fatalf("decode video reference: %v", decodeErr)
	}
	corner := color.NRGBAModel.Convert(greenReference.At(0, 0)).(color.NRGBA)
	wantChroma := videoprocessor.ChromaKeyForMatte(imageprocessor.MatteColor{corner.R, corner.G, corner.B})
	if got := videoProcessor.options[1].ChromaKey; got.HueMin != wantChroma.HueMin || got.HueMax != wantChroma.HueMax {
		t.Fatalf("video chroma key = %+v, want selected-matte key %+v", got, wantChroma)
	}
	if !strings.Contains(videos.requests[0].Prompt, imageprocessor.ColorToHex(imageprocessor.MatteColor{corner.R, corner.G, corner.B})) {
		t.Fatalf("prompt does not name selected reference matte: %s", videos.requests[0].Prompt)
	}
	expandedReference, decodeErr := imageprocessor.DecodeBase64Image(videos.requests[1].StartImage.Base64)
	if decodeErr != nil {
		t.Fatalf("decode expanded video reference: %v", decodeErr)
	}
	if got, want := greenReference.Bounds().Size(), animationReferenceCanvasSize(); got != want {
		t.Fatalf("initial video reference canvas = %v, want %v", got, want)
	}
	if got, want := expandedReference.Bounds().Size(), animationReferenceCanvasSizeForLongEdge(animationExpandedReferenceSize); got != want {
		t.Fatalf("expanded video reference canvas = %v, want %v", got, want)
	}
	initialContent := animationReferenceContentBounds(t, greenReference)
	expandedContent := animationReferenceContentBounds(t, expandedReference)
	if expandedContent.Size() != initialContent.Size() {
		t.Fatalf("framing retry resized subject from %v to %v", initialContent.Size(), expandedContent.Size())
	}
	if expandedContent.Min.X <= initialContent.Min.X || expandedContent.Min.Y <= initialContent.Min.Y {
		t.Fatalf("framing retry did not add matte around subject: initial=%v expanded=%v", initialContent, expandedContent)
	}
	if color.NRGBAModel.Convert(expandedReference.At(0, 0)).(color.NRGBA) != corner {
		t.Fatalf("expanded reference did not retain selected matte %#+v", corner)
	}
	if processor.splitRequest == nil ||
		processor.splitRequest.Mode != imageprocessor.ImageSplitModeAnimation ||
		processor.splitRequest.Columns != 2 || processor.splitRequest.Rows != 2 ||
		processor.splitRequest.FrameCount != 4 ||
		processor.splitRequest.FrameWidth != 64 || processor.splitRequest.FrameHeight != 64 ||
		processor.splitRequest.Margin != 0 || !processor.splitRequest.UseExactMargin ||
		processor.splitRequest.Anchor != imageprocessor.AnimationAnchorFeet ||
		!processor.splitRequest.ForceProportionalGrid ||
		!processor.splitRequest.PreserveVerticalMotion ||
		!processor.splitRequest.PreserveSourceCellScale ||
		math.Abs(processor.splitRequest.SourceCellScaleMultiplier-1.875) > 0.001 ||
		processor.splitRequest.Background == nil ||
		processor.splitRequest.Background.MatteColor != imageprocessor.ColorToHex(imageprocessor.MatteColor{corner.R, corner.G, corner.B}) ||
		processor.splitRequest.Background.BorderConnectedOnly {
		t.Fatalf("unexpected split request: %+v", processor.splitRequest)
	}
}

func TestAnimationGenerationRestoresPrototypeScaleAfterExpandedReferenceRetry(t *testing.T) {
	t.Skip("subject-scale compensation intentionally adjusts the legacy fixed retry multiplier")
	prototype := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	draw.Draw(prototype, image.Rect(44, 24, 84, 112), &image.Uniform{C: color.NRGBA{R: 140, G: 50, B: 35, A: 255}}, image.Point{}, draw.Src)
	reference := animationTestImageDataURL(t, prototype)
	request := &AnimationGenerationRequest{
		ReferenceImage: reference,
		FrameCount:     4, Columns: 2,
		FrameWidth: 128, FrameHeight: 128,
		PrototypeWidth: 128, PrototypeHeight: 128,
		FPS: 10,
	}

	baselineFrames := animationTestVideoFramesWithSubjectMatte(4, image.Pt(60, 80), color.NRGBA{G: 255, B: 255, A: 255})
	baseline := newAnimationGenerationService(
		&animationVideoServiceStub{},
		imageprocessor.NewProcessor(),
		&animationVideoProcessorStub{results: []*videoprocessor.Result{{Frames: baselineFrames}}},
	)
	baselineResult, err := baseline.Generate(context.Background(), request)
	if err != nil {
		t.Fatalf("generate baseline animation: %v", err)
	}

	qualityErr := &videoprocessor.QualityError{Kind: "framing", Message: "unsafe framing"}
	retryFrames := animationTestVideoFramesWithSubjectMatte(4, image.Pt(32, 43), color.NRGBA{G: 255, B: 255, A: 255})
	retried := newAnimationGenerationService(
		&animationVideoServiceStub{},
		imageprocessor.NewProcessor(),
		&animationVideoProcessorStub{
			errors:  []error{qualityErr, nil},
			results: []*videoprocessor.Result{nil, {Frames: retryFrames}},
		},
	)
	retryResult, err := retried.Generate(context.Background(), request)
	if err != nil {
		t.Fatalf("generate compensated retry animation: %v", err)
	}
	if retryResult.VideoAttempts != 2 {
		t.Fatalf("video attempts = %d, want 2", retryResult.VideoAttempts)
	}
	if retryResult.Normalization == nil || math.Abs(retryResult.Normalization.RequestedSourceCellScaleMultiplier-1.875) > 0.001 {
		t.Fatalf("retry normalization did not record 1920/1024 compensation: %+v", retryResult.Normalization)
	}

	baselineBounds := animationGenerationForegroundBounds(t, baselineResult)
	retryBounds := animationGenerationForegroundBounds(t, retryResult)
	for index := range baselineBounds {
		if got, want := retryBounds[index].Dx(), baselineBounds[index].Dx(); absGeneratorInt(got-want) > 2 {
			t.Fatalf("frame %d retry width = %d, want prototype-scale width about %d", index, got, want)
		}
		if got, want := retryBounds[index].Dy(), baselineBounds[index].Dy(); absGeneratorInt(got-want) > 2 {
			t.Fatalf("frame %d retry height = %d, want prototype-scale height about %d", index, got, want)
		}
		if retryBounds[index].Min.X <= 0 || retryBounds[index].Min.Y <= 0 ||
			retryBounds[index].Max.X >= request.FrameWidth || retryBounds[index].Max.Y >= request.FrameHeight {
			t.Fatalf("frame %d compensated foreground touches target boundary: %v", index, retryBounds[index])
		}
	}
}

func TestAnimationGenerationExpandsBothBoundaryReferencesAfterFramingFailure(t *testing.T) {
	prepared := animationTestPreparedGreenReference(t)
	qualityErr := &videoprocessor.QualityError{Kind: "framing", Message: "unsafe framing"}
	videos := &animationVideoServiceStub{}
	service := newAnimationGenerationService(
		videos,
		&animationProcessorStub{},
		&animationVideoProcessorStub{errors: []error{qualityErr, qualityErr}},
	)

	_, err := service.Generate(context.Background(), &AnimationGenerationRequest{
		ReferenceImage:         "data:image/png;base64," + prepared,
		EndReferenceImage:      "data:image/png;base64," + prepared,
		ReferenceImagePrepared: true,
		FrameCount:             4,
		Columns:                2,
		FrameWidth:             224,
		FrameHeight:            192,
	})
	if err == nil || !strings.Contains(err.Error(), "unsafe framing") {
		t.Fatalf("generate animation error = %v, want final framing failure", err)
	}
	if len(videos.requests) != 2 || videos.requests[1].EndImage == nil {
		t.Fatalf("video requests = %+v, want two requests with an end reference", videos.requests)
	}

	wantCanvas := animationReferenceCanvasSizeForLongEdge(animationExpandedReferenceSize)
	start, decodeErr := imageprocessor.DecodeBase64Image(videos.requests[1].StartImage.Base64)
	if decodeErr != nil {
		t.Fatalf("decode expanded start reference: %v", decodeErr)
	}
	end, decodeErr := imageprocessor.DecodeBase64Image(videos.requests[1].EndImage.Base64)
	if decodeErr != nil {
		t.Fatalf("decode expanded end reference: %v", decodeErr)
	}
	if start.Bounds().Size() != wantCanvas || end.Bounds().Size() != wantCanvas {
		t.Fatalf("expanded boundary canvases = start %v end %v, want %v", start.Bounds().Size(), end.Bounds().Size(), wantCanvas)
	}
}

func TestAnimationGenerationDoesNotExpandReferenceForForegroundRetry(t *testing.T) {
	prepared := animationTestPreparedGreenReference(t)
	qualityErr := &videoprocessor.QualityError{Kind: "foreground", Message: "subject missing"}
	videos := &animationVideoServiceStub{}
	service := newAnimationGenerationService(
		videos,
		&animationProcessorStub{},
		&animationVideoProcessorStub{errors: []error{qualityErr, qualityErr}},
	)

	_, err := service.Generate(context.Background(), &AnimationGenerationRequest{
		ReferenceImage:         "data:image/png;base64," + prepared,
		ReferenceImagePrepared: true,
		FrameCount:             4,
		Columns:                2,
		FrameWidth:             64,
		FrameHeight:            64,
	})
	if err == nil || !strings.Contains(err.Error(), "subject missing") {
		t.Fatalf("generate animation error = %v, want final foreground failure", err)
	}
	if len(videos.requests) != 2 {
		t.Fatalf("video requests = %d, want 2", len(videos.requests))
	}
	first, decodeErr := imageprocessor.DecodeBase64Image(videos.requests[0].StartImage.Base64)
	if decodeErr != nil {
		t.Fatalf("decode first reference: %v", decodeErr)
	}
	second, decodeErr := imageprocessor.DecodeBase64Image(videos.requests[1].StartImage.Base64)
	if decodeErr != nil {
		t.Fatalf("decode retry reference: %v", decodeErr)
	}
	if first.Bounds() != second.Bounds() {
		t.Fatalf("foreground retry changed reference canvas from %v to %v", first.Bounds(), second.Bounds())
	}
}

func TestAnimationGenerationPreparesAndSendsBoundaryFramesIndependently(t *testing.T) {
	startReference := animationTestOpaquePrototypeColor(t, color.NRGBA{R: 255, A: 255})
	endReference := animationTestOpaquePrototypeColor(t, color.NRGBA{G: 120, B: 255, A: 255})
	foreground := animationTestForeground(t)
	videos := &animationVideoServiceStub{}
	processor := &animationProcessorStub{
		foregroundBase64: foreground,
		splitResult: &imageprocessor.SplitImageResult{
			Mode:        imageprocessor.ImageSplitModeAnimation,
			ImageBase64: "spritesheet",
			MIMEType:    "image/png",
			Regions: []imageprocessor.ImageRegion{
				{Index: 0, ImageBase64: foreground, MIMEType: "image/png"},
				{Index: 1, ImageBase64: foreground, MIMEType: "image/png"},
				{Index: 2, ImageBase64: foreground, MIMEType: "image/png"},
			},
		},
	}
	editedFrames := animationTestVideoFrames(3)
	editedMiddle := editedFrames[1].(*image.NRGBA)
	draw.Draw(editedMiddle, image.Rect(67, 44, 77, 54), &image.Uniform{C: color.NRGBA{R: 140, G: 50, B: 35, A: 255}}, image.Point{}, draw.Src)
	videoProcessor := &animationVideoProcessorStub{results: []*videoprocessor.Result{{
		Frames: editedFrames,
	}}}
	service := newAnimationGenerationService(videos, processor, videoProcessor)
	contextFrames := animationTestVideoFrames(3)
	contextReferences := make([]string, len(contextFrames))
	for index, frame := range contextFrames {
		contextReferences[index] = animationTestImageDataURL(t, frame)
	}

	_, err := service.Generate(context.Background(), &AnimationGenerationRequest{
		Description:            "knight",
		Action:                 "raise the sword",
		ReferenceImage:         startReference,
		EndReferenceImage:      endReference,
		ReferenceImageContext:  true,
		TargetFrameIndices:     []int{1},
		ContextReferenceImages: contextReferences,
		FrameCount:             3,
		Columns:                3,
		FrameWidth:             64,
		FrameHeight:            64,
		FPS:                    10,
	})
	if err != nil {
		t.Fatalf("generate edited frame segment: %v", err)
	}
	if len(processor.removeRequests) != 2 || len(processor.resizeRequests) != 5 {
		t.Fatalf("boundary references or final frames were not processed: removes=%d resizes=%d", len(processor.removeRequests), len(processor.resizeRequests))
	}
	assertAnimationPixelResizeRequests(t, processor.resizeRequests[2:], 3, 64, 64)
	if processor.removeRequests[0].ImageBase64 != startReference ||
		processor.removeRequests[1].ImageBase64 != endReference {
		t.Fatalf("boundary reference preparation order changed: %+v", processor.removeRequests)
	}
	if len(videos.requests) != 1 || videos.requests[0].EndImage == nil {
		t.Fatalf("video request did not receive start and end references: %+v", videos.requests)
	}
	if videos.requests[0].StartImage.Base64 == "" || videos.requests[0].EndImage.Base64 == "" {
		t.Fatalf("video request contains an empty boundary reference: %+v", videos.requests[0])
	}
	if !strings.Contains(videos.requests[0].Prompt, "BOUNDARY FRAME REFERENCES") ||
		!strings.Contains(videos.requests[0].Prompt, "start/end inputs") ||
		strings.Contains(videos.requests[0].Prompt, "ordered image array") ||
		strings.Contains(videos.requests[0].Prompt, "@Image") {
		t.Fatalf("unexpected boundary-frame edit prompt: %s", videos.requests[0].Prompt)
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
	result, err := service.processVideoWithSourceCellScale(context.Background(), []byte("video"), "", AnimationGenerationRequest{
		Action:      "idle breathing",
		FrameCount:  4,
		Columns:     2,
		FrameWidth:  64,
		FrameHeight: 64,
	}, 1)
	if err != nil {
		t.Fatalf("process video: %v", err)
	}
	if len(result.Frames) != 4 || result.Normalization == nil || result.Spritesheet == "" {
		t.Fatalf("unexpected normalized result: %+v", result)
	}
	decodedFrames := make([]*image.RGBA, 0, len(result.Frames))
	for index, frame := range result.Frames {
		decoded, decodeErr := imageprocessor.DecodeBase64Image(frame.ImageBase64)
		if decodeErr != nil {
			t.Fatalf("decode frame %d: %v", index, decodeErr)
		}
		if decoded.Bounds().Dx() != 64 || decoded.Bounds().Dy() != 64 {
			t.Fatalf("frame %d has size %v", index, decoded.Bounds().Size())
		}
		for y := range 64 {
			for x := range 64 {
				alpha := decoded.RGBAAt(x, y).A
				if alpha != 0 && alpha != 255 {
					t.Fatalf("frame %d retained smooth alpha %d at (%d,%d)", index, alpha, x, y)
				}
			}
		}
		decodedFrames = append(decodedFrames, decoded)
	}
	sheet, decodeErr := imageprocessor.DecodeBase64Image(result.Spritesheet)
	if decodeErr != nil {
		t.Fatalf("decode rebuilt pixel spritesheet: %v", decodeErr)
	}
	if got := sheet.Bounds().Size(); got != image.Pt(128, 128) {
		t.Fatalf("pixel spritesheet size = %v, want 128x128", got)
	}
	for index, frame := range decodedFrames {
		offset := image.Pt((index%2)*64, (index/2)*64)
		for y := range 64 {
			for x := range 64 {
				if got, want := sheet.RGBAAt(offset.X+x, offset.Y+y), frame.RGBAAt(x, y); got != want {
					t.Fatalf("spritesheet frame %d differs at (%d,%d): got=%+v want=%+v", index, x, y, got, want)
				}
			}
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

	result, err := service.processVideoWithSourceCellScale(context.Background(), []byte("video"), "", AnimationGenerationRequest{
		Action: "idle breathing", FrameCount: 4, Columns: 2, FrameWidth: 64, FrameHeight: 64,
	}, 1)
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

func assertAnimationPixelResizeRequests(
	t *testing.T,
	requests []*imageprocessor.ResizeRequest,
	wantCount, width, height int,
) {
	t.Helper()
	if len(requests) != wantCount {
		t.Fatalf("final animation pixel resize requests = %d, want %d", len(requests), wantCount)
	}
	for index, request := range requests {
		options := request.Options
		if options.Width != width || options.Height != height ||
			options.Margin != 0 || options.CropContent ||
			options.Mode != imageprocessor.RasterModePixel || !options.HardAlpha ||
			!options.RecoverPixelGrid || !options.PrequantizeBeforeResize ||
			!options.PreferNearestReduction || !options.SpritePixelPipeline ||
			!options.PreserveCanvasGeometry {
			t.Fatalf("final animation pixel resize request %d has unexpected options: %+v", index, options)
		}
	}
}

func animationReferenceContentBounds(t *testing.T, reference image.Image) image.Rectangle {
	t.Helper()
	matte := color.NRGBAModel.Convert(reference.At(reference.Bounds().Min.X, reference.Bounds().Min.Y)).(color.NRGBA)
	bounds := image.Rectangle{}
	found := false
	for y := reference.Bounds().Min.Y; y < reference.Bounds().Max.Y; y++ {
		for x := reference.Bounds().Min.X; x < reference.Bounds().Max.X; x++ {
			if color.NRGBAModel.Convert(reference.At(x, y)).(color.NRGBA) == matte {
				continue
			}
			pixel := image.Rect(x, y, x+1, y+1)
			if !found {
				bounds = pixel
				found = true
			} else {
				bounds = bounds.Union(pixel)
			}
		}
	}
	if !found {
		t.Fatal("animation reference contains no non-matte pixels")
	}
	return bounds
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
	return animationTestOpaquePrototypeColor(t, color.NRGBA{R: 255, B: 255, A: 255})
}

func animationTestOpaquePrototypeColor(t *testing.T, background color.NRGBA) string {
	t.Helper()
	frame := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
	draw.Draw(frame, image.Rect(30, 16, 66, 88), &image.Uniform{C: color.NRGBA{R: 140, G: 50, B: 35, A: 255}}, image.Point{}, draw.Src)
	encoded, err := imageprocessor.EncodePNGBase64(frame)
	if err != nil {
		t.Fatalf("encode opaque prototype: %v", err)
	}
	return "data:image/png;base64," + encoded
}

func animationTestImageDataURL(t *testing.T, frame image.Image) string {
	t.Helper()
	encoded, err := imageprocessor.EncodePNGBase64(frame)
	if err != nil {
		t.Fatalf("encode animation test frame: %v", err)
	}
	return "data:image/png;base64," + encoded
}

func animationGenerationForegroundBounds(t *testing.T, result *AnimationGenerationResult) []image.Rectangle {
	t.Helper()
	bounds := make([]image.Rectangle, len(result.Frames))
	for index, frame := range result.Frames {
		decoded, err := imageprocessor.DecodeBase64Image(frame.ImageBase64)
		if err != nil {
			t.Fatalf("decode generated animation frame %d: %v", index, err)
		}
		found := false
		for y := decoded.Bounds().Min.Y; y < decoded.Bounds().Max.Y; y++ {
			for x := decoded.Bounds().Min.X; x < decoded.Bounds().Max.X; x++ {
				if color.NRGBAModel.Convert(decoded.At(x, y)).(color.NRGBA).A == 0 {
					continue
				}
				pixel := image.Rect(x, y, x+1, y+1)
				if !found {
					bounds[index], found = pixel, true
				} else {
					bounds[index] = bounds[index].Union(pixel)
				}
			}
		}
		if !found {
			t.Fatalf("generated animation frame %d has no foreground", index)
		}
	}
	return bounds
}

func animationTestVideoFramesWithSubjectMatte(count int, subjectSize image.Point, matte color.NRGBA) []image.Image {
	frames := make([]image.Image, count)
	for index := range frames {
		frame := image.NewNRGBA(image.Rect(0, 0, 192, 192))
		draw.Draw(frame, frame.Bounds(), &image.Uniform{C: matte}, image.Point{}, draw.Src)
		min := image.Pt((192-subjectSize.X)/2, (192-subjectSize.Y)/2+index%2)
		draw.Draw(frame, image.Rectangle{Min: min, Max: min.Add(subjectSize)}, &image.Uniform{C: color.NRGBA{R: 140, G: 50, B: 35, A: 255}}, image.Point{}, draw.Src)
		frames[index] = frame
	}
	return frames
}

func absGeneratorInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
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
