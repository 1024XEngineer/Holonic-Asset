package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
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
	images imageclient.ImageGenerationService
	assets AssetWriter
}

// NewExecutor creates the image-to-asset workflow used by task handlers.
// Image processing is intentionally deferred; generated images are stored as
// data URLs until a processor and object-storage flow are available.
func NewExecutor(
	images imageclient.ImageGenerationService,
	assets AssetWriter,
) Executor {
	return &executor{images: images, assets: assets}
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
	case GenerateCharacterAnimation:
		request := CreateCharacterAnimationPayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateAnimation(ctx, taskType, request.ParentID, request.AssetName, request.CreativeBrief)
	case GenerateObjectAnimation:
		request := CreateObjectAnimationPayload{}
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
	value, err := newPrototypeAsset(
		assetdomain.AssetTypeCharacter,
		payload.AssetName,
		payload.ProjectID,
		payload.CreativeBrief,
		payload.Perspective,
		payload.DirectionCount,
	)
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
	created, err := e.assets.CreateCharacterAsset(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("generator: create character asset: %w", err)
	}
	if created == nil || created.ID == 0 {
		return nil, fmt.Errorf("generator: create character asset: empty result")
	}
	if err := e.assets.UpdatePrototypeImages(ctx, created.ID, prototypeResources(generated)); err != nil {
		return nil, fmt.Errorf("generator: update character asset %d prototype: %w", created.ID, err)
	}
	return encodeExecutionResult(ExecutionResult{AssetID: created.ID})
}

func (e *executor) generateObjectPrototype(
	ctx context.Context,
	payload CreateObjectPrototypePayload,
) (json.RawMessage, error) {
	value, err := newPrototypeAsset(
		assetdomain.AssetTypeObject,
		payload.AssetName,
		payload.ProjectID,
		payload.CreativeBrief,
		payload.Perspective,
		"",
	)
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
	assetID, err := e.assets.CreateObjectAsset(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("generator: create object asset: %w", err)
	}
	if assetID == 0 {
		return nil, fmt.Errorf("generator: create object asset: empty result")
	}
	if err := e.assets.UpdatePrototypeImages(ctx, assetID, prototypeResources(generated)); err != nil {
		return nil, fmt.Errorf("generator: update object asset %d prototype: %w", assetID, err)
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
	return result, nil
}

func newPrototypeAsset(
	assetType assetdomain.AssetType,
	name string,
	projectID uint,
	description string,
	perspective string,
	directionCount string,
) (*assetdomain.Asset, error) {
	content := assetdomain.NewAssetContent(assetType)
	content.ViewMode = viewMode(perspective)
	if directionCount != "" {
		value, err := strconv.ParseUint(directionCount, 10, 0)
		if err != nil {
			return nil, fmt.Errorf("generator: parse direction count %q: %w", directionCount, err)
		}
		content.DirectionCount = uint(value)
	}
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

func viewMode(perspective string) assetdomain.ViewMode {
	normalized := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToLower(strings.TrimSpace(perspective)))
	switch normalized {
	case "side_on", "sideon":
		return assetdomain.ViewModeSideOn
	case "top_down", "topdown":
		return assetdomain.ViewModeTopDown
	default:
		return ""
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
