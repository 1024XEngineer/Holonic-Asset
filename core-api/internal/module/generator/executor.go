package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
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
	GetDetail(context.Context, uint) (assetdomain.Asset, error)
	CreateCharacterAsset(context.Context, *assetdomain.Asset) (*assetdomain.Asset, error)
	CreateObjectAsset(context.Context, *assetdomain.Asset) (uint, error)
	CreateAnimation(context.Context, uint, string, []assetdomain.Frame) (uint, error)
	UpdatePrototypeImages(context.Context, uint, []assetdomain.ImageResource) error
	CreateRecord(context.Context, *assetdomain.AssetRecord) (*assetdomain.AssetRecord, error)
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
	Version     uint `json:"version,omitempty"`
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

func (e *executor) editObjectPrototype(
	ctx context.Context,
	payload EditObjectPrototypePayload,
) (json.RawMessage, error) {
	asset, err := e.assets.GetDetail(ctx, payload.AssetID)
	if err != nil {
		return nil, fmt.Errorf("generator: load object asset %d: %w", payload.AssetID, err)
	}
	if asset.ID == 0 {
		return nil, fmt.Errorf("generator: object asset %d not found", payload.AssetID)
	}
	if asset.Type != assetdomain.AssetTypeObject {
		return nil, fmt.Errorf("generator: object prototype edit is unsupported for asset type %q", asset.Type)
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
		return nil, fmt.Errorf("generator: decode object asset %d content: %w", asset.ID, err)
	}
	originalReferences, err := prototypeReferences(content.Prototype)
	if err != nil {
		return nil, fmt.Errorf("generator: load object asset %d prototype: %w", asset.ID, err)
	}
	directionCount := asset.Perspective.CharacterDirectionCount()
	resources, err := e.generatePrototypeResources(
		ctx,
		EditObjectProtoType,
		prompts.EditObjectPrototype(
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
		return nil, fmt.Errorf("generator: encode edited object asset %d content: %w", asset.ID, err)
	}
	record, err := e.assets.CreateRecord(ctx, &assetdomain.AssetRecord{
		AssetID: asset.ID,
		Content: encoded,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: create object asset %d version: %w", asset.ID, err)
	}
	if record == nil || record.Version == 0 {
		return nil, fmt.Errorf("generator: create object asset %d version: empty result", asset.ID)
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
	frames, err := e.persistAnimationFrames(ctx, generated)
	if err != nil {
		return nil, err
	}
	animationID, err := e.assets.CreateAnimation(ctx, payload.AssetID, animationName, frames)
	if err != nil {
		return nil, fmt.Errorf("generator: create animation for asset %d: %w", payload.AssetID, err)
	}
	if animationID == 0 {
		return nil, fmt.Errorf("generator: create animation for asset %d: empty result", payload.AssetID)
	}
	return encodeExecutionResult(ExecutionResult{AssetID: payload.AssetID, AnimationID: animationID})
}

func animationReference(asset assetdomain.Asset, direction string) (string, bool, error) {
	if asset.Type != assetdomain.AssetTypeCharacter && asset.Type != assetdomain.AssetTypeObject {
		return "", false, fmt.Errorf("generator: asset type %q does not support animation generation", asset.Type)
	}
	// Select the requested direction and resolve its -unprocessed image-hosting
	// reference.
	content, err := asset.DecodeContent()
	if err != nil {
		return "", false, fmt.Errorf("generator: decode animation asset %d content: %w", asset.ID, err)
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
	references, err = e.resolveReferences(ctx, taskType, references)
	if err != nil {
		return nil, err
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
func parsePerspective(perspective string) (assetdomain.Perspective, error) {
	value := assetdomain.Perspective(strings.TrimSpace(perspective))
	if !value.Valid() {
		return "", fmt.Errorf("generator: invalid perspective %q", perspective)
	}
	return value, nil
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

func (e *executor) persistAnimationFrames(
	ctx context.Context,
	result *AnimationGenerationResult,
) ([]assetdomain.Frame, error) {
	if e.references == nil {
		return nil, ErrAnimationReferenceStoreRequired
	}
	frames := make([]assetdomain.Frame, len(result.Frames))
	for index, frame := range result.Frames {
		mediaType := strings.TrimSpace(frame.MIMEType)
		if mediaType == "" {
			mediaType = "image/png"
		}
		dataURL := "data:" + mediaType + ";base64," + frame.ImageBase64
		objectKey, err := e.references.PersistReference(ctx, dataURL)
		if err != nil {
			return nil, fmt.Errorf("generator: persist animation frame %d: %w", index+1, err)
		}
		objectKey = strings.TrimSpace(objectKey)
		if objectKey == "" {
			return nil, fmt.Errorf("generator: persist animation frame %d: empty object key", index+1)
		}
		if strings.HasPrefix(objectKey, "data:") ||
			strings.HasPrefix(objectKey, "http://") ||
			strings.HasPrefix(objectKey, "https://") {
			return nil, fmt.Errorf("generator: persist animation frame %d: storage returned a non-object-key reference", index+1)
		}
		frames[index] = assetdomain.Frame{
			ID:       uint(index + 1),
			URL:      &objectKey,
			Duration: result.FrameDurationMS,
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
