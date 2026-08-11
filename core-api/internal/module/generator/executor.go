package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
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
	CreateRecord(context.Context, *assetdomain.AssetRecord) (*assetdomain.AssetRecord, error)
	UpdateAnimationFrames(context.Context, uint, uint, []assetdomain.Frame) error
	GetDetail(context.Context, uint) (assetdomain.Asset, error)
}

// ReferenceStore is the object-storage boundary used by the worker. It keeps
// storage-specific credentials and URL formats out of generation logic.
type ReferenceStore interface {
	ResolveReference(context.Context, string) (string, error)
	PersistReference(context.Context, string) (string, error)
	NewObjectKey(string) (string, error)
	PersistReferenceAt(context.Context, string, string) error
}

type executor struct {
	images     imageclient.ImageGenerationService
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

type ExecutionResult struct {
	AssetID     uint `json:"asset_id"`
	AnimationID uint `json:"animation_id,omitempty"`
	Version     uint `json:"version,omitempty"`
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
	if e.processor == nil && (taskType == GenerateCharacterProtoType || taskType == EditCharacterProtoType || taskType == GenerateObjectProtoType || taskType == GenerateAnimation) {
		return nil, ErrImageProcessorRequired
	}

	switch taskType {
	case GenerateCharacterProtoType:
		request := CreateCharacterPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateCharacterPrototype(ctx, request)
	case EditCharacterProtoType:
		request := EditCharacterPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.editCharacterPrototype(ctx, request)
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
	directionCount := perspective.CharacterDirectionCount()
	resources, err := e.generatePrototypeResources(
		ctx,
		GenerateCharacterProtoType,
		prompts.CharacterPrototype(
			payload.CreativeBrief,
			payload.Perspective,
			prompts.SolidMatteBackground(imageprocessor.DefaultMatteColor),
		),
		payload.Dimensions,
		directionCount,
		referenceImages(payload.Reference),
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
		payload.Dimensions,
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

func (e *executor) editCharacterPrototype(
	ctx context.Context,
	payload EditCharacterPrototypePayload,
) (json.RawMessage, error) {
	asset, err := e.assets.GetDetail(ctx, payload.AssetID)
	if err != nil {
		return nil, fmt.Errorf("generator: load character asset %d: %w", payload.AssetID, err)
	}
	if asset.ID == 0 {
		return nil, fmt.Errorf("generator: character asset %d not found", payload.AssetID)
	}
	if asset.Type != assetdomain.AssetTypeCharacter {
		return nil, fmt.Errorf("generator: character prototype edit is unsupported for asset type %q", asset.Type)
	}
	if !asset.Perspective.Valid() {
		return nil, fmt.Errorf("generator: invalid perspective %q", asset.Perspective)
	}
	var dimensions assetdomain.Size
	if err := json.Unmarshal(asset.Dimensions, &dimensions); err != nil {
		return nil, fmt.Errorf("generator: decode asset %d dimensions: %w", asset.ID, err)
	}
	if err := assetdomain.ValidateDimensions(asset.Type, asset.Dimensions); err != nil {
		return nil, err
	}
	content, err := asset.DecodeContent()
	if err != nil {
		return nil, fmt.Errorf("generator: decode character asset %d content: %w", asset.ID, err)
	}
	originalReferences, err := prototypeReferences(content.Prototype)
	if err != nil {
		return nil, fmt.Errorf("generator: load character asset %d prototype: %w", asset.ID, err)
	}
	directionCount := asset.Perspective.CharacterDirectionCount()
	resources, err := e.generatePrototypeResources(
		ctx,
		EditCharacterProtoType,
		prompts.EditCharacterPrototype(
			asset.Description,
			payload.EditInstructions,
			string(asset.Perspective),
			uint(len(originalReferences)),
			prompts.SolidMatteBackground(imageprocessor.DefaultMatteColor),
		),
		dimensions,
		directionCount,
		originalReferences,
	)
	if err != nil {
		return nil, err
	}
	prototype := assetdomain.Prototype(resources)
	content.Prototype = &prototype
	content.DirectionCount = directionCount
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		return nil, fmt.Errorf("generator: encode edited character asset %d content: %w", asset.ID, err)
	}
	record, err := e.assets.CreateRecord(ctx, &assetdomain.AssetRecord{
		AssetID: asset.ID,
		Content: encoded,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: create character asset %d version: %w", asset.ID, err)
	}
	if record == nil || record.Version == 0 {
		return nil, fmt.Errorf("generator: create character asset %d version: empty result", asset.ID)
	}
	return encodeExecutionResult(ExecutionResult{AssetID: asset.ID, Version: record.Version})
}

func (e *executor) generateObjectPrototype(
	ctx context.Context,
	payload CreateObjectPrototypePayload,
) (json.RawMessage, error) {
	perspective, err := parsePerspective(payload.Perspective)
	if err != nil {
		return nil, err
	}
	directionCount := perspective.CharacterDirectionCount()
	resources, err := e.generatePrototypeResources(
		ctx,
		GenerateObjectProtoType,
		prompts.ObjectPrototype(
			payload.CreativeBrief,
			payload.Perspective,
			prompts.SolidMatteBackground(imageprocessor.DefaultMatteColor),
		),
		payload.Dimensions,
		directionCount,
		referenceImages(payload.Reference),
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
		payload.Dimensions,
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
	taskType TaskType,
	assetID uint,
	name string,
	prompt string,
) (json.RawMessage, error) {
	asset, err := e.assets.GetDetail(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("generator: load asset %d dimensions: %w", assetID, err)
	}
	if asset.Type != assetdomain.AssetTypeCharacter && asset.Type != assetdomain.AssetTypeObject {
		return nil, fmt.Errorf("generator: animation is unsupported for asset type %q", asset.Type)
	}
	var dimensions assetdomain.Size
	if err := json.Unmarshal(asset.Dimensions, &dimensions); err != nil {
		return nil, fmt.Errorf("generator: decode asset %d dimensions: %w", assetID, err)
	}
	if err := assetdomain.ValidateDimensions(asset.Type, asset.Dimensions); err != nil {
		return nil, err
	}
	generated, err := e.generateImages(ctx, taskType, prompt, dimensions, "")
	if err != nil {
		return nil, err
	}
	frames, err := e.animationFrames(ctx, generated)
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
	if err := e.assets.UpdateAnimationFrames(ctx, assetID, animationID, frames); err != nil {
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
	dimensions assetdomain.Size,
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
	})
	if err != nil {
		return nil, fmt.Errorf("generator: generate %s images: %w", taskType, err)
	}
	if result == nil || len(result.Images) == 0 {
		return nil, fmt.Errorf("generator: generate %s images: %w", taskType, ErrImageResultRequired)
	}
	if taskType != GenerateCharacterProtoType && taskType != GenerateObjectProtoType && taskType != GenerateAnimation {
		return result, nil
	}
	if dimensions.Width == 0 || dimensions.Height == 0 {
		return nil, fmt.Errorf("generator: process %s images: dimensions must be positive", taskType)
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
			Options:     imageprocessor.DefaultResizeOptions(int(dimensions.Width), int(dimensions.Height)),
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

func (e *executor) generatePrototypeResources(
	ctx context.Context,
	taskType TaskType,
	prompt string,
	dimensions assetdomain.Size,
	directionCount uint,
	references []string,
) ([]assetdomain.ImageResource, error) {
	if directionCount == 0 {
		return nil, fmt.Errorf("generator: prototype direction count must be positive")
	}
	if dimensions.Width == 0 || dimensions.Height == 0 {
		return nil, fmt.Errorf("generator: process %s images: dimensions must be positive", taskType)
	}
	columns, rows, err := directionGrid(directionCount)
	if err != nil {
		return nil, err
	}
	resolvedReferences, err := e.resolveReferences(ctx, taskType, references)
	if err != nil {
		return nil, err
	}
	result, err := e.images.Generate(ctx, &imageclient.GenerateRequest{
		Prompt:          prompt,
		ReferenceImages: resolvedReferences,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: generate %s images: %w", taskType, err)
	}
	if result == nil || len(result.Images) == 0 {
		return nil, fmt.Errorf("generator: generate %s images: %w", taskType, ErrImageResultRequired)
	}
	if len(result.Images) != 1 {
		return nil, fmt.Errorf("generator: generate %s images: expected one direction sheet, got %d", taskType, len(result.Images))
	}

	backgroundRemoved, err := e.processor.RemoveBackground(ctx, &imageprocessor.RemoveBackgroundRequest{
		ImageBase64: result.Images[0].Base64,
		MatteColor:  imageprocessor.DefaultMatteColor,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: remove %s background: %w", taskType, err)
	}
	if backgroundRemoved == nil || backgroundRemoved.ImageBase64 == "" {
		return nil, fmt.Errorf("generator: remove %s background: empty result", taskType)
	}
	split, err := e.processor.SplitImage(ctx, &imageprocessor.SplitImageRequest{
		ImageBase64:           backgroundRemoved.ImageBase64,
		Mode:                  imageprocessor.ImageSplitModeGrid,
		Columns:               columns,
		Rows:                  rows,
		ForceProportionalGrid: true,
		CropToContent:         true,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: split %s direction sheet: %w", taskType, err)
	}
	if split == nil || len(split.Regions) != int(directionCount) {
		got := 0
		if split != nil {
			got = len(split.Regions)
		}
		return nil, fmt.Errorf("generator: split %s direction sheet: got %d regions, want %d", taskType, got, directionCount)
	}

	var baseKey string
	if e.references != nil {
		baseKey, err = e.references.NewObjectKey("image/png")
		if err != nil {
			return nil, fmt.Errorf("generator: allocate %s image key: %w", taskType, err)
		}
	}
	resources := make([]assetdomain.ImageResource, 0, len(split.Regions))
	for index, region := range split.Regions {
		if region.ImageBase64 == "" {
			return nil, fmt.Errorf("generator: split %s direction %d is empty", taskType, index)
		}
		unprocessedURL := generatedImageDataURL(imageclient.GeneratedImage{
			Base64:    region.ImageBase64,
			MediaType: region.MIMEType,
		})
		finalKey := ""
		if e.references != nil {
			finalKey = addObjectKeySuffix(baseKey, fmt.Sprintf("-%d", index))
			if err := e.references.PersistReferenceAt(
				ctx,
				addObjectKeySuffix(finalKey, "-unprocessed"),
				unprocessedURL,
			); err != nil {
				return nil, fmt.Errorf("generator: persist %s direction %d unprocessed image: %w", taskType, index, err)
			}
		}

		resized, err := e.processor.Resize(ctx, &imageprocessor.ResizeRequest{
			ImageBase64: region.ImageBase64,
			Options:     imageprocessor.DefaultResizeOptions(int(dimensions.Width), int(dimensions.Height)),
		})
		if err != nil {
			return nil, fmt.Errorf("generator: resize %s direction %d image: %w", taskType, index, err)
		}
		if resized == nil || resized.ImageBase64 == "" {
			return nil, fmt.Errorf("generator: resize %s direction %d image: empty result", taskType, index)
		}
		finalURL := generatedImageDataURL(imageclient.GeneratedImage{
			Base64:    resized.ImageBase64,
			MediaType: resized.MIMEType,
		})
		if e.references != nil {
			if err := e.references.PersistReferenceAt(ctx, finalKey, finalURL); err != nil {
				return nil, fmt.Errorf("generator: persist %s direction %d image: %w", taskType, index, err)
			}
			finalURL = finalKey
		}
		resources = append(resources, assetdomain.ImageResource{
			ID:  uint(index + 1),
			URL: &finalURL,
		})
	}
	return resources, nil
}

func (e *executor) resolveReferences(
	ctx context.Context,
	taskType TaskType,
	references []string,
) ([]string, error) {
	resolved := append([]string(nil), references...)
	if e.references == nil {
		return resolved, nil
	}
	for index, reference := range resolved {
		value, err := e.references.ResolveReference(ctx, reference)
		if err != nil {
			return nil, fmt.Errorf("generator: resolve %s reference %d: %w", taskType, index+1, err)
		}
		resolved[index] = value
	}
	return resolved, nil
}

func prototypeReferences(prototype *assetdomain.Prototype) ([]string, error) {
	if prototype == nil || len(*prototype) == 0 {
		return nil, fmt.Errorf("prototype images are required")
	}
	references := make([]string, len(*prototype))
	for index, resource := range *prototype {
		if resource.URL == nil || *resource.URL == "" {
			return nil, fmt.Errorf("prototype image %d URL is required", index+1)
		}
		references[index] = *resource.URL
	}
	return references, nil
}

func referenceImages(reference string) []string {
	if reference == "" {
		return nil
	}
	return []string{reference}
}

func directionGrid(directionCount uint) (int, int, error) {
	switch directionCount {
	case 2:
		return 2, 1, nil
	case 4:
		return 2, 2, nil
	case 8:
		return 4, 2, nil
	default:
		return 0, 0, fmt.Errorf("generator: unsupported prototype direction count %d", directionCount)
	}
}

func addObjectKeySuffix(objectKey, suffix string) string {
	lastSlash := strings.LastIndex(objectKey, "/")
	lastDot := strings.LastIndex(objectKey, ".")
	if lastDot <= lastSlash {
		return objectKey + suffix
	}
	return objectKey[:lastDot] + suffix + objectKey[lastDot:]
}

func newPrototypeAsset(
	assetType assetdomain.AssetType,
	name string,
	projectID uint,
	description string,
	perspective assetdomain.Perspective,
	dimensions assetdomain.Size,
	directionCount uint,
	prototype []assetdomain.ImageResource,
) (*assetdomain.Asset, error) {
	content := assetdomain.NewAssetContent(assetType)
	prototypeValue := assetdomain.Prototype(prototype)
	content.Prototype = &prototypeValue
	content.DirectionCount = directionCount
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		return nil, fmt.Errorf("generator: encode prototype asset content: %w", err)
	}
	dimensionsValue, err := json.Marshal(dimensions)
	if err != nil {
		return nil, fmt.Errorf("generator: encode prototype asset dimensions: %w", err)
	}
	return &assetdomain.Asset{
		Name:        name,
		ProjectID:   projectID,
		Type:        assetType,
		Description: description,
		Perspective: perspective,
		Dimensions:  dimensionsValue,
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

func (e *executor) animationFrames(
	ctx context.Context,
	result *imageclient.GenerateResult,
) ([]assetdomain.Frame, error) {
	frames := make([]assetdomain.Frame, len(result.Images))
	for index, image := range result.Images {
		url := generatedImageDataURL(image)
		if e.references != nil {
			objectKey, err := e.references.PersistReference(ctx, url)
			if err != nil {
				return nil, fmt.Errorf("generator: persist animation frame %d: %w", index+1, err)
			}
			url = objectKey
		}
		frames[index] = assetdomain.Frame{ID: uint(index + 1), URL: &url}
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
