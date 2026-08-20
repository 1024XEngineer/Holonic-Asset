package generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const maxPrototypeReferenceBytes = 32 << 20

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
			payload.Dimensions,
			prompts.SolidMatteBackground(imageprocessor.DefaultMatteColor),
			prototypeReferenceState(payload.ProjectReference, payload.Reference),
		),
		payload.Dimensions,
		directionCount,
		referenceImages(payload.ProjectReference, payload.Reference),
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
			dimensions,
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
	candidate := assetdomain.AssetContent{
		DirectionCount: directionCount,
		Prototype:      &prototype,
	}
	encoded, err := assetdomain.EncodeContent(candidate)
	if err != nil {
		return nil, fmt.Errorf("generator: encode edited character asset %d content: %w", asset.ID, err)
	}
	return encodeExecutionResult(ExecutionResult{
		AssetID:            asset.ID,
		Version:            asset.Version,
		Content:            encoded,
		GeneratedResources: generatedPrototypeResourceKeys(resources),
	})
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
			payload.Dimensions,
			prompts.SolidMatteBackground(imageprocessor.DefaultMatteColor),
			prototypeReferenceState(payload.ProjectReference, payload.Reference),
		),
		payload.Dimensions,
		directionCount,
		referenceImages(payload.ProjectReference, payload.Reference),
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
		Params:          imageclient.Params{"quality": "high"},
		MaxAttempts:     3,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: generate %s images: %w", taskType, err)
	}
	generatedImage, err := singlePrototypeSheet(taskType, "generated", result)
	if err != nil {
		return nil, err
	}

	backgroundRemoved, err := e.processor.RemoveBackground(ctx, &imageprocessor.RemoveBackgroundRequest{
		ImageBase64:               generatedImage.Base64,
		MatteColor:                imageprocessor.DefaultMatteColor,
		AllowSampledMatteFallback: true,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: remove %s background: %w", taskType, err)
	}
	if backgroundRemoved == nil || backgroundRemoved.ImageBase64 == "" {
		return nil, fmt.Errorf("generator: remove %s background: empty result", taskType)
	}
	// Prototype directions are static views of one subject, not independent
	// component crops. Normalize the known grid as an animation sequence so
	// every direction shares one content scale and one centre anchor. Cropping
	// each cell independently makes a model-generated subject appear at a
	// different size (and preserves any off-centre placement) in each direction.
	split, err := e.processor.SplitImage(ctx, &imageprocessor.SplitImageRequest{
		ImageBase64:           backgroundRemoved.ImageBase64,
		Mode:                  imageprocessor.ImageSplitModeAnimation,
		Columns:               columns,
		Rows:                  rows,
		ForceProportionalGrid: true,
		FrameWidth:            int(dimensions.Width),
		FrameHeight:           int(dimensions.Height),
		Margin:                imageprocessor.AnimationFrameMargin(int(dimensions.Width), int(dimensions.Height)),
		Anchor:                imageprocessor.AnimationAnchorCenter,
		NormalizeContentScale: true,
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

		// Animation-mode splitting has already fixed the final frame geometry,
		// shared scale, and centre anchor. Run pixel cleanup on that exact canvas
		// without cropping, adding another margin, or changing pixel positions.
		resized, err := e.processor.Resize(ctx, &imageprocessor.ResizeRequest{
			ImageBase64: region.ImageBase64,
			Options: prototypePixelPostProcessOptions(
				taskType,
				int(dimensions.Width),
				int(dimensions.Height),
			),
		})
		if err != nil {
			return nil, fmt.Errorf("generator: pixel-process %s direction %d image: %w", taskType, index, err)
		}
		if resized == nil || resized.ImageBase64 == "" {
			return nil, fmt.Errorf("generator: pixel-process %s direction %d image: empty result", taskType, index)
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

func prototypePixelPostProcessOptions(taskType TaskType, width, height int) imageprocessor.ResizeOptions {
	var options imageprocessor.ResizeOptions
	switch taskType {
	case GenerateCharacterProtoType, EditCharacterProtoType:
		options = imageprocessor.CharacterPrototypePixelResizeOptions(width, height)
	default:
		options = imageprocessor.PrototypePixelResizeOptions(width, height)
	}
	// SplitImage already produced a canonical target-size frame with the shared
	// prototype/animation margin. Preserve that canvas exactly while applying
	// hard alpha, source-colour palette reduction, and block repair.
	options.Margin = 0
	options.CropContent = false
	return options
}

func singlePrototypeSheet(
	taskType TaskType,
	stage string,
	result *imageclient.GenerateResult,
) (imageclient.GeneratedImage, error) {
	if result == nil || len(result.Images) == 0 {
		return imageclient.GeneratedImage{}, fmt.Errorf(
			"generator: generate %s %s images: %w",
			taskType,
			stage,
			ErrImageResultRequired,
		)
	}
	if len(result.Images) != 1 {
		return imageclient.GeneratedImage{}, fmt.Errorf(
			"generator: generate %s %s images: expected one direction sheet, got %d",
			taskType,
			stage,
			len(result.Images),
		)
	}
	if result.Images[0].Base64 == "" {
		return imageclient.GeneratedImage{}, fmt.Errorf(
			"generator: generate %s %s images: %w",
			taskType,
			stage,
			ErrImageResultRequired,
		)
	}
	return result.Images[0], nil
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
	for index, reference := range resolved {
		value := reference
		if e.references != nil {
			var err error
			value, err = e.references.ResolveReference(ctx, reference)
			if err != nil {
				return nil, fmt.Errorf("generator: resolve %s reference %d: %w", taskType, index+1, err)
			}
		} else if isHTTPReference(reference) {
			return nil, fmt.Errorf(
				"generator: resolve %s reference %d: object-storage reference store is required for URL references",
				taskType,
				index+1,
			)
		}
		normalized, err := e.normalizePrototypeReference(ctx, value)
		if err != nil {
			return nil, fmt.Errorf("generator: normalize %s reference %d: %w", taskType, index+1, err)
		}
		resolved[index] = normalized
	}
	return resolved, nil
}

func isHTTPReference(reference string) bool {
	reference = strings.ToLower(strings.TrimSpace(reference))
	return strings.HasPrefix(reference, "http://") || strings.HasPrefix(reference, "https://")
}

func (e *executor) normalizePrototypeReference(ctx context.Context, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", fmt.Errorf("reference image is required")
	}

	imageBase64 := reference
	if isHTTPReference(reference) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, reference, nil)
		if err != nil {
			return "", fmt.Errorf("create reference download request: %w", err)
		}
		if err := validatePrototypeReferenceURL(request.URL); err != nil {
			return "", err
		}
		client := e.referenceHTTPClient
		if client == nil {
			client = newPrototypeReferenceHTTPClient()
		}
		response, err := client.Do(request)
		if err != nil {
			return "", fmt.Errorf("download reference: %w", err)
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
			return "", fmt.Errorf("download reference: HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, maxPrototypeReferenceBytes+1))
		if err != nil {
			return "", fmt.Errorf("read reference: %w", err)
		}
		if len(body) > maxPrototypeReferenceBytes {
			return "", fmt.Errorf("reference exceeds %d bytes", maxPrototypeReferenceBytes)
		}
		if len(body) == 0 {
			return "", fmt.Errorf("download reference: empty response")
		}
		imageBase64 = base64.StdEncoding.EncodeToString(body)
	} else if !strings.HasPrefix(strings.ToLower(reference), "data:image/") {
		// Non-URL values are storage/provider-specific references that cannot be
		// decoded locally. Preserve the legacy pass-through behavior.
		return reference, nil
	}

	normalized, err := e.processor.NormalizeReference(ctx, &imageprocessor.NormalizeReferenceRequest{
		ImageBase64: imageBase64,
	})
	if err != nil {
		return "", err
	}
	if normalized == nil || strings.TrimSpace(normalized.ImageBase64) == "" {
		return "", fmt.Errorf("empty normalized reference")
	}
	if !normalized.Report.Upscaled {
		return reference, nil
	}
	return generatedImageDataURL(imageclient.GeneratedImage{
		Base64: normalized.ImageBase64, MediaType: normalized.MIMEType,
	}), nil
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

func prototypeReferenceState(projectReference, userReference string) prompts.PrototypeReferenceState {
	return prompts.PrototypeReferenceState{
		HasProjectReference: strings.TrimSpace(projectReference) != "",
		HasUserReference:    strings.TrimSpace(userReference) != "",
	}
}

func referenceImages(references ...string) []string {
	result := make([]string, 0, len(references))
	for _, reference := range references {
		if strings.TrimSpace(reference) != "" {
			result = append(result, reference)
		}
	}
	return result
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
			dimensions,
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
	candidate := assetdomain.AssetContent{
		DirectionCount: directionCount,
		Prototype:      &prototype,
	}
	encoded, err := assetdomain.EncodeContent(candidate)
	if err != nil {
		return nil, fmt.Errorf("generator: encode edited object asset %d content: %w", asset.ID, err)
	}
	return encodeExecutionResult(ExecutionResult{
		AssetID:            asset.ID,
		Version:            asset.Version,
		Content:            encoded,
		GeneratedResources: generatedPrototypeResourceKeys(resources),
	})
}
