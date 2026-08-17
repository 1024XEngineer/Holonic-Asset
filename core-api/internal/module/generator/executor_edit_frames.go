package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

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
	contextStart := max(indices[0]-1, 0)
	contextEnd := min(indices[len(indices)-1]+1, len(animation.Frames)-1)
	contextCount := contextEnd - contextStart + 1
	if contextCount > 32 {
		return nil, fmt.Errorf("generator: edit frames boundary interval contains %d frames; maximum is 32", contextCount)
	}
	targetFrameIndices := make([]int, len(indices))
	for index, frameIndex := range indices {
		targetFrameIndices[index] = frameIndex - contextStart
	}
	if animation.Generation == nil {
		return nil, fmt.Errorf(
			"generator: animation %d in asset %d has no generation configuration for frame editing",
			payload.AnimationID,
			payload.AssetID,
		)
	}
	generation := animation.Generation
	description := strings.TrimSpace(asset.Description)
	if description == "" {
		description = strings.TrimSpace(asset.Name)
	}

	// Use the unprocessed frames immediately outside the selected interval as
	// the image-to-video start and end anchors. Clamp at animation boundaries so
	// selecting the first or last frame remains valid. Never combine frames into
	// a contact sheet or spritesheet.
	startFrame := animation.Frames[contextStart]
	endFrame := animation.Frames[contextEnd]
	if startFrame.URL == nil || strings.TrimSpace(*startFrame.URL) == "" {
		return nil, fmt.Errorf("generator: frame %d has no image URL", startFrame.ID)
	}
	if endFrame.URL == nil || strings.TrimSpace(*endFrame.URL) == "" {
		return nil, fmt.Errorf("generator: frame %d has no image URL", endFrame.ID)
	}
	request := &AnimationGenerationRequest{
		Description:           description,
		Style:                 generation.Style,
		Action:                prompt,
		ReferenceImage:        animationUnprocessedImageURL(strings.TrimSpace(*startFrame.URL)),
		EndReferenceImage:     animationUnprocessedImageURL(strings.TrimSpace(*endFrame.URL)),
		ReferenceImageContext: true,
		TargetFrameIndices:    targetFrameIndices,
		// Generate one ordered context segment, then replace only the requested
		// samples in the original animation.
		FrameCount:  contextCount,
		Columns:     min(contextCount, 4),
		FrameWidth:  generation.FrameWidth,
		FrameHeight: generation.FrameHeight,
		FPS:         generation.FPS,
		Resolution:  generation.Resolution,
		Duration:    generation.Duration,
		AspectRatio: generation.AspectRatio,
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
	for index, frameIndex := range indices {
		frame := animation.Frames[frameIndex]
		generatedIndex := targetFrameIndices[index]
		persisted, persistErr := e.persistAnimationFrame(ctx, generated.Frames[generatedIndex], rawFrameAt(generated.RawFrames, generatedIndex), generated.FrameDurationMS)
		if persistErr != nil {
			return nil, persistErr
		}
		persisted.ID = frame.ID
		persisted.Metadata = append(json.RawMessage(nil), frame.Metadata...)
		if persisted.Duration == 0 {
			persisted.Duration = frame.Duration
		}
		updated[frameIndex] = persisted
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
