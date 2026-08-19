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
			prompts.AdaptiveMatteBackground(),
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
			prompts.AdaptiveMatteBackground(),
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
	}, asset.Version)
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
			prompts.AdaptiveMatteBackground(),
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
		MaxAttempts:     3,
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
		MatteColor:  "auto",
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
			// Prototype cells share the padded canonical-frame contract used by
			// animation references and final animation frames. This leaves room
			// for large animation poses without making the prototype appear
			// larger at the same nominal dimensions.
			Options: imageprocessor.AnimationFrameResizeOptions(
				int(dimensions.Width),
				int(dimensions.Height),
			),
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
			prompts.AdaptiveMatteBackground(),
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
	}, asset.Version)
	if err != nil {
		return nil, fmt.Errorf("generator: create object asset %d version: %w", asset.ID, err)
	}
	if record == nil || record.Version == 0 {
		return nil, fmt.Errorf("generator: create object asset %d version: empty result", asset.ID)
	}
	return encodeExecutionResult(ExecutionResult{AssetID: asset.ID, Version: record.Version})
}
