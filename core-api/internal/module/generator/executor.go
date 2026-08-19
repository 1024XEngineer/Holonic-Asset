package generator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
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
	DeleteObjects(context.Context, []string) error
}

// ResourceStore publishes generated resources under stable object keys and
// removes them when the enclosing asset workflow cannot be completed.
type ResourceStore interface {
	PutObject(context.Context, string, string, []byte) error
	DeleteObject(context.Context, string) error
}

type ExecutionResult struct {
	AssetID            uint            `json:"asset_id"`
	AnimationID        uint            `json:"animation_id,omitempty"`
	Version            uint            `json:"version,omitempty"`
	Content            json.RawMessage `json:"content,omitempty"`
	GeneratedResources []string        `json:"generated_resources,omitempty"`
}

// AssetWriter is the subset of Workspace asset operations used by generation.
type AssetWriter interface {
	GetDetail(context.Context, uint) (assetdomain.Asset, error)
	CreateCharacterAsset(context.Context, *assetdomain.Asset) (*assetdomain.Asset, error)
	CreateObjectAsset(context.Context, *assetdomain.Asset) (uint, error)
	CreateSceneryAsset(context.Context, *assetdomain.Asset) (uint, error)
	CreateTileSetAsset(context.Context, *assetdomain.Asset) (uint, error)
}

type executor struct {
	images     imageclient.ImageGenerationService
	llm        llmclient.LLMService
	animations AnimationGenerationService
	processor  imageprocessor.Processor
	assets     AssetWriter
	projects   ProjectReader
	references ReferenceStore
	resources  ResourceStore
	logger     logger.Logger
}

// ExecutorDependencies contains optional workflow integrations.
type ExecutorDependencies struct {
	References ReferenceStore
	Projects   ProjectReader
	LLM        llmclient.LLMService
	Animations AnimationGenerationService
	Resources  ResourceStore
	Logger     logger.Logger
}

// NewExecutorWithDependencies creates an executor with explicit optional
// workflow integrations.
func NewExecutorWithDependencies(
	images imageclient.ImageGenerationService,
	processor imageprocessor.Processor,
	assets AssetWriter,
	dependencies ExecutorDependencies,
) Executor {
	return &executor{
		images:     images,
		llm:        dependencies.LLM,
		animations: dependencies.Animations,
		processor:  processor,
		assets:     assets,
		projects:   dependencies.Projects,
		references: dependencies.References,
		resources:  dependencies.Resources,
		logger:     dependencies.Logger,
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
	case EditObjectProtoType:
		if err := e.requirePrototypeDependencies(); err != nil {
			return nil, err
		}
		request := EditObjectPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.editObjectPrototype(ctx, request)
	case GenerateObjectProtoType:
		if err := e.requirePrototypeDependencies(); err != nil {
			return nil, err
		}
		request := CreateObjectPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateObjectPrototype(ctx, request)
	case EditFrames:
		if e.assets == nil {
			return nil, ErrAssetWriterRequired
		}
		if e.animations == nil {
			return nil, ErrAnimationServiceRequired
		}
		if e.references == nil {
			return nil, ErrAnimationReferenceStoreRequired
		}
		request := EditFramesPayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.editFrames(ctx, request)
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
	case GenerateScenery:
		if err := e.requireSceneryDependencies(); err != nil {
			return nil, err
		}
		request := CreateSceneryPayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		if err := validateSceneryPayload(request); err != nil {
			return nil, err
		}
		return e.generateScenery(ctx, request)
	case EditAnimation:
		if e.assets == nil {
			return nil, ErrAssetWriterRequired
		}
		if e.animations == nil {
			return nil, ErrAnimationServiceRequired
		}
		if e.references == nil {
			return nil, ErrAnimationReferenceStoreRequired
		}
		request := EditAnimationPayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.editAnimation(ctx, request)
	case GenerateTileSet:
		if err := e.requireTileSetDependencies(); err != nil {
			return nil, err
		}
		request := CreateTileSetPayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateTileSet(ctx, request)
	case EditTiles:
		if err := e.requireTileSetDependencies(); err != nil {
			return nil, err
		}
		request := EditTilesPayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.editTileSetTiles(ctx, request)
	case EditTilesetItem:
		if err := e.requireTileSetDependencies(); err != nil {
			return nil, err
		}
		request := EditTilesetItemPayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.editTileSetItem(ctx, request)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedTaskType, taskType)
	}
}

func (e *executor) requireSceneryDependencies() error {
	if e.images == nil {
		return ErrImageServiceRequired
	}
	if e.assets == nil {
		return ErrAssetWriterRequired
	}
	if e.processor == nil {
		return ErrImageProcessorRequired
	}
	if e.llm == nil {
		return ErrLLMServiceRequired
	}
	if e.resources == nil {
		return ErrResourceStoreRequired
	}
	return nil
}

func (e *executor) requireTileSetDependencies() error {
	if e.images == nil {
		return ErrImageServiceRequired
	}
	if e.processor == nil {
		return ErrImageProcessorRequired
	}
	if e.assets == nil {
		return ErrAssetWriterRequired
	}
	if e.projects == nil {
		return fmt.Errorf("generator: project reader is required for Tileset execution")
	}
	if e.references == nil {
		return fmt.Errorf("generator: reference store is required for Tileset execution")
	}
	return nil
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

func generatedPrototypeResourceKeys(resources []assetdomain.ImageResource) []string {
	keys := make([]string, 0, len(resources)*2)
	for _, resource := range resources {
		if resource.URL == nil || *resource.URL == "" {
			continue
		}
		keys = append(keys, *resource.URL, addObjectKeySuffix(*resource.URL, "-unprocessed"))
	}
	return keys
}

func generatedFrameResourceKeys(frames []assetdomain.Frame) []string {
	keys := make([]string, 0, len(frames))
	for _, frame := range frames {
		if frame.URL != nil && *frame.URL != "" {
			keys = append(keys, *frame.URL)
		}
	}
	return keys
}

var _ Executor = (*executor)(nil)
