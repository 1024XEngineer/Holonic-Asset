package image

import (
	"context"
	"fmt"
	"image"
	"sort"
)

const defaultImageSplitAlphaThreshold uint8 = 8

// ImageSplitMode controls how source images are split into regions.
type ImageSplitMode string

const (
	// ImageSplitModeGrid performs structural extraction from a known rows x
	// columns grid. It keeps each source cell unchanged and does not register
	// animation subjects inside those cells.
	ImageSplitModeGrid ImageSplitMode = "grid"
	// ImageSplitModeAnimation returns final, same-sized animation frames. It
	// registers every subject to one shared anchor and uses one shared crop and
	// global scale, preventing per-frame crop displacement.
	ImageSplitModeAnimation ImageSplitMode = "animation"
	// ImageSplitModeComponents extracts each connected non-transparent
	// region and crops it to its visible bounds.
	ImageSplitModeComponents ImageSplitMode = "components"
	// ImageSplitModeProjection groups nearby disconnected alpha components into
	// larger visual regions using expanded bounds and union-find. This is useful
	// for a pose whose body, weapon, or shadow is not one connected component.
	ImageSplitModeProjection ImageSplitMode = "projection"
)

// SplitImageRequest describes a deterministic image splitting job. Animation
// mode accepts either transparent input or a flat opaque background; when
// Background is nil for opaque input, its matte colour is detected from the
// source edges automatically.
type SplitImageRequest struct {
	ImageBase64              string                      `json:"image_base64"`
	Mode                     ImageSplitMode              `json:"mode,omitempty"`
	Columns                  int                         `json:"columns,omitempty"`
	Rows                     int                         `json:"rows,omitempty"`
	ForceProportionalGrid    bool                        `json:"force_proportional_grid,omitempty"`
	DetectGridBounds         bool                        `json:"detect_grid_bounds,omitempty"`
	AlphaThreshold           uint8                       `json:"alpha_threshold,omitempty"`
	MinComponentPixels       int                         `json:"min_component_pixels,omitempty"`
	MinBandSize              int                         `json:"min_band_size,omitempty"`
	ProjectionMergeGap       int                         `json:"projection_merge_gap,omitempty"`
	CropToContent            bool                        `json:"crop_to_content,omitempty"`
	AllowEmptyRegions        bool                        `json:"allow_empty_regions,omitempty"`
	FrameCount               int                         `json:"frame_count,omitempty"`
	FrameWidth               int                         `json:"frame_width,omitempty"`
	FrameHeight              int                         `json:"frame_height,omitempty"`
	Margin                   int                         `json:"margin,omitempty"`
	CropPadding              int                         `json:"crop_padding,omitempty"`
	Anchor                   AnimationAnchor             `json:"anchor,omitempty"`
	PreserveHorizontalMotion bool                        `json:"preserve_horizontal_motion,omitempty"`
	PreserveVerticalMotion   bool                        `json:"preserve_vertical_motion,omitempty"`
	MaxStabilizationShift    int                         `json:"max_stabilization_shift,omitempty"`
	Background               *AnimationBackgroundOptions `json:"background,omitempty"`
}

// ImageRegion is one independently encoded PNG region. SourceBounds are
// relative to the input image. ContentBounds are relative to the returned
// region image and are nil when the region has no visible pixels.
type ImageRegion struct {
	Index         int               `json:"index"`
	ImageBase64   string            `json:"image_base64"`
	MIMEType      string            `json:"mime_type"`
	SourceBounds  AlphaBoundingBox  `json:"source_bounds"`
	ContentBounds *AlphaBoundingBox `json:"content_bounds,omitempty"`
	SourceAnchor  *AnimationPoint   `json:"source_anchor,omitempty"`
	OutputAnchor  *AnimationPoint   `json:"output_anchor,omitempty"`
	Translation   *AnimationOffset  `json:"translation,omitempty"`
}

// SplitImageResult contains regions in deterministic reading order.
type SplitImageResult struct {
	Mode            ImageSplitMode                `json:"mode"`
	Width           int                           `json:"width"`
	Height          int                           `json:"height"`
	OutputWidth     int                           `json:"output_width,omitempty"`
	OutputHeight    int                           `json:"output_height,omitempty"`
	FrameWidth      int                           `json:"frame_width,omitempty"`
	FrameHeight     int                           `json:"frame_height,omitempty"`
	Columns         int                           `json:"columns,omitempty"`
	Rows            int                           `json:"rows,omitempty"`
	ImageBase64     string                        `json:"image_base64,omitempty"`
	MIMEType        string                        `json:"mime_type,omitempty"`
	AnimationReport *AnimationNormalizationReport `json:"animation_report,omitempty"`
	Regions         []ImageRegion                 `json:"regions"`
}

// SplitImage separates a transparent image or a larger image of
// multiple poses into independently encoded PNG regions.
func (p *processor) SplitImage(ctx context.Context, request *SplitImageRequest) (*SplitImageResult, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, fmt.Errorf("split image request is required")
	}

	input, err := decodeBase64Image(request.ImageBase64)
	if err != nil {
		return nil, fmt.Errorf("decode input image: %w", err)
	}
	result, err := splitImage(input.image, *request)
	if err != nil {
		return nil, fmt.Errorf("split image: %w", err)
	}
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func splitImage(src image.Image, request SplitImageRequest) (*SplitImageResult, error) {
	if src == nil {
		return nil, fmt.Errorf("source image is required")
	}
	mode := request.Mode
	if mode == "" {
		if request.Columns > 0 || request.Rows > 0 {
			mode = ImageSplitModeAnimation
		} else {
			mode = ImageSplitModeComponents
		}
	}
	switch mode {
	case ImageSplitModeGrid, ImageSplitModeAnimation, ImageSplitModeComponents, ImageSplitModeProjection:
	default:
		return nil, fmt.Errorf("unsupported splitting mode %q", mode)
	}
	threshold := request.AlphaThreshold
	if threshold == 0 {
		threshold = defaultImageSplitAlphaThreshold
	}
	if request.MinComponentPixels < 0 {
		return nil, fmt.Errorf("min component pixels must not be negative")
	}
	if request.MinBandSize < 0 {
		return nil, fmt.Errorf("min band size must not be negative")
	}
	if request.ProjectionMergeGap < 0 {
		return nil, fmt.Errorf("projection merge gap must not be negative")
	}

	input := toNRGBA(src)
	if mode == ImageSplitModeAnimation {
		return splitAnimation(input, request, threshold)
	}
	var regions []splitRegion
	columns, rows := 0, 0
	switch mode {
	case ImageSplitModeGrid:
		if request.Columns <= 0 || request.Rows <= 0 {
			return nil, fmt.Errorf("grid splitting requires positive columns and rows")
		}
		if request.Columns > input.Bounds().Dx() || request.Rows > input.Bounds().Dy() {
			return nil, fmt.Errorf("grid dimensions %dx%d exceed source image size %dx%d", request.Columns, request.Rows, input.Bounds().Dx(), input.Bounds().Dy())
		}
		columns, rows = request.Columns, request.Rows
		regions = gridRegions(input, columns, rows, request.DetectGridBounds && !request.ForceProportionalGrid, threshold)
	case ImageSplitModeComponents:
		minPixels := request.MinComponentPixels
		if minPixels == 0 {
			minPixels = 4
		}
		regions = componentRegions(input, threshold, minPixels)
	case ImageSplitModeProjection:
		minBandSize := request.MinBandSize
		if minBandSize == 0 {
			minBandSize = 2
		}
		regions = projectionRegions(input, threshold, minBandSize, request.ProjectionMergeGap)
	}
	if len(regions) == 0 {
		return nil, fmt.Errorf("no image regions found")
	}

	outputRegions := make([]ImageRegion, 0, len(regions))
	for index, sourceRegion := range regions {
		output := cloneNRGBA(input.SubImage(sourceRegion.bounds))
		content, hasContent := alphaBoundsNRGBA(output, threshold)
		if !hasContent && !request.AllowEmptyRegions {
			return nil, fmt.Errorf("region %d is empty", index)
		}
		if hasContent && request.CropToContent {
			output = cloneNRGBA(output.SubImage(content))
			content = image.Rect(0, 0, content.Dx(), content.Dy())
		}
		encoded, err := EncodePNGBase64(output)
		if err != nil {
			return nil, fmt.Errorf("encode region %d: %w", index, err)
		}
		var contentBox *AlphaBoundingBox
		if hasContent {
			box := rectangleToAlphaBoundingBox(content)
			contentBox = &box
		}
		outputRegions = append(outputRegions, ImageRegion{
			Index:         index,
			ImageBase64:   encoded,
			MIMEType:      pngMIMEType,
			SourceBounds:  rectangleToAlphaBoundingBox(sourceRegion.bounds),
			ContentBounds: contentBox,
		})
	}
	return &SplitImageResult{
		Mode: mode, Width: input.Bounds().Dx(), Height: input.Bounds().Dy(),
		Columns: columns, Rows: rows, Regions: outputRegions,
	}, nil
}

func splitAnimation(input *image.NRGBA, request SplitImageRequest, threshold uint8) (*SplitImageResult, error) {
	if request.CropToContent {
		return nil, fmt.Errorf("animation mode does not support per-frame crop_to_content; it uses one shared crop to prevent displacement")
	}
	background := request.Background
	if background == nil && !hasTransparentPixel(input, threshold) {
		background = &AnimationBackgroundOptions{MatteColor: "auto"}
	}
	result, err := normalizeAnimationImage(input, normalizeAnimationRequest{
		Columns: request.Columns, Rows: request.Rows, FrameCount: request.FrameCount,
		FrameWidth: request.FrameWidth, FrameHeight: request.FrameHeight,
		Margin: request.Margin, CropPadding: request.CropPadding,
		AlphaThreshold: request.AlphaThreshold, Anchor: request.Anchor,
		PreserveHorizontalMotion: request.PreserveHorizontalMotion,
		PreserveVerticalMotion:   request.PreserveVerticalMotion,
		MaxStabilizationShift:    request.MaxStabilizationShift,
		DetectGridBounds:         request.DetectGridBounds && !request.ForceProportionalGrid,
		AllowEmptyFrames:         request.AllowEmptyRegions,
		Background:               background,
	})
	if err != nil {
		return nil, err
	}

	regions := make([]ImageRegion, 0, len(result.Frames))
	for _, frame := range result.Frames {
		translation := frame.Translation
		regions = append(regions, ImageRegion{
			Index: frame.Index, ImageBase64: frame.ImageBase64, MIMEType: frame.MIMEType,
			SourceBounds: frame.SourceBounds, ContentBounds: frame.ContentBounds,
			SourceAnchor: frame.SourceAnchor, OutputAnchor: frame.OutputAnchor,
			Translation: &translation,
		})
	}
	report := result.Report
	return &SplitImageResult{
		Mode:  ImageSplitModeAnimation,
		Width: report.SourceWidth, Height: report.SourceHeight,
		OutputWidth: report.SheetWidth, OutputHeight: report.SheetHeight,
		FrameWidth: report.FrameWidth, FrameHeight: report.FrameHeight,
		Columns: report.Columns, Rows: report.Rows,
		ImageBase64: result.ImageBase64, MIMEType: result.MIMEType,
		AnimationReport: &report, Regions: regions,
	}, nil
}

type splitRegion struct {
	bounds image.Rectangle
}

func gridRegions(img *image.NRGBA, columns, rows int, detectBounds bool, threshold uint8) []splitRegion {
	xBounds, yBounds := proportionalBounds(img.Bounds().Dx(), columns), proportionalBounds(img.Bounds().Dy(), rows)
	if detectBounds {
		xBounds, yBounds = detectGridBoundsNRGBA(img, columns, rows, false, threshold)
	}
	regions := make([]splitRegion, 0, columns*rows)
	for row := range rows {
		for column := range columns {
			regions = append(regions, splitRegion{bounds: image.Rect(
				xBounds[column], yBounds[row], xBounds[column+1], yBounds[row+1],
			)})
		}
	}
	return regions
}

func detectGridBoundsNRGBA(img *image.NRGBA, columns, rows int, forceProportional bool, threshold uint8) ([]int, []int) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if forceProportional {
		return proportionalBounds(w, columns), proportionalBounds(h, rows)
	}
	xCounts := make([]int, w)
	yCounts := make([]int, h)
	for y := range h {
		for x := range w {
			if img.NRGBAAt(x, y).A > threshold {
				xCounts[x]++
				yCounts[y]++
			}
		}
	}
	xRuns := projectionRuns(xCounts, splitMaxInt(1, h/200), splitMaxInt(1, w/160), splitMaxInt(1, w/80))
	yRuns := projectionRuns(yCounts, splitMaxInt(1, w/200), splitMaxInt(1, h/160), splitMaxInt(1, h/80))
	xBounds, xOK := boundariesFromRuns(xRuns, columns, w)
	yBounds, yOK := boundariesFromRuns(yRuns, rows, h)
	if !xOK {
		xBounds = proportionalBounds(w, columns)
	}
	if !yOK {
		yBounds = proportionalBounds(h, rows)
	}
	return xBounds, yBounds
}

func projectionRegions(img *image.NRGBA, threshold uint8, minBandSize, mergeGap int) []splitRegion {
	// Projection mode starts with alpha-connected regions, then joins regions
	// that are close enough to belong to the same pose. This handles common
	// generated character sheets where a head, body, weapon, or shadow is
	// separated by transparent pixels but still occupies one visual column.
	components := componentRegions(img, threshold, splitMaxInt(1, minBandSize*minBandSize))
	if len(components) < 2 {
		return components
	}

	gap := mergeGap
	if gap == 0 {
		gap = splitMaxInt(2, splitMinInt(img.Bounds().Dx(), img.Bounds().Dy())/16)
	}
	parent := make([]int, len(components))
	for index := range parent {
		parent[index] = index
	}
	var find func(int) int
	find = func(index int) int {
		if parent[index] == index {
			return index
		}
		parent[index] = find(parent[index])
		return parent[index]
	}
	union := func(left, right int) {
		leftRoot, rightRoot := find(left), find(right)
		if leftRoot != rightRoot {
			parent[rightRoot] = leftRoot
		}
	}

	for left := range components {
		for right := left + 1; right < len(components); right++ {
			if splitExpandRect(components[left].bounds, gap).Overlaps(splitExpandRect(components[right].bounds, gap)) {
				union(left, right)
			}
		}
	}

	merged := make(map[int]image.Rectangle, len(components))
	for index, component := range components {
		root := find(index)
		if bounds, ok := merged[root]; ok {
			merged[root] = bounds.Union(component.bounds)
		} else {
			merged[root] = component.bounds
		}
	}
	regions := make([]splitRegion, 0, len(merged))
	for _, bounds := range merged {
		regions = append(regions, splitRegion{bounds: bounds})
	}
	sort.SliceStable(regions, func(i, j int) bool {
		left, right := regions[i].bounds, regions[j].bounds
		if left.Min.Y != right.Min.Y {
			return left.Min.Y < right.Min.Y
		}
		return left.Min.X < right.Min.X
	})
	return regions
}

func splitExpandRect(rect image.Rectangle, gap int) image.Rectangle {
	return image.Rect(rect.Min.X-gap, rect.Min.Y-gap, rect.Max.X+gap, rect.Max.Y+gap)
}

func componentRegions(img *image.NRGBA, threshold uint8, minPixels int) []splitRegion {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	visited := make([]bool, width*height)
	regions := make([]splitRegion, 0)
	for y := range height {
		for x := range width {
			index := y*width + x
			if visited[index] || img.NRGBAAt(x, y).A <= threshold {
				continue
			}
			visited[index] = true
			queue := []image.Point{{X: x, Y: y}}
			minX, maxX, minY, maxY, pixels := x, x, y, y, 0
			for head := 0; head < len(queue); head++ {
				point := queue[head]
				pixels++
				minX, maxX = splitMinInt(minX, point.X), splitMaxInt(maxX, point.X)
				minY, maxY = splitMinInt(minY, point.Y), splitMaxInt(maxY, point.Y)
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						nx, ny := point.X+dx, point.Y+dy
						if nx < 0 || nx >= width || ny < 0 || ny >= height {
							continue
						}
						n := ny*width + nx
						if visited[n] || img.NRGBAAt(nx, ny).A <= threshold {
							continue
						}
						visited[n] = true
						queue = append(queue, image.Point{X: nx, Y: ny})
					}
				}
			}
			if pixels >= minPixels {
				regions = append(regions, splitRegion{bounds: image.Rect(minX, minY, maxX+1, maxY+1)})
			}
		}
	}
	sort.SliceStable(regions, func(i, j int) bool {
		left, right := regions[i].bounds, regions[j].bounds
		if left.Min.Y != right.Min.Y {
			return left.Min.Y < right.Min.Y
		}
		if left.Min.X != right.Min.X {
			return left.Min.X < right.Min.X
		}
		return left.Max.X*left.Max.Y < right.Max.X*right.Max.Y
	})
	return regions
}

func projectionRuns(counts []int, threshold, mergeGap, minSize int) []imageRun {
	runs := make([]imageRun, 0)
	start := -1
	for i, count := range counts {
		if count > threshold && start < 0 {
			start = i
		} else if count <= threshold && start >= 0 {
			runs = append(runs, imageRun{Start: start, End: i})
			start = -1
		}
	}
	if start >= 0 {
		runs = append(runs, imageRun{Start: start, End: len(counts)})
	}
	merged := make([]imageRun, 0, len(runs))
	for _, run := range runs {
		if len(merged) == 0 || run.Start-merged[len(merged)-1].End > mergeGap {
			merged = append(merged, run)
			continue
		}
		merged[len(merged)-1].End = run.End
	}
	filtered := merged[:0]
	for _, run := range merged {
		if run.End-run.Start >= minSize {
			filtered = append(filtered, run)
		}
	}
	return filtered
}

type imageRun struct{ Start, End int }

func boundariesFromRuns(runs []imageRun, expected, size int) ([]int, bool) {
	if len(runs) != expected {
		return nil, false
	}
	if expected == 1 {
		return []int{0, size}, true
	}
	centers := make([]float64, len(runs))
	for i, run := range runs {
		centers[i] = float64(run.Start+run.End) / 2
	}
	distances := make([]float64, len(centers)-1)
	for i := range distances {
		distances[i] = centers[i+1] - centers[i]
	}
	median := medianFloat64(distances)
	if median <= 0 {
		return nil, false
	}
	bounds := make([]int, expected+1)
	bounds[0] = splitMaxInt(0, int(centers[0]-median/2+0.5))
	for i := range len(centers) - 1 {
		bounds[i+1] = int((centers[i]+centers[i+1])/2 + 0.5)
	}
	bounds[len(bounds)-1] = splitMinInt(size, int(centers[len(centers)-1]+median/2+0.5))
	for i := range len(bounds) - 1 {
		if bounds[i+1] <= bounds[i] {
			return nil, false
		}
	}
	return bounds, true
}

func proportionalBounds(size, count int) []int {
	bounds := make([]int, count+1)
	for i := range bounds {
		bounds[i] = (i*size + count/2) / count
	}
	return bounds
}

func medianFloat64(values []float64) float64 {
	values = append([]float64(nil), values...)
	sort.Float64s(values)
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return (values[middle-1] + values[middle]) / 2
}

func alphaBoundsNRGBA(img *image.NRGBA, threshold uint8) (image.Rectangle, bool) {
	minX, minY := img.Bounds().Max.X, img.Bounds().Max.Y
	maxX, maxY := img.Bounds().Min.X, img.Bounds().Min.Y
	found := false
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.NRGBAAt(x, y).A <= threshold {
				continue
			}
			found = true
			minX, maxX = splitMinInt(minX, x), splitMaxInt(maxX, x+1)
			minY, maxY = splitMinInt(minY, y), splitMaxInt(maxY, y+1)
		}
	}
	if !found {
		return image.Rectangle{}, false
	}
	return image.Rect(minX, minY, maxX, maxY), true
}

func rectangleToAlphaBoundingBox(rect image.Rectangle) AlphaBoundingBox {
	return AlphaBoundingBox{X: rect.Min.X, Y: rect.Min.Y, Width: rect.Dx(), Height: rect.Dy()}
}

func cloneNRGBA(src image.Image) *image.NRGBA {
	return toNRGBA(src)
}

func splitMinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func splitMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
