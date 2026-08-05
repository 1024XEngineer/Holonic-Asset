package image

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"sort"
	"strings"
)

// AnimationAnchor identifies the stable point used to register generated
// animation frames before a shared crop is calculated.
type AnimationAnchor string

const (
	AnimationAnchorAuto   AnimationAnchor = "auto"
	AnimationAnchorFeet   AnimationAnchor = "feet"
	AnimationAnchorCenter AnimationAnchor = "center"
	AnimationAnchorHead   AnimationAnchor = "head"
)

// AnimationBackgroundOptions optionally removes a generated flat background
// before frame registration. Leave nil when the input already has alpha.
type AnimationBackgroundOptions struct {
	MatteColor       string   `json:"matte_color,omitempty"`
	Material         Material `json:"material,omitempty"`
	Threshold        *float64 `json:"threshold,omitempty"`
	Softness         *float64 `json:"softness,omitempty"`
	SpillSuppression *float64 `json:"spill_suppression,omitempty"`
}

// normalizeAnimationRequest is the private animation engine input assembled
// by SplitImage animation mode.
type normalizeAnimationRequest struct {
	Columns                  int
	Rows                     int
	FrameCount               int
	FrameWidth               int
	FrameHeight              int
	Margin                   int
	CropPadding              int
	AlphaThreshold           uint8
	Anchor                   AnimationAnchor
	PreserveHorizontalMotion bool
	PreserveVerticalMotion   bool
	MaxStabilizationShift    int
	DetectGridBounds         bool
	AllowEmptyFrames         bool
	Background               *AnimationBackgroundOptions
}

type AnimationPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type AnimationOffset struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// animationFrame is one normalized PNG frame. Every frame has identical
// dimensions and is rendered from the same shared crop with one global scale.
type animationFrame struct {
	Index         int
	ImageBase64   string
	MIMEType      string
	SourceBounds  AlphaBoundingBox
	ContentBounds *AlphaBoundingBox
	SourceAnchor  *AnimationPoint
	OutputAnchor  *AnimationPoint
	Translation   AnimationOffset
}

type AnimationNormalizationReport struct {
	SourceWidth             int               `json:"source_width"`
	SourceHeight            int               `json:"source_height"`
	FrameWidth              int               `json:"frame_width"`
	FrameHeight             int               `json:"frame_height"`
	SheetWidth              int               `json:"sheet_width"`
	SheetHeight             int               `json:"sheet_height"`
	Columns                 int               `json:"columns"`
	Rows                    int               `json:"rows"`
	FrameCount              int               `json:"frame_count"`
	Margin                  int               `json:"margin"`
	CropPadding             int               `json:"crop_padding"`
	SharedCrop              AlphaBoundingBox  `json:"shared_crop"`
	Scale                   float64           `json:"scale"`
	Anchor                  AnimationAnchor   `json:"anchor"`
	SourceAnchorMedian      AnimationPoint    `json:"source_anchor_median"`
	SourceAnchorRange       AnimationPoint    `json:"source_anchor_range"`
	OutputAnchorRange       AnimationPoint    `json:"output_anchor_range"`
	AppliedMaxShift         AnimationPoint    `json:"applied_max_shift"`
	TranslationClamped      int               `json:"translation_clamped"`
	GridXBounds             []int             `json:"grid_x_bounds"`
	GridYBounds             []int             `json:"grid_y_bounds"`
	GridPolicy              string            `json:"grid_policy"`
	RegistrationPolicy      string            `json:"registration_policy"`
	BackgroundRemovalReport *ExtractionReport `json:"background_removal_report,omitempty"`
	Warnings                []string          `json:"warnings,omitempty"`
}

type normalizedAnimation struct {
	ImageBase64 string
	MIMEType    string
	Frames      []animationFrame
	Report      AnimationNormalizationReport
}

type animationCell struct {
	image      *image.NRGBA
	bbox       image.Rectangle
	visible    bool
	anchor     AnimationPoint
	shift      image.Point
	sourceCell image.Rectangle
}

// normalizeAnimationImage is the private engine behind SplitImage animation
// mode. A single shared coordinate system prevents crop-induced jitter.
func normalizeAnimationImage(src image.Image, request normalizeAnimationRequest) (*normalizedAnimation, error) {
	if request.Columns <= 0 || request.Rows <= 0 {
		return nil, fmt.Errorf("columns and rows must be positive")
	}
	capacity := request.Columns * request.Rows
	frameCount := request.FrameCount
	if frameCount == 0 {
		frameCount = capacity
	}
	if frameCount < 1 || frameCount > capacity {
		return nil, fmt.Errorf("frame count must be between 1 and grid capacity %d", capacity)
	}
	if request.Margin < 0 || request.CropPadding < 0 || request.MaxStabilizationShift < 0 {
		return nil, fmt.Errorf("margin, crop padding, and max stabilization shift must not be negative")
	}
	anchor := request.Anchor
	if anchor == "" || anchor == AnimationAnchorAuto {
		anchor = AnimationAnchorFeet
	}
	switch anchor {
	case AnimationAnchorFeet, AnimationAnchorCenter, AnimationAnchorHead:
	default:
		return nil, fmt.Errorf("unsupported animation anchor %q", request.Anchor)
	}
	threshold := request.AlphaThreshold
	if threshold == 0 {
		threshold = defaultImageSplitAlphaThreshold
	}

	input := toNRGBA(src)
	var extractionReport *ExtractionReport
	if request.Background != nil {
		prepared, report, err := removeAnimationBackground(input, *request.Background)
		if err != nil {
			return nil, err
		}
		input, extractionReport = prepared, &report
	}
	if !hasTransparentPixel(input, threshold) {
		return nil, fmt.Errorf("source has no transparent pixels; configure background removal or call RemoveBackground first")
	}

	w, h := input.Bounds().Dx(), input.Bounds().Dy()
	if request.Columns > w || request.Rows > h {
		return nil, fmt.Errorf("grid dimensions %dx%d exceed source image size %dx%d", request.Columns, request.Rows, w, h)
	}
	xBounds, yBounds := proportionalBounds(w, request.Columns), proportionalBounds(h, request.Rows)
	gridPolicy := "proportional_fixed_cells"
	if request.DetectGridBounds {
		xBounds, yBounds = detectGridBoundsNRGBA(input, request.Columns, request.Rows, false, threshold)
		gridPolicy = "content_detected_opt_in_with_proportional_fallback"
	}
	cells, cellW, cellH := extractAnimationCells(input, xBounds, yBounds, frameCount, threshold, anchor)
	visible := 0
	for i := range cells {
		if cells[i].visible {
			visible++
		} else if !request.AllowEmptyFrames {
			return nil, fmt.Errorf("source frame %d has no visible pixels", i)
		}
	}
	if visible == 0 {
		return nil, fmt.Errorf("no visible animation pixels found")
	}

	frameWidth, frameHeight := request.FrameWidth, request.FrameHeight
	if frameWidth == 0 && frameHeight == 0 {
		frameWidth, frameHeight = cellW, cellH
	}
	if frameWidth <= 0 || frameHeight <= 0 {
		return nil, fmt.Errorf("frame width and height must both be positive or both be omitted")
	}
	margin := request.Margin
	if margin == 0 {
		margin = defaultAssetMargin(frameWidth, frameHeight)
	}
	if margin*2 >= frameWidth || margin*2 >= frameHeight {
		return nil, fmt.Errorf("animation margin must be less than half the frame dimensions")
	}

	report := AnimationNormalizationReport{
		SourceWidth: w, SourceHeight: h,
		FrameWidth: frameWidth, FrameHeight: frameHeight,
		SheetWidth: frameWidth * request.Columns, SheetHeight: frameHeight * request.Rows,
		Columns: request.Columns, Rows: request.Rows, FrameCount: frameCount,
		Margin: margin, Anchor: anchor, GridXBounds: append([]int(nil), xBounds...), GridYBounds: append([]int(nil), yBounds...),
		GridPolicy:              gridPolicy,
		RegistrationPolicy:      "median_root_anchor_shared_union_crop_single_global_scale_no_per_frame_recentering",
		BackgroundRemovalReport: extractionReport,
	}
	stabilizeAnimationCells(cells, cellW, cellH, request, &report)

	pad := max(6, int(math.Round(float64(min(cellW, cellH))*.14)))
	pad = max(pad, int(math.Ceil(max(report.AppliedMaxShift.X, report.AppliedMaxShift.Y))))
	workingBounds := image.Rect(0, 0, cellW+2*pad, cellH+2*pad)
	working := make([]*image.NRGBA, len(cells))
	shared := image.Rectangle{}
	sharedSet := false
	for i := range cells {
		canvas := image.NewNRGBA(workingBounds)
		dst := image.Rect(pad+cells[i].shift.X, pad+cells[i].shift.Y, pad+cells[i].shift.X+cellW, pad+cells[i].shift.Y+cellH)
		draw.Draw(canvas, dst, cells[i].image, image.Point{}, draw.Over)
		working[i] = canvas
		bbox, ok := alphaBoundsNRGBA(canvas, threshold)
		if !ok {
			continue
		}
		if !sharedSet {
			shared, sharedSet = bbox, true
		} else {
			shared = shared.Union(bbox)
		}
	}
	if !sharedSet {
		return nil, fmt.Errorf("no visible pixels after frame registration")
	}
	cropPadding := request.CropPadding
	if cropPadding == 0 {
		cropPadding = max(2, int(math.Round(float64(max(shared.Dx(), shared.Dy()))*.025)))
	}
	shared = expandAnimationRect(shared, cropPadding, workingBounds)
	report.CropPadding = cropPadding
	report.SharedCrop = rectangleToAlphaBoundingBox(shared)

	innerW, innerH := frameWidth-2*margin, frameHeight-2*margin
	scale := math.Min(float64(innerW)/float64(shared.Dx()), float64(innerH)/float64(shared.Dy()))
	if scale <= 0 {
		return nil, fmt.Errorf("target frame is too small")
	}
	drawW := max(1, int(math.Round(float64(shared.Dx())*scale)))
	drawH := max(1, int(math.Round(float64(shared.Dy())*scale)))
	destX, destY := (frameWidth-drawW)/2, (frameHeight-drawH)/2
	report.Scale = scale

	sheet := image.NewNRGBA(image.Rect(0, 0, report.SheetWidth, report.SheetHeight))
	frames := make([]animationFrame, 0, len(cells))
	outputAnchors := make([]AnimationPoint, 0, visible)
	for i := range cells {
		frame := image.NewNRGBA(image.Rect(0, 0, frameWidth, frameHeight))
		if cells[i].visible {
			cropped := cloneNRGBA(working[i].SubImage(shared))
			resized := qualityResize(ToRGBA(cropped), cropped.Bounds(), drawW, drawH)
			draw.Draw(frame, image.Rect(destX, destY, destX+drawW, destY+drawH), resized, image.Point{}, draw.Over)
		}
		scrubTransparentNRGBA(frame)
		encoded, err := EncodePNGBase64(frame)
		if err != nil {
			return nil, fmt.Errorf("encode normalized frame %d: %w", i, err)
		}
		var contentBox *AlphaBoundingBox
		if bbox, ok := alphaBoundsNRGBA(frame, threshold); ok {
			box := rectangleToAlphaBoundingBox(bbox)
			contentBox = &box
		}
		var sourceAnchor, outputAnchor *AnimationPoint
		if cells[i].visible {
			sourceValue := cells[i].anchor
			outputValue := AnimationPoint{
				X: float64(destX) + (float64(pad+cells[i].shift.X)+cells[i].anchor.X-float64(shared.Min.X))*scale,
				Y: float64(destY) + (float64(pad+cells[i].shift.Y)+cells[i].anchor.Y-float64(shared.Min.Y))*scale,
			}
			sourceAnchor, outputAnchor = &sourceValue, &outputValue
			outputAnchors = append(outputAnchors, outputValue)
		}
		frames = append(frames, animationFrame{
			Index: i, ImageBase64: encoded, MIMEType: pngMIMEType,
			SourceBounds: rectangleToAlphaBoundingBox(cells[i].sourceCell), ContentBounds: contentBox,
			SourceAnchor: sourceAnchor, OutputAnchor: outputAnchor,
			Translation: AnimationOffset{X: cells[i].shift.X, Y: cells[i].shift.Y},
		})
		col, row := i%request.Columns, i/request.Columns
		draw.Draw(sheet, image.Rect(col*frameWidth, row*frameHeight, (col+1)*frameWidth, (row+1)*frameHeight), frame, image.Point{}, draw.Over)
	}
	report.OutputAnchorRange = animationPointRange(outputAnchors)
	if report.TranslationClamped > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d frame(s) exceeded the configured stabilization shift", report.TranslationClamped))
	}
	encodedSheet, err := EncodePNGBase64(sheet)
	if err != nil {
		return nil, fmt.Errorf("encode normalized spritesheet: %w", err)
	}
	return &normalizedAnimation{ImageBase64: encodedSheet, MIMEType: pngMIMEType, Frames: frames, Report: report}, nil
}

func removeAnimationBackground(input image.Image, options AnimationBackgroundOptions) (*image.NRGBA, ExtractionReport, error) {
	matteValue := strings.TrimSpace(options.MatteColor)
	if matteValue == "" {
		matteValue = "auto"
	}
	matteColor, autoMatte, err := ParseMatteColorOrAuto(matteValue)
	if err != nil {
		return nil, ExtractionReport{}, fmt.Errorf("parse animation matte color: %w", err)
	}
	var matte *MatteColor
	if !autoMatte {
		matte = &matteColor
	}
	settings := ResolveChromaSettings(options.Material, options.Threshold, options.Softness, options.SpillSuppression)
	output, report := ExtractChromaWithReport(input, matte, settings)
	return toNRGBA(output), report, nil
}

func hasTransparentPixel(img *image.NRGBA, threshold uint8) bool {
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			if img.NRGBAAt(x, y).A <= threshold {
				return true
			}
		}
	}
	return false
}

func extractAnimationCells(input *image.NRGBA, xBounds, yBounds []int, frameCount int, threshold uint8, anchor AnimationAnchor) ([]animationCell, int, int) {
	cellW, cellH := 0, 0
	for i := 0; i+1 < len(xBounds); i++ {
		cellW = max(cellW, xBounds[i+1]-xBounds[i])
	}
	for i := 0; i+1 < len(yBounds); i++ {
		cellH = max(cellH, yBounds[i+1]-yBounds[i])
	}
	cells := make([]animationCell, 0, frameCount)
	for row := 0; row+1 < len(yBounds) && len(cells) < frameCount; row++ {
		for col := 0; col+1 < len(xBounds) && len(cells) < frameCount; col++ {
			sourceRect := image.Rect(xBounds[col], yBounds[row], xBounds[col+1], yBounds[row+1])
			source := cloneNRGBA(input.SubImage(sourceRect))
			canvas := image.NewNRGBA(image.Rect(0, 0, cellW, cellH))
			offset := image.Pt((cellW-source.Bounds().Dx())/2, (cellH-source.Bounds().Dy())/2)
			draw.Draw(canvas, image.Rectangle{Min: offset, Max: offset.Add(source.Bounds().Size())}, source, image.Point{}, draw.Over)
			bbox, visible := alphaBoundsNRGBA(canvas, threshold)
			cell := animationCell{image: canvas, bbox: bbox, visible: visible, sourceCell: sourceRect}
			if visible {
				cell.anchor = animationAnchorFor(canvas, bbox, anchor)
			}
			cells = append(cells, cell)
		}
	}
	return cells, cellW, cellH
}

func stabilizeAnimationCells(cells []animationCell, width, height int, request normalizeAnimationRequest, report *AnimationNormalizationReport) {
	xs, ys := make([]float64, 0, len(cells)), make([]float64, 0, len(cells))
	for i := range cells {
		if cells[i].visible {
			xs, ys = append(xs, cells[i].anchor.X), append(ys, cells[i].anchor.Y)
		}
	}
	medianX, medianY := medianAnimationValue(xs), medianAnimationValue(ys)
	report.SourceAnchorMedian = AnimationPoint{X: medianX, Y: medianY}
	report.SourceAnchorRange = AnimationPoint{X: rangeAnimationValue(xs), Y: rangeAnimationValue(ys)}
	// Register all raster frames against one integer target. Rounding each
	// (median-anchor) delta independently makes even frame counts diverge by one
	// source pixel when their median lies on a half pixel (for example +7.5 and
	// -7.5 both round away from zero). Rounding the target once avoids that
	// artificial residual displacement.
	targetX, targetY := math.Round(medianX), math.Round(medianY)
	maxShift := request.MaxStabilizationShift
	if maxShift == 0 {
		maxShift = max(4, int(math.Round(float64(min(width, height))*.20)))
	}
	for i := range cells {
		if !cells[i].visible {
			continue
		}
		dx := int(math.Round(targetX - cells[i].anchor.X))
		dy := int(math.Round(targetY - cells[i].anchor.Y))
		if request.PreserveHorizontalMotion {
			dx = 0
		}
		if request.PreserveVerticalMotion {
			dy = 0
		}
		clampedX, clampedY := clampAnimationInt(dx, -maxShift, maxShift), clampAnimationInt(dy, -maxShift, maxShift)
		if clampedX != dx || clampedY != dy {
			report.TranslationClamped++
		}
		cells[i].shift = image.Pt(clampedX, clampedY)
		report.AppliedMaxShift.X = math.Max(report.AppliedMaxShift.X, math.Abs(float64(clampedX)))
		report.AppliedMaxShift.Y = math.Max(report.AppliedMaxShift.Y, math.Abs(float64(clampedY)))
	}
}

func animationAnchorFor(img *image.NRGBA, bbox image.Rectangle, anchor AnimationAnchor) AnimationPoint {
	switch anchor {
	case AnimationAnchorHead:
		top := image.Rect(bbox.Min.X, bbox.Min.Y, bbox.Max.X, bbox.Min.Y+max(1, int(math.Round(float64(bbox.Dy())*.45))))
		return AnimationPoint{X: animationWeightedMedianX(img, top), Y: animationWeightedQuantileY(img, top, .48)}
	case AnimationAnchorCenter:
		return AnimationPoint{X: animationWeightedMedianX(img, bbox), Y: animationWeightedQuantileY(img, bbox, .5)}
	default:
		band := image.Rect(bbox.Min.X, bbox.Min.Y+int(math.Round(float64(bbox.Dy())*.30)), bbox.Max.X, bbox.Min.Y+int(math.Round(float64(bbox.Dy())*.74)))
		x := animationWeightedMedianX(img, band)
		half := max(8, int(math.Round(float64(bbox.Dx())*.22)))
		corridor := image.Rect(max(bbox.Min.X, int(math.Round(x))-half), bbox.Min.Y, min(bbox.Max.X, int(math.Round(x))+half+1), bbox.Max.Y)
		return AnimationPoint{X: x, Y: animationWeightedQuantileY(img, corridor, .992)}
	}
}

func animationWeightedMedianX(img *image.NRGBA, rect image.Rectangle) float64 {
	rect = rect.Intersect(img.Bounds())
	if rect.Empty() {
		return 0
	}
	weights := make([]float64, rect.Dx())
	total := 0.0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			a := float64(img.NRGBAAt(x, y).A)
			if a > 24 {
				weights[x-rect.Min.X] += a
				total += a
			}
		}
	}
	if total == 0 {
		return float64(rect.Min.X+rect.Max.X) / 2
	}
	cumulative := 0.0
	for i, weight := range weights {
		cumulative += weight
		if cumulative >= total*.5 {
			return float64(rect.Min.X + i)
		}
	}
	return float64(rect.Max.X - 1)
}

func animationWeightedQuantileY(img *image.NRGBA, rect image.Rectangle, quantile float64) float64 {
	rect = rect.Intersect(img.Bounds())
	if rect.Empty() {
		return 0
	}
	weights := make([]float64, rect.Dy())
	total := 0.0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			a := float64(img.NRGBAAt(x, y).A)
			if a > 24 {
				weights[y-rect.Min.Y] += a
				total += a
			}
		}
	}
	if total == 0 {
		return float64(rect.Min.Y+rect.Max.Y) / 2
	}
	cumulative := 0.0
	for i, weight := range weights {
		cumulative += weight
		if cumulative >= total*quantile {
			return float64(rect.Min.Y + i)
		}
	}
	return float64(rect.Max.Y - 1)
}

func medianAnimationValue(values []float64) float64 {
	values = append([]float64(nil), values...)
	sort.Float64s(values)
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return (values[middle-1] + values[middle]) / 2
}

func rangeAnimationValue(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minValue, maxValue := values[0], values[0]
	for _, value := range values[1:] {
		minValue, maxValue = math.Min(minValue, value), math.Max(maxValue, value)
	}
	return maxValue - minValue
}

func animationPointRange(points []AnimationPoint) AnimationPoint {
	if len(points) == 0 {
		return AnimationPoint{}
	}
	xs, ys := make([]float64, len(points)), make([]float64, len(points))
	for i, point := range points {
		xs[i], ys[i] = point.X, point.Y
	}
	return AnimationPoint{X: rangeAnimationValue(xs), Y: rangeAnimationValue(ys)}
}

func expandAnimationRect(rect image.Rectangle, padding int, limit image.Rectangle) image.Rectangle {
	return image.Rect(rect.Min.X-padding, rect.Min.Y-padding, rect.Max.X+padding, rect.Max.Y+padding).Intersect(limit)
}

func clampAnimationInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func scrubTransparentNRGBA(img *image.NRGBA) {
	for y := range img.Bounds().Dy() {
		row := y * img.Stride
		for x := range img.Bounds().Dx() {
			offset := row + x*4
			if img.Pix[offset+3] <= TransparentAlphaMax {
				img.Pix[offset], img.Pix[offset+1], img.Pix[offset+2] = 0, 0, 0
			}
		}
	}
}
