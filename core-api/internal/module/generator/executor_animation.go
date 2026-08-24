package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	videoclient "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/video_client"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	videoprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/video"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const (
	defaultAnimationFrameCount          = 16
	defaultAnimationFrameWidth          = 256
	defaultAnimationFrameHeight         = 256
	defaultAnimationFPS                 = 10
	defaultAnimationResolution          = "720p"
	defaultAnimationDuration            = 5
	defaultAnimationAspectRatio         = "1:1"
	animationVideoAttempts              = 2
	animationReferenceSize              = 1024
	maxAnimationReferenceBytes          = 32 << 20
	defaultAnimationReferenceMaxRetries = 3
	defaultAnimationReferenceTimeout    = 45 * time.Second
	defaultAnimationReferenceRetryDelay = 2 * time.Second
	animationAnalysisFPS                = 12
	animationMinLoopSpanFrames          = 4
	animationMinLoopSpanRatio           = 0.50
	animationMinStartWindow             = 1
	animationInitialWindowRatio         = 0.20
	animationMinForegroundRatio         = 0.05
	animationEndpointQuantile           = 0.35
	animationRichnessQuantile           = 0.90
	animationMotionQuantile             = 0.25
	animationSeamWarningMSE             = 0.015

	animationEndpointWeight          = 1.0
	animationRichnessWeight          = 0.45
	animationCentroidStabilityWeight = 0.45
	animationTranslationWeight       = 0.20
	animationInitialFrameWeight      = 0.45
	animationLoopCompactnessWeight   = 1.15
	animationPoseCoverageWeight      = 0.65
	animationMotionCoverageWeight    = 0.65
	animationRecoveryWeight          = 0.35

	// Green remains the default animation matte. AutoDetect allows generated
	// videos to use an alternate matte when the subject contains green.
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
	Description       string
	Style             string
	Action            string
	OriginalAction    string
	ReferenceImage    string
	EndReferenceImage string
	// ReferenceImagePrepared marks an original high-resolution green-screen
	// asset that does not need image-model redrawing. The executor selects one
	// character or object direction before this service is called.
	ReferenceImagePrepared bool
	// ReferenceImageContext marks a local frame edit whose generated samples are
	// mapped back to selected frames. ReferenceImage and EndReferenceImage are
	// the original boundary frames bracketing the selected frame interval.
	ReferenceImageContext bool
	// TargetFrameIndices identifies the zero-based output samples that will be
	// replaced when ReferenceImageContext is true.
	TargetFrameIndices []int
	// ContextReferenceImages contains the original unprocessed frames for the
	// complete local-edit interval. The continuity gate temporarily replaces the
	// target positions in this sequence before accepting generated output.
	ContextReferenceImages []string
	FrameCount             int
	Columns                int
	FrameWidth             int
	FrameHeight            int
	FPS                    int
	Resolution             string
	Duration               int
	AspectRatio            string

	continuityReferenceFrames []image.Image
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
	referenceMaxRetries int
	referenceTimeout    time.Duration
	referenceRetryDelay time.Duration
	logger              logger.Logger
}

// AnimationGenerationDependencies configures infrastructure used by the
// animation pipeline outside the video provider itself.
type AnimationGenerationDependencies struct {
	ReferenceResolver   AnimationReferenceResolver
	ReferenceHTTPClient *http.Client
	Logger              logger.Logger
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
	return NewAnimationGenerationServiceWithDependencies(
		videos,
		processor,
		AnimationGenerationDependencies{ReferenceResolver: resolver},
	)
}

// NewAnimationGenerationServiceWithDependencies creates the animation pipeline
// with explicit logging and reference-download infrastructure.
func NewAnimationGenerationServiceWithDependencies(
	videos videoclient.VideoGenerationService,
	processor imageprocessor.Processor,
	dependencies AnimationGenerationDependencies,
) AnimationGenerationService {
	client := dependencies.ReferenceHTTPClient
	if client == nil {
		client = newDefaultAnimationReferenceHTTPClient()
	}
	return &animationGenerationService{
		videos:              videos,
		processor:           processor,
		videoProcessor:      videoprocessor.NewProcessor(),
		referenceResolver:   dependencies.ReferenceResolver,
		referenceHTTPClient: client,
		referenceMaxRetries: defaultAnimationReferenceMaxRetries,
		referenceTimeout:    defaultAnimationReferenceTimeout,
		referenceRetryDelay: defaultAnimationReferenceRetryDelay,
		logger:              dependencies.Logger,
	}
}

func newDefaultAnimationReferenceHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSHandshakeTimeout = defaultAnimationReferenceTimeout
	return &http.Client{
		Transport: transport,
		Timeout:   defaultAnimationReferenceTimeout,
	}
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
		referenceHTTPClient: newDefaultAnimationReferenceHTTPClient(),
		referenceMaxRetries: defaultAnimationReferenceMaxRetries,
		referenceTimeout:    defaultAnimationReferenceTimeout,
		referenceRetryDelay: defaultAnimationReferenceRetryDelay,
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
	if options.ReferenceImageContext {
		options.continuityReferenceFrames, err = s.loadAnimationContextFrames(ctx, options.ContextReferenceImages)
		if err != nil {
			return nil, err
		}
	}
	promptOptions := prompts.AnimationOptions{
		Description:        options.Description,
		Style:              options.Style,
		Action:             options.Action,
		OriginalAction:     options.OriginalAction,
		FrameCount:         options.FrameCount,
		LocalFrameEdit:     options.ReferenceImageContext,
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
	var endReference *videoclient.ReferenceImage
	if options.EndReferenceImage != "" {
		greenEndReference, prepareErr := s.prepareAnimationReference(
			ctx,
			options.EndReferenceImage,
			options.ReferenceImagePrepared,
		)
		if prepareErr != nil {
			return nil, fmt.Errorf("generator: prepare end animation reference: %w", prepareErr)
		}
		endReference = &videoclient.ReferenceImage{Base64: greenEndReference, MediaType: "image/png"}
	}

	baseVideoPrompt := prompts.BuildAnimationVideo(promptOptions)
	videoPrompt := baseVideoPrompt
	var lastQualityError error
	for attempt := 1; attempt <= animationVideoAttempts; attempt++ {
		videoResult, generateErr := s.videos.Generate(ctx, &videoclient.GenerateRequest{
			Prompt:        videoPrompt,
			StartImage:    videoclient.ReferenceImage{Base64: greenReference, MediaType: "image/png"},
			EndImage:      endReference,
			Resolution:    options.Resolution,
			Duration:      options.Duration,
			AspectRatio:   options.AspectRatio,
			GenerateAudio: false,
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
	value.OriginalAction = strings.TrimSpace(value.OriginalAction)
	value.TargetFrameIndices = append([]int(nil), value.TargetFrameIndices...)
	value.ContextReferenceImages = append([]string(nil), value.ContextReferenceImages...)
	for index := range value.ContextReferenceImages {
		value.ContextReferenceImages[index] = strings.TrimSpace(value.ContextReferenceImages[index])
	}
	value.ReferenceImage = strings.TrimSpace(value.ReferenceImage)
	value.EndReferenceImage = strings.TrimSpace(value.EndReferenceImage)
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
	if value.ReferenceImageContext && value.EndReferenceImage == "" {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation frame edit end reference image is required")
	}
	if value.ReferenceImageContext && len(value.ContextReferenceImages) != value.FrameCount {
		return AnimationGenerationRequest{}, fmt.Errorf(
			"generator: animation frame edit requires %d context reference images; got %d",
			value.FrameCount,
			len(value.ContextReferenceImages),
		)
	}
	for index, reference := range value.ContextReferenceImages {
		if reference == "" {
			return AnimationGenerationRequest{}, fmt.Errorf("generator: animation frame edit context reference image %d is required", index+1)
		}
	}
	return value, nil
}

func (s *animationGenerationService) processVideo(
	ctx context.Context,
	video []byte,
	request AnimationGenerationRequest,
) (*AnimationGenerationResult, error) {
	var loop AnimationLoopSelection
	processed, err := s.videoProcessor.Process(ctx, video, videoprocessor.ProcessOptions{
		AnalysisFPS: animationAnalysisFPS,
		ChromaKey:   animationVideoChromaKey(),
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
	if request.ReferenceImageContext {
		if err := validateEditFrameContinuity(request, processed.Frames); err != nil {
			return nil, err
		}
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
			MatteColor:          "auto",
			BorderConnectedOnly: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("generator: normalize sampled animation frames: %w", err)
	}
	if normalized == nil || len(normalized.Regions) != request.FrameCount || strings.TrimSpace(normalized.ImageBase64) == "" {
		return nil, fmt.Errorf("generator: normalize sampled animation frames: empty or incomplete result")
	}
	pixelFrames, pixelSpritesheet, err := s.pixelProcessAnimationFrames(
		ctx,
		normalized.Regions,
		request.Columns,
		request.FrameWidth,
		request.FrameHeight,
	)
	if err != nil {
		return nil, err
	}
	return &AnimationGenerationResult{
		Frames: pixelFrames, RawFrames: rawFrames, Spritesheet: pixelSpritesheet,
		MIMEType: "image/png", Normalization: normalized.AnimationReport,
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
	bestStart, bestEnd := -1, -1
	for start := 0; start < len(analysis.Frames); {
		for start < len(analysis.Frames) && !analysis.Frames[start].Safe {
			start++
		}
		end := start
		for end < len(analysis.Frames) && analysis.Frames[end].Safe {
			end++
		}
		if end-start > bestEnd-bestStart {
			bestStart, bestEnd = start, end
		}
		start = max(end, start+1)
	}
	if bestEnd-bestStart < frameCount {
		return nil, &videoprocessor.QualityError{
			Kind: "framing", Message: fmt.Sprintf("generator: edit context video has no continuous safe interval with %d frames", frameCount),
		}
	}
	indices := make([]int, frameCount)
	if frameCount == 1 {
		indices[0] = bestStart + (bestEnd-bestStart-1)/2
	} else {
		last := bestEnd - bestStart - 1
		for index := range frameCount {
			indices[index] = bestStart + (index*last+(frameCount-1)/2)/(frameCount-1)
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

type generatedAnimationCandidate struct {
	Name       string                                 `json:"name"`
	Frames     []assetdomain.Frame                    `json:"frames"`
	Generation *assetdomain.AnimationGenerationConfig `json:"generation,omitempty"`
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
	animation := generatedAnimationCandidate{
		Name:   animationName,
		Frames: frames,
		Generation: &assetdomain.AnimationGenerationConfig{
			Direction:   strings.ToLower(strings.TrimSpace(payload.Direction)),
			Style:       generationRequest.Style,
			Action:      generationRequest.Action,
			FrameCount:  generationRequest.FrameCount,
			Columns:     generationRequest.Columns,
			FrameWidth:  generationRequest.FrameWidth,
			FrameHeight: generationRequest.FrameHeight,
			FPS:         generationRequest.FPS,
			Resolution:  generationRequest.Resolution,
			Duration:    generationRequest.Duration,
			AspectRatio: generationRequest.AspectRatio,
		},
	}
	encoded, err := json.Marshal(struct {
		Animations []generatedAnimationCandidate `json:"animations"`
	}{
		Animations: []generatedAnimationCandidate{animation},
	})
	if err != nil {
		return nil, fmt.Errorf("generator: encode animation result for asset %d: %w", payload.AssetID, err)
	}
	return encodeExecutionResult(ExecutionResult{
		AssetID:            payload.AssetID,
		Version:            asset.Version,
		Content:            encoded,
		GeneratedResources: generatedFrameResourceKeys(frames),
	})
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
	animation.Frames = append([]assetdomain.Frame(nil), frames...)
	encoded, err := assetdomain.EncodeContent(assetdomain.AssetContent{
		Animations: []assetdomain.Animation{*animation},
	})
	if err != nil {
		return nil, fmt.Errorf("generator: encode animation %d result for asset %d: %w", payload.AnimationID, payload.AssetID, err)
	}
	return encodeExecutionResult(ExecutionResult{
		AssetID:            payload.AssetID,
		AnimationID:        payload.AnimationID,
		Version:            asset.Version,
		Content:            encoded,
		GeneratedResources: generatedFrameResourceKeys(frames),
	})
}
