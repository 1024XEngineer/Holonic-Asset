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
	"net/http"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	videoclient "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/video_client"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
)

const (
	defaultAnimationFrameCount  = 16
	defaultAnimationColumns     = 4
	defaultAnimationFrameWidth  = 256
	defaultAnimationFrameHeight = 256
	defaultAnimationFPS         = 10
	defaultAnimationResolution  = "720p"
	defaultAnimationDuration    = 5
	defaultAnimationAspectRatio = "1:1"
	animationVideoAttempts      = 2
	animationReferenceSize      = 1024
	maxAnimationReferenceBytes  = 32 << 20
)

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
	// direction before this service is called.
	ReferenceImagePrepared bool
	FrameCount             int
	Columns                int
	FrameWidth             int
	FrameHeight            int
	FPS                    int
	Resolution             string
	Duration               int
	AspectRatio            string
}

type AnimationGenerationResult struct {
	Frames          []imageprocessor.ImageRegion
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
	extractor           animationFrameExtractor
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
		ffmpegAnimationFrameExtractor{},
		resolver,
	)
}

func newAnimationGenerationService(
	videos videoclient.VideoGenerationService,
	processor imageprocessor.Processor,
	extractor animationFrameExtractor,
) AnimationGenerationService {
	return newAnimationGenerationServiceWithResolver(videos, processor, extractor, nil)
}

func newAnimationGenerationServiceWithResolver(
	videos videoclient.VideoGenerationService,
	processor imageprocessor.Processor,
	extractor animationFrameExtractor,
	resolver AnimationReferenceResolver,
) AnimationGenerationService {
	return &animationGenerationService{
		videos:              videos,
		processor:           processor,
		extractor:           extractor,
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
	if s.extractor == nil {
		return nil, ErrVideoFrameExtractorRequired
	}
	options, err := normalizeAnimationGenerationRequest(request)
	if err != nil {
		return nil, err
	}

	promptOptions := prompts.AnimationOptions{
		Description: options.Description,
		Style:       options.Style,
		Action:      options.Action,
		FrameCount:  options.FrameCount,
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
		var qualityError *AnimationVideoQualityError
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
	value.ReferenceImage = strings.TrimSpace(value.ReferenceImage)
	value.Resolution = strings.TrimSpace(value.Resolution)
	value.AspectRatio = strings.TrimSpace(value.AspectRatio)
	if value.Description == "" {
		value.Description = "preserve the supplied character exactly"
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
		value.Columns = defaultAnimationColumns
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
	frames, err := s.extractor.Extract(ctx, video, animationCandidateFPS)
	if err != nil {
		return nil, err
	}
	indices, loop, err := selectAnimationLoopFrames(frames, request.FrameCount, animationCandidateFPS)
	if err != nil {
		return nil, err
	}
	if err := validateAnimationMotionSafeAreaAtIndices(frames, indices); err != nil {
		return nil, err
	}
	selected := make([]image.Image, 0, len(indices))
	for _, index := range indices {
		selected = append(selected, frames[index])
	}
	rawSheet, err := packAnimationVideoFrames(selected, request.Columns)
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
		Background: &imageprocessor.AnimationBackgroundOptions{
			MatteColor: "#00ff00",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("generator: normalize sampled animation frames: %w", err)
	}
	if normalized == nil || len(normalized.Regions) != request.FrameCount || strings.TrimSpace(normalized.ImageBase64) == "" {
		return nil, fmt.Errorf("generator: normalize sampled animation frames: empty or incomplete result")
	}
	return &AnimationGenerationResult{
		Frames: normalized.Regions, Spritesheet: normalized.ImageBase64,
		MIMEType: normalized.MIMEType, Normalization: normalized.AnimationReport,
		Loop: loop,
	}, nil
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

func animationFrameMetadata(
	frame imageprocessor.ImageRegion,
	result *AnimationGenerationResult,
) (json.RawMessage, error) {
	metadata, err := json.Marshal(struct {
		Index          int                                          `json:"index"`
		SourceAnchor   *imageprocessor.AnimationPoint               `json:"source_anchor,omitempty"`
		OutputAnchor   *imageprocessor.AnimationPoint               `json:"output_anchor,omitempty"`
		Translation    *imageprocessor.AnimationOffset              `json:"translation,omitempty"`
		Loop           AnimationLoopSelection                       `json:"loop"`
		Normalization  *imageprocessor.AnimationNormalizationReport `json:"normalization,omitempty"`
		VideoRequestID string                                       `json:"video_request_id,omitempty"`
		VideoAttempts  int                                          `json:"video_attempts"`
	}{
		Index:          frame.Index,
		SourceAnchor:   frame.SourceAnchor,
		OutputAnchor:   frame.OutputAnchor,
		Translation:    frame.Translation,
		Loop:           result.Loop,
		Normalization:  result.Normalization,
		VideoRequestID: result.VideoRequestID,
		VideoAttempts:  result.VideoAttempts,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: encode animation frame metadata: %w", err)
	}
	return metadata, nil
}

var _ AnimationGenerationService = (*animationGenerationService)(nil)
