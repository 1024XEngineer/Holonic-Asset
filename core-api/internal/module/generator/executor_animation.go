package generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"math"
	"net/http"
	"slices"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	videoclient "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/video_client"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	videoprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/video"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const (
	defaultAnimationFrameCount  = 16
	defaultAnimationFrameWidth  = 256
	defaultAnimationFrameHeight = 256
	defaultAnimationFPS         = 10
	defaultAnimationResolution  = "720p"
	defaultAnimationDuration    = 5
	defaultAnimationAspectRatio = "1:1"
	animationVideoAttempts      = 2
	animationReferenceSize      = 1024
	maxAnimationReferenceBytes  = 32 << 20
	animationAnalysisFPS        = 12
	animationMinLoopSpanFrames  = 4
	animationMinLoopSpanRatio   = 0.50
	animationMinStartWindow     = 1
	animationInitialWindowRatio = 0.20
	animationMinForegroundRatio = 0.05
	animationEndpointQuantile   = 0.35
	animationRichnessQuantile   = 0.90
	animationMotionQuantile     = 0.25
	animationSeamWarningMSE     = 0.015

	animationEndpointWeight          = 1.0
	animationRichnessWeight          = 0.45
	animationCentroidStabilityWeight = 0.45
	animationTranslationWeight       = 0.20
	animationInitialFrameWeight      = 0.45
	animationLoopCompactnessWeight   = 1.15
	animationPoseCoverageWeight      = 0.65
	animationMotionCoverageWeight    = 0.65
	animationRecoveryWeight          = 0.35

	animationChromaHueMin              = 30
	animationChromaHueMax              = 90
	animationChromaHighSaturationMin   = 80
	animationChromaHighValueMin        = 80
	animationChromaBrightSaturationMin = 50
	animationChromaBrightValueMin      = 180
)

func animationFrameSelectionOptions(frameCount int) videoprocessor.FrameIntervalSelectionOptions {
	return videoprocessor.FrameIntervalSelectionOptions{
		SampleCount:              frameCount,
		MinimumSpanFrames:        animationMinLoopSpanFrames,
		MinimumSpanRatio:         animationMinLoopSpanRatio,
		MinimumStartWindowFrames: animationMinStartWindow,
		StartWindowRatio:         animationInitialWindowRatio,
		PreferFirstFrame:         true,
		MinimumForegroundRatio:   animationMinForegroundRatio,
		EndpointMSEQuantile:      animationEndpointQuantile,
		ChangeScaleQuantile:      animationRichnessQuantile,
		ChangeBaselineQuantile:   animationMotionQuantile,
		Weights: videoprocessor.FrameIntervalSelectionWeights{
			EndpointSimilarity:    animationEndpointWeight,
			MeanAdjacentMSE:       animationRichnessWeight,
			CentroidStability:     animationCentroidStabilityWeight,
			LinearCentroidMotion:  animationTranslationWeight,
			FirstFrameSimilarity:  animationInitialFrameWeight,
			Compactness:           animationLoopCompactnessWeight,
			GeometryCoverage:      animationPoseCoverageWeight,
			ChangeCoverage:        animationMotionCoverageWeight,
			PostIntervalStability: animationRecoveryWeight,
		},
	}
}

type AnimationLoopSelection struct {
	CandidateFPS       int     `json:"candidate_fps"`
	StartFrame         int     `json:"start_frame"`
	EndFrame           int     `json:"end_frame"`
	SpanFrames         int     `json:"span_frames"`
	Score              float64 `json:"score"`
	EndpointSimilarity float64 `json:"endpoint_similarity"`
	Richness           float64 `json:"richness"`
	PoseCoverage       float64 `json:"pose_coverage"`
	SpanRatio          float64 `json:"span_ratio"`
	CentroidStability  float64 `json:"centroid_stability"`
	SeamWarning        string  `json:"seam_warning,omitempty"`
	Method             string  `json:"method"`
}

// AnimationGenerationService turns one asset reference into normalized,
// transparent animation frames. Provider calls stay in videoclient;
// deterministic reference and frame normalization stays in processor/image.
type AnimationGenerationService interface {
	Generate(context.Context, *AnimationGenerationRequest) (*AnimationGenerationResult, error)
}

// AnimationReferenceResolver converts a persisted prototype reference into a
// short-lived URL that the generator can read. The generator only depends on
// this small read boundary; upload/storage credentials stay in the upload
// module.
type AnimationReferenceResolver interface {
	ResolveReference(context.Context, string) (string, error)
}

type AnimationGenerationRequest struct {
	Description    string
	Style          string
	Action         string
	ReferenceImage string
	// ReferenceImagePrepared marks an original high-resolution green-screen
	// asset that does not need image-model redrawing. The executor selects one
	// character or object direction before this service is called.
	ReferenceImagePrepared bool
	// ReferenceImageContext marks an ordered context strip used for local frame edits.
	ReferenceImageContext bool
	// TargetFrameIndices identifies the zero-based output samples that will be
	// replaced when ReferenceImageContext is true.
	TargetFrameIndices []int
	FrameCount         int
	Columns            int
	FrameWidth         int
	FrameHeight        int
	FPS                int
	Resolution         string
	Duration           int
	AspectRatio        string
}

type AnimationGenerationResult struct {
	Frames []imageprocessor.ImageRegion
	// RawFrames contains the sampled video frames before background removal and
	// normalization. They are persisted beside processed frames with the
	// -unprocessed suffix for future frame edits.
	RawFrames       []imageprocessor.ImageRegion
	Spritesheet     string
	MIMEType        string
	Normalization   *imageprocessor.AnimationNormalizationReport
	Loop            AnimationLoopSelection
	VideoRequestID  string
	VideoAttempts   int
	FrameDurationMS uint
}

type animationGenerationService struct {
	videos              videoclient.VideoGenerationService
	processor           imageprocessor.Processor
	videoProcessor      videoprocessor.Processor
	referenceResolver   AnimationReferenceResolver
	referenceHTTPClient *http.Client
}

// NewAnimationGenerationService creates the formal image-to-video animation
// pipeline. ffmpeg is resolved lazily from FFMPEG_PATH or PATH.
func NewAnimationGenerationService(
	videos videoclient.VideoGenerationService,
	processor imageprocessor.Processor,
	resolvers ...AnimationReferenceResolver,
) AnimationGenerationService {
	var resolver AnimationReferenceResolver
	if len(resolvers) > 0 {
		resolver = resolvers[0]
	}
	return newAnimationGenerationServiceWithResolver(
		videos,
		processor,
		videoprocessor.NewProcessor(),
		resolver,
	)
}

func newAnimationGenerationService(
	videos videoclient.VideoGenerationService,
	processor imageprocessor.Processor,
	videoProcessor videoprocessor.Processor,
) AnimationGenerationService {
	return newAnimationGenerationServiceWithResolver(videos, processor, videoProcessor, nil)
}

func newAnimationGenerationServiceWithResolver(
	videos videoclient.VideoGenerationService,
	processor imageprocessor.Processor,
	videoProcessor videoprocessor.Processor,
	resolver AnimationReferenceResolver,
) AnimationGenerationService {
	return &animationGenerationService{
		videos:              videos,
		processor:           processor,
		videoProcessor:      videoProcessor,
		referenceResolver:   resolver,
		referenceHTTPClient: http.DefaultClient,
	}
}

func (s *animationGenerationService) Generate(
	ctx context.Context,
	request *AnimationGenerationRequest,
) (*AnimationGenerationResult, error) {
	if s.videos == nil {
		return nil, ErrVideoServiceRequired
	}
	if s.processor == nil {
		return nil, ErrImageProcessorRequired
	}
	if s.videoProcessor == nil {
		return nil, ErrVideoFrameExtractorRequired
	}
	options, err := normalizeAnimationGenerationRequest(request)
	if err != nil {
		return nil, err
	}
	promptOptions := prompts.AnimationOptions{
		Description:        options.Description,
		Style:              options.Style,
		Action:             options.Action,
		FrameCount:         options.FrameCount,
		ContextSheet:       options.ReferenceImageContext,
		TargetFrameIndices: options.TargetFrameIndices,
	}
	greenReference, err := s.prepareAnimationReference(
		ctx,
		options.ReferenceImage,
		options.ReferenceImagePrepared,
	)
	if err != nil {
		return nil, err
	}

	baseVideoPrompt := prompts.BuildAnimationVideo(promptOptions)
	videoPrompt := baseVideoPrompt
	var lastQualityError error
	for attempt := 1; attempt <= animationVideoAttempts; attempt++ {
		videoResult, generateErr := s.videos.Generate(ctx, &videoclient.GenerateRequest{
			Prompt:                  videoPrompt,
			ReferenceImageBase64:    greenReference,
			ReferenceImageMediaType: "image/png",
			Resolution:              options.Resolution,
			Duration:                options.Duration,
			AspectRatio:             options.AspectRatio,
			GenerateAudio:           false,
		})
		if generateErr != nil {
			return nil, fmt.Errorf("generator: generate animation video: %w", generateErr)
		}
		if videoResult == nil || strings.TrimSpace(videoResult.VideoURL) == "" {
			return nil, fmt.Errorf("generator: generate animation video: empty result")
		}
		video, downloadErr := s.videos.Download(ctx, videoResult.VideoURL)
		if downloadErr != nil {
			return nil, fmt.Errorf("generator: download animation video: %w", downloadErr)
		}
		processed, processErr := s.processVideo(ctx, video, options)
		if processErr == nil {
			processed.VideoRequestID = videoResult.RequestID
			processed.VideoAttempts = attempt
			processed.FrameDurationMS = uint((1000 + options.FPS/2) / options.FPS)
			return processed, nil
		}
		var qualityError *videoprocessor.QualityError
		if !errors.As(processErr, &qualityError) || attempt == animationVideoAttempts {
			return nil, fmt.Errorf("generator: process animation video: %w", processErr)
		}
		lastQualityError = processErr
		videoPrompt = prompts.BuildAnimationVideoRetry(baseVideoPrompt, qualityError.Kind)
	}
	return nil, fmt.Errorf("generator: process animation video: %w", lastQualityError)
}

func normalizeAnimationGenerationRequest(
	request *AnimationGenerationRequest,
) (AnimationGenerationRequest, error) {
	if request == nil {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation generation request is required")
	}
	value := *request
	value.Description = strings.TrimSpace(value.Description)
	value.Style = strings.TrimSpace(value.Style)
	value.Action = strings.TrimSpace(value.Action)
	value.TargetFrameIndices = append([]int(nil), value.TargetFrameIndices...)
	value.ReferenceImage = strings.TrimSpace(value.ReferenceImage)
	value.Resolution = strings.TrimSpace(value.Resolution)
	value.AspectRatio = strings.TrimSpace(value.AspectRatio)
	if value.Description == "" {
		value.Description = "preserve the supplied subject exactly"
	}
	if value.Style == "" {
		value.Style = prompts.DefaultAnimationStyle
	}
	if value.Action == "" {
		value.Action = "idle"
	}
	if value.ReferenceImage == "" {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation reference image is required")
	}
	if value.FrameCount == 0 {
		value.FrameCount = defaultAnimationFrameCount
	}
	if value.Columns == 0 {
		value.Columns = animationGridColumns(value.FrameCount)
	}
	if value.FrameWidth == 0 {
		value.FrameWidth = defaultAnimationFrameWidth
	}
	if value.FrameHeight == 0 {
		value.FrameHeight = defaultAnimationFrameHeight
	}
	if value.FPS == 0 {
		value.FPS = defaultAnimationFPS
	}
	if value.Resolution == "" {
		value.Resolution = defaultAnimationResolution
	}
	if value.Duration == 0 {
		value.Duration = defaultAnimationDuration
	}
	if value.AspectRatio == "" {
		value.AspectRatio = defaultAnimationAspectRatio
	}
	if value.FrameCount < 1 || value.FrameCount > 32 {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation frame count must be between 1 and 32")
	}
	if value.Columns < 1 || value.Columns > 8 {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation columns must be between 1 and 8")
	}
	if animationRows(value.FrameCount, value.Columns) > 8 {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation grid must not exceed 8 rows")
	}
	if value.FrameWidth < 32 || value.FrameWidth > 1024 || value.FrameHeight < 32 || value.FrameHeight > 1024 {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation frame dimensions must be between 32 and 1024 pixels")
	}
	if value.FPS < 1 || value.FPS > 60 {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation FPS must be between 1 and 60")
	}
	if value.Duration < 4 || value.Duration > 15 {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation video duration must be between 4 and 15 seconds")
	}
	if len(value.TargetFrameIndices) > 0 && !value.ReferenceImageContext {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: target frame indices require an animation context reference")
	}
	previousTarget := -1
	for _, target := range value.TargetFrameIndices {
		if target < 0 || target >= value.FrameCount {
			return AnimationGenerationRequest{}, fmt.Errorf("generator: target frame index %d is outside the %d-frame context", target, value.FrameCount)
		}
		if target <= previousTarget {
			return AnimationGenerationRequest{}, fmt.Errorf("generator: target frame indices must be unique and ordered")
		}
		previousTarget = target
	}
	return value, nil
}

// prepareAnimationReference keeps an explicitly prepared green-screen reference
// pixel-identical. Other callers retain the legacy transparent-prototype
// normalization path.
func (s *animationGenerationService) prepareAnimationReference(
	ctx context.Context,
	reference string,
	prepared bool,
) (string, error) {
	reference, err := s.loadAnimationReference(ctx, reference)
	if err != nil {
		return "", err
	}
	if !prepared {
		return s.prepareGreenReference(ctx, reference)
	}
	referenceImage, err := imageprocessor.DecodeBase64Image(reference)
	if err != nil {
		return "", fmt.Errorf("generator: decode prepared animation reference: %w", err)
	}
	encoded, err := imageprocessor.EncodePNGBase64(referenceImage)
	if err != nil {
		return "", fmt.Errorf("generator: encode prepared animation reference: %w", err)
	}
	return encoded, nil
}

func (s *animationGenerationService) loadAnimationReference(ctx context.Context, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", fmt.Errorf("generator: animation reference image is required")
	}

	if s.referenceResolver != nil {
		resolved, err := s.referenceResolver.ResolveReference(ctx, reference)
		if err != nil {
			return "", fmt.Errorf("generator: resolve animation reference: %w", err)
		}
		reference = strings.TrimSpace(resolved)
		if reference == "" {
			return "", fmt.Errorf("generator: resolve animation reference: empty result")
		}
	}

	if !strings.HasPrefix(reference, "http://") && !strings.HasPrefix(reference, "https://") {
		return reference, nil
	}

	client := s.referenceHTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, reference, nil)
	if err != nil {
		return "", fmt.Errorf("generator: create animation reference download request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("generator: download animation reference: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return "", fmt.Errorf("generator: download animation reference: HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxAnimationReferenceBytes+1))
	if err != nil {
		return "", fmt.Errorf("generator: read animation reference: %w", err)
	}
	if len(body) > maxAnimationReferenceBytes {
		return "", fmt.Errorf("generator: animation reference exceeds %d bytes", maxAnimationReferenceBytes)
	}
	if len(body) == 0 {
		return "", fmt.Errorf("generator: download animation reference: empty response")
	}
	encoded := base64.StdEncoding.EncodeToString(body)
	decoded, err := imageprocessor.DecodeBase64Image(encoded)
	if err != nil {
		return "", fmt.Errorf("generator: decode downloaded animation reference: %w", err)
	}
	canonical, err := imageprocessor.EncodePNGBase64(decoded)
	if err != nil {
		return "", fmt.Errorf("generator: encode downloaded animation reference: %w", err)
	}
	return canonical, nil
}

func (s *animationGenerationService) prepareGreenReference(
	ctx context.Context,
	referenceBase64 string,
) (string, error) {
	foregroundBase64 := referenceBase64
	reference, err := imageprocessor.DecodeBase64Image(referenceBase64)
	if err != nil {
		return "", fmt.Errorf("generator: decode animation reference: %w", err)
	}
	if !animationImageHasTransparency(reference) {
		removed, removeErr := s.processor.RemoveBackground(ctx, &imageprocessor.RemoveBackgroundRequest{
			ImageBase64: referenceBase64,
			MatteColor:  "auto",
		})
		if removeErr != nil {
			return "", fmt.Errorf("generator: remove animation reference background: %w", removeErr)
		}
		if removed == nil || strings.TrimSpace(removed.ImageBase64) == "" {
			return "", fmt.Errorf("generator: remove animation reference background: empty result")
		}
		foregroundBase64 = removed.ImageBase64
	}

	// Use the same padded canonical-frame layout as prototype sprites. The
	// animation reference must not be made smaller by an extra safety margin;
	// the margin is part of the shared prototype/animation canvas contract.
	resizeOptions := imageprocessor.AnimationFrameResizeOptions(animationReferenceSize, animationReferenceSize)
	resized, err := s.processor.Resize(ctx, &imageprocessor.ResizeRequest{
		ImageBase64: foregroundBase64,
		Options:     resizeOptions,
	})
	if err != nil {
		return "", fmt.Errorf("generator: normalize animation reference: %w", err)
	}
	if resized == nil || strings.TrimSpace(resized.ImageBase64) == "" {
		return "", fmt.Errorf("generator: normalize animation reference: empty result")
	}
	foreground, err := imageprocessor.DecodeBase64Image(resized.ImageBase64)
	if err != nil {
		return "", fmt.Errorf("generator: decode normalized animation reference: %w", err)
	}
	canvas := image.NewNRGBA(image.Rect(0, 0, foreground.Bounds().Dx(), foreground.Bounds().Dy()))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), foreground, foreground.Bounds().Min, draw.Over)
	encoded, err := imageprocessor.EncodePNGBase64(canvas)
	if err != nil {
		return "", fmt.Errorf("generator: encode green animation reference: %w", err)
	}
	return encoded, nil
}

func animationImageHasTransparency(source image.Image) bool {
	for y := source.Bounds().Min.Y; y < source.Bounds().Max.Y; y++ {
		for x := source.Bounds().Min.X; x < source.Bounds().Max.X; x++ {
			_, _, _, alpha := source.At(x, y).RGBA()
			if alpha < 0xffff {
				return true
			}
		}
	}
	return false
}

func (s *animationGenerationService) processVideo(
	ctx context.Context,
	video []byte,
	request AnimationGenerationRequest,
) (*AnimationGenerationResult, error) {
	var loop AnimationLoopSelection
	processed, err := s.videoProcessor.Process(ctx, video, videoprocessor.ProcessOptions{
		AnalysisFPS: animationAnalysisFPS,
		ChromaKey: videoprocessor.ChromaKey{
			HueMin: animationChromaHueMin, HueMax: animationChromaHueMax,
			HighSaturationMin: animationChromaHighSaturationMin, HighValueMin: animationChromaHighValueMin,
			BrightSaturationMin: animationChromaBrightSaturationMin, BrightValueMin: animationChromaBrightValueMin,
		},
		Select: func(analysis videoprocessor.FrameSequenceAnalysis) ([]int, error) {
			if request.ReferenceImageContext {
				indices, selectErr := selectEditFrameContextIndices(analysis, request.FrameCount)
				if selectErr != nil {
					return nil, selectErr
				}
				loop = AnimationLoopSelection{
					CandidateFPS: analysis.FPS, StartFrame: indices[0], EndFrame: indices[len(indices)-1],
					SpanFrames: indices[len(indices)-1] - indices[0], Method: "ordered_context_segment",
				}
				return indices, nil
			}
			selected, selectErr := videoprocessor.SelectFrameInterval(
				analysis,
				animationFrameSelectionOptions(request.FrameCount),
			)
			if selectErr != nil {
				return nil, animationFrameSelectionError(selectErr, analysis, request.FrameCount)
			}
			loop = animationLoopSelection(selected, analysis.FPS)
			return selected.Indices, nil
		},
	})
	if err != nil {
		return nil, err
	}
	rawFrames := make([]imageprocessor.ImageRegion, 0, len(processed.Frames))
	for index, frame := range processed.Frames {
		encodedFrame, encodeErr := imageprocessor.EncodePNGBase64(frame)
		if encodeErr != nil {
			return nil, fmt.Errorf("generator: encode raw animation frame %d: %w", index+1, encodeErr)
		}
		rawFrames = append(rawFrames, imageprocessor.ImageRegion{
			Index: index, ImageBase64: encodedFrame, MIMEType: "image/png",
		})
	}
	rawSheet, err := packAnimationVideoFrames(processed.Frames, request.Columns)
	if err != nil {
		return nil, err
	}
	encoded, err := imageprocessor.EncodePNGBase64(rawSheet)
	if err != nil {
		return nil, fmt.Errorf("generator: encode sampled animation sheet: %w", err)
	}
	normalized, err := s.processor.SplitImage(ctx, &imageprocessor.SplitImageRequest{
		ImageBase64:             encoded,
		Mode:                    imageprocessor.ImageSplitModeAnimation,
		Columns:                 request.Columns,
		Rows:                    animationRows(request.FrameCount, request.Columns),
		FrameCount:              request.FrameCount,
		FrameWidth:              request.FrameWidth,
		FrameHeight:             request.FrameHeight,
		Margin:                  imageprocessor.AnimationFrameMargin(request.FrameWidth, request.FrameHeight),
		Anchor:                  imageprocessor.AnimationAnchorFeet,
		ForceProportionalGrid:   true,
		PreserveVerticalMotion:  true,
		PreserveSourceCellScale: true,
		// The video prompt asks for a matte, but providers can return a
		// different (or compressed) background colour. Sample the frame
		// borders instead of assuming an exact pure-green key; otherwise
		// normalization may receive an entirely opaque sheet and cannot
		// compute the character bounds.
		Background: &imageprocessor.AnimationBackgroundOptions{
			MatteColor: "auto",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("generator: normalize sampled animation frames: %w", err)
	}
	if normalized == nil || len(normalized.Regions) != request.FrameCount || strings.TrimSpace(normalized.ImageBase64) == "" {
		return nil, fmt.Errorf("generator: normalize sampled animation frames: empty or incomplete result")
	}
	return &AnimationGenerationResult{
		Frames: normalized.Regions, RawFrames: rawFrames, Spritesheet: normalized.ImageBase64,
		MIMEType: normalized.MIMEType, Normalization: normalized.AnimationReport,
		Loop: loop,
	}, nil
}

func selectEditFrameContextIndices(analysis videoprocessor.FrameSequenceAnalysis, frameCount int) ([]int, error) {
	if frameCount < 1 {
		return nil, fmt.Errorf("generator: edit frame count must be positive")
	}
	if analysis.ForegroundRatio < animationMinForegroundRatio {
		return nil, &videoprocessor.QualityError{
			Kind: "foreground", Message: fmt.Sprintf("generator: chroma-key separation failed: foreground ratio %.3f", analysis.ForegroundRatio),
		}
	}
	if len(analysis.Frames) < frameCount {
		return nil, fmt.Errorf("generator: video has %d candidate frames; need at least %d", len(analysis.Frames), frameCount)
	}
	safeIndices := make([]int, 0, len(analysis.Frames))
	for index, frame := range analysis.Frames {
		if frame.Safe {
			safeIndices = append(safeIndices, index)
		}
	}
	if len(safeIndices) < frameCount {
		return nil, &videoprocessor.QualityError{
			Kind: "framing", Message: fmt.Sprintf("generator: edit context video has %d safe candidate frames; need at least %d", len(safeIndices), frameCount),
		}
	}
	indices := make([]int, frameCount)
	if frameCount == 1 {
		indices[0] = safeIndices[(len(safeIndices)-1)/2]
	} else {
		last := len(safeIndices) - 1
		for index := range frameCount {
			indices[index] = safeIndices[(index*last+(frameCount-1)/2)/(frameCount-1)]
		}
	}
	return indices, nil
}

func animationFrameSelectionError(
	err error,
	analysis videoprocessor.FrameSequenceAnalysis,
	frameCount int,
) error {
	var qualityError *videoprocessor.QualityError
	if errors.As(err, &qualityError) {
		switch qualityError.Kind {
		case "foreground":
			return &videoprocessor.QualityError{
				Kind: "foreground", Message: fmt.Sprintf("generator: chroma-key separation failed: foreground ratio %.3f", analysis.ForegroundRatio),
			}
		case "framing":
			return &videoprocessor.QualityError{
				Kind:    "framing",
				Message: fmt.Sprintf("generator: no candidate interval has %d sampled frames inside the outer 2.5%% safety band; interval still needs at least %.0f%% of the source duration", frameCount, animationMinLoopSpanRatio*100),
			}
		}
	}
	if frameCount <= 0 || len(analysis.Frames) < frameCount+1 {
		return fmt.Errorf("generator: video has %d candidate frames; need at least %d", len(analysis.Frames), frameCount+1)
	}
	return fmt.Errorf("generator: loop search produced no candidate: %w", err)
}

func animationLoopSelection(
	selected videoprocessor.FrameIntervalSelection,
	fps int,
) AnimationLoopSelection {
	warning := ""
	if selected.EndpointMSE > animationSeamWarningMSE {
		warning = fmt.Sprintf("foreground seam MSE %.4f exceeds 0.015; inspect the source video", selected.EndpointMSE)
	}
	return AnimationLoopSelection{
		CandidateFPS: fps, StartFrame: selected.StartFrame, EndFrame: selected.EndFrame,
		SpanFrames: selected.EndFrame - selected.StartFrame, Score: roundAnimationValue(selected.Score),
		EndpointSimilarity: roundAnimationValue(selected.EndpointSimilarity),
		Richness:           roundAnimationValue(selected.MeanAdjacentMSE),
		PoseCoverage:       roundAnimationValue(selected.GeometryCoverage),
		SpanRatio:          roundAnimationValue(selected.SpanRatio),
		CentroidStability:  roundAnimationValue(selected.CentroidStability),
		SeamWarning:        warning, Method: "subject_mse_full_cycle",
	}
}

func roundAnimationValue(value float64) float64 {
	return math.Round(value*1e6) / 1e6
}

func packAnimationVideoFrames(frames []image.Image, columns int) (*image.NRGBA, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("generator: sampled animation frames are required")
	}
	if columns <= 0 {
		return nil, fmt.Errorf("generator: animation columns must be positive")
	}
	bounds := frames[0].Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("generator: sampled animation frame dimensions must be positive")
	}
	rows := animationRows(len(frames), columns)
	sheet := image.NewNRGBA(image.Rect(0, 0, width*columns, height*rows))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	for index, frame := range frames {
		if frame.Bounds().Dx() != width || frame.Bounds().Dy() != height {
			return nil, fmt.Errorf("generator: sampled animation frame %d dimensions differ", index+1)
		}
		column, row := index%columns, index/columns
		destination := image.Rect(column*width, row*height, (column+1)*width, (row+1)*height)
		draw.Draw(sheet, destination, frame, frame.Bounds().Min, draw.Src)
	}
	return sheet, nil
}

func animationRows(frameCount, columns int) int {
	return (frameCount + columns - 1) / columns
}

var _ AnimationGenerationService = (*animationGenerationService)(nil)

var animationDirectionLayouts = map[uint][]string{
	2: {
		AnimationDirectionLeft,
		AnimationDirectionRight,
	},
	4: {
		AnimationDirectionFront,
		AnimationDirectionRight,
		AnimationDirectionBack,
		AnimationDirectionLeft,
	},
	8: {
		AnimationDirectionFront,
		AnimationDirectionFrontRight,
		AnimationDirectionRight,
		AnimationDirectionBackRight,
		AnimationDirectionBack,
		AnimationDirectionBackLeft,
		AnimationDirectionLeft,
		AnimationDirectionFrontLeft,
	},
}

func animationDirectionIndex(direction string, directionCount uint) (int, error) {
	if directionCount > 8 {
		return 0, fmt.Errorf("generator: animation supports at most 8 directions, asset has %d", directionCount)
	}
	layout, ok := animationDirectionLayouts[directionCount]
	if !ok {
		return 0, fmt.Errorf("generator: animation asset direction count must be one of 2, 4, or 8, got %d", directionCount)
	}

	direction = strings.ToLower(strings.TrimSpace(direction))
	if direction == "" {
		return 0, fmt.Errorf("generator: animation direction is required; available directions: %s", strings.Join(layout, ", "))
	}
	index := slices.Index(layout, direction)
	if index < 0 {
		return 0, fmt.Errorf(
			"generator: animation direction %q is unavailable for an asset with %d directions; available directions: %s",
			direction,
			directionCount,
			strings.Join(layout, ", "),
		)
	}
	return index, nil
}

func animationFrameDimensions(asset assetdomain.Asset) (assetdomain.Size, error) {
	var dimensions assetdomain.Size
	if err := json.Unmarshal(asset.Dimensions, &dimensions); err != nil {
		return assetdomain.Size{}, fmt.Errorf(
			"generator: decode animation asset %d dimensions: %w",
			asset.ID,
			err,
		)
	}
	if dimensions.Width == 0 || dimensions.Height == 0 {
		return assetdomain.Size{}, fmt.Errorf(
			"generator: animation asset %d dimensions must be positive",
			asset.ID,
		)
	}
	return dimensions, nil
}

func (e *executor) generateAnimation(
	ctx context.Context,
	payload CreateAnimationPayload,
) (json.RawMessage, error) {
	if payload.AssetID == 0 {
		return nil, fmt.Errorf("generator: animation asset is required")
	}
	animationName := strings.TrimSpace(payload.AnimationName)
	if animationName == "" {
		return nil, fmt.Errorf("generator: animation name is required")
	}
	asset, err := e.assets.GetDetail(ctx, payload.AssetID)
	if err != nil {
		return nil, fmt.Errorf("generator: get animation asset %d: %w", payload.AssetID, err)
	}
	if asset.ID == 0 {
		return nil, fmt.Errorf("generator: animation asset %d not found", payload.AssetID)
	}
	if payload.ProjectID != 0 && asset.ProjectID != payload.ProjectID {
		return nil, fmt.Errorf(
			"generator: animation asset %d belongs to project %d, not project %d",
			payload.AssetID,
			asset.ProjectID,
			payload.ProjectID,
		)
	}
	reference, referencePrepared, err := animationReference(asset, payload.Direction)
	if err != nil {
		return nil, err
	}
	dimensions, err := animationFrameDimensions(asset)
	if err != nil {
		return nil, err
	}
	description := strings.TrimSpace(asset.Description)
	if description == "" {
		description = strings.TrimSpace(asset.Name)
	}
	generationRequest, err := normalizeAnimationGenerationRequest(&AnimationGenerationRequest{
		Description:            description,
		Style:                  payload.Style,
		Action:                 payload.CreativeBrief,
		ReferenceImage:         reference,
		ReferenceImagePrepared: referencePrepared,
		FrameCount:             payload.FrameCount,
		FrameWidth:             int(dimensions.Width),
		FrameHeight:            int(dimensions.Height),
		FPS:                    payload.FPS,
		Resolution:             payload.Resolution,
		Duration:               payload.Duration,
		AspectRatio:            defaultAnimationAspectRatio,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: normalize animation request: %w", err)
	}
	generated, err := e.animations.Generate(ctx, &generationRequest)
	if err != nil {
		return nil, fmt.Errorf("generator: generate animation frames: %w", err)
	}
	if generated == nil || len(generated.Frames) == 0 {
		return nil, fmt.Errorf("generator: generate animation frames: empty result")
	}
	frames, err := e.persistAnimationFrames(ctx, generated)
	if err != nil {
		return nil, err
	}
	animationID, err := e.assets.CreateAnimation(ctx, payload.AssetID, assetdomain.Animation{
		Name:   animationName,
		Frames: frames,
		Generation: &assetdomain.AnimationGenerationConfig{
			Direction:   strings.ToLower(strings.TrimSpace(payload.Direction)),
			Style:       generationRequest.Style,
			FrameCount:  generationRequest.FrameCount,
			Columns:     generationRequest.Columns,
			FrameWidth:  generationRequest.FrameWidth,
			FrameHeight: generationRequest.FrameHeight,
			FPS:         generationRequest.FPS,
			Resolution:  generationRequest.Resolution,
			Duration:    generationRequest.Duration,
			AspectRatio: generationRequest.AspectRatio,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("generator: create animation for asset %d: %w", payload.AssetID, err)
	}
	if animationID == 0 {
		return nil, fmt.Errorf("generator: create animation for asset %d: empty result", payload.AssetID)
	}
	return encodeExecutionResult(ExecutionResult{AssetID: payload.AssetID, AnimationID: animationID})
}

func (e *executor) editAnimation(
	ctx context.Context,
	payload EditAnimationPayload,
) (json.RawMessage, error) {
	if payload.AssetID == 0 {
		return nil, fmt.Errorf("generator: animation asset is required")
	}
	if payload.AnimationID == 0 {
		return nil, fmt.Errorf("generator: animation id is required")
	}
	creativeBrief := strings.TrimSpace(payload.CreativeBrief)
	if creativeBrief == "" {
		return nil, fmt.Errorf("generator: animation creative brief is required")
	}

	asset, err := e.assets.GetDetail(ctx, payload.AssetID)
	if err != nil {
		return nil, fmt.Errorf("generator: get animation asset %d: %w", payload.AssetID, err)
	}
	if asset.ID == 0 {
		return nil, fmt.Errorf("generator: animation asset %d not found", payload.AssetID)
	}
	if payload.ProjectID != 0 && asset.ProjectID != payload.ProjectID {
		return nil, fmt.Errorf(
			"generator: animation asset %d belongs to project %d, not project %d",
			payload.AssetID,
			asset.ProjectID,
			payload.ProjectID,
		)
	}

	content, err := asset.DecodeContent()
	if err != nil {
		return nil, fmt.Errorf("generator: decode animation asset %d content: %w", payload.AssetID, err)
	}
	var animation *assetdomain.Animation
	for index := range content.Animations {
		if content.Animations[index].ID == payload.AnimationID {
			animation = &content.Animations[index]
			break
		}
	}
	if animation == nil {
		return nil, fmt.Errorf(
			"generator: animation %d not found in asset %d",
			payload.AnimationID,
			payload.AssetID,
		)
	}
	if animation.Generation == nil {
		return nil, fmt.Errorf(
			"generator: animation %d in asset %d has no generation configuration",
			payload.AnimationID,
			payload.AssetID,
		)
	}

	generation := *animation.Generation
	reference, _, err := animationReference(asset, generation.Direction)
	if err != nil {
		return nil, err
	}
	description := strings.TrimSpace(asset.Description)
	if description == "" {
		description = strings.TrimSpace(asset.Name)
	}
	generationRequest := AnimationGenerationRequest{
		Description:            description,
		Style:                  generation.Style,
		Action:                 creativeBrief,
		ReferenceImage:         reference,
		ReferenceImagePrepared: false,
		FrameCount:             generation.FrameCount,
		Columns:                generation.Columns,
		FrameWidth:             generation.FrameWidth,
		FrameHeight:            generation.FrameHeight,
		FPS:                    generation.FPS,
		Resolution:             generation.Resolution,
		Duration:               generation.Duration,
		AspectRatio:            generation.AspectRatio,
	}
	generated, err := e.animations.Generate(ctx, &generationRequest)
	if err != nil {
		return nil, fmt.Errorf("generator: regenerate animation frames: %w", err)
	}
	if generated == nil || len(generated.Frames) == 0 {
		return nil, fmt.Errorf("generator: regenerate animation frames: empty result")
	}
	frames, err := e.persistAnimationFrames(ctx, generated)
	if err != nil {
		return nil, err
	}
	if err := e.assets.UpdateAnimationFrames(
		ctx,
		payload.AssetID,
		payload.AnimationID,
		frames,
	); err != nil {
		return nil, fmt.Errorf(
			"generator: update animation %d frames for asset %d: %w",
			payload.AnimationID,
			payload.AssetID,
			err,
		)
	}
	return encodeExecutionResult(ExecutionResult{
		AssetID:     payload.AssetID,
		AnimationID: payload.AnimationID,
	})
}
