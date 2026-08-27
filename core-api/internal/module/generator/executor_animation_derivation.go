package generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const animationDerivationImageModel = "openai/gpt-image-2"

var animationMirrorDirections = map[string]string{
	AnimationDirectionLeft:       AnimationDirectionRight,
	AnimationDirectionRight:      AnimationDirectionLeft,
	AnimationDirectionFrontLeft:  AnimationDirectionFrontRight,
	AnimationDirectionFrontRight: AnimationDirectionFrontLeft,
	AnimationDirectionBackLeft:   AnimationDirectionBackRight,
	AnimationDirectionBackRight:  AnimationDirectionBackLeft,
}

func (e *executor) deriveAnimation(
	ctx context.Context,
	payload DeriveAnimationPayload,
) (json.RawMessage, error) {
	if payload.AssetID == 0 {
		return nil, fmt.Errorf("generator: animation derivation asset is required")
	}
	if payload.ProjectID == 0 {
		return nil, fmt.Errorf("generator: animation derivation project is required")
	}
	if payload.SourceAnimationID == 0 {
		return nil, fmt.Errorf("generator: animation derivation source animation is required")
	}
	if len(payload.TargetDirections) == 0 {
		return nil, fmt.Errorf("generator: animation derivation target directions are required")
	}

	asset, err := e.assets.GetDetail(ctx, payload.AssetID)
	if err != nil {
		return nil, fmt.Errorf("generator: get animation derivation asset %d: %w", payload.AssetID, err)
	}
	if asset.ID == 0 {
		return nil, fmt.Errorf("generator: animation derivation asset %d not found", payload.AssetID)
	}
	if asset.ProjectID != payload.ProjectID {
		return nil, fmt.Errorf(
			"generator: animation derivation asset %d belongs to project %d, not project %d",
			payload.AssetID,
			asset.ProjectID,
			payload.ProjectID,
		)
	}
	content, err := asset.DecodeContent()
	if err != nil {
		return nil, fmt.Errorf("generator: decode animation derivation asset %d content: %w", payload.AssetID, err)
	}
	if _, ok := animationDirectionLayouts[content.DirectionCount]; !ok {
		return nil, fmt.Errorf("generator: animation derivation requires 2, 4, or 8 asset directions; got %d", content.DirectionCount)
	}

	sourceIndex := -1
	for index := range content.Animations {
		if content.Animations[index].ID == payload.SourceAnimationID {
			sourceIndex = index
			break
		}
	}
	if sourceIndex < 0 {
		return nil, fmt.Errorf(
			"generator: animation derivation source %d not found in asset %d",
			payload.SourceAnimationID,
			payload.AssetID,
		)
	}
	source := content.Animations[sourceIndex]
	if source.Generation == nil {
		return nil, fmt.Errorf("generator: animation derivation source %d has no generation configuration", source.ID)
	}
	if len(source.Frames) == 0 {
		return nil, fmt.Errorf("generator: animation derivation source %d has no frames", source.ID)
	}
	if _, err := animationDirectionIndex(source.Generation.Direction, content.DirectionCount); err != nil {
		return nil, fmt.Errorf("generator: invalid source animation direction: %w", err)
	}

	targets := make([]string, len(payload.TargetDirections))
	seenTargets := make(map[string]struct{}, len(targets))
	for index, rawDirection := range payload.TargetDirections {
		direction := strings.ToLower(strings.TrimSpace(rawDirection))
		if _, err := animationDirectionIndex(direction, content.DirectionCount); err != nil {
			return nil, fmt.Errorf("generator: invalid animation derivation target %d: %w", index+1, err)
		}
		if _, duplicate := seenTargets[direction]; duplicate {
			return nil, fmt.Errorf("generator: animation derivation target direction %q is duplicated", direction)
		}
		seenTargets[direction] = struct{}{}
		targets[index] = direction
	}

	groupID := source.GroupID
	if groupID == 0 {
		groupID = source.ID
	}
	groupByDirection := make(map[string]assetdomain.Animation, int(content.DirectionCount))
	for _, animation := range content.Animations {
		if !animationBelongsToGroup(animation, source, groupID) || animation.Generation == nil {
			continue
		}
		direction := strings.ToLower(strings.TrimSpace(animation.Generation.Direction))
		if direction == "" {
			continue
		}
		if _, duplicate := groupByDirection[direction]; duplicate {
			return nil, fmt.Errorf("generator: animation group %d contains duplicate direction %q", groupID, direction)
		}
		groupByDirection[direction] = animation
	}
	for _, direction := range targets {
		if _, exists := groupByDirection[direction]; exists {
			return nil, fmt.Errorf("generator: animation group %d already contains direction %q", groupID, direction)
		}
	}

	baseRequest, err := derivationGenerationRequest(asset, source)
	if err != nil {
		return nil, err
	}
	type derivationResult struct {
		candidate generatedAnimationCandidate
		resources []string
	}
	results := make([]derivationResult, len(targets))
	generatedCandidates := make([]generatedAnimationCandidate, 0, len(targets))
	generatedResources := make([]string, 0, len(targets)*baseRequest.FrameCount)
	completed := false
	defer func() {
		if !completed && len(generatedResources) > 0 {
			_ = e.references.DeleteObjects(context.WithoutCancel(ctx), generatedResources)
		}
	}()

	deriveContext, cancel := context.WithCancel(ctx)
	defer cancel()
	var group sync.WaitGroup
	var firstError error
	var errorOnce sync.Once
	for index, direction := range targets {
		group.Go(func() {
			var frames []assetdomain.Frame
			mirrorDirection, hasMirror := animationMirrorDirections[direction]
			mirrorAnimation, canUseImage := groupByDirection[mirrorDirection]
			var deriveErr error
			if hasMirror && canUseImage {
				frames, deriveErr = e.deriveAnimationFramesWithImage(
					deriveContext,
					asset,
					baseRequest,
					direction,
					mirrorAnimation,
				)
			} else {
				frames, deriveErr = e.deriveAnimationFramesWithVideo(
					deriveContext,
					asset,
					baseRequest,
					direction,
					source,
				)
			}
			if deriveErr != nil {
				errorOnce.Do(func() {
					firstError = fmt.Errorf("generator: derive animation direction %q: %w", direction, deriveErr)
					cancel()
				})
				return
			}
			generation := *source.Generation
			generation.Direction = direction
			generation.Style = baseRequest.Style
			generation.Action = baseRequest.Action
			generation.FrameCount = baseRequest.FrameCount
			generation.Columns = baseRequest.Columns
			generation.FrameWidth = baseRequest.FrameWidth
			generation.FrameHeight = baseRequest.FrameHeight
			generation.FPS = baseRequest.FPS
			generation.Resolution = baseRequest.Resolution
			generation.Duration = baseRequest.Duration
			generation.AspectRatio = baseRequest.AspectRatio
			candidate := generatedAnimationCandidate{
				GroupID:    groupID,
				Name:       source.Name,
				Frames:     frames,
				Generation: &generation,
			}
			results[index] = derivationResult{
				candidate: candidate,
				resources: generatedFrameResourceKeys(frames),
			}
		})
	}
	group.Wait()
	for _, result := range results {
		generatedResources = append(generatedResources, result.resources...)
	}
	if firstError != nil {
		return nil, firstError
	}
	if err := deriveContext.Err(); err != nil {
		return nil, err
	}
	for _, result := range results {
		generatedCandidates = append(generatedCandidates, result.candidate)
	}

	encoded, err := json.Marshal(struct {
		Animations []generatedAnimationCandidate `json:"animations"`
	}{Animations: generatedCandidates})
	if err != nil {
		return nil, fmt.Errorf("generator: encode animation derivation result for asset %d: %w", payload.AssetID, err)
	}
	completed = true
	return encodeExecutionResult(ExecutionResult{
		AssetID:            payload.AssetID,
		AnimationID:        payload.SourceAnimationID,
		Version:            asset.Version,
		Content:            encoded,
		GeneratedResources: generatedResources,
	})
}

func animationBelongsToGroup(animation, source assetdomain.Animation, groupID uint) bool {
	return animation.ID == source.ID || animation.ID == groupID || animation.GroupID == groupID
}

func derivationGenerationRequest(
	asset assetdomain.Asset,
	source assetdomain.Animation,
) (AnimationGenerationRequest, error) {
	generation := source.Generation
	if generation == nil {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation derivation source generation configuration is required")
	}
	dimensions, err := animationFrameDimensions(asset)
	if err != nil {
		return AnimationGenerationRequest{}, err
	}
	frameWidth, frameHeight, err := resolveAnimationFrameDimensions(
		dimensions,
		generation.FrameWidth,
		generation.FrameHeight,
	)
	if err != nil {
		return AnimationGenerationRequest{}, err
	}
	frameCount := len(source.Frames)
	columns := generation.Columns
	if columns <= 0 || columns > 8 || animationRows(frameCount, columns) > 8 {
		columns = animationGridColumns(frameCount)
	}
	description := strings.TrimSpace(asset.Description)
	if description == "" {
		description = strings.TrimSpace(asset.Name)
	}
	request, err := normalizeAnimationGenerationRequest(&AnimationGenerationRequest{
		Description:     description,
		Style:           generation.Style,
		Action:          generation.Action,
		ReferenceImage:  "derivation-reference",
		FrameCount:      frameCount,
		Columns:         columns,
		FrameWidth:      frameWidth,
		FrameHeight:     frameHeight,
		PrototypeWidth:  int(dimensions.Width),
		PrototypeHeight: int(dimensions.Height),
		FPS:             generation.FPS,
		Resolution:      generation.Resolution,
		Duration:        generation.Duration,
		AspectRatio:     generation.AspectRatio,
	})
	if err != nil {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: normalize animation derivation settings: %w", err)
	}
	return request, nil
}

func (e *executor) deriveAnimationFramesWithVideo(
	ctx context.Context,
	asset assetdomain.Asset,
	base AnimationGenerationRequest,
	targetDirection string,
	source assetdomain.Animation,
) ([]assetdomain.Frame, error) {
	targetReference, _, err := animationReference(asset, targetDirection)
	if err != nil {
		return nil, err
	}
	sourceSheet, err := e.animationDerivationFrameSheet(ctx, source)
	if err != nil {
		return nil, err
	}
	encodedSheet, err := imageprocessor.EncodePNGBase64(sourceSheet)
	if err != nil {
		return nil, fmt.Errorf("encode source animation frame sheet: %w", err)
	}
	request := base
	request.ReferenceImage = targetReference
	request.ReferenceImagePrepared = false
	request.DerivationSourceImage = "data:image/png;base64," + encodedSheet
	request.TargetOrientation = animationDirectionDescription(targetDirection)
	request.SourceOrientation = animationDirectionDescription(source.Generation.Direction)
	generated, err := e.animations.Generate(ctx, &request)
	if err != nil {
		return nil, fmt.Errorf("generate direction video: %w", err)
	}
	if generated == nil || len(generated.Frames) != base.FrameCount {
		return nil, fmt.Errorf(
			"direction video returned %d frames; expected %d",
			resultFrameCount(generated),
			base.FrameCount,
		)
	}
	return e.persistAnimationFrames(ctx, generated)
}

func (e *executor) deriveAnimationFramesWithImage(
	ctx context.Context,
	asset assetdomain.Asset,
	base AnimationGenerationRequest,
	targetDirection string,
	source assetdomain.Animation,
) ([]assetdomain.Frame, error) {
	composite, err := e.animationDerivationComposite(ctx, asset, targetDirection, source)
	if err != nil {
		return nil, err
	}
	encodedComposite, err := imageprocessor.EncodePNGBase64(composite)
	if err != nil {
		return nil, fmt.Errorf("encode animation derivation composite: %w", err)
	}
	rows := animationRows(base.FrameCount, base.Columns)
	result, err := e.images.Generate(ctx, &imageclient.GenerateRequest{
		Prompt: prompts.BuildAnimationDerivationImage(prompts.AnimationImageDerivationOptions{
			Description:       base.Description,
			Style:             base.Style,
			Action:            base.Action,
			TargetOrientation: animationDirectionDescription(targetDirection),
			SourceOrientation: animationDirectionDescription(source.Generation.Direction),
			FrameCount:        base.FrameCount,
			Columns:           base.Columns,
			Rows:              rows,
			FrameWidth:        base.FrameWidth,
			FrameHeight:       base.FrameHeight,
		}),
		ReferenceImages: []string{
			"data:image/png;base64," + encodedComposite,
		},
		N:           1,
		Model:       animationDerivationImageModel,
		Size:        fmt.Sprintf("%dx%d", base.FrameWidth*base.Columns, base.FrameHeight*rows),
		Params:      imageclient.Params{"quality": "high"},
		MaxAttempts: 3,
	})
	if err != nil {
		return nil, fmt.Errorf("edit animation direction frame sheet: %w", err)
	}
	if result == nil || len(result.Images) != 1 || strings.TrimSpace(result.Images[0].Base64) == "" {
		return nil, fmt.Errorf("edit animation direction frame sheet: expected exactly one generated image")
	}
	generatedBase64 := result.Images[0].Base64
	generatedImage, err := imageprocessor.DecodeBase64Image(generatedBase64)
	if err != nil {
		return nil, fmt.Errorf("decode generated direction frame sheet: %w", err)
	}
	if !animationImageHasTransparency(generatedImage) {
		generatedBase64, err = e.removeAnimationDerivationBackground(ctx, generatedBase64, generatedImage)
		if err != nil {
			return nil, err
		}
	}
	split, err := e.processor.SplitImage(ctx, &imageprocessor.SplitImageRequest{
		ImageBase64:            generatedBase64,
		Mode:                   imageprocessor.ImageSplitModeAnimation,
		Columns:                base.Columns,
		Rows:                   rows,
		FrameCount:             base.FrameCount,
		FrameWidth:             base.FrameWidth,
		FrameHeight:            base.FrameHeight,
		Margin:                 0,
		UseExactMargin:         true,
		Anchor:                 imageprocessor.AnimationAnchorFeet,
		ForceProportionalGrid:  true,
		PreserveVerticalMotion: true,
	})
	if err != nil {
		return nil, fmt.Errorf("split generated direction frame sheet: %w", err)
	}
	if split == nil || len(split.Regions) != base.FrameCount {
		got := 0
		if split != nil {
			got = len(split.Regions)
		}
		return nil, fmt.Errorf("split generated direction frame sheet into %d frames; expected %d", got, base.FrameCount)
	}
	rawFrames := append([]imageprocessor.ImageRegion(nil), split.Regions...)
	processed, sheet, err := pixelProcessAnimationFrames(
		ctx,
		e.processor,
		split.Regions,
		base.Columns,
		base.FrameWidth,
		base.FrameHeight,
	)
	if err != nil {
		return nil, err
	}
	duration := uint((1000 + base.FPS/2) / base.FPS)
	generated := &AnimationGenerationResult{
		Frames:          processed,
		RawFrames:       rawFrames,
		Spritesheet:     sheet,
		MIMEType:        "image/png",
		FrameDurationMS: duration,
	}
	return e.persistAnimationFrames(ctx, generated)
}

func (e *executor) removeAnimationDerivationBackground(
	ctx context.Context,
	imageBase64 string,
	generated image.Image,
) (string, error) {
	green, err := imageprocessor.ParseMatteColor(imageprocessor.DefaultMatteColor)
	if err != nil {
		return "", fmt.Errorf("parse animation derivation matte: %w", err)
	}
	mattes := []string{imageprocessor.DefaultMatteColor}
	sampled := imageprocessor.EstimateMatteColor(generated)
	if imageprocessor.EuclideanColorDistance(sampled, green) > imageprocessor.DefaultChromaThreshold {
		// Image models sometimes add a white outer border or grid gutters despite
		// the prompt. Edge sampling finds that secondary matte before the green
		// pass scrubs transparent RGB and makes it undiscoverable.
		mattes = append(mattes, imageprocessor.ColorToHex(sampled))
	}

	current := imageBase64
	for _, matte := range mattes {
		removed, removeErr := e.processor.RemoveBackground(ctx, &imageprocessor.RemoveBackgroundRequest{
			ImageBase64: current,
			MatteColor:  matte,
		})
		if removeErr != nil {
			return "", fmt.Errorf("remove generated direction frame sheet matte %s: %w", matte, removeErr)
		}
		if removed == nil || strings.TrimSpace(removed.ImageBase64) == "" {
			return "", fmt.Errorf("remove generated direction frame sheet matte %s: empty result", matte)
		}
		current = removed.ImageBase64
	}
	return current, nil
}

func (e *executor) animationDerivationComposite(
	ctx context.Context,
	asset assetdomain.Asset,
	targetDirection string,
	source assetdomain.Animation,
) (*image.NRGBA, error) {
	target, err := e.loadAnimationPrototypeImage(ctx, asset, targetDirection)
	if err != nil {
		return nil, err
	}
	sourceSheet, err := e.animationDerivationFrameSheet(ctx, source)
	if err != nil {
		return nil, err
	}
	columns := effectiveAnimationColumns(source)
	rows := animationRows(len(source.Frames), columns)
	if rows <= 0 {
		return nil, fmt.Errorf("source animation rows are required")
	}
	frameHeight := sourceSheet.Bounds().Dy() / rows
	panelWidth := max(sourceSheet.Bounds().Dx(), target.Bounds().Dx())
	panelHeight := max(frameHeight, target.Bounds().Dy())
	canvas := image.NewNRGBA(image.Rect(
		0,
		0,
		panelWidth,
		panelHeight+sourceSheet.Bounds().Dy(),
	))
	matte := &image.Uniform{C: color.NRGBA{G: 255, A: 255}}
	draw.Draw(canvas, canvas.Bounds(), matte, image.Point{}, draw.Src)
	targetPosition := image.Pt(
		(canvas.Bounds().Dx()-target.Bounds().Dx())/2,
		(panelHeight-target.Bounds().Dy())/2,
	)
	draw.Draw(
		canvas,
		target.Bounds().Add(targetPosition.Sub(target.Bounds().Min)),
		target,
		target.Bounds().Min,
		draw.Over,
	)
	sheetPosition := image.Pt(
		(canvas.Bounds().Dx()-sourceSheet.Bounds().Dx())/2,
		panelHeight,
	)
	draw.Draw(
		canvas,
		sourceSheet.Bounds().Add(sheetPosition.Sub(sourceSheet.Bounds().Min)),
		sourceSheet,
		sourceSheet.Bounds().Min,
		draw.Src,
	)
	return canvas, nil
}

func (e *executor) animationDerivationFrameSheet(
	ctx context.Context,
	animation assetdomain.Animation,
) (*image.NRGBA, error) {
	if animation.Generation == nil {
		return nil, fmt.Errorf("source animation generation configuration is required")
	}
	if len(animation.Frames) == 0 {
		return nil, fmt.Errorf("source animation frames are required")
	}
	columns := effectiveAnimationColumns(animation)
	frames := make([]image.Image, len(animation.Frames))
	for index, frame := range animation.Frames {
		if frame.URL == nil || strings.TrimSpace(*frame.URL) == "" {
			return nil, fmt.Errorf("source animation frame %d has no image URL", frame.ID)
		}
		loaded, err := e.loadAnimationFrameImage(ctx, strings.TrimSpace(*frame.URL))
		if err != nil {
			return nil, fmt.Errorf("load source animation frame %d: %w", frame.ID, err)
		}
		frames[index] = loaded
	}
	return packAnimationDerivationFrames(frames, columns)
}

func effectiveAnimationColumns(animation assetdomain.Animation) int {
	columns := 0
	if animation.Generation != nil {
		columns = animation.Generation.Columns
	}
	if columns <= 0 || columns > 8 || animationRows(len(animation.Frames), columns) > 8 {
		columns = animationGridColumns(len(animation.Frames))
	}
	return columns
}

func packAnimationDerivationFrames(frames []image.Image, columns int) (*image.NRGBA, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("animation derivation frames are required")
	}
	if columns <= 0 {
		return nil, fmt.Errorf("animation derivation columns must be positive")
	}
	width, height := frames[0].Bounds().Dx(), frames[0].Bounds().Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("animation derivation frame dimensions must be positive")
	}
	rows := animationRows(len(frames), columns)
	canvas := image.NewNRGBA(image.Rect(0, 0, width*columns, height*rows))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	for index, frame := range frames {
		if frame == nil || frame.Bounds().Dx() != width || frame.Bounds().Dy() != height {
			return nil, fmt.Errorf("animation derivation frame %d dimensions differ", index+1)
		}
		column, row := index%columns, index/columns
		destination := image.Rect(column*width, row*height, (column+1)*width, (row+1)*height)
		draw.Draw(canvas, destination, frame, frame.Bounds().Min, draw.Over)
	}
	return canvas, nil
}

func (e *executor) loadAnimationPrototypeImage(
	ctx context.Context,
	asset assetdomain.Asset,
	direction string,
) (image.Image, error) {
	content, err := asset.DecodeContent()
	if err != nil {
		return nil, fmt.Errorf("decode animation prototype asset %d: %w", asset.ID, err)
	}
	index, err := animationDirectionIndex(direction, content.DirectionCount)
	if err != nil {
		return nil, err
	}
	if content.Prototype == nil || index >= len(*content.Prototype) {
		return nil, fmt.Errorf("animation asset %d has no prototype for direction %q", asset.ID, direction)
	}
	resource := (*content.Prototype)[index]
	if resource.URL == nil || strings.TrimSpace(*resource.URL) == "" {
		return nil, fmt.Errorf("animation asset %d prototype direction %q has no image URL", asset.ID, direction)
	}
	reference := strings.TrimSpace(*resource.URL)
	unprocessed := animationUnprocessedImageURL(reference)
	if unprocessed != reference {
		if loaded, loadErr := e.loadAnimationDerivationImage(ctx, unprocessed); loadErr == nil {
			return loaded, nil
		}
	}
	loaded, err := e.loadAnimationDerivationImage(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("load animation prototype direction %q: %w", direction, err)
	}
	return loaded, nil
}

func (e *executor) loadAnimationFrameImage(
	ctx context.Context,
	reference string,
) (image.Image, error) {
	loaded, processedErr := e.loadAnimationDerivationImage(ctx, reference)
	if processedErr == nil {
		return loaded, nil
	}
	unprocessed := animationUnprocessedImageURL(reference)
	if unprocessed != reference {
		loaded, err := e.loadAnimationDerivationImage(ctx, unprocessed)
		if err == nil {
			return loaded, nil
		}
		return nil, fmt.Errorf("load processed frame: %v; load unprocessed frame: %w", processedErr, err)
	}
	return nil, processedErr
}

func (e *executor) loadAnimationDerivationImage(
	ctx context.Context,
	reference string,
) (image.Image, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return nil, fmt.Errorf("image reference is required")
	}
	resolved := reference
	if !strings.HasPrefix(strings.ToLower(reference), "data:image/") {
		var err error
		resolved, err = e.references.ResolveReference(ctx, reference)
		if err != nil {
			return nil, fmt.Errorf("resolve image reference: %w", err)
		}
		resolved = strings.TrimSpace(resolved)
	}
	if !isHTTPReference(resolved) {
		decoded, err := imageprocessor.DecodeBase64Image(resolved)
		if err != nil {
			return nil, fmt.Errorf("decode image reference: %w", err)
		}
		return decoded, nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, resolved, nil)
	if err != nil {
		return nil, fmt.Errorf("create image reference download request: %w", err)
	}
	if err := validatePrototypeReferenceURL(request.URL); err != nil {
		return nil, err
	}
	client := e.referenceHTTPClient
	if client == nil {
		client = newPrototypeReferenceHTTPClient()
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download image reference: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return nil, fmt.Errorf("download image reference: HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxAnimationReferenceBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read image reference: %w", err)
	}
	if len(body) == 0 || len(body) > maxAnimationReferenceBytes {
		return nil, fmt.Errorf("downloaded image reference size %d is invalid", len(body))
	}
	decoded, err := imageprocessor.DecodeBase64Image(base64.StdEncoding.EncodeToString(body))
	if err != nil {
		return nil, fmt.Errorf("decode downloaded image reference: %w", err)
	}
	return decoded, nil
}

func animationDirectionDescription(direction string) string {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case AnimationDirectionFront:
		return "Front / South / screen-down view"
	case AnimationDirectionFrontRight:
		return "Front-right / Southeast isometric view"
	case AnimationDirectionRight:
		return "Right / East / screen-right view"
	case AnimationDirectionBackRight:
		return "Back-right / Northeast isometric view"
	case AnimationDirectionBack:
		return "Back / North / screen-up view"
	case AnimationDirectionBackLeft:
		return "Back-left / Northwest isometric view"
	case AnimationDirectionLeft:
		return "Left / West / screen-left view"
	case AnimationDirectionFrontLeft:
		return "Front-left / Southwest isometric view"
	default:
		return strings.TrimSpace(direction)
	}
}
