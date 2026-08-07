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

// ReferenceStore is the object-storage boundary used by the worker. It keeps
// storage-specific credentials and URL formats out of generation logic.
type ReferenceStore interface {
	ResolveReference(context.Context, string) (string, error)
	PersistReference(context.Context, string) (string, error)
}

type executor struct {
	images     imageclient.ImageGenerationService
	animations AnimationGenerationService
	processor  imageprocessor.Processor
	assets     AssetWriter
	references ReferenceStore
}

// NewExecutor creates the image-to-asset workflow used by task handlers.
func NewExecutor(
	images imageclient.ImageGenerationService,
	processor imageprocessor.Processor,
	assets AssetWriter,
	references ...ReferenceStore,
) Executor {
	var referenceStore ReferenceStore
	if len(references) > 0 {
		referenceStore = references[0]
	}
	return &executor{
		images: images, processor: processor, assets: assets, references: referenceStore,
	}
}

// NewExecutorWithAnimation creates the complete generation workflow, including
// image-to-video animation generation. NewExecutor remains available for
// prototype-only callers and tests that do not need animation generation.
func NewExecutorWithAnimation(
	images imageclient.ImageGenerationService,
	animations AnimationGenerationService,
	processor imageprocessor.Processor,
	assets AssetWriter,
	references ...ReferenceStore,
) Executor {
	var referenceStore ReferenceStore
	if len(references) > 0 {
		referenceStore = references[0]
	}
	return &executor{
		images: images, animations: animations, processor: processor, assets: assets, references: referenceStore,
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
	resources, err := e.prototypeResources(ctx, generated)
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
		resources,
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
	directionCount, err := parseDirectionCount(payload.DirectionCount)
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
	resources, err := e.prototypeResources(ctx, generated)
	if err != nil {
		return nil, err
	}
	value, err := newPrototypeAsset(
		assetdomain.AssetTypeObject,
		payload.AssetName,
		payload.ProjectID,
		payload.CreativeBrief,
		perspective,
		directionCount,
		resources,
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

func animationReference(asset assetdomain.Asset, direction string) (string, bool, error) {
	content, err := asset.DecodeContent()
	if err != nil {
		return "", false, fmt.Errorf("generator: decode animation asset %d content: %w", asset.ID, err)
	}

	if asset.Type != assetdomain.AssetTypeCharacter && asset.Type != assetdomain.AssetTypeObject {
		return "", false, fmt.Errorf("generator: asset type %q does not support animation generation", asset.Type)
	}
	prototypeIndex, err := animationDirectionIndex(direction, content.DirectionCount)
	if err != nil {
		return "", false, err
	}

	if content.Prototype == nil || prototypeIndex >= len(*content.Prototype) {
		return "", false, fmt.Errorf("generator: animation asset %d has no prototype for direction %q", asset.ID, direction)
	}
	prototype := (*content.Prototype)[prototypeIndex]
	if prototype.URL == nil || strings.TrimSpace(*prototype.URL) == "" {
		return "", false, fmt.Errorf("generator: animation asset %d prototype direction %q has no image URL", asset.ID, direction)
	}
	unprocessedURL := animationUnprocessedImageURL(strings.TrimSpace(*prototype.URL))
	return unprocessedURL, false, nil
}

func animationUnprocessedImageURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "data:") {
		return value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return addAnimationUnprocessedSuffix(value)
	}
	parsed.Path = addAnimationUnprocessedSuffix(parsed.Path)
	return parsed.String()
}

func addAnimationUnprocessedSuffix(path string) string {
	lastSlash := strings.LastIndex(path, "/")
	lastDot := strings.LastIndex(path, ".")
	if lastDot <= lastSlash {
		return path + "-unprocessed"
	}
	return path[:lastDot] + "-unprocessed" + path[lastDot:]
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
		if e.references != nil {
			resolved, err := e.references.ResolveReference(ctx, reference)
			if err != nil {
				return nil, fmt.Errorf("generator: resolve %s reference: %w", taskType, err)
			}
			reference = resolved
		}
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
		resizeOptions := imageprocessor.DefaultResizeOptions(resizeWidth, resizeHeight)
		if taskType == GenerateCharacterProtoType {
			// Character prototypes are the canonical scale contract for later
			// image-to-video generation. Keep a larger transparent safety area so
			// held props can move without forcing animation frames to shrink.
			resizeOptions = imageprocessor.AnimationFrameResizeOptions(resizeWidth, resizeHeight)
		}
		resized, resizeErr := e.processor.Resize(ctx, &imageprocessor.ResizeRequest{
			ImageBase64: imageBase64,
			Options:     resizeOptions,
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
	directionCount = strings.TrimSpace(directionCount)
	if directionCount == "" {
		return 0, fmt.Errorf("generator: direction count is required; use 2, 4, or 8")
	}
	value, err := strconv.ParseUint(directionCount, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("generator: parse direction count %q: %w", directionCount, err)
	}
	switch value {
	case 2, 4, 8:
		return uint(value), nil
	default:
		return 0, fmt.Errorf("generator: invalid direction count %q; use 2, 4, or 8", directionCount)
	}
}

func (e *executor) prototypeResources(
	ctx context.Context,
	result *imageclient.GenerateResult,
) ([]assetdomain.ImageResource, error) {
	resources := make([]assetdomain.ImageResource, len(result.Images))
	for index, image := range result.Images {
		url := generatedImageDataURL(image)
		if e.references != nil {
			objectKey, err := e.references.PersistReference(ctx, url)
			if err != nil {
				return nil, fmt.Errorf("generator: persist prototype image %d: %w", index+1, err)
			}
			url = objectKey
		}
		resources[index] = assetdomain.ImageResource{ID: uint(index + 1), URL: &url}
	}
	return resources, nil
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
