package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

// Executor owns generation and any resulting asset creation.
const animationReferenceMetadataKey = "animation_reference"

type Executor interface {
	Generate(ctx context.Context, taskType TaskType, payload json.RawMessage) (json.RawMessage, error)
}

// AssetWriter is the subset of Workspace asset operations used by generation.
type AssetWriter interface {
	GetDetail(context.Context, uint) (assetdomain.Asset, error)
	CreateCharacterAsset(context.Context, *assetdomain.Asset) (*assetdomain.Asset, error)
	CreateObjectAsset(context.Context, *assetdomain.Asset) (uint, error)
	CreateAnimation(context.Context, uint, string) (uint, error)
	UpdatePrototypeImages(context.Context, uint, []assetdomain.ImageResource) error
	UpdateAnimationFrames(context.Context, uint, uint, []assetdomain.Frame) error
}

type executor struct {
	images     imageclient.ImageGenerationService
	animations AnimationGenerationService
	processor  imageprocessor.Processor
	assets     AssetWriter
}

// NewExecutor creates the image-to-asset workflow used by task handlers.
func NewExecutor(
	images imageclient.ImageGenerationService,
	processor imageprocessor.Processor,
	assets AssetWriter,
) Executor {
	return &executor{images: images, processor: processor, assets: assets}
}

// NewExecutorWithAnimation creates the complete generation workflow, including
// image-to-video animation generation. NewExecutor remains available for
// prototype-only callers and tests that do not need animation generation.
func NewExecutorWithAnimation(
	images imageclient.ImageGenerationService,
	animations AnimationGenerationService,
	processor imageprocessor.Processor,
	assets AssetWriter,
) Executor {
	return &executor{
		images: images, animations: animations, processor: processor, assets: assets,
	}
}

type ExecutionResult struct {
	AssetID     uint `json:"asset_id"`
	AnimationID uint `json:"animation_id,omitempty"`
}

func (e *executor) Generate(
	ctx context.Context,
	taskType TaskType,
	payload json.RawMessage,
) (json.RawMessage, error) {
	switch taskType {
	case GenerateCharacterProtoType:
		if err := e.requirePrototypeDependencies(); err != nil {
			return nil, err
		}
		request := CreateCharacterPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateCharacterPrototype(ctx, request)
	case GenerateObjectProtoType:
		if err := e.requirePrototypeDependencies(); err != nil {
			return nil, err
		}
		request := CreateObjectPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateObjectPrototype(ctx, request)
	case GenerateAnimation:
		if e.assets == nil {
			return nil, ErrAssetWriterRequired
		}
		if e.animations == nil {
			return nil, ErrAnimationServiceRequired
		}
		request := CreateAnimationPayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateAnimation(ctx, request)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedTaskType, taskType)
	}
}

func (e *executor) requirePrototypeDependencies() error {
	if e.images == nil {
		return ErrImageServiceRequired
	}
	if e.assets == nil {
		return ErrAssetWriterRequired
	}
	if e.processor == nil {
		return ErrImageProcessorRequired
	}
	return nil
}

func (e *executor) generateCharacterPrototype(
	ctx context.Context,
	payload CreateCharacterPrototypePayload,
) (json.RawMessage, error) {
	perspective, err := parsePerspective(payload.Perspective)
	if err != nil {
		return nil, err
	}
	directionCount, err := parseDirectionCount(payload.DirectionCount)
	if err != nil {
		return nil, err
	}
	generated, err := e.generateImages(
		ctx,
		GenerateCharacterProtoType,
		payload.CreativeBrief,
		payload.CanvasSize,
		payload.Reference,
	)
	if err != nil {
		return nil, err
	}
	value, err := newPrototypeAsset(
		assetdomain.AssetTypeCharacter,
		payload.AssetName,
		payload.ProjectID,
		payload.CreativeBrief,
		perspective,
		directionCount,
		prototypeResources(generated),
	)
	if err != nil {
		return nil, err
	}
	created, err := e.assets.CreateCharacterAsset(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("generator: create character asset: %w", err)
	}
	if created == nil || created.ID == 0 {
		return nil, fmt.Errorf("generator: create character asset: empty result")
	}
	return encodeExecutionResult(ExecutionResult{AssetID: created.ID})
}

func (e *executor) generateObjectPrototype(
	ctx context.Context,
	payload CreateObjectPrototypePayload,
) (json.RawMessage, error) {
	perspective, err := parsePerspective(payload.Perspective)
	if err != nil {
		return nil, err
	}
	generated, err := e.generateImages(
		ctx,
		GenerateObjectProtoType,
		payload.CreativeBrief,
		payload.CanvasSize,
		payload.Reference,
	)
	if err != nil {
		return nil, err
	}
	value, err := newPrototypeAsset(
		assetdomain.AssetTypeObject,
		payload.AssetName,
		payload.ProjectID,
		payload.CreativeBrief,
		perspective,
		0,
		prototypeResources(generated),
	)
	if err != nil {
		return nil, err
	}
	assetID, err := e.assets.CreateObjectAsset(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("generator: create object asset: %w", err)
	}
	if assetID == 0 {
		return nil, fmt.Errorf("generator: create object asset: empty result")
	}
	return encodeExecutionResult(ExecutionResult{AssetID: assetID})
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
	description := strings.TrimSpace(asset.Description)
	if description == "" {
		description = strings.TrimSpace(asset.Name)
	}
	generated, err := e.animations.Generate(ctx, &AnimationGenerationRequest{
		Description:            description,
		Style:                  payload.Style,
		Action:                 payload.CreativeBrief,
		ReferenceImage:         reference,
		ReferenceImagePrepared: referencePrepared,
		FrameCount:             payload.FrameCount,
		Columns:                payload.Columns,
		FrameWidth:             payload.FrameWidth,
		FrameHeight:            payload.FrameHeight,
		FPS:                    payload.FPS,
		Resolution:             payload.Resolution,
		Duration:               payload.Duration,
		AspectRatio:            payload.AspectRatio,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: generate animation frames: %w", err)
	}
	if generated == nil || len(generated.Frames) == 0 {
		return nil, fmt.Errorf("generator: generate animation frames: empty result")
	}
	frames, err := animationFrames(generated)
	if err != nil {
		return nil, err
	}

	animationID, err := e.assets.CreateAnimation(ctx, payload.AssetID, animationName)
	if err != nil {
		return nil, fmt.Errorf("generator: create animation for asset %d: %w", payload.AssetID, err)
	}
	if animationID == 0 {
		return nil, fmt.Errorf("generator: create animation for asset %d: empty result", payload.AssetID)
	}
	if err := e.assets.UpdateAnimationFrames(ctx, payload.AssetID, animationID, frames); err != nil {
		return nil, fmt.Errorf(
			"generator: update asset %d animation %d frames: %w",
			payload.AssetID,
			animationID,
			err,
		)
	}
	return encodeExecutionResult(ExecutionResult{AssetID: payload.AssetID, AnimationID: animationID})
}

func animationReference(asset assetdomain.Asset, direction int) (string, bool, error) {
	content, err := asset.DecodeContent()
	if err != nil {
		return "", false, fmt.Errorf("generator: decode animation asset %d content: %w", asset.ID, err)
	}
	directionCount := content.DirectionCount
	if directionCount == 0 {
		directionCount = 1
	}
	if direction < 0 || uint(direction) >= directionCount {
		return "", false, fmt.Errorf("generator: animation direction %d is out of range for asset %d with %d directions", direction, asset.ID, directionCount)
	}
	if content.DirectionCount > 1 {
		if content.Prototype == nil || direction >= len(*content.Prototype) {
			return "", false, fmt.Errorf("generator: animation asset %d has no prototype for direction %d", asset.ID, direction)
		}
		prototype := (*content.Prototype)[direction]
		if prototype.URL == nil || strings.TrimSpace(*prototype.URL) == "" {
			return "", false, fmt.Errorf("generator: animation asset %d prototype direction %d has no image URL", asset.ID, direction)
		}
		rowURL := animationRowImageURL(strings.TrimSpace(*prototype.URL))
		// TODO: Ask the image-storage module for the uncompressed, background-removed
		// "-row" variant and pass its bytes to image-to-video. The storage contract
		// is not implemented in this change, so the URL is kept as the reference
		// value until that loader is available.
		return rowURL, false, nil
	}
	if reference := animationReferenceFromMetadata(content.Metadata); reference != "" {
		return reference, true, nil
	}
	if content.Prototype == nil {
		return "", false, fmt.Errorf("generator: animation asset %d has no prototype", asset.ID)
	}
	for _, prototype := range *content.Prototype {
		if prototype.URL != nil && strings.TrimSpace(*prototype.URL) != "" {
			return strings.TrimSpace(*prototype.URL), false, nil
		}
	}
	return "", false, fmt.Errorf("generator: animation asset %d has no prototype image", asset.ID)
}

func animationRowImageURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "data:") {
		return value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return addAnimationRowSuffix(value)
	}
	parsed.Path = addAnimationRowSuffix(parsed.Path)
	return parsed.String()
}

func addAnimationRowSuffix(path string) string {
	lastSlash := strings.LastIndex(path, "/")
	lastDot := strings.LastIndex(path, ".")
	if lastDot <= lastSlash {
		return path + "-row"
	}
	return path[:lastDot] + "-row" + path[lastDot:]
}

func (e *executor) generateImages(
	ctx context.Context,
	taskType TaskType,
	prompt string,
	size string,
	reference string,
) (*imageclient.GenerateResult, error) {
	references := []string(nil)
	if reference != "" {
		references = []string{reference}
	}
	result, err := e.images.Generate(ctx, &imageclient.GenerateRequest{
		Prompt:          prompt,
		ReferenceImages: references,
		Size:            size,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: generate %s images: %w", taskType, err)
	}
	if result == nil || len(result.Images) == 0 {
		return nil, fmt.Errorf("generator: generate %s images: %w", taskType, ErrImageResultRequired)
	}
	if taskType != GenerateCharacterProtoType && taskType != GenerateObjectProtoType {
		return result, nil
	}
	resizeWidth, resizeHeight, err := parseCanvasSize(size)
	if err != nil {
		return nil, fmt.Errorf("generator: process %s images: %w", taskType, err)
	}
	processed := &imageclient.GenerateResult{Images: make([]imageclient.GeneratedImage, len(result.Images)), Model: result.Model, Size: result.Size, CreatedAt: result.CreatedAt, Usage: result.Usage}
	for index, generated := range result.Images {
		imageBase64 := generated.Base64
		if taskType == GenerateObjectProtoType {
			backgroundRemoved, processErr := e.processor.RemoveBackground(ctx, &imageprocessor.RemoveBackgroundRequest{ImageBase64: imageBase64})
			if processErr != nil {
				return nil, fmt.Errorf("generator: remove %s image %d background: %w", taskType, index+1, processErr)
			}
			if backgroundRemoved == nil || backgroundRemoved.ImageBase64 == "" {
				return nil, fmt.Errorf("generator: remove %s image %d background: empty result", taskType, index+1)
			}
			imageBase64 = backgroundRemoved.ImageBase64
		}
		resized, resizeErr := e.processor.Resize(ctx, &imageprocessor.ResizeRequest{
			ImageBase64: imageBase64,
			Options:     imageprocessor.DefaultResizeOptions(resizeWidth, resizeHeight),
		})
		if resizeErr != nil {
			return nil, fmt.Errorf("generator: resize %s image %d: %w", taskType, index+1, resizeErr)
		}
		if resized == nil || resized.ImageBase64 == "" {
			return nil, fmt.Errorf("generator: resize %s image %d: empty result", taskType, index+1)
		}
		processed.Images[index] = imageclient.GeneratedImage{Base64: resized.ImageBase64, MediaType: resized.MIMEType}
	}
	return processed, nil
}

func parseCanvasSize(size string) (int, int, error) {
	var width, height int
	if _, err := fmt.Sscanf(size, "%dx%d", &width, &height); err != nil || width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("invalid canvas size %q", size)
	}
	return width, height, nil
}

func newPrototypeAsset(
	assetType assetdomain.AssetType,
	name string,
	projectID uint,
	description string,
	perspective assetdomain.Perspective,
	directionCount uint,
	prototype []assetdomain.ImageResource,
) (*assetdomain.Asset, error) {
	content := assetdomain.NewAssetContent(assetType)
	content.Perspective = perspective
	prototypeValue := assetdomain.Prototype(prototype)
	content.Prototype = &prototypeValue
	content.DirectionCount = directionCount
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		return nil, fmt.Errorf("generator: encode prototype asset content: %w", err)
	}
	return &assetdomain.Asset{
		Name:        name,
		ProjectID:   projectID,
		Type:        assetType,
		Description: description,
		Content:     encoded,
	}, nil
}

func parsePerspective(perspective string) (assetdomain.Perspective, error) {
	value := assetdomain.Perspective(perspective)
	if !value.Valid() {
		return "", fmt.Errorf("generator: invalid perspective %q", perspective)
	}
	return value, nil
}

func parseDirectionCount(directionCount string) (uint, error) {
	if directionCount == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(directionCount, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("generator: parse direction count %q: %w", directionCount, err)
	}
	switch value {
	case 1, 2, 4, 8:
		return uint(value), nil
	default:
		return 0, fmt.Errorf("generator: invalid direction count %q", directionCount)
	}
}

func prototypeResources(result *imageclient.GenerateResult) []assetdomain.ImageResource {
	resources := make([]assetdomain.ImageResource, len(result.Images))
	for index, image := range result.Images {
		url := generatedImageDataURL(image)
		resources[index] = assetdomain.ImageResource{ID: uint(index + 1), URL: &url}
	}
	return resources
}

func animationReferenceFromMetadata(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	reference, _ := metadata[animationReferenceMetadataKey].(string)
	return strings.TrimSpace(reference)
}

func animationFrames(result *AnimationGenerationResult) ([]assetdomain.Frame, error) {
	frames := make([]assetdomain.Frame, len(result.Frames))
	for index, frame := range result.Frames {
		mediaType := strings.TrimSpace(frame.MIMEType)
		if mediaType == "" {
			mediaType = "image/png"
		}
		url := "data:" + mediaType + ";base64," + frame.ImageBase64
		metadata, err := animationFrameMetadata(frame, result)
		if err != nil {
			return nil, err
		}
		frames[index] = assetdomain.Frame{
			ID:       uint(index + 1),
			URL:      &url,
			Duration: result.FrameDurationMS,
			Metadata: metadata,
		}
	}
	return frames, nil
}

func generatedImageDataURL(image imageclient.GeneratedImage) string {
	mediaType := image.MediaType
	if mediaType == "" {
		mediaType = "image/png"
	}
	return "data:" + mediaType + ";base64," + image.Base64
}

func decodeExecutionPayload(taskType TaskType, payload json.RawMessage, target any) error {
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("generator: decode %s execution payload: %w", taskType, err)
	}
	return nil
}

func encodeExecutionResult(result ExecutionResult) (json.RawMessage, error) {
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("generator: encode execution result: %w", err)
	}
	return encoded, nil
}

var _ Executor = (*executor)(nil)
