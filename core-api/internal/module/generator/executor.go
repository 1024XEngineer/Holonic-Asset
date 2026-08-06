package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

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
	CreateCharacterAsset(context.Context, *assetdomain.Asset) (*assetdomain.Asset, error)
	CreateObjectAsset(context.Context, *assetdomain.Asset) (uint, error)
	CreateAnimation(context.Context, uint, string) (uint, error)
	UpdatePrototypeImages(context.Context, uint, []assetdomain.ImageResource) error
	UpdateAnimationFrames(context.Context, uint, uint, []assetdomain.Frame) error
}

type executor struct {
	images    imageclient.ImageGenerationService
	processor imageprocessor.Processor
	assets    AssetWriter
}

// NewExecutor creates the image-to-asset workflow used by task handlers.
func NewExecutor(
	images imageclient.ImageGenerationService,
	processor imageprocessor.Processor,
	assets AssetWriter,
) Executor {
	return &executor{images: images, processor: processor, assets: assets}
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
	if e.images == nil {
		return nil, ErrImageServiceRequired
	}
	if e.assets == nil {
		return nil, ErrAssetWriterRequired
	}
	if e.processor == nil && (taskType == GenerateCharacterProtoType || taskType == GenerateObjectProtoType) {
		return nil, ErrImageProcessorRequired
	}

	switch taskType {
	case GenerateCharacterProtoType:
		request := CreateCharacterPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateCharacterPrototype(ctx, request)
	case GenerateObjectProtoType:
		request := CreateObjectPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateObjectPrototype(ctx, request)
	case GenerateAnimation:
		request := CreateAnimationPayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateAnimation(ctx, taskType, request.ParentID, request.AssetName, request.CreativeBrief)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedTaskType, taskType)
	}
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
	taskType TaskType,
	assetID uint,
	name string,
	prompt string,
) (json.RawMessage, error) {
	generated, err := e.generateImages(ctx, taskType, prompt, "", "")
	if err != nil {
		return nil, err
	}

	animationID, err := e.assets.CreateAnimation(ctx, assetID, name)
	if err != nil {
		return nil, fmt.Errorf("generator: create animation for asset %d: %w", assetID, err)
	}
	if animationID == 0 {
		return nil, fmt.Errorf("generator: create animation for asset %d: empty result", assetID)
	}
	if err := e.assets.UpdateAnimationFrames(ctx, assetID, animationID, animationFrames(generated)); err != nil {
		return nil, fmt.Errorf(
			"generator: update asset %d animation %d frames: %w",
			assetID,
			animationID,
			err,
		)
	}
	return encodeExecutionResult(ExecutionResult{AssetID: assetID, AnimationID: animationID})
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

func animationFrames(result *imageclient.GenerateResult) []assetdomain.Frame {
	frames := make([]assetdomain.Frame, len(result.Images))
	for index, image := range result.Images {
		url := generatedImageDataURL(image)
		frames[index] = assetdomain.Frame{ID: uint(index + 1), URL: &url}
	}
	return frames
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
