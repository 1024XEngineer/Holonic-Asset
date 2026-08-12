package executor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

// AssetWriter is the subset of Workspace asset operations used by generation.
type AssetWriter interface {
	GetDetail(context.Context, uint) (assetdomain.Asset, error)
	CreateCharacterAsset(context.Context, *assetdomain.Asset) (*assetdomain.Asset, error)
	CreateObjectAsset(context.Context, *assetdomain.Asset) (uint, error)
	CreateAnimation(context.Context, uint, string, []assetdomain.Frame) (uint, error)
	CreateRecord(context.Context, *assetdomain.AssetRecord) (*assetdomain.AssetRecord, error)
}

type executor struct {
	images     imageclient.ImageGenerationService
	animations AnimationGenerationService
	processor  imageprocessor.Processor
	assets     AssetWriter
	references generator.ReferenceStore
}

// NewExecutor creates the image-to-asset workflow used by task handlers.
func NewExecutor(
	images imageclient.ImageGenerationService,
	processor imageprocessor.Processor,
	assets AssetWriter,
	references ...generator.ReferenceStore,
) generator.Executor {
	var referenceStore generator.ReferenceStore
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
	references ...generator.ReferenceStore,
) generator.Executor {
	var referenceStore generator.ReferenceStore
	if len(references) > 0 {
		referenceStore = references[0]
	}
	return &executor{
		images: images, animations: animations, processor: processor, assets: assets, references: referenceStore,
	}
}

func (e *executor) Generate(
	ctx context.Context,
	taskType generator.TaskType,
	payload json.RawMessage,
) (json.RawMessage, error) {
	switch taskType {
	case generator.GenerateCharacterProtoType:
		if err := e.requirePrototypeDependencies(); err != nil {
			return nil, err
		}
		request := generator.CreateCharacterPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateCharacterPrototype(ctx, request)
	case generator.EditCharacterProtoType:
		if err := e.requirePrototypeDependencies(); err != nil {
			return nil, err
		}
		request := generator.EditCharacterPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.editCharacterPrototype(ctx, request)
	case generator.GenerateObjectProtoType:
		if err := e.requirePrototypeDependencies(); err != nil {
			return nil, err
		}
		request := generator.CreateObjectPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateObjectPrototype(ctx, request)
	case generator.GenerateAnimation:
		if e.assets == nil {
			return nil, generator.ErrAssetWriterRequired
		}
		if e.animations == nil {
			return nil, generator.ErrAnimationServiceRequired
		}
		if e.references == nil {
			return nil, generator.ErrAnimationReferenceStoreRequired
		}
		request := generator.CreateAnimationPayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateAnimation(ctx, request)
	default:
		return nil, fmt.Errorf("%w: %q", generator.ErrUnsupportedTaskType, taskType)
	}
}

func (e *executor) requirePrototypeDependencies() error {
	if e.images == nil {
		return generator.ErrImageServiceRequired
	}
	if e.assets == nil {
		return generator.ErrAssetWriterRequired
	}
	if e.processor == nil {
		return generator.ErrImageProcessorRequired
	}
	return nil
}

func generatedImageDataURL(image imageclient.GeneratedImage) string {
	mediaType := image.MediaType
	if mediaType == "" {
		mediaType = "image/png"
	}
	return "data:" + mediaType + ";base64," + image.Base64
}

func decodeExecutionPayload(taskType generator.TaskType, payload json.RawMessage, target any) error {
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("generator: decode %s execution payload: %w", taskType, err)
	}
	return nil
}

func encodeExecutionResult(result generator.ExecutionResult) (json.RawMessage, error) {
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("generator: encode execution result: %w", err)
	}
	return encoded, nil
}

var _ generator.Executor = (*executor)(nil)
