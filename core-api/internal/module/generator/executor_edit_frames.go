package generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const editFrameContextPadding = 4

func (e *executor) editFrames(ctx context.Context, payload EditFramesPayload) (json.RawMessage, error) {
	if payload.AssetID == 0 {
		return nil, fmt.Errorf("generator: edit frames asset is required")
	}
	if payload.ProjectID == 0 {
		return nil, fmt.Errorf("generator: edit frames project is required")
	}
	if payload.AnimationID == 0 {
		return nil, fmt.Errorf("generator: edit frames animation is required")
	}
	if len(payload.FrameIDs) == 0 {
		return nil, fmt.Errorf("generator: edit frames frame ids are required")
	}
	prompt := strings.TrimSpace(payload.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("generator: edit frames prompt is required")
	}

	updater, ok := e.assets.(AnimationFrameUpdater)
	if !ok {
		return nil, ErrAssetWriterRequired
	}
	asset, err := e.assets.GetDetail(ctx, payload.AssetID)
	if err != nil {
		return nil, fmt.Errorf("generator: get edit frames asset %d: %w", payload.AssetID, err)
	}
	if asset.ID == 0 {
		return nil, fmt.Errorf("generator: edit frames asset %d not found", payload.AssetID)
	}
	if asset.ProjectID != payload.ProjectID {
		return nil, fmt.Errorf("generator: edit frames asset %d belongs to project %d, not project %d", payload.AssetID, asset.ProjectID, payload.ProjectID)
	}
	content, err := asset.DecodeContent()
	if err != nil {
		return nil, fmt.Errorf("generator: decode edit frames asset %d content: %w", payload.AssetID, err)
	}
	animationIndex := slices.IndexFunc(content.Animations, func(value assetdomain.Animation) bool { return value.ID == payload.AnimationID })
	if animationIndex < 0 {
		return nil, fmt.Errorf("generator: animation %d not found in asset %d", payload.AnimationID, payload.AssetID)
	}
	animation := content.Animations[animationIndex]
	if len(animation.Frames) == 0 {
		return nil, fmt.Errorf("generator: animation %d has no frames", payload.AnimationID)
	}

	targets := make(map[uint]struct{}, len(payload.FrameIDs))
	indices := make([]int, 0, len(payload.FrameIDs))
	for _, frameID := range payload.FrameIDs {
		if frameID == 0 {
			return nil, fmt.Errorf("generator: edit frames frame id must be positive")
		}
		if _, exists := targets[frameID]; exists {
			return nil, fmt.Errorf("generator: edit frames frame id %d is duplicated", frameID)
		}
		targets[frameID] = struct{}{}
		index := slices.IndexFunc(animation.Frames, func(value assetdomain.Frame) bool { return value.ID == frameID })
		if index < 0 {
			return nil, fmt.Errorf("generator: frame %d not found in animation %d", frameID, payload.AnimationID)
		}
		indices = append(indices, index)
	}
	slices.Sort(indices)
	contextStart := max(indices[0]-editFrameContextPadding, 0)
	contextEnd := indices[len(indices)-1] + editFrameContextPadding
	if contextEnd >= len(animation.Frames) {
		contextEnd = len(animation.Frames) - 1
	}
	contextCount := contextEnd - contextStart + 1
	targetFrameIndices := make([]int, len(indices))
	for index, frameIndex := range indices {
		targetFrameIndices[index] = frameIndex - contextStart
	}
	if contextCount > 32 {
		return nil, fmt.Errorf("generator: edit frames context contains %d frames; maximum is 32", contextCount)
	}

	contextImages := make([]image.Image, 0, contextCount)
	for index := contextStart; index <= contextEnd; index++ {
		frame := animation.Frames[index]
		if frame.URL == nil || strings.TrimSpace(*frame.URL) == "" {
			return nil, fmt.Errorf("generator: frame %d has no image URL", frame.ID)
		}
		imageValue, loadErr := e.loadFrameImage(ctx, animationUnprocessedImageURL(*frame.URL), *frame.URL)
		if loadErr != nil {
			return nil, fmt.Errorf("generator: load context frame %d: %w", frame.ID, loadErr)
		}
		contextImages = append(contextImages, imageValue)
	}
	columns := min(contextCount, 4)
	sheet, err := packAnimationVideoFrames(contextImages, columns)
	if err != nil {
		return nil, fmt.Errorf("generator: build edit frames context: %w", err)
	}
	contextSheet, err := imageprocessor.EncodePNGBase64(sheet)
	if err != nil {
		return nil, fmt.Errorf("generator: encode edit frames context: %w", err)
	}

	description := strings.TrimSpace(asset.Description)
	if description == "" {
		description = strings.TrimSpace(asset.Name)
	}
	generation := animation.Generation
	request := &AnimationGenerationRequest{
		Description:            description,
		Style:                  prompts.DefaultAnimationStyle,
		Action:                 prompt,
		ReferenceImage:         "data:image/png;base64," + contextSheet,
		ReferenceImagePrepared: true,
		ReferenceImageContext:  true,
		TargetFrameIndices:     targetFrameIndices,
		FrameCount:             contextCount,
		Columns:                columns,
		FrameWidth:             defaultAnimationFrameWidth,
		FrameHeight:            defaultAnimationFrameHeight,
		FPS:                    defaultAnimationFPS,
		Resolution:             defaultAnimationResolution,
		Duration:               defaultAnimationDuration,
		AspectRatio:            defaultAnimationAspectRatio,
	}
	if generation != nil {
		request.Style = generation.Style
		request.FrameCount = contextCount
		request.Columns = columns
		request.FrameWidth = generation.FrameWidth
		request.FrameHeight = generation.FrameHeight
		request.FPS = generation.FPS
		request.Resolution = generation.Resolution
		request.Duration = generation.Duration
		request.AspectRatio = generation.AspectRatio
	}
	requestValue, err := normalizeAnimationGenerationRequest(request)
	if err != nil {
		return nil, fmt.Errorf("generator: normalize edit frames request: %w", err)
	}
	generated, err := e.animations.Generate(ctx, &requestValue)
	if err != nil {
		return nil, fmt.Errorf("generator: generate edited frames: %w", err)
	}
	if generated == nil || len(generated.Frames) != contextCount {
		return nil, fmt.Errorf("generator: edited frame result contains %d frames; expected %d", resultFrameCount(generated), contextCount)
	}
	if len(generated.RawFrames) != contextCount {
		return nil, fmt.Errorf("generator: edited raw frame result contains %d frames; expected %d", len(generated.RawFrames), contextCount)
	}

	updated := append([]assetdomain.Frame(nil), animation.Frames...)
	for index := contextStart; index <= contextEnd; index++ {
		if _, selected := targets[animation.Frames[index].ID]; !selected {
			continue
		}
		generatedIndex := index - contextStart
		persisted, persistErr := e.persistAnimationFrame(ctx, generated.Frames[generatedIndex], rawFrameAt(generated.RawFrames, generatedIndex), generated.FrameDurationMS)
		if persistErr != nil {
			return nil, persistErr
		}
		persisted.ID = animation.Frames[index].ID
		persisted.Metadata = append(json.RawMessage(nil), animation.Frames[index].Metadata...)
		if persisted.Duration == 0 {
			persisted.Duration = animation.Frames[index].Duration
		}
		updated[index] = persisted
	}
	if err := updater.UpdateAnimationFrames(ctx, payload.AssetID, payload.AnimationID, updated); err != nil {
		return nil, fmt.Errorf("generator: update animation %d frames: %w", payload.AnimationID, err)
	}
	updatedAsset, err := e.assets.GetDetail(ctx, payload.AssetID)
	if err != nil {
		return nil, fmt.Errorf("generator: get updated edit frames asset %d: %w", payload.AssetID, err)
	}
	return encodeExecutionResult(ExecutionResult{AssetID: payload.AssetID, AnimationID: payload.AnimationID, Version: updatedAsset.Version})
}

func resultFrameCount(result *AnimationGenerationResult) int {
	if result == nil {
		return 0
	}
	return len(result.Frames)
}

func rawFrameAt(frames []imageprocessor.ImageRegion, index int) *imageprocessor.ImageRegion {
	if len(frames) == 0 || index < 0 || index >= len(frames) {
		return nil
	}
	return &frames[index]
}

func (e *executor) persistAnimationFrame(ctx context.Context, frame imageprocessor.ImageRegion, raw *imageprocessor.ImageRegion, duration uint) (assetdomain.Frame, error) {
	mediaType := strings.TrimSpace(frame.MIMEType)
	if mediaType == "" {
		mediaType = "image/png"
	}
	key, err := e.references.PersistReference(ctx, "data:"+mediaType+";base64,"+frame.ImageBase64)
	if err != nil {
		return assetdomain.Frame{}, fmt.Errorf("generator: persist edited animation frame: %w", err)
	}
	key = strings.TrimSpace(key)
	if key == "" || strings.HasPrefix(key, "data:") || strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
		return assetdomain.Frame{}, fmt.Errorf("generator: persist edited animation frame: storage returned a non-object-key reference")
	}
	if raw != nil {
		rawType := strings.TrimSpace(raw.MIMEType)
		if rawType == "" {
			rawType = "image/png"
		}
		if strings.TrimSpace(raw.ImageBase64) == "" {
			return assetdomain.Frame{}, fmt.Errorf("generator: persist edited animation frame: raw frame is empty")
		}
		if err := e.references.PersistReferenceAt(ctx, addObjectKeySuffix(key, "-unprocessed"), "data:"+rawType+";base64,"+raw.ImageBase64); err != nil {
			return assetdomain.Frame{}, fmt.Errorf("generator: persist raw edited animation frame: %w", err)
		}
	}
	return assetdomain.Frame{URL: &key, Duration: duration}, nil
}

func (e *executor) loadFrameImage(ctx context.Context, rawReference, fallbackReference string) (image.Image, error) {
	value, err := e.loadFrameReference(ctx, rawReference)
	if err != nil && rawReference != fallbackReference {
		value, err = e.loadFrameReference(ctx, fallbackReference)
	}
	if err != nil {
		return nil, err
	}
	decoded, err := imageprocessor.DecodeBase64Image(value)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func (e *executor) loadFrameReference(ctx context.Context, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", fmt.Errorf("frame reference is empty")
	}
	if e.references != nil && !strings.HasPrefix(reference, "data:") {
		resolved, err := e.references.ResolveReference(ctx, reference)
		if err != nil {
			return "", err
		}
		reference = strings.TrimSpace(resolved)
	}
	if strings.HasPrefix(reference, "data:") {
		return reference, nil
	}
	parsed, err := url.Parse(reference)
	if err != nil || !parsed.IsAbs() {
		return "", fmt.Errorf("frame reference is not readable")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, reference, nil)
	if err != nil {
		return "", err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("frame reference download returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxAnimationReferenceBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > maxAnimationReferenceBytes {
		return "", fmt.Errorf("frame reference exceeds %d bytes", maxAnimationReferenceBytes)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(body), nil
}
