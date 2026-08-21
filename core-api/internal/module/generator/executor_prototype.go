package generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const (
	minimumPrototypeSheetPixels    uint64 = 655_360
	maximumPrototypeSheetPixels    uint64 = 8_294_400
	maximumPrototypeSheetDimension uint64 = 3840
	prototypeSheetAlignment        uint64 = 16
	maximumPrototypeSheetAspect    uint64 = 3
	maximumPrototypeCandidates            = 3
	maxPrototypeReferenceBytes            = 32 << 20
)

type prototypeSheetSpec struct {
	Size               string
	GridBoundaryMargin int
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
			prompts.AdaptiveMatteBackground(),
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
			prompts.AdaptiveMatteBackground(),
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
	sheet, err := derivePrototypeSheetSpec(dimensions, columns, rows)
	if err != nil {
		return nil, fmt.Errorf("generator: derive %s direction sheet size: %w", taskType, err)
	}
	resolvedReferences, err := e.resolveReferences(ctx, taskType, references)
	if err != nil {
		return nil, err
	}
	var split *imageprocessor.SplitImageResult
	for candidate := 1; candidate <= maximumPrototypeCandidates; candidate++ {
		result, generateErr := e.images.Generate(ctx, &imageclient.GenerateRequest{
			Prompt:          prompt,
			ReferenceImages: resolvedReferences,
			Size:            sheet.Size,
			MaxAttempts:     3,
		})
		if generateErr != nil {
			return nil, fmt.Errorf("generator: generate %s images: %w", taskType, generateErr)
		}
		if result == nil || len(result.Images) == 0 {
			return nil, fmt.Errorf("generator: generate %s images: %w", taskType, ErrImageResultRequired)
		}
		if len(result.Images) != 1 {
			return nil, fmt.Errorf("generator: generate %s images: expected one direction sheet, got %d", taskType, len(result.Images))
		}
		backgroundRemoved, removeErr := e.processor.RemoveBackground(ctx, &imageprocessor.RemoveBackgroundRequest{
			ImageBase64: result.Images[0].Base64,
			MatteColor:  "auto",
		})
		if removeErr != nil {
			return nil, fmt.Errorf("generator: remove %s background: %w", taskType, removeErr)
		}
		if backgroundRemoved == nil || backgroundRemoved.ImageBase64 == "" {
			return nil, fmt.Errorf("generator: remove %s background: empty result", taskType)
		}
		// Prototype directions are static views of one subject, not independent
		// component crops. Animation mode keeps one content scale and centre anchor
		// while boundary validation still rejects unsafe generated sheets.
		split, err = e.processor.SplitImage(ctx, &imageprocessor.SplitImageRequest{
			ImageBase64:               backgroundRemoved.ImageBase64,
			Mode:                      imageprocessor.ImageSplitModeAnimation,
			Columns:                   columns,
			Rows:                      rows,
			ForceProportionalGrid:     true,
			FrameWidth:                int(dimensions.Width),
			FrameHeight:               int(dimensions.Height),
			Margin:                    imageprocessor.AnimationFrameMargin(int(dimensions.Width), int(dimensions.Height)),
			Anchor:                    imageprocessor.AnimationAnchorCenter,
			NormalizeContentScale:     true,
			RejectGridBoundaryContent: true,
			GridBoundaryMargin:        sheet.GridBoundaryMargin,
		})
		if err == nil {
			break
		}
		if !errors.Is(err, imageprocessor.ErrGridBoundaryContent) {
			return nil, fmt.Errorf("generator: split %s direction sheet: %w", taskType, err)
		}
		if candidate == maximumPrototypeCandidates {
			return nil, fmt.Errorf(
				"generator: split %s direction sheet: all %d generated candidates crossed an internal grid boundary: %w",
				taskType,
				maximumPrototypeCandidates,
				err,
			)
		}
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

		// Animation-mode splitting has already produced the final canonical PNG
		// at the requested dimensions. Persist those bytes directly. Running the
		// frame through Resize again performs a redundant raster resample, which
		// can damage fine seams and asymmetric details even when the canvas size
		// does not change.
		finalURL := unprocessedURL
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

func derivePrototypeSheetSpec(dimensions assetdomain.Size, columns, rows int) (prototypeSheetSpec, error) {
	if dimensions.Width == 0 || dimensions.Height == 0 {
		return prototypeSheetSpec{}, fmt.Errorf("dimensions must be positive")
	}
	if columns <= 0 || rows <= 0 {
		return prototypeSheetSpec{}, fmt.Errorf("grid dimensions must be positive")
	}

	targetWidth, targetHeight := uint64(dimensions.Width), uint64(dimensions.Height)
	columnCount, rowCount := uint64(columns), uint64(rows)
	if targetWidth > math.MaxUint64/columnCount || targetHeight > math.MaxUint64/rowCount {
		return prototypeSheetSpec{}, fmt.Errorf(
			"target dimensions %dx%d with grid %dx%d overflow sheet dimensions",
			dimensions.Width,
			dimensions.Height,
			columns,
			rows,
		)
	}

	baseWidth, baseHeight := targetWidth*columnCount, targetHeight*rowCount
	longEdge, shortEdge := max(baseWidth, baseHeight), min(baseWidth, baseHeight)
	if longEdge/shortEdge > maximumPrototypeSheetAspect ||
		(longEdge/shortEdge == maximumPrototypeSheetAspect && longEdge%shortEdge != 0) {
		return prototypeSheetSpec{}, fmt.Errorf(
			"target dimensions %dx%d with grid %dx%d require sheet aspect ratio %d:%d, exceeding %d:1",
			dimensions.Width,
			dimensions.Height,
			columns,
			rows,
			longEdge,
			shortEdge,
			maximumPrototypeSheetAspect,
		)
	}

	widthScale := prototypeSheetAlignment / prototypeGCD(baseWidth, prototypeSheetAlignment)
	heightScale := prototypeSheetAlignment / prototypeGCD(baseHeight, prototypeSheetAlignment)
	scaleStep := prototypeLCM(widthScale, heightScale)
	if baseWidth <= maximumPrototypeSheetDimension && baseHeight <= maximumPrototypeSheetDimension {
		maxScale := min(maximumPrototypeSheetDimension/baseWidth, maximumPrototypeSheetDimension/baseHeight)
		for scale := scaleStep; scale <= maxScale; scale += scaleStep {
			width, height := baseWidth*scale, baseHeight*scale
			pixels := width * height
			if pixels < minimumPrototypeSheetPixels {
				continue
			}
			if pixels > maximumPrototypeSheetPixels {
				break
			}
			return newPrototypeSheetSpec(width, height, columnCount, rowCount), nil
		}
	}

	if fallback, ok := closestLegalPrototypeSheet(baseWidth, baseHeight, columnCount, rowCount); ok {
		return fallback, nil
	}

	return prototypeSheetSpec{}, fmt.Errorf(
		"no legal sheet for target dimensions %dx%d and grid %dx%d satisfies provider constraints",
		dimensions.Width,
		dimensions.Height,
		columns,
		rows,
	)
}

func closestLegalPrototypeSheet(baseWidth, baseHeight, columns, rows uint64) (prototypeSheetSpec, bool) {
	basePixels := float64(baseWidth) * float64(baseHeight)
	desiredScale := 1.0
	if basePixels < float64(minimumPrototypeSheetPixels) {
		desiredScale = math.Sqrt(float64(minimumPrototypeSheetPixels) / basePixels)
	} else if basePixels > float64(maximumPrototypeSheetPixels) {
		desiredScale = math.Sqrt(float64(maximumPrototypeSheetPixels) / basePixels)
	}
	desiredScale = min(
		desiredScale,
		float64(maximumPrototypeSheetDimension)/float64(baseWidth),
		float64(maximumPrototypeSheetDimension)/float64(baseHeight),
	)
	desiredPixels := basePixels * desiredScale * desiredScale

	bestScore := math.Inf(1)
	var bestWidth, bestHeight uint64
	for width := prototypeSheetAlignment; width <= maximumPrototypeSheetDimension; width += prototypeSheetAlignment {
		idealHeight := float64(width) * float64(baseHeight) / float64(baseWidth)
		alignedHeight := uint64(math.Round(idealHeight/float64(prototypeSheetAlignment))) * prototypeSheetAlignment
		for _, height := range []uint64{
			alignedHeight - min(alignedHeight, prototypeSheetAlignment),
			alignedHeight,
			alignedHeight + prototypeSheetAlignment,
		} {
			if height == 0 || height > maximumPrototypeSheetDimension {
				continue
			}
			pixels := width * height
			if pixels < minimumPrototypeSheetPixels || pixels > maximumPrototypeSheetPixels {
				continue
			}
			longEdge, shortEdge := max(width, height), min(width, height)
			if longEdge > shortEdge*maximumPrototypeSheetAspect {
				continue
			}

			aspectError := math.Abs(math.Log(
				(float64(width) / float64(height)) / (float64(baseWidth) / float64(baseHeight)),
			))
			areaError := math.Abs(math.Log(float64(pixels) / desiredPixels))
			score := aspectError*1000 + areaError
			if score < bestScore {
				bestScore, bestWidth, bestHeight = score, width, height
			}
		}
	}
	if bestWidth == 0 || bestHeight == 0 {
		return prototypeSheetSpec{}, false
	}
	return newPrototypeSheetSpec(bestWidth, bestHeight, columns, rows), true
}

func newPrototypeSheetSpec(width, height, columns, rows uint64) prototypeSheetSpec {
	shortCellEdge := min(width/columns, height/rows)
	margin := shortCellEdge / 32
	if margin == 0 {
		margin = 1
	}
	return prototypeSheetSpec{
		Size:               fmt.Sprintf("%dx%d", width, height),
		GridBoundaryMargin: int(margin),
	}
}

func prototypeGCD(left, right uint64) uint64 {
	for right != 0 {
		left, right = right, left%right
	}
	return left
}

func prototypeLCM(left, right uint64) uint64 {
	return left / prototypeGCD(left, right) * right
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
