package generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

const (
	maxTileSetEditImageBytes         = 32 << 20
	maxTileSetEditGenerationAttempts = 3
	tileSetTileEditSeamBorder        = 1
)

type tileSetResolvedTarget struct {
	itemIndex int
	tileIndex int
}

type tileSetEditContext struct {
	asset      assetdomain.Asset
	content    assetdomain.AssetContent
	dimensions assetdomain.TileSetDimensions
	project    *projectdomain.Project
}

func (e *executor) editTileSetTiles(
	ctx context.Context,
	request EditTilesPayload,
) (json.RawMessage, error) {
	if err := validateEditTilesPayload(&request); err != nil {
		return nil, err
	}
	edit, err := e.loadTileSetEditContext(ctx, request.AssetID, request.ProjectID)
	if err != nil {
		return nil, err
	}
	targets, err := resolveTileSetTargets(request.Targets, edit.content, edit.dimensions)
	if err != nil {
		return nil, err
	}
	editReferences, err := e.resolveTileSetEditReferences(ctx, edit.project, request.Reference)
	if err != nil {
		return nil, err
	}
	uploads := make([]tileSetTileUpload, 0, len(targets))
	allocated := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		item := edit.content.Items[target.itemIndex]
		tile := item.Tiles[target.tileIndex]
		region, generateErr := e.generateTileSetTileEdit(
			ctx, request.CreativeBrief, edit.project, item.Name, tile, edit.dimensions, editReferences,
		)
		if generateErr != nil {
			return nil, generateErr
		}
		upload, allocateErr := allocateTileSetEditUpload(
			e.references, target, tile.Position, region, allocated,
		)
		if allocateErr != nil {
			return nil, allocateErr
		}
		uploads = append(uploads, upload)
	}
	return e.commitTileSetEdits(ctx, request.AssetID, edit.asset.Version, edit.content, uploads)
}

func (e *executor) editTileSetItem(
	ctx context.Context,
	request EditTilesetItemPayload,
) (json.RawMessage, error) {
	if err := validateEditTilesetItemPayload(&request); err != nil {
		return nil, err
	}
	edit, err := e.loadTileSetEditContext(ctx, request.AssetID, request.ProjectID)
	if err != nil {
		return nil, err
	}
	targets, err := resolveTileSetTargets([]TileSetEditTarget{*request.Target}, edit.content, edit.dimensions)
	if err != nil {
		return nil, err
	}
	itemIndex := targets[0].itemIndex
	editReferences, err := e.resolveTileSetEditReferences(ctx, edit.project, request.Reference)
	if err != nil {
		return nil, err
	}
	regions, err := e.generateTileSetItemEdit(
		ctx, request.CreativeBrief, edit.project, edit.content.Items[itemIndex], edit.dimensions, editReferences,
	)
	if err != nil {
		return nil, err
	}
	uploads := make([]tileSetTileUpload, 0, len(regions))
	allocated := make(map[string]struct{}, len(regions))
	for tileIndex, region := range regions {
		tile := edit.content.Items[itemIndex].Tiles[tileIndex]
		upload, allocateErr := allocateTileSetEditUpload(
			e.references,
			tileSetResolvedTarget{itemIndex: itemIndex, tileIndex: tileIndex},
			tile.Position,
			region,
			allocated,
		)
		if allocateErr != nil {
			return nil, allocateErr
		}
		uploads = append(uploads, upload)
	}
	return e.commitTileSetEdits(ctx, request.AssetID, edit.asset.Version, edit.content, uploads)
}

func (e *executor) loadTileSetEditContext(
	ctx context.Context,
	assetID uint,
	projectID uint,
) (*tileSetEditContext, error) {
	asset, err := e.assets.GetDetail(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("generator: load Tileset asset %d: %w", assetID, err)
	}
	if asset.ID == 0 {
		return nil, fmt.Errorf("generator: Tileset asset %d not found", assetID)
	}
	if asset.Type != assetdomain.AssetTypeTileSet {
		return nil, fmt.Errorf("generator: asset %d must have type %s", assetID, assetdomain.AssetTypeTileSet)
	}
	if asset.ProjectID != projectID {
		return nil, fmt.Errorf("generator: asset %d does not belong to project %d", assetID, projectID)
	}
	if err := assetdomain.ValidateDimensions(asset.Type, asset.Dimensions); err != nil {
		return nil, fmt.Errorf("generator: validate Tileset asset %d dimensions: %w", assetID, err)
	}
	var dimensions assetdomain.TileSetDimensions
	if err := json.Unmarshal(asset.Dimensions, &dimensions); err != nil {
		return nil, fmt.Errorf("generator: decode Tileset asset %d dimensions: %w", assetID, err)
	}
	if dimensions.TileSize.Width > maxTileEdge || dimensions.TileSize.Height > maxTileEdge ||
		uint64(dimensions.TileAmount.Columns)*uint64(dimensions.TileAmount.Rows) > maxTileSetGridTiles {
		return nil, fmt.Errorf("generator: Tileset asset %d dimensions exceed processing limits", assetID)
	}
	content, err := asset.DecodeContent()
	if err != nil {
		return nil, fmt.Errorf("generator: decode Tileset asset %d content: %w", assetID, err)
	}
	project, err := e.projects.GetDetail(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("generator: load project %d for Tileset editing: %w", projectID, err)
	}
	if project == nil {
		return nil, fmt.Errorf("generator: project %d is required for Tileset editing", projectID)
	}
	return &tileSetEditContext{asset: asset, content: content, dimensions: dimensions, project: project}, nil
}

func resolveTileSetTargets(
	requested []TileSetEditTarget,
	content assetdomain.AssetContent,
	dimensions assetdomain.TileSetDimensions,
) ([]tileSetResolvedTarget, error) {
	positions := make(map[assetdomain.TilePosition]tileSetResolvedTarget)
	for itemIndex, item := range content.Items {
		for tileIndex, tile := range item.Tiles {
			position := tile.Position
			if position.X < 0 || position.Y < 0 || uint(position.X) >= dimensions.TileAmount.Columns ||
				uint(position.Y) >= dimensions.TileAmount.Rows {
				return nil, invalidTaskPayload("Tileset content position (%d,%d) is out of grid", position.X, position.Y)
			}
			if _, duplicate := positions[position]; duplicate {
				return nil, invalidTaskPayload("Tileset content contains duplicate Tile position (%d,%d)", position.X, position.Y)
			}
			if tile.URL == nil || strings.TrimSpace(*tile.URL) == "" {
				return nil, invalidTaskPayload("Tileset Tile at (%d,%d) has no resource", position.X, position.Y)
			}
			positions[position] = tileSetResolvedTarget{itemIndex: itemIndex, tileIndex: tileIndex}
		}
	}
	resolved := make([]tileSetResolvedTarget, len(requested))
	for index, target := range requested {
		position := assetdomain.TilePosition{X: *target.Position.X, Y: *target.Position.Y}
		match, found := positions[position]
		if !found {
			return nil, invalidTaskPayload("Tileset target position (%d,%d) is not occupied", position.X, position.Y)
		}
		resolved[index] = match
	}
	return resolved, nil
}

func (e *executor) resolveTileSetEditReferences(
	ctx context.Context,
	project *projectdomain.Project,
	editReference string,
) ([]string, error) {
	references := make([]string, 0, 2)
	if editReference != "" {
		resolved, err := e.resolveReferences(ctx, EditTiles, []string{editReference})
		if err != nil {
			return nil, err
		}
		references = append(references, resolved...)
	}
	projectReferences, err := e.resolveReferences(ctx, EditTiles, referenceImages(project.Reference))
	if err != nil {
		return nil, err
	}
	return append(references, projectReferences...), nil
}

func (e *executor) generateTileSetTileEdit(
	ctx context.Context,
	brief string,
	project *projectdomain.Project,
	itemName string,
	tile assetdomain.Tile,
	dimensions assetdomain.TileSetDimensions,
	editReferences []string,
) (imageprocessor.ImageRegion, error) {
	resolved, err := e.references.ResolveReference(ctx, strings.TrimSpace(*tile.URL))
	if err != nil {
		return imageprocessor.ImageRegion{}, fmt.Errorf("generator: resolve Tile at (%d,%d): %w", tile.Position.X, tile.Position.Y, err)
	}
	original, err := loadTileSetImage(ctx, resolved)
	if err != nil {
		return imageprocessor.ImageRegion{}, fmt.Errorf("generator: load Tile at (%d,%d): %w", tile.Position.X, tile.Position.Y, err)
	}
	tileWidth, tileHeight := int(dimensions.TileSize.Width), int(dimensions.TileSize.Height)
	if original.Bounds().Dx() != tileWidth || original.Bounds().Dy() != tileHeight {
		return imageprocessor.ImageRegion{}, fmt.Errorf("generator: Tile at (%d,%d) size is %dx%d, want %dx%d",
			tile.Position.X, tile.Position.Y, original.Bounds().Dx(), original.Bounds().Dy(), tileWidth, tileHeight)
	}
	if !tileSetImageHasVisiblePixels(original) {
		return imageprocessor.ImageRegion{}, fmt.Errorf(
			"generator: Tile at (%d,%d) has no visible pixels", tile.Position.X, tile.Position.Y,
		)
	}
	originalBase64, err := imageprocessor.EncodePNGBase64(original)
	if err != nil {
		return imageprocessor.ImageRegion{}, err
	}
	references := append([]string{"data:image/png;base64," + originalBase64}, editReferences...)
	prompt := prompts.TileSetTileEdit(
		brief, formatTileSetProjectContext(project), itemName, tileWidth, tileHeight, string(project.Perspective),
	)
	var attemptErrors []error
	attemptsPerformed := 0
	for attempt := 1; attempt <= maxTileSetEditGenerationAttempts; attempt++ {
		attemptsPerformed = attempt
		if err := ctx.Err(); err != nil {
			return imageprocessor.ImageRegion{}, err
		}
		result, generateErr := e.images.Generate(ctx, &imageclient.GenerateRequest{
			Prompt: prompt, ReferenceImages: references, N: 2, MaxAttempts: 2,
		})
		if generateErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return imageprocessor.ImageRegion{}, ctxErr
			}
			attemptErrors = append(attemptErrors, fmt.Errorf("attempt %d provider: %w", attempt, generateErr))
			break
		}
		if result == nil || len(result.Images) == 0 {
			attemptErrors = append(attemptErrors, fmt.Errorf("attempt %d provider returned no images", attempt))
			continue
		}
		for candidateIndex, candidate := range result.Images {
			processed, processErr := e.processTileSetTileEditCandidate(
				ctx, candidate, originalBase64, tileWidth, tileHeight,
			)
			if processErr == nil {
				return processed, nil
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return imageprocessor.ImageRegion{}, ctxErr
			}
			attemptErrors = append(attemptErrors, fmt.Errorf(
				"attempt %d candidate %d: %w", attempt, candidateIndex, processErr,
			))
		}
	}
	return imageprocessor.ImageRegion{}, fmt.Errorf(
		"generator: edit Tile at (%d,%d) after %d attempts: %w",
		tile.Position.X, tile.Position.Y, attemptsPerformed, errors.Join(attemptErrors...),
	)
}

func tileSetImageHasVisiblePixels(img image.Image) bool {
	if img == nil {
		return false
	}
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			_, _, _, alpha := img.At(x, y).RGBA()
			if alpha>>8 > uint32(imageprocessor.TransparentAlphaMax) {
				return true
			}
		}
	}
	return false
}

func (e *executor) processTileSetTileEditCandidate(
	ctx context.Context,
	candidate imageclient.GeneratedImage,
	originalBase64 string,
	width int,
	height int,
) (imageprocessor.ImageRegion, error) {
	processed, err := e.processTileSetEditImage(
		ctx, &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{candidate}},
		width, height, []TileSetCoordinate{{0, 0}},
	)
	if err != nil {
		return imageprocessor.ImageRegion{}, fmt.Errorf("process: %w", err)
	}
	stabilized, err := stabilizeTileSetTileEdit(originalBase64, processed.ImageBase64, width, height)
	if err != nil {
		return imageprocessor.ImageRegion{}, fmt.Errorf("stabilize: %w", err)
	}
	if err := validateTileSetTileEditStructure(originalBase64, stabilized); err != nil {
		return imageprocessor.ImageRegion{}, fmt.Errorf("structure: %w", err)
	}
	if err := validateTileSetEditChanged(originalBase64, stabilized); err != nil {
		return imageprocessor.ImageRegion{}, fmt.Errorf("change: %w", err)
	}
	if err := verifyTileSetImage(ctx, e.processor, stabilized, width, height, true, false); err != nil {
		return imageprocessor.ImageRegion{}, fmt.Errorf("verify: %w", err)
	}
	processed.ImageBase64 = stabilized
	processed.Index = 0
	return processed, nil
}

func validateTileSetTileEditStructure(originalBase64, editedBase64 string) error {
	original, err := imageprocessor.DecodeBase64Image(originalBase64)
	if err != nil {
		return fmt.Errorf("decode original Tile: %w", err)
	}
	edited, err := imageprocessor.DecodeBase64Image(editedBase64)
	if err != nil {
		return fmt.Errorf("decode edited Tile: %w", err)
	}
	if original.Bounds().Dx() != edited.Bounds().Dx() || original.Bounds().Dy() != edited.Bounds().Dy() {
		return fmt.Errorf("edited Tile dimensions changed")
	}
	width, height := original.Bounds().Dx(), original.Bounds().Dy()
	for y := range height {
		for x := range width {
			before := original.RGBAAt(original.Bounds().Min.X+x, original.Bounds().Min.Y+y)
			after := edited.RGBAAt(edited.Bounds().Min.X+x, edited.Bounds().Min.Y+y)
			if before.A != after.A {
				return fmt.Errorf("edited Tile alpha silhouette changed at (%d,%d)", x, y)
			}
			if tileSetTileEditInSeamBorder(x, y, width, height) && before != after {
				return fmt.Errorf("edited Tile seam border changed at (%d,%d)", x, y)
			}
		}
	}
	return nil
}

func validateTileSetEditChanged(originalBase64, editedBase64 string) error {
	original, err := imageprocessor.DecodeBase64Image(originalBase64)
	if err != nil {
		return fmt.Errorf("decode original Tileset image: %w", err)
	}
	edited, err := imageprocessor.DecodeBase64Image(editedBase64)
	if err != nil {
		return fmt.Errorf("decode edited Tileset image: %w", err)
	}
	if original.Bounds().Dx() != edited.Bounds().Dx() || original.Bounds().Dy() != edited.Bounds().Dy() {
		return fmt.Errorf("edited Tileset image dimensions changed")
	}
	for y := range original.Bounds().Dy() {
		for x := range original.Bounds().Dx() {
			before := original.RGBAAt(original.Bounds().Min.X+x, original.Bounds().Min.Y+y)
			after := edited.RGBAAt(edited.Bounds().Min.X+x, edited.Bounds().Min.Y+y)
			if (before.A > imageprocessor.TransparentAlphaMax || after.A > imageprocessor.TransparentAlphaMax) && before != after {
				return nil
			}
		}
	}
	return fmt.Errorf("candidate left every pixel unchanged")
}

func tileSetTileEditInSeamBorder(x, y, width, height int) bool {
	return x < tileSetTileEditSeamBorder || y < tileSetTileEditSeamBorder ||
		x >= width-tileSetTileEditSeamBorder || y >= height-tileSetTileEditSeamBorder
}

// stabilizeTileSetTileEdit keeps persisted geometry authoritative. The model
// supplies colour and interior detail only; alpha and the one-pixel seam edge
// come from the current Tile.
func stabilizeTileSetTileEdit(
	originalBase64 string,
	generatedBase64 string,
	width int,
	height int,
) (string, error) {
	original, err := imageprocessor.DecodeBase64Image(originalBase64)
	if err != nil {
		return "", fmt.Errorf("decode original Tile: %w", err)
	}
	generated, err := imageprocessor.DecodeBase64Image(generatedBase64)
	if err != nil {
		return "", fmt.Errorf("decode generated Tile: %w", err)
	}
	if width <= 0 || height <= 0 || original.Bounds().Dx() != width || original.Bounds().Dy() != height ||
		generated.Bounds().Dx() != width || generated.Bounds().Dy() != height {
		return "", fmt.Errorf("tile images must both be %dx%d", width, height)
	}
	output := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			originalPixel := original.RGBAAt(original.Bounds().Min.X+x, original.Bounds().Min.Y+y)
			if originalPixel.A == 0 {
				continue
			}
			if tileSetTileEditInSeamBorder(x, y, width, height) {
				output.SetRGBA(x, y, originalPixel)
				continue
			}
			generatedPixel := generated.RGBAAt(generated.Bounds().Min.X+x, generated.Bounds().Min.Y+y)
			if tileSetEditPixelContaminated(generatedPixel) {
				generatedPixel = originalPixel
			}
			generatedPixel.A = originalPixel.A
			output.SetRGBA(x, y, generatedPixel)
		}
	}
	return imageprocessor.EncodePNGBase64(output)
}

func (e *executor) generateTileSetItemEdit(
	ctx context.Context,
	brief string,
	project *projectdomain.Project,
	item assetdomain.TileSetItem,
	dimensions assetdomain.TileSetDimensions,
	editReferences []string,
) ([]imageprocessor.ImageRegion, error) {
	itemBase64, shape, columns, rows, err := e.reconstructTileSetItem(ctx, item, dimensions)
	if err != nil {
		return nil, err
	}
	tileWidth, tileHeight := int(dimensions.TileSize.Width), int(dimensions.TileSize.Height)
	guide, err := buildTileSetShapeGuide(shape, columns, rows, tileWidth, tileHeight)
	if err != nil {
		return nil, err
	}
	mask, err := buildTileSetShapeMask(shape, columns, rows, tileWidth, tileHeight)
	if err != nil {
		return nil, err
	}
	references := []string{
		"data:image/png;base64," + guide,
		"data:image/png;base64," + itemBase64,
	}
	references = append(references, editReferences...)
	prompt := prompts.TileSetItemEdit(
		brief, formatTileSetProjectContext(project), item.Name, formatTileSetCoordinates(shape),
		tileWidth, tileHeight, string(project.Perspective),
	)
	var attemptErrors []error
	attemptsPerformed := 0
	for attempt := 1; attempt <= maxTileSetEditGenerationAttempts; attempt++ {
		attemptsPerformed = attempt
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		result, generateErr := e.images.Generate(ctx, &imageclient.GenerateRequest{
			Prompt: prompt, ReferenceImages: references,
			MaskImage: "data:image/png;base64," + mask, N: 2,
			MaxAttempts: 2,
		})
		if generateErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			attemptErrors = append(attemptErrors, fmt.Errorf("attempt %d provider: %w", attempt, generateErr))
			break
		}
		if result == nil || len(result.Images) == 0 {
			attemptErrors = append(attemptErrors, fmt.Errorf("attempt %d provider returned no images", attempt))
			continue
		}
		for candidateIndex, candidate := range result.Images {
			regions, processErr := e.processTileSetItemEditCandidate(
				ctx, candidate, itemBase64, shape, columns, rows, tileWidth, tileHeight,
			)
			if processErr == nil {
				return regions, nil
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			attemptErrors = append(attemptErrors, fmt.Errorf(
				"attempt %d candidate %d: %w", attempt, candidateIndex, processErr,
			))
		}
	}
	return nil, fmt.Errorf(
		"generator: edit complete Tileset Item %q after %d attempts: %w",
		item.Name, attemptsPerformed, errors.Join(attemptErrors...),
	)
}

func (e *executor) processTileSetItemEditCandidate(
	ctx context.Context,
	candidate imageclient.GeneratedImage,
	originalBase64 string,
	shape []TileSetCoordinate,
	columns int,
	rows int,
	tileWidth int,
	tileHeight int,
) ([]imageprocessor.ImageRegion, error) {
	processed, err := e.processTileSetEditImage(
		ctx, &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{candidate}},
		columns*tileWidth, rows*tileHeight, shape,
	)
	if err != nil {
		return nil, fmt.Errorf("process: %w", err)
	}
	aligned, err := alignTileSetImageToShape(
		processed.ImageBase64, shape, columns, rows, tileWidth, tileHeight,
	)
	if err != nil {
		return nil, fmt.Errorf("align: %w", err)
	}
	aligned, err = stabilizeTileSetItemEdit(originalBase64, aligned, tileWidth, tileHeight)
	if err != nil {
		return nil, fmt.Errorf("stabilize: %w", err)
	}
	if err := validateTileSetEditChanged(originalBase64, aligned); err != nil {
		return nil, fmt.Errorf("change: %w", err)
	}
	split, err := e.processor.SplitImage(ctx, &imageprocessor.SplitImageRequest{
		ImageBase64: aligned, Mode: imageprocessor.ImageSplitModeGrid, Columns: columns, Rows: rows,
		ForceProportionalGrid: true, AllowEmptyRegions: true,
	})
	if err != nil || split == nil || len(split.Regions) != columns*rows {
		if err == nil {
			err = fmt.Errorf("expected %d regions", columns*rows)
		}
		return nil, fmt.Errorf("split: %w", err)
	}
	regions := make([]imageprocessor.ImageRegion, len(shape))
	for index, cell := range shape {
		region := split.Regions[cell[1]*columns+cell[0]]
		if err := verifyTileSetImage(ctx, e.processor, region.ImageBase64, tileWidth, tileHeight, true, true); err != nil {
			return nil, fmt.Errorf("verify cell %d: %w", index, err)
		}
		region.Index = index
		regions[index] = region
	}
	return regions, nil
}

// stabilizeTileSetItemEdit preserves the persisted Item footprint and every
// cross-Tile seam while accepting generated colour, material, and interior
// detail. Transparent generated gaps fall back to the current Item.
func stabilizeTileSetItemEdit(
	originalBase64 string,
	generatedBase64 string,
	tileWidth int,
	tileHeight int,
) (string, error) {
	original, err := imageprocessor.DecodeBase64Image(originalBase64)
	if err != nil {
		return "", fmt.Errorf("decode original Item: %w", err)
	}
	generated, err := imageprocessor.DecodeBase64Image(generatedBase64)
	if err != nil {
		return "", fmt.Errorf("decode generated Item: %w", err)
	}
	width, height := original.Bounds().Dx(), original.Bounds().Dy()
	if tileWidth <= 0 || tileHeight <= 0 || width == 0 || height == 0 ||
		width%tileWidth != 0 || height%tileHeight != 0 ||
		generated.Bounds().Dx() != width || generated.Bounds().Dy() != height {
		return "", fmt.Errorf("item images must share a positive Tile-aligned canvas")
	}
	output := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			originalPixel := original.RGBAAt(original.Bounds().Min.X+x, original.Bounds().Min.Y+y)
			if originalPixel.A == 0 {
				continue
			}
			localX, localY := x%tileWidth, y%tileHeight
			if localX == 0 || localY == 0 || localX == tileWidth-1 || localY == tileHeight-1 {
				output.SetRGBA(x, y, originalPixel)
				continue
			}
			generatedPixel := generated.RGBAAt(generated.Bounds().Min.X+x, generated.Bounds().Min.Y+y)
			if tileSetEditPixelContaminated(generatedPixel) {
				generatedPixel = originalPixel
			}
			generatedPixel.A = originalPixel.A
			output.SetRGBA(x, y, generatedPixel)
		}
	}
	return imageprocessor.EncodePNGBase64(output)
}

func tileSetEditPixelContaminated(pixel color.RGBA) bool {
	if pixel.A == 0 {
		return true
	}
	// Image-edit providers sometimes flatten transparent input onto a white or
	// chroma-key matte, or leak black pixels from the occupancy guide. None are
	// valid edit detail; retaining the current pixel prevents guide/matte blocks
	// from entering the persisted footprint.
	nearWhite := pixel.R >= 238 && pixel.G >= 238 && pixel.B >= 238
	nearGreenMatte := pixel.G >= 220 && pixel.R <= 48 && pixel.B <= 48
	nearBlackGuide := pixel.R <= 12 && pixel.G <= 12 && pixel.B <= 12
	return nearWhite || nearGreenMatte || nearBlackGuide
}

func (e *executor) processTileSetEditImage(
	ctx context.Context,
	result *imageclient.GenerateResult,
	width int,
	height int,
	shape []TileSetCoordinate,
) (imageprocessor.ImageRegion, error) {
	if result == nil || len(result.Images) == 0 {
		return imageprocessor.ImageRegion{}, fmt.Errorf("expected at least one image")
	}
	var candidateErrors []error
	for index, candidate := range result.Images {
		if strings.TrimSpace(candidate.Base64) == "" {
			candidateErrors = append(candidateErrors, fmt.Errorf("candidate %d is empty", index))
			continue
		}
		removed, err := e.processor.RemoveBackground(ctx, &imageprocessor.RemoveBackgroundRequest{
			ImageBase64:               candidate.Base64,
			MatteColor:                imageprocessor.DefaultMatteColor,
			AllowSampledMatteFallback: true,
		})
		if err != nil || removed == nil || strings.TrimSpace(removed.ImageBase64) == "" {
			if err == nil {
				err = fmt.Errorf("empty background-removal result")
			}
			candidateErrors = append(candidateErrors, fmt.Errorf("candidate %d: %w", index, err))
			continue
		}
		resized, err := e.processor.Resize(ctx, &imageprocessor.ResizeRequest{
			ImageBase64: removed.ImageBase64,
			Options: imageprocessor.ResizeOptions{
				Width: width, Height: height, Margin: 0, CropContent: false,
				HardAlpha: true, Mode: imageprocessor.RasterModePixel,
			},
		})
		if err != nil || resized == nil || strings.TrimSpace(resized.ImageBase64) == "" {
			if err == nil {
				err = fmt.Errorf("empty resize result")
			}
			candidateErrors = append(candidateErrors, fmt.Errorf("candidate %d: %w", index, err))
			continue
		}
		return imageprocessor.ImageRegion{ImageBase64: resized.ImageBase64, MIMEType: "image/png"}, nil
	}
	return imageprocessor.ImageRegion{}, errors.Join(candidateErrors...)
}

func (e *executor) reconstructTileSetItem(
	ctx context.Context,
	item assetdomain.TileSetItem,
	dimensions assetdomain.TileSetDimensions,
) (string, []TileSetCoordinate, int, int, error) {
	if len(item.Tiles) == 0 {
		return "", nil, 0, 0, fmt.Errorf("generator: Tileset Item %q has no Tiles", item.Name)
	}
	minX, minY := item.Tiles[0].Position.X, item.Tiles[0].Position.Y
	maxX, maxY := minX, minY
	seen := make(map[assetdomain.TilePosition]struct{}, len(item.Tiles))
	for index, tile := range item.Tiles {
		if tile.URL == nil || strings.TrimSpace(*tile.URL) == "" {
			return "", nil, 0, 0, fmt.Errorf("generator: Tileset Item %q Tile %d has no resource", item.Name, index)
		}
		if _, duplicate := seen[tile.Position]; duplicate {
			return "", nil, 0, 0, fmt.Errorf("generator: Tileset Item %q has duplicate position (%d,%d)", item.Name, tile.Position.X, tile.Position.Y)
		}
		seen[tile.Position] = struct{}{}
		minX, minY = min(minX, tile.Position.X), min(minY, tile.Position.Y)
		maxX, maxY = max(maxX, tile.Position.X), max(maxY, tile.Position.Y)
	}
	columns, rows := maxX-minX+1, maxY-minY+1
	tileWidth, tileHeight := int(dimensions.TileSize.Width), int(dimensions.TileSize.Height)
	if columns <= 0 || rows <= 0 || columns*tileWidth > maxGeneratedItemImageEdge || rows*tileHeight > maxGeneratedItemImageEdge {
		return "", nil, 0, 0, fmt.Errorf("generator: Tileset Item %q footprint exceeds processing limits", item.Name)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, columns*tileWidth, rows*tileHeight))
	shape := make([]TileSetCoordinate, len(item.Tiles))
	for index, tile := range item.Tiles {
		resolved, err := e.references.ResolveReference(ctx, strings.TrimSpace(*tile.URL))
		if err != nil {
			return "", nil, 0, 0, fmt.Errorf("generator: resolve Tileset Item %q Tile %d: %w", item.Name, index, err)
		}
		decoded, err := loadTileSetImage(ctx, resolved)
		if err != nil {
			return "", nil, 0, 0, fmt.Errorf("generator: load Tileset Item %q Tile %d: %w", item.Name, index, err)
		}
		if decoded.Bounds().Dx() != tileWidth || decoded.Bounds().Dy() != tileHeight {
			return "", nil, 0, 0, fmt.Errorf("generator: Tileset Item %q Tile %d size is %dx%d, want %dx%d",
				item.Name, index, decoded.Bounds().Dx(), decoded.Bounds().Dy(), tileWidth, tileHeight)
		}
		cell := TileSetCoordinate{tile.Position.X - minX, tile.Position.Y - minY}
		shape[index] = cell
		destination := image.Rect(cell[0]*tileWidth, cell[1]*tileHeight, (cell[0]+1)*tileWidth, (cell[1]+1)*tileHeight)
		draw.Draw(canvas, destination, decoded, decoded.Bounds().Min, draw.Src)
	}
	encoded, err := imageprocessor.EncodePNGBase64(canvas)
	if err != nil {
		return "", nil, 0, 0, err
	}
	return encoded, shape, columns, rows, nil
}

func loadTileSetImage(ctx context.Context, reference string) (image.Image, error) {
	reference = strings.TrimSpace(reference)
	if strings.HasPrefix(reference, "http://") || strings.HasPrefix(reference, "https://") {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, reference, nil)
		if err != nil {
			return nil, err
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return nil, err
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("HTTP %d", response.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, maxTileSetEditImageBytes+1))
		if err != nil {
			return nil, err
		}
		if len(body) > maxTileSetEditImageBytes {
			return nil, fmt.Errorf("image exceeds %d bytes", maxTileSetEditImageBytes)
		}
		reference = base64.StdEncoding.EncodeToString(body)
	}
	return imageprocessor.DecodeBase64Image(reference)
}

func allocateTileSetEditUpload(
	references ReferenceStore,
	target tileSetResolvedTarget,
	position assetdomain.TilePosition,
	region imageprocessor.ImageRegion,
	allocated map[string]struct{},
) (tileSetTileUpload, error) {
	key, err := references.NewObjectKey("image/png")
	if err != nil {
		return tileSetTileUpload{}, fmt.Errorf("generator: allocate edited Tile at (%d,%d): %w", position.X, position.Y, err)
	}
	if _, duplicate := allocated[key]; duplicate {
		return tileSetTileUpload{}, fmt.Errorf("generator: allocate edited Tile at (%d,%d): duplicate object key %q", position.X, position.Y, key)
	}
	allocated[key] = struct{}{}
	return tileSetTileUpload{
		itemIndex: target.itemIndex, tileIndex: target.tileIndex,
		position: TileSetCoordinate{position.X, position.Y}, region: region, objectKey: key,
	}, nil
}

func (e *executor) commitTileSetEdits(
	ctx context.Context,
	assetID uint,
	expectedVersion uint,
	content assetdomain.AssetContent,
	uploads []tileSetTileUpload,
) (json.RawMessage, error) {
	uploadedKeys, err := e.persistTileSetUploads(ctx, uploads)
	if err != nil {
		return nil, err
	}
	cleanup := func(cause error) error {
		if cleanupErr := e.references.DeleteObjects(context.WithoutCancel(ctx), uploadedKeys); cleanupErr != nil {
			return errors.Join(cause, fmt.Errorf("generator: clean up edited Tileset uploads: %w", cleanupErr))
		}
		return cause
	}
	for _, upload := range uploads {
		key := upload.objectKey
		content.Items[upload.itemIndex].Tiles[upload.tileIndex].URL = &key
	}
	editedIndexes := make(map[int][]int, len(uploads))
	for _, upload := range uploads {
		editedIndexes[upload.itemIndex] = append(editedIndexes[upload.itemIndex], upload.tileIndex)
	}
	indexes := make([]int, 0, len(editedIndexes))
	for index := range editedIndexes {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	editedItems := make([]assetdomain.TileSetItem, 0, len(indexes))
	for _, index := range indexes {
		tileIndexes := editedIndexes[index]
		sort.Ints(tileIndexes)
		tiles := make([]assetdomain.Tile, 0, len(tileIndexes))
		for _, tileIndex := range tileIndexes {
			tiles = append(tiles, content.Items[index].Tiles[tileIndex])
		}
		editedItems = append(editedItems, assetdomain.TileSetItem{
			Name:  content.Items[index].Name,
			Tiles: tiles,
		})
	}
	encoded, err := assetdomain.EncodeContent(assetdomain.AssetContent{Items: editedItems})
	if err != nil {
		return nil, cleanup(fmt.Errorf("generator: encode edited Tileset asset %d content: %w", assetID, err))
	}
	return encodeExecutionResult(ExecutionResult{
		AssetID:            assetID,
		Version:            expectedVersion,
		Content:            encoded,
		GeneratedResources: uploadedKeys,
	})
}

func formatTileSetCoordinates(cells []TileSetCoordinate) string {
	values := make([]string, len(cells))
	for index, cell := range cells {
		values[index] = "[" + strconv.Itoa(cell[0]) + "," + strconv.Itoa(cell[1]) + "]"
	}
	sort.Strings(values)
	return "[" + strings.Join(values, ", ") + "]"
}
