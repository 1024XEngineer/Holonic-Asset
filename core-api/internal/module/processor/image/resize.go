package image

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
)

// RasterMode selects the local final-size conversion strategy.
type RasterMode string

const (
	resizeSamplingArea     = "alpha-aware-area"
	resizeSamplingBilinear = "alpha-aware-bilinear"

	// RasterModeSmooth is intended for regular 2D game art. It uses alpha-aware
	// area resampling and preserves semi-transparent edge coverage.
	RasterModeSmooth RasterMode = "smooth"
	// RasterModePixel is intended for deliberate pixel art. It still uses area
	// resampling when reducing a large source; nearest-neighbour is only suitable
	// for enlarging an already-small sprite for preview.
	RasterModePixel RasterMode = "pixel"
)

// ResizeOptions controls deterministic local conversion from a generated
// illustration to a final game-asset canvas. Mode determines whether the
// result is smooth 2D art or deliberate pixel art.
type ResizeOptions struct {
	Width       int        `json:"width"`
	Height      int        `json:"height"`
	Margin      int        `json:"margin"`       // -1 chooses a proportional margin (about 6.25%).
	PaletteSize int        `json:"palette_size"` // 0 preserves the source colours.
	CropContent bool       `json:"crop_content"`
	HardAlpha   bool       `json:"hard_alpha"`
	Mode        RasterMode `json:"mode"`
}

// DefaultResizeOptions returns non-destructive defaults for regular 2D game
// assets: content cropping, proportional transparent margin, full colour, and
// smooth alpha edges.
func DefaultResizeOptions(width, height int) ResizeOptions {
	return ResizeOptions{
		Width:       width,
		Height:      height,
		Margin:      -1,
		PaletteSize: 0,
		CropContent: true,
		HardAlpha:   false,
		Mode:        RasterModeSmooth,
	}
}

// AnimationFrameMargin returns the shared safety margin used by canonical
// prototype frames and generated animation frames. Keeping this policy in the
// processor prevents the prototype and animation pipelines from choosing
// different subject scales. A quarter-frame margin leaves the centre half of
// the frame available for the canonical pose while leaving room for held
// objects and their motion.
func AnimationFrameMargin(width, height int) int {
	margin := min(width, height) / 4
	if margin < 1 {
		return 1
	}
	return margin
}

// AnimationFrameResizeOptions returns the canonical padded-frame layout. The
// source is cropped to its visible subject, then placed on a fixed canvas using
// the proportional margin expected by the animation generator. Animation
// splitting can then preserve that full canvas instead of fitting action
// bounds. Callers may still change Mode, PaletteSize, or HardAlpha for
// pixel-art output.
func AnimationFrameResizeOptions(width, height int) ResizeOptions {
	options := DefaultResizeOptions(width, height)
	options.Margin = AnimationFrameMargin(width, height)
	return options
}

type ResizeReport struct {
	InputWidth       int        `json:"input_width"`
	InputHeight      int        `json:"input_height"`
	OutputWidth      int        `json:"output_width"`
	OutputHeight     int        `json:"output_height"`
	CroppedToContent bool       `json:"cropped_to_content"`
	Margin           int        `json:"margin"`
	PaletteSize      int        `json:"palette_size"`
	HardAlpha        bool       `json:"hard_alpha"`
	Mode             RasterMode `json:"mode"`
	Sampling         string     `json:"sampling"`
}

// ResizeImage optionally crops transparent padding, downsamples with
// alpha-aware area filtering, and returns a final-size PNG-ready canvas.
//
// The old implementation sampled one source pixel per destination pixel. That
// discarded most source information and produced jagged silhouettes. Area
// filtering instead integrates every covered source pixel with alpha-weighted
// straight colours, avoiding transparent-edge halos and preserving small
// features.
func ResizeImage(input image.Image, opts ResizeOptions) (*image.RGBA, ResizeReport, error) {
	if input == nil {
		return nil, ResizeReport{}, fmt.Errorf("input image is required")
	}
	if opts.Width <= 0 || opts.Height <= 0 {
		return nil, ResizeReport{}, fmt.Errorf("asset dimensions must be positive")
	}
	if opts.PaletteSize < 0 {
		return nil, ResizeReport{}, fmt.Errorf("palette size cannot be negative")
	}
	mode := opts.Mode
	if mode == "" {
		mode = RasterModeSmooth
	}
	if mode != RasterModeSmooth && mode != RasterModePixel {
		return nil, ResizeReport{}, fmt.Errorf("raster mode must be smooth or pixel")
	}
	margin := opts.Margin
	if margin == -1 {
		margin = defaultAssetMargin(opts.Width, opts.Height)
	}
	if margin < 0 || margin*2 >= opts.Width || margin*2 >= opts.Height {
		return nil, ResizeReport{}, fmt.Errorf("asset margin must be between 0 and half the target dimensions")
	}

	img := toNRGBA(input)
	inW, inH := img.Bounds().Dx(), img.Bounds().Dy()
	if inW <= 0 || inH <= 0 {
		return nil, ResizeReport{}, fmt.Errorf("resize source must not be empty")
	}
	crop := image.Rect(0, 0, inW, inH)
	cropped := false
	if opts.CropContent {
		if bounds, ok := alphaBounds(img); ok {
			crop = bounds
			cropped = crop != image.Rect(0, 0, inW, inH)
		}
	}

	innerW, innerH := opts.Width-2*margin, opts.Height-2*margin
	out, placement, sampling := resizeContain(img, crop, innerW, innerH)
	if opts.PaletteSize > 0 {
		applyPalette(out, opts.PaletteSize)
	}
	if opts.HardAlpha {
		applyHardAlpha(out, 112)
	}

	canvas := image.NewNRGBA(image.Rect(0, 0, opts.Width, opts.Height))
	for y := range out.Bounds().Dy() {
		for x := range out.Bounds().Dx() {
			canvas.SetNRGBA(
				margin+placement.X+x,
				margin+placement.Y+y,
				out.NRGBAAt(x, y),
			)
		}
	}
	scrubTransparentNRGBA(canvas)
	return ToRGBA(canvas), ResizeReport{
		InputWidth: inW, InputHeight: inH,
		OutputWidth: opts.Width, OutputHeight: opts.Height,
		CroppedToContent: cropped, Margin: margin,
		PaletteSize: opts.PaletteSize, HardAlpha: opts.HardAlpha,
		Mode: mode, Sampling: sampling,
	}, nil
}

func defaultAssetMargin(width, height int) int {
	margin := min(width, height) / 16
	if margin < 1 {
		return 1
	}
	return margin
}

func alphaBounds(img image.Image) (image.Rectangle, bool) {
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	found := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if colorChannel8(a) <= TransparentAlphaMax {
				continue
			}
			found = true
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x >= maxX {
				maxX = x + 1
			}
			if y >= maxY {
				maxY = y + 1
			}
		}
	}
	if !found {
		return image.Rectangle{}, false
	}
	return image.Rect(minX-b.Min.X, minY-b.Min.Y, maxX-b.Min.X, maxY-b.Min.Y), true
}

// resizeContain preserves the selected source rectangle's aspect ratio and
// centres the scaled result inside the available destination area.
func resizeContain(
	src *image.NRGBA,
	crop image.Rectangle,
	dstW, dstH int,
) (*image.NRGBA, image.Point, string) {
	scaledW, scaledH := containDimensions(crop.Dx(), crop.Dy(), dstW, dstH)
	placement := image.Pt((dstW-scaledW)/2, (dstH-scaledH)/2)
	resized, sampling := qualityResize(src, crop, scaledW, scaledH)
	return resized, placement, sampling
}

func containDimensions(srcW, srcH, dstW, dstH int) (int, int) {
	if int64(srcW)*int64(dstH) > int64(srcH)*int64(dstW) {
		scaledH := roundedScale(srcH, dstW, srcW)
		return dstW, max(1, min(dstH, scaledH))
	}
	scaledW := roundedScale(srcW, dstH, srcH)
	return max(1, min(dstW, scaledW)), dstH
}

func roundedScale(value, numerator, denominator int) int {
	scaled := int64(value) * int64(numerator)
	return int((scaled + int64(denominator)/2) / int64(denominator))
}

// qualityResize integrates source coverage for reductions. For enlargement it
// uses bilinear interpolation; nearest-neighbour enlargement is a presentation
// choice that should be done by the game engine for an existing pixel sprite.
func qualityResize(src *image.NRGBA, crop image.Rectangle, dstW, dstH int) (*image.NRGBA, string) {
	if crop.Dx() <= dstW && crop.Dy() <= dstH {
		return bilinearResize(src, crop, dstW, dstH), resizeSamplingBilinear
	}
	return areaResize(src, crop, dstW, dstH), resizeSamplingArea
}

// areaResize performs exact box/area resampling with alpha-weighted straight
// colours. It considers all source pixels covered by a destination pixel
// rather than a single nearest sample, which is essential when reducing
// 1024px art to 64px.
func areaResize(src *image.NRGBA, crop image.Rectangle, dstW, dstH int) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	scaleX := float64(crop.Dx()) / float64(dstW)
	scaleY := float64(crop.Dy()) / float64(dstH)
	for dy := range dstH {
		sy0 := float64(crop.Min.Y) + float64(dy)*scaleY
		sy1 := float64(crop.Min.Y) + float64(dy+1)*scaleY
		y0, y1 := int(math.Floor(sy0)), int(math.Ceil(sy1))
		for dx := range dstW {
			sx0 := float64(crop.Min.X) + float64(dx)*scaleX
			sx1 := float64(crop.Min.X) + float64(dx+1)*scaleX
			x0, x1 := int(math.Floor(sx0)), int(math.Ceil(sx1))
			var sumR, sumG, sumB, sumAlpha, sumWeight float64
			for sy := y0; sy < y1; sy++ {
				if sy < crop.Min.Y || sy >= crop.Max.Y {
					continue
				}
				wy := math.Min(sy1, float64(sy+1)) - math.Max(sy0, float64(sy))
				if wy <= 0 {
					continue
				}
				for sx := x0; sx < x1; sx++ {
					if sx < crop.Min.X || sx >= crop.Max.X {
						continue
					}
					wx := math.Min(sx1, float64(sx+1)) - math.Max(sx0, float64(sx))
					weight := wx * wy
					if weight <= 0 {
						continue
					}
					p := src.NRGBAAt(sx, sy)
					alphaWeight := float64(p.A) * weight
					sumR += float64(p.R) * alphaWeight
					sumG += float64(p.G) * alphaWeight
					sumB += float64(p.B) * alphaWeight
					sumAlpha += alphaWeight
					sumWeight += weight
				}
			}
			if sumWeight == 0 || sumAlpha == 0 {
				continue
			}
			out.SetNRGBA(dx, dy, color.NRGBA{
				R: clampByte(sumR / sumAlpha), G: clampByte(sumG / sumAlpha),
				B: clampByte(sumB / sumAlpha), A: clampByte(sumAlpha / sumWeight),
			})
		}
	}
	return out
}

func bilinearResize(src *image.NRGBA, crop image.Rectangle, dstW, dstH int) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	for dy := range dstH {
		sy := float64(crop.Min.Y) + (float64(dy)+0.5)*float64(crop.Dy())/float64(dstH) - 0.5
		y0 := int(math.Floor(sy))
		fy := sy - float64(y0)
		if y0 < crop.Min.Y {
			y0, fy = crop.Min.Y, 0
		}
		y1 := min(y0+1, crop.Max.Y-1)
		for dx := range dstW {
			sx := float64(crop.Min.X) + (float64(dx)+0.5)*float64(crop.Dx())/float64(dstW) - 0.5
			x0 := int(math.Floor(sx))
			fx := sx - float64(x0)
			if x0 < crop.Min.X {
				x0, fx = crop.Min.X, 0
			}
			x1 := min(x0+1, crop.Max.X-1)
			pixels := [4]color.NRGBA{
				src.NRGBAAt(x0, y0),
				src.NRGBAAt(x1, y0),
				src.NRGBAAt(x0, y1),
				src.NRGBAAt(x1, y1),
			}
			weights := [4]float64{
				(1 - fx) * (1 - fy),
				fx * (1 - fy),
				(1 - fx) * fy,
				fx * fy,
			}
			out.SetNRGBA(dx, dy, alphaWeightedNRGBA(pixels, weights))
		}
	}
	return out
}

func alphaWeightedNRGBA(pixels [4]color.NRGBA, weights [4]float64) color.NRGBA {
	var sumR, sumG, sumB, sumAlpha, sumWeight float64
	for index, pixel := range pixels {
		weight := weights[index]
		alphaWeight := float64(pixel.A) * weight
		sumR += float64(pixel.R) * alphaWeight
		sumG += float64(pixel.G) * alphaWeight
		sumB += float64(pixel.B) * alphaWeight
		sumAlpha += alphaWeight
		sumWeight += weight
	}
	if sumWeight == 0 || sumAlpha == 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{
		R: clampByte(sumR / sumAlpha),
		G: clampByte(sumG / sumAlpha),
		B: clampByte(sumB / sumAlpha),
		A: clampByte(sumAlpha / sumWeight),
	}
}

func clampByte(value float64) uint8 {
	if value <= 0 {
		return 0
	}
	if value >= 255 {
		return 255
	}
	return uint8(math.Round(value))
}

func applyHardAlpha(img *image.NRGBA, threshold uint8) {
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			p := img.NRGBAAt(x, y)
			if p.A < threshold {
				img.SetNRGBA(x, y, color.NRGBA{})
				continue
			}
			p.A = 255
			img.SetNRGBA(x, y, p)
		}
	}
}

func scrubTransparentNRGBA(img *image.NRGBA) {
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			pixel := img.NRGBAAt(x, y)
			if pixel.A <= TransparentAlphaMax {
				img.SetNRGBA(x, y, color.NRGBA{})
			}
		}
	}
}

type palettePoint struct {
	r, g, b float64
	weight  uint64
}

type paletteBox struct {
	points      []palettePoint
	minR, maxR  float64
	minG, maxG  float64
	minB, maxB  float64
	totalWeight uint64
}

func newPaletteBox(points []palettePoint) paletteBox {
	box := paletteBox{points: points, minR: 255, minG: 255, minB: 255}
	for _, p := range points {
		box.minR, box.maxR = math.Min(box.minR, p.r), math.Max(box.maxR, p.r)
		box.minG, box.maxG = math.Min(box.minG, p.g), math.Max(box.maxG, p.g)
		box.minB, box.maxB = math.Min(box.minB, p.b), math.Max(box.maxB, p.b)
		box.totalWeight += p.weight
	}
	return box
}

func (box paletteBox) score() float64 {
	rangeMax := math.Max(box.maxR-box.minR, math.Max(box.maxG-box.minG, box.maxB-box.minB))
	return rangeMax * rangeMax * math.Sqrt(float64(box.totalWeight))
}

func splitPaletteBox(box paletteBox) (paletteBox, paletteBox, bool) {
	if len(box.points) < 2 {
		return paletteBox{}, paletteBox{}, false
	}
	rRange, gRange, bRange := box.maxR-box.minR, box.maxG-box.minG, box.maxB-box.minB
	channel := 0
	if gRange >= rRange && gRange >= bRange {
		channel = 1
	} else if bRange >= rRange && bRange >= gRange {
		channel = 2
	}
	sort.Slice(box.points, func(i, j int) bool {
		a, b := box.points[i], box.points[j]
		switch channel {
		case 1:
			return a.g < b.g
		case 2:
			return a.b < b.b
		default:
			return a.r < b.r
		}
	})
	half, accumulated, split := box.totalWeight/2, uint64(0), 1
	for i, point := range box.points[:len(box.points)-1] {
		accumulated += point.weight
		if accumulated >= half {
			split = i + 1
			break
		}
	}
	return newPaletteBox(box.points[:split]), newPaletteBox(box.points[split:]), true
}

// applyPalette uses weighted median-cut rather than retaining only the most
// frequent histogram buckets. This keeps isolated accent colours and small
// equipment details much more reliably.
func applyPalette(img *image.NRGBA, limit int) {
	if limit <= 0 {
		return
	}
	type accumulator struct{ r, g, b, weight uint64 }
	hist := make(map[int]*accumulator)
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			p := img.NRGBAAt(x, y)
			if p.A <= TransparentAlphaMax {
				continue
			}
			key := int(p.R>>3)<<10 | int(p.G>>3)<<5 | int(p.B>>3)
			entry := hist[key]
			if entry == nil {
				entry = &accumulator{}
				hist[key] = entry
			}
			weight := uint64(p.A)
			entry.r += uint64(p.R) * weight
			entry.g += uint64(p.G) * weight
			entry.b += uint64(p.B) * weight
			entry.weight += weight
		}
	}
	points := make([]palettePoint, 0, len(hist))
	for _, entry := range hist {
		if entry.weight == 0 {
			continue
		}
		points = append(points, palettePoint{
			r:      float64(entry.r) / float64(entry.weight),
			g:      float64(entry.g) / float64(entry.weight),
			b:      float64(entry.b) / float64(entry.weight),
			weight: entry.weight,
		})
	}
	if len(points) == 0 {
		return
	}
	boxes := []paletteBox{newPaletteBox(points)}
	for len(boxes) < limit {
		best := -1
		bestScore := -1.0
		for i, box := range boxes {
			if len(box.points) > 1 && box.score() > bestScore {
				best, bestScore = i, box.score()
			}
		}
		if best < 0 {
			break
		}
		left, right, ok := splitPaletteBox(boxes[best])
		if !ok {
			break
		}
		boxes[best] = left
		boxes = append(boxes, right)
	}
	palette := make([]color.RGBA, 0, len(boxes))
	for _, box := range boxes {
		var r, g, b float64
		for _, point := range box.points {
			weight := float64(point.weight)
			r += point.r * weight
			g += point.g * weight
			b += point.b * weight
		}
		weight := math.Max(1, float64(box.totalWeight))
		palette = append(palette, color.RGBA{R: clampByte(r / weight), G: clampByte(g / weight), B: clampByte(b / weight), A: 255})
	}
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			p := img.NRGBAAt(x, y)
			if p.A <= TransparentAlphaMax {
				continue
			}
			best, bestDist := palette[0], int64(math.MaxInt64)
			for _, candidate := range palette {
				dr := int64(p.R) - int64(candidate.R)
				dg := int64(p.G) - int64(candidate.G)
				db := int64(p.B) - int64(candidate.B)
				distance := 2*dr*dr + 4*dg*dg + 3*db*db
				if distance < bestDist {
					best, bestDist = candidate, distance
				}
			}
			img.SetNRGBA(x, y, color.NRGBA{
				R: best.R,
				G: best.G,
				B: best.B,
				A: p.A,
			})
		}
	}
}
