package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"
	"sync"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

const (
	maxTileSetItemConcurrency        = 4
	maxTileSetItemGenerationAttempts = 3
)

type processedTileSetItem struct {
	Index          int
	Name           string
	Columns        int
	Rows           int
	ImageBase64    string
	MIMEType       string
	RawImageBase64 string
	RawMediaType   string
	LocalShape     []TileSetCoordinate
	Tiles          []imageprocessor.ImageRegion
	Perspective    assetdomain.Perspective
}

func (e *executor) generateTileSet(
	ctx context.Context,
	request CreateTileSetPayload,
) (json.RawMessage, error) {
	if err := validateCreateTileSetPayload(&request); err != nil {
		return nil, err
	}
	items, err := e.processTileSetItems(ctx, request)
	if err != nil {
		return nil, err
	}
	layout, err := assignTileSetLayout(request)
	if err != nil {
		return nil, err
	}
	return e.publishTileSet(ctx, request, items, layout)
}

func (e *executor) processTileSetItems(
	ctx context.Context,
	request CreateTileSetPayload,
) ([]processedTileSetItem, error) {
	if e.images == nil {
		return nil, ErrImageServiceRequired
	}
	if e.processor == nil {
		return nil, ErrImageProcessorRequired
	}
	if e.projects == nil {
		return nil, fmt.Errorf("generator: project reader is required for Tileset processing")
	}
	if err := validateCreateTileSetPayload(&request); err != nil {
		return nil, err
	}
	project, err := e.projects.GetDetail(ctx, request.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("generator: load project %d for Tileset processing: %w", request.ProjectID, err)
	}
	if project == nil {
		return nil, fmt.Errorf("generator: project %d is required for Tileset processing", request.ProjectID)
	}
	projectReferences, err := e.resolveReferences(ctx, GenerateTileSet, referenceImages(project.Reference))
	if err != nil {
		return nil, err
	}

	processed := make([]processedTileSetItem, len(request.Items))
	processContext, cancel := context.WithCancel(ctx)
	defer cancel()
	semaphore := make(chan struct{}, maxTileSetItemConcurrency)
	var group sync.WaitGroup
	var firstError error
	var errorOnce sync.Once
	for index, item := range request.Items {
		group.Go(func() {
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-processContext.Done():
				return
			}
			result, itemErr := e.processTileSetItem(
				processContext,
				request,
				project,
				item,
				index,
				projectReferences,
			)
			if itemErr != nil {
				errorOnce.Do(func() {
					firstError = itemErr
					cancel()
				})
				return
			}
			processed[index] = *result
		})
	}
	group.Wait()
	if firstError != nil {
		return nil, firstError
	}
	if err := processContext.Err(); err != nil {
		return nil, err
	}
	return processed, nil
}

func (e *executor) processTileSetItem(
	ctx context.Context,
	request CreateTileSetPayload,
	project *projectdomain.Project,
	item TileSetItemDefinition,
	index int,
	projectReferences []string,
) (*processedTileSetItem, error) {
	minX, minY, maxX, maxY := tileSetItemBounds(item.Shape)
	columns, rows := maxX-minX+1, maxY-minY+1
	localShape := make([]TileSetCoordinate, len(item.Shape))
	shapeText := make([]string, len(item.Shape))
	for coordinateIndex, coordinate := range item.Shape {
		localShape[coordinateIndex] = TileSetCoordinate{coordinate[0] - minX, coordinate[1] - minY}
		shapeText[coordinateIndex] = "[" + strconv.Itoa(localShape[coordinateIndex][0]) + "," +
			strconv.Itoa(localShape[coordinateIndex][1]) + "]"
	}
	tileWidth := int(request.Dimensions.TileSize.Width)
	tileHeight := int(request.Dimensions.TileSize.Height)
	shape := "[" + strings.Join(shapeText, ", ") + "]"
	prompt := prompts.TileSetItem(
		request.CreativeBrief,
		formatTileSetProjectContext(project),
		item.Name,
		item.Description,
		shape,
		tileWidth,
		tileHeight,
		string(project.Perspective),
	)
	shapeGuide, err := buildTileSetShapeGuide(localShape, columns, rows, tileWidth, tileHeight)
	if err != nil {
		return nil, fmt.Errorf("generator: build Tileset Item %d Shape guide: %w", index, err)
	}
	shapeMask, err := buildTileSetShapeMask(localShape, columns, rows, tileWidth, tileHeight)
	if err != nil {
		return nil, fmt.Errorf("generator: build Tileset Item %d Shape mask: %w", index, err)
	}
	references := append(
		[]string{"data:image/png;base64," + shapeGuide},
		projectReferences...,
	)
	var attemptErrors []error
	attemptsPerformed := 0
	for attempt := 1; attempt <= maxTileSetItemGenerationAttempts; attempt++ {
		attemptsPerformed = attempt
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		generated, err := e.images.Generate(ctx, &imageclient.GenerateRequest{
			Prompt:          prompt,
			ReferenceImages: references,
			MaskImage:       "data:image/png;base64," + shapeMask,
			N:               2,
			MaxAttempts:     2,
		})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			attemptErrors = append(attemptErrors, fmt.Errorf("attempt %d provider: %w", attempt, err))
			break
		}
		if generated == nil || len(generated.Images) == 0 {
			attemptErrors = append(attemptErrors, fmt.Errorf("attempt %d provider returned no images", attempt))
			continue
		}
		for candidateIndex, candidate := range generated.Images {
			result, candidateErr := e.processTileSetItemCandidate(
				ctx, request, item, index, candidateIndex, columns, rows, localShape, candidate,
				assetdomain.Perspective(project.Perspective),
			)
			if candidateErr == nil {
				return result, nil
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			attemptErrors = append(attemptErrors, fmt.Errorf(
				"attempt %d candidate %d: %w", attempt, candidateIndex, candidateErr,
			))
		}
	}
	return nil, fmt.Errorf(
		"generator: process Tileset Item %d after %d attempts: %w",
		index,
		attemptsPerformed,
		errors.Join(attemptErrors...),
	)
}

func (e *executor) processTileSetItemCandidate(
	ctx context.Context,
	request CreateTileSetPayload,
	item TileSetItemDefinition,
	itemIndex int,
	candidateIndex int,
	columns int,
	rows int,
	localShape []TileSetCoordinate,
	candidate imageclient.GeneratedImage,
	perspective assetdomain.Perspective,
) (*processedTileSetItem, error) {
	if strings.TrimSpace(candidate.Base64) == "" {
		return nil, fmt.Errorf("candidate %d is empty", candidateIndex)
	}
	index := itemIndex
	tileWidth := int(request.Dimensions.TileSize.Width)
	tileHeight := int(request.Dimensions.TileSize.Height)
	removed, err := e.processor.RemoveBackground(ctx, &imageprocessor.RemoveBackgroundRequest{
		ImageBase64:               candidate.Base64,
		MatteColor:                imageprocessor.DefaultMatteColor,
		AllowSampledMatteFallback: true,
	})
	if err != nil || removed == nil || strings.TrimSpace(removed.ImageBase64) == "" {
		if err == nil {
			err = fmt.Errorf("empty result")
		}
		return nil, fmt.Errorf("generator: remove Tileset Item %d background: %w", index, err)
	}
	verified, err := e.processor.Verify(ctx, &imageprocessor.VerifyRequest{
		ImageBase64: removed.ImageBase64,
		Profile:     imageprocessor.ProfileGeneric,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: verify Tileset Item %d alpha: %w", index, err)
	}
	if verified == nil || !verified.HasAlpha || verified.NontransparentPixels == 0 {
		return nil, fmt.Errorf("generator: verify Tileset Item %d alpha: invalid transparent image", index)
	}
	resized, err := e.processor.Resize(ctx, &imageprocessor.ResizeRequest{
		ImageBase64: removed.ImageBase64,
		Options: imageprocessor.ResizeOptions{
			Width:       columns * tileWidth,
			Height:      rows * tileHeight,
			Margin:      0,
			CropContent: false,
			HardAlpha:   true,
			Mode:        imageprocessor.RasterModePixel,
		},
	})
	if err != nil || resized == nil || strings.TrimSpace(resized.ImageBase64) == "" {
		if err == nil {
			err = fmt.Errorf("empty result")
		}
		return nil, fmt.Errorf("generator: resize Tileset Item %d: %w", index, err)
	}
	aligned, err := alignTileSetImageToShape(
		resized.ImageBase64,
		localShape,
		columns,
		rows,
		tileWidth,
		tileHeight,
	)
	if err != nil {
		return nil, fmt.Errorf("generator: align Tileset Item %d to Shape: %w", index, err)
	}
	if err := verifyTileSetImage(
		ctx,
		e.processor,
		aligned,
		columns*tileWidth,
		rows*tileHeight,
		true,
		false,
	); err != nil {
		return nil, fmt.Errorf("generator: verify resized Tileset Item %d: %w", index, err)
	}
	split, err := e.processor.SplitImage(ctx, &imageprocessor.SplitImageRequest{
		ImageBase64:           aligned,
		Mode:                  imageprocessor.ImageSplitModeGrid,
		Columns:               columns,
		Rows:                  rows,
		ForceProportionalGrid: true,
		AllowEmptyRegions:     true,
	})
	if err != nil || split == nil || len(split.Regions) != columns*rows {
		if err == nil {
			err = fmt.Errorf("expected %d regions", columns*rows)
		}
		return nil, fmt.Errorf("generator: split Tileset Item %d: %w", index, err)
	}
	occupied := make(map[TileSetCoordinate]struct{}, len(localShape))
	for _, coordinate := range localShape {
		occupied[coordinate] = struct{}{}
	}
	for regionIndex := range split.Regions {
		cell := TileSetCoordinate{regionIndex % columns, regionIndex / columns}
		_, shouldContain := occupied[cell]
		if !shouldContain {
			transparent, encodeErr := transparentTileSetRegion(regionIndex, tileWidth, tileHeight)
			if encodeErr != nil {
				return nil, fmt.Errorf("generator: mask Tileset Item %d candidate %d cell %d: %w", index, candidateIndex, regionIndex, encodeErr)
			}
			split.Regions[regionIndex] = transparent
		}
		if shouldContain {
			if err := verifyTileSetNoGuideLeak(split.Regions[regionIndex].ImageBase64); err != nil {
				return nil, fmt.Errorf("generator: verify Tileset Item %d cell %d guide residue: %w", index, regionIndex, err)
			}
		}
		if err := verifyTileSetImage(
			ctx,
			e.processor,
			split.Regions[regionIndex].ImageBase64,
			tileWidth,
			tileHeight,
			shouldContain,
			shouldContain,
		); err != nil {
			return nil, fmt.Errorf("generator: verify Tileset Item %d cell %d: %w", index, regionIndex, err)
		}
	}
	tiles := make([]imageprocessor.ImageRegion, len(localShape))
	for coordinateIndex, coordinate := range localShape {
		tiles[coordinateIndex] = split.Regions[coordinate[1]*columns+coordinate[0]]
	}
	rawPNGBase64, err := normalizeTileSetCandidatePNG(candidate)
	if err != nil {
		return nil, fmt.Errorf("generator: normalize Tileset Item %d candidate %d to PNG: %w", index, candidateIndex, err)
	}
	return &processedTileSetItem{
		Index:          index,
		Name:           item.Name,
		Columns:        columns,
		Rows:           rows,
		ImageBase64:    aligned,
		MIMEType:       "image/png",
		RawImageBase64: rawPNGBase64,
		RawMediaType:   "image/png",
		LocalShape:     localShape,
		Tiles:          tiles,
		Perspective:    perspective,
	}, nil
}

func normalizeTileSetCandidatePNG(candidate imageclient.GeneratedImage) (string, error) {
	decoded, err := imageprocessor.DecodeBase64Image(candidate.Base64)
	if err != nil {
		return "", fmt.Errorf("decode candidate image: %w", err)
	}
	return imageprocessor.EncodePNGBase64(decoded)
}

func (e *executor) publishTileSet(
	ctx context.Context,
	request CreateTileSetPayload,
	items []processedTileSetItem,
	layout []tileSetPlacement,
) (json.RawMessage, error) {
	uploads, err := buildTileSetUploads(e.references, items, layout)
	if err != nil {
		return nil, err
	}
	uploadedKeys, err := e.persistTileSetUploads(ctx, uploads)
	if err != nil {
		return nil, err
	}
	cleanup := func(cause error) error {
		if cleanupErr := e.references.DeleteObjects(context.WithoutCancel(ctx), uploadedKeys); cleanupErr != nil {
			return errors.Join(cause, fmt.Errorf("generator: clean up Tileset uploads: %w", cleanupErr))
		}
		return cause
	}
	dimensions, err := json.Marshal(request.Dimensions)
	if err != nil {
		return nil, cleanup(fmt.Errorf("generator: encode Tileset dimensions: %w", err))
	}
	contentItems := buildTileSetContentItems(items, uploads)
	content, err := assetdomain.EncodeContent(assetdomain.AssetContent{Items: contentItems})
	if err != nil {
		return nil, cleanup(fmt.Errorf("generator: encode Tileset content: %w", err))
	}
	perspective := assetdomain.PerspectiveTopDown
	if len(items) > 0 {
		perspective = items[0].Perspective
	}
	if !perspective.Valid() {
		return nil, cleanup(fmt.Errorf("generator: invalid Tileset project perspective %q", perspective))
	}
	assetID, err := e.assets.CreateTileSetAsset(ctx, &assetdomain.Asset{
		Name: request.AssetName, ProjectID: request.ProjectID, Type: assetdomain.AssetTypeTileSet,
		Description: request.CreativeBrief, Perspective: perspective,
		Dimensions: dimensions, Content: content,
	})
	if err != nil {
		return nil, cleanup(fmt.Errorf("generator: create Tileset asset: %w", err))
	}
	if assetID == 0 {
		return nil, cleanup(fmt.Errorf("generator: create Tileset asset: empty ID"))
	}
	return encodeExecutionResult(ExecutionResult{AssetID: assetID, Version: 1})
}

func buildTileSetShapeGuide(
	shape []TileSetCoordinate,
	columns int,
	rows int,
	tileWidth int,
	tileHeight int,
) (string, error) {
	occupied, err := validateTileSetShapeGrid(shape, columns, rows, tileWidth, tileHeight)
	if err != nil {
		return "", fmt.Errorf("shape guide: %w", err)
	}
	guide := image.NewRGBA(image.Rect(0, 0, columns*tileWidth, rows*tileHeight))
	draw.Draw(
		guide,
		guide.Bounds(),
		&image.Uniform{C: color.RGBA{G: 255, A: 255}},
		image.Point{},
		draw.Src,
	)
	paintTileSetShape(guide, occupied, tileWidth, tileHeight, color.RGBA{A: 255})
	return imageprocessor.EncodePNGBase64(guide)
}

func buildTileSetShapeMask(
	shape []TileSetCoordinate,
	columns int,
	rows int,
	tileWidth int,
	tileHeight int,
) (string, error) {
	occupied, err := validateTileSetShapeGrid(shape, columns, rows, tileWidth, tileHeight)
	if err != nil {
		return "", fmt.Errorf("shape mask: %w", err)
	}
	mask := image.NewRGBA(image.Rect(0, 0, columns*tileWidth, rows*tileHeight))
	draw.Draw(mask, mask.Bounds(), &image.Uniform{C: color.RGBA{A: 255}}, image.Point{}, draw.Src)
	paintTileSetShape(mask, occupied, tileWidth, tileHeight, color.RGBA{})
	return imageprocessor.EncodePNGBase64(mask)
}

func validateTileSetShapeGrid(
	shape []TileSetCoordinate,
	columns int,
	rows int,
	tileWidth int,
	tileHeight int,
) (map[TileSetCoordinate]struct{}, error) {
	if columns <= 0 || rows <= 0 || tileWidth <= 0 || tileHeight <= 0 || len(shape) == 0 {
		return nil, fmt.Errorf("requires a non-empty positive grid")
	}
	occupied := make(map[TileSetCoordinate]struct{}, len(shape))
	for _, cell := range shape {
		if cell[0] < 0 || cell[0] >= columns || cell[1] < 0 || cell[1] >= rows {
			return nil, fmt.Errorf("cell [%d,%d] is outside Item grid", cell[0], cell[1])
		}
		if _, duplicate := occupied[cell]; duplicate {
			return nil, fmt.Errorf("cell [%d,%d] is duplicated", cell[0], cell[1])
		}
		occupied[cell] = struct{}{}
	}
	return occupied, nil
}

func paintTileSetShape(
	target *image.RGBA,
	occupied map[TileSetCoordinate]struct{},
	tileWidth int,
	tileHeight int,
	paint color.RGBA,
) {
	margin := tileSetShapeSafetyMargin(tileWidth, tileHeight)
	for cell := range occupied {
		bounds := tileSetOccupiedCellBounds(cell, occupied, tileWidth, tileHeight, margin)
		draw.Draw(target, bounds, &image.Uniform{C: paint}, image.Point{}, draw.Src)
	}
}

func transparentTileSetRegion(index int, width int, height int) (imageprocessor.ImageRegion, error) {
	encoded, err := imageprocessor.EncodePNGBase64(image.NewRGBA(image.Rect(0, 0, width, height)))
	if err != nil {
		return imageprocessor.ImageRegion{}, err
	}
	return imageprocessor.ImageRegion{
		Index:       index,
		ImageBase64: encoded,
		MIMEType:    "image/png",
	}, nil
}

// alignTileSetImageToShape translates or shrinks content before splitting. It
// prefers the safety-margin-inset occupied bounds and then clips any remaining
// model spill from omitted cells. Intentional empty space inside occupied cells
// is preserved.
func alignTileSetImageToShape(
	imageBase64 string,
	shape []TileSetCoordinate,
	columns int,
	rows int,
	tileWidth int,
	tileHeight int,
) (string, error) {
	decoded, err := imageprocessor.DecodeBase64Image(imageBase64)
	if err != nil {
		return "", fmt.Errorf("decode Item image: %w", err)
	}
	expectedWidth, expectedHeight := columns*tileWidth, rows*tileHeight
	if columns <= 0 || rows <= 0 || tileWidth <= 0 || tileHeight <= 0 ||
		decoded.Bounds().Dx() != expectedWidth || decoded.Bounds().Dy() != expectedHeight {
		return "", fmt.Errorf(
			"image size is %dx%d, want %dx%d",
			decoded.Bounds().Dx(),
			decoded.Bounds().Dy(),
			expectedWidth,
			expectedHeight,
		)
	}
	occupied := make(map[TileSetCoordinate]struct{}, len(shape))
	for _, cell := range shape {
		if cell[0] < 0 || cell[0] >= columns || cell[1] < 0 || cell[1] >= rows {
			return "", fmt.Errorf("shape cell [%d,%d] is outside Item grid", cell[0], cell[1])
		}
		occupied[cell] = struct{}{}
	}
	visibleBounds, visible := tileSetAlphaBounds(decoded)
	if !visible {
		return "", fmt.Errorf("item has no visible pixels")
	}
	if len(occupied) == columns*rows {
		return imageBase64, nil
	}
	safetyMargin := tileSetShapeSafetyMargin(tileWidth, tileHeight)
	best := tileSetShapePlacement{}
	relaxed := tileSetShapePlacement{}
	for scalePercent := 100; scalePercent >= 50; scalePercent -= 2 {
		scaledWidth := max(1, (visibleBounds.Dx()*scalePercent+50)/100)
		scaledHeight := max(1, (visibleBounds.Dy()*scalePercent+50)/100)
		scaled, _, resizeErr := imageprocessor.ResizeImage(decoded, imageprocessor.ResizeOptions{
			Width:       scaledWidth,
			Height:      scaledHeight,
			Margin:      0,
			CropContent: true,
			HardAlpha:   true,
			Mode:        imageprocessor.RasterModePixel,
		})
		if resizeErr != nil {
			return "", fmt.Errorf("resize Shape candidate to %d%%: %w", scalePercent, resizeErr)
		}
		candidate := findTileSetShapeImagePlacement(
			scaled,
			occupied,
			expectedWidth,
			expectedHeight,
			tileWidth,
			tileHeight,
			safetyMargin,
			visibleBounds,
		)
		candidate.scalePercent = scalePercent
		if candidate.valid && (!best.valid || tileSetShapePlacementLess(candidate, best)) {
			best = candidate
		}
		if candidate.valid && candidate.outsidePixels == 0 {
			best = candidate
			break
		}
		fallback := findTileSetShapeImagePlacement(
			scaled,
			occupied,
			expectedWidth,
			expectedHeight,
			tileWidth,
			tileHeight,
			0,
			visibleBounds,
		)
		fallback.scalePercent = scalePercent
		if fallback.valid && fallback.outsidePixels == 0 &&
			(!relaxed.valid || tileSetShapePlacementLess(fallback, relaxed)) {
			relaxed = fallback
		}
	}
	if (!best.valid || best.outsidePixels != 0) && relaxed.valid {
		best = relaxed
	}
	if !best.valid {
		return "", fmt.Errorf("no placement keeps visible Item content inside the occupied Shape")
	}
	aligned := image.NewRGBA(image.Rect(0, 0, expectedWidth, expectedHeight))
	destination := image.Rectangle{
		Min: best.position,
		Max: best.position.Add(best.image.Bounds().Size()),
	}
	draw.Draw(aligned, destination, best.image, best.image.Bounds().Min, draw.Src)
	transparent := &image.Uniform{C: color.RGBA{}}
	for row := range rows {
		for column := range columns {
			if _, allowed := occupied[TileSetCoordinate{column, row}]; allowed {
				continue
			}
			bounds := image.Rect(
				column*tileWidth,
				row*tileHeight,
				(column+1)*tileWidth,
				(row+1)*tileHeight,
			)
			draw.Draw(aligned, bounds, transparent, image.Point{}, draw.Src)
		}
	}
	return imageprocessor.EncodePNGBase64(aligned)
}

func tileSetShapeSafetyMargin(tileWidth int, tileHeight int) int {
	return max(1, min(tileWidth, tileHeight)/32)
}

func tileSetOccupiedCellBounds(
	cell TileSetCoordinate,
	occupied map[TileSetCoordinate]struct{},
	tileWidth int,
	tileHeight int,
	safetyMargin int,
) image.Rectangle {
	bounds := image.Rect(
		cell[0]*tileWidth,
		cell[1]*tileHeight,
		(cell[0]+1)*tileWidth,
		(cell[1]+1)*tileHeight,
	)
	if _, adjacent := occupied[TileSetCoordinate{cell[0] - 1, cell[1]}]; !adjacent {
		bounds.Min.X += safetyMargin
	}
	if _, adjacent := occupied[TileSetCoordinate{cell[0] + 1, cell[1]}]; !adjacent {
		bounds.Max.X -= safetyMargin
	}
	if _, adjacent := occupied[TileSetCoordinate{cell[0], cell[1] - 1}]; !adjacent {
		bounds.Min.Y += safetyMargin
	}
	if _, adjacent := occupied[TileSetCoordinate{cell[0], cell[1] + 1}]; !adjacent {
		bounds.Max.Y -= safetyMargin
	}
	return bounds
}

type tileSetShapePlacement struct {
	image         *image.RGBA
	position      image.Point
	scalePercent  int
	insidePixels  int
	outsidePixels int
	distance      int
	valid         bool
}

func findTileSetShapeImagePlacement(
	content *image.RGBA,
	occupied map[TileSetCoordinate]struct{},
	canvasWidth int,
	canvasHeight int,
	tileWidth int,
	tileHeight int,
	safetyMargin int,
	originalBounds image.Rectangle,
) tileSetShapePlacement {
	integral, _, visible := tileSetAlphaIntegral(content)
	if !visible || content.Bounds().Dx() > canvasWidth || content.Bounds().Dy() > canvasHeight {
		return tileSetShapePlacement{}
	}
	totalPixels := tileSetIntegralArea(integral, content.Bounds())
	preferred := image.Pt(
		(originalBounds.Min.X+originalBounds.Max.X-content.Bounds().Dx())/2,
		(originalBounds.Min.Y+originalBounds.Max.Y-content.Bounds().Dy())/2,
	)
	maxX := canvasWidth - content.Bounds().Dx()
	maxY := canvasHeight - content.Bounds().Dy()
	step := max(1, min(tileWidth, tileHeight)/16)
	best := tileSetShapePlacement{}
	consider := func(position image.Point) {
		inside, valid := tileSetShapeScore(
			integral,
			occupied,
			position,
			tileWidth,
			tileHeight,
			content.Bounds().Dx(),
			content.Bounds().Dy(),
			safetyMargin,
		)
		if !valid {
			return
		}
		candidate := tileSetShapePlacement{
			image:         content,
			position:      position,
			insidePixels:  inside,
			outsidePixels: totalPixels - inside,
			distance: tileSetAbsolute(position.X-preferred.X) +
				tileSetAbsolute(position.Y-preferred.Y),
			valid: true,
		}
		if !best.valid || tileSetShapePlacementLess(candidate, best) {
			best = candidate
		}
	}
	for y := 0; y <= maxY; y += step {
		for x := 0; x <= maxX; x += step {
			consider(image.Pt(x, y))
		}
	}
	consider(image.Pt(maxX, maxY))
	if !best.valid || step == 1 {
		return best
	}
	coarse := best
	best = tileSetShapePlacement{}
	for y := max(0, coarse.position.Y-step); y <= min(maxY, coarse.position.Y+step); y++ {
		for x := max(0, coarse.position.X-step); x <= min(maxX, coarse.position.X+step); x++ {
			consider(image.Pt(x, y))
		}
	}
	return best
}

func tileSetShapePlacementLess(candidate tileSetShapePlacement, current tileSetShapePlacement) bool {
	if candidate.outsidePixels != current.outsidePixels {
		return candidate.outsidePixels < current.outsidePixels
	}
	if candidate.scalePercent != current.scalePercent {
		return candidate.scalePercent > current.scalePercent
	}
	return candidate.distance < current.distance
}

type tileSetIntegralImage struct {
	values []int
	stride int
}

func tileSetAlphaIntegral(value *image.RGBA) (tileSetIntegralImage, image.Rectangle, bool) {
	width, height := value.Bounds().Dx(), value.Bounds().Dy()
	integral := tileSetIntegralImage{
		values: make([]int, (width+1)*(height+1)),
		stride: width + 1,
	}
	visibleBounds := image.Rectangle{Min: image.Pt(width, height), Max: image.Point{}}
	found := false
	for y := range height {
		row := 0
		for x := range width {
			if value.RGBAAt(x, y).A != 0 {
				row++
				found = true
				visibleBounds.Min.X = min(visibleBounds.Min.X, x)
				visibleBounds.Min.Y = min(visibleBounds.Min.Y, y)
				visibleBounds.Max.X = max(visibleBounds.Max.X, x+1)
				visibleBounds.Max.Y = max(visibleBounds.Max.Y, y+1)
			}
			integral.values[(y+1)*integral.stride+x+1] =
				integral.values[y*integral.stride+x+1] + row
		}
	}
	return integral, visibleBounds, found
}

func tileSetAlphaBounds(value *image.RGBA) (image.Rectangle, bool) {
	width, height := value.Bounds().Dx(), value.Bounds().Dy()
	visibleBounds := image.Rectangle{Min: image.Pt(width, height), Max: image.Point{}}
	found := false
	for y := range height {
		for x := range width {
			if value.RGBAAt(x, y).A == 0 {
				continue
			}
			found = true
			visibleBounds.Min.X = min(visibleBounds.Min.X, x)
			visibleBounds.Min.Y = min(visibleBounds.Min.Y, y)
			visibleBounds.Max.X = max(visibleBounds.Max.X, x+1)
			visibleBounds.Max.Y = max(visibleBounds.Max.Y, y+1)
		}
	}
	return visibleBounds, found
}

func tileSetShapeScore(
	integral tileSetIntegralImage,
	occupied map[TileSetCoordinate]struct{},
	position image.Point,
	tileWidth int,
	tileHeight int,
	width int,
	height int,
	safetyMargin int,
) (int, bool) {
	inside := 0
	for cell := range occupied {
		rectangle := tileSetOccupiedCellBounds(
			cell,
			occupied,
			tileWidth,
			tileHeight,
			safetyMargin,
		).Sub(position).Intersect(image.Rect(0, 0, width, height))
		cellPixels := tileSetIntegralArea(integral, rectangle)
		inside += cellPixels
	}
	return inside, inside > 0
}

func tileSetIntegralArea(integral tileSetIntegralImage, rectangle image.Rectangle) int {
	index := func(x int, y int) int {
		return y*integral.stride + x
	}
	return integral.values[index(rectangle.Max.X, rectangle.Max.Y)] -
		integral.values[index(rectangle.Max.X, rectangle.Min.Y)] -
		integral.values[index(rectangle.Min.X, rectangle.Max.Y)] +
		integral.values[index(rectangle.Min.X, rectangle.Min.Y)]
}

func tileSetAbsolute(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func verifyTileSetImage(
	ctx context.Context,
	processor imageprocessor.Processor,
	imageBase64 string,
	expectedWidth int,
	expectedHeight int,
	shouldContainContent bool,
	allowEmptyContent bool,
) error {
	report, err := processor.Verify(ctx, &imageprocessor.VerifyRequest{
		ImageBase64: imageBase64,
		Profile:     imageprocessor.ProfileGeneric,
	})
	if err != nil {
		return err
	}
	if report == nil {
		return fmt.Errorf("missing verification report")
	}
	if !report.IsPNG {
		return fmt.Errorf("image must be a PNG")
	}
	if expectedWidth <= 0 || expectedHeight <= 0 {
		return fmt.Errorf("expected image dimensions must be positive")
	}
	totalPixels := uint64(expectedWidth) * uint64(expectedHeight)
	if !report.HasAlpha && (!shouldContainContent || report.NontransparentPixels != totalPixels) {
		return fmt.Errorf("non-opaque image must be a PNG with alpha")
	}
	if report.Width != expectedWidth || report.Height != expectedHeight {
		return fmt.Errorf("image size is %dx%d, want %dx%d", report.Width, report.Height, expectedWidth, expectedHeight)
	}
	if !report.TransparentRGBScrubbed {
		return fmt.Errorf("transparent pixels contain residual RGB values")
	}
	minimumPixels := uint64(max(1, expectedWidth*expectedHeight/100))
	if shouldContainContent && !allowEmptyContent && report.NontransparentPixels < minimumPixels {
		return fmt.Errorf(
			"occupied cell has %d content pixels, need at least %d",
			report.NontransparentPixels,
			minimumPixels,
		)
	}
	if !shouldContainContent && report.NontransparentPixels != 0 {
		return fmt.Errorf("outside-Shape cell is not transparent")
	}
	return nil
}
