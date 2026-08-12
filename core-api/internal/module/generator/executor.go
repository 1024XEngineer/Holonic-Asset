package generator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

// Executor executes queued generation workflows.
type Executor interface {
	Generate(context.Context, TaskType, json.RawMessage) (json.RawMessage, error)
}

// ReferenceStore is the object-storage boundary shared by run preparation and
// generation execution.
type ReferenceStore interface {
	ResolveReference(context.Context, string) (string, error)
	PersistReference(context.Context, string) (string, error)
	NewObjectKey(string) (string, error)
	PersistReferenceAt(context.Context, string, string) error
}

type ExecutionResult struct {
	AssetID     uint `json:"asset_id"`
	AnimationID uint `json:"animation_id,omitempty"`
	Version     uint `json:"version,omitempty"`
}

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
	case EditCharacterProtoType:
		if err := e.requirePrototypeDependencies(); err != nil {
			return nil, err
		}
		request := EditCharacterPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.editCharacterPrototype(ctx, request)
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
		if e.references == nil {
			return nil, ErrAnimationReferenceStoreRequired
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
