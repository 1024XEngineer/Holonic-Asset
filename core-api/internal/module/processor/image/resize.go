package image

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// RasterMode selects the local final-size conversion strategy.
type RasterMode string

const (
	resizeSamplingArea               = "alpha-aware-area"
	resizeSamplingBilinear           = "alpha-aware-bilinear"
	resizeSamplingPixelArea          = "alpha-aware-area-then-block-recolour"
	resizeSamplingNearest            = "nearest-neighbour"
	pixelGridSamplingNearestFallback = "recovered-pixel-grid-nearest"
	// PixelAlphaThreshold is the coverage cutoff shared by prototype splitting
	// and final pixel cleanup. Keeping the two stages on the same cutoff avoids
	// normalizing faint antialias pixels that are removed immediately after.
	PixelAlphaThreshold = uint8(112)
	hardAlphaThreshold  = PixelAlphaThreshold

	// RasterModeSmooth is intended for regular 2D game art. It uses alpha-aware
	// area resampling and preserves semi-transparent edge coverage.
	RasterModeSmooth RasterMode = "smooth"
	// RasterModePixel is intended for deliberate pixel art. Reduction first uses
	// alpha-aware area sampling to lock geometry, then repairs colours and alpha
	// only on the final logical pixel grid. Enlargement uses nearest-neighbour.
	RasterModePixel RasterMode = "pixel"

	// PrototypeRenderScale keeps prototype silhouettes above the final logical
	// pixel grid until the last resize. This preserves more contour evidence
	// for area sampling and avoids turning a smooth 32x32 frame into a jagged
	// mask before palette cleanup has a chance to run.
	PrototypeRenderScale = 4
)

// ResizeOptions controls deterministic local conversion from a generated
// illustration to a final game-asset canvas. Mode determines whether the
// result is smooth 2D art or deliberate pixel art.
type ResizeOptions struct {
	Width                   int        `json:"width"`
	Height                  int        `json:"height"`
	Margin                  int        `json:"margin"`       // -1 chooses a proportional margin (about 6.25%).
	PaletteSize             int        `json:"palette_size"` // 0 preserves the source colours.
	CropContent             bool       `json:"crop_content"`
	CoverCanvas             bool       `json:"cover_canvas"` // Crops to fill the full target and requires Margin 0.
	HardAlpha               bool       `json:"hard_alpha"`
	Mode                    RasterMode `json:"mode"`
	RecoverPixelGrid        bool       `json:"recover_pixel_grid"`        // Samples supersampled prototype frames on their recovered logical grid.
	PrequantizeBeforeResize bool       `json:"prequantize_before_resize"` // Quantizes source colours before geometry reduction, like a pixel-art converter.
	PreferNearestReduction  bool       `json:"prefer_nearest_reduction"`  // Uses nearest-neighbour for pixel-art reduction when no integral grid is available.
	SpritePixelPipeline     bool       `json:"sprite_pixel_pipeline"`     // Uses quantize-before-nearest conversion with no shape-repair heuristics.
	PreserveCanvasGeometry  bool       `json:"preserve_canvas_geometry"`  // Keeps a pre-padded fixed frame from being refit to visible content.
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

type ResizeReport struct {
	InputWidth       int        `json:"input_width"`
	InputHeight      int        `json:"input_height"`
	OutputWidth      int        `json:"output_width"`
	OutputHeight     int        `json:"output_height"`
	CroppedToContent bool       `json:"cropped_to_content"`
	CoveredCanvas    bool       `json:"covered_canvas"`
	Margin           int        `json:"margin"`
	PaletteSize      int        `json:"palette_size"`
	HardAlpha        bool       `json:"hard_alpha"`
	Mode             RasterMode `json:"mode"`
	Sampling         string     `json:"sampling"`
}

// ResizeImage optionally crops transparent padding and returns a final-size
// PNG-ready canvas. Smooth mode uses alpha-aware area or bilinear filtering.
// Pixel mode uses the Sprite-compatible nearest path when requested and the
// generic quality sampler otherwise. Palette mapping is deliberately limited to
// colour replacement; shape-repair heuristics are not part of resizing.
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
	if opts.CoverCanvas && margin != 0 {
		return nil, ResizeReport{}, fmt.Errorf("cover canvas requires zero margin")
	}

	img := toNRGBA(input)
	if mode == RasterModePixel && opts.PrequantizeBeforeResize && opts.PaletteSize > 0 {
		img = prequantizePixelArtSource(img, opts)
	}
	inW, inH := img.Bounds().Dx(), img.Bounds().Dy()
	if inW <= 0 || inH <= 0 {
		return nil, ResizeReport{}, fmt.Errorf("resize source must not be empty")
	}
	crop := image.Rect(0, 0, inW, inH)
	cropped := false
	if opts.CropContent {
		alphaThreshold := TransparentAlphaMax
		if opts.HardAlpha {
			alphaThreshold = hardAlphaThreshold - 1
			if opts.SpritePixelPipeline {
				alphaThreshold = spriteAIAlphaThreshold
			}
		}
		if bounds, ok := alphaBounds(img, alphaThreshold); ok {
			crop = bounds
			cropped = crop != image.Rect(0, 0, inW, inH)
		}
	}

	innerW, innerH := opts.Width-2*margin, opts.Height-2*margin
	var out *image.NRGBA
	var placement image.Point
	var sampling string
	if opts.CoverCanvas {
		crop = coverCrop(crop, innerW, innerH)
		out, sampling = resizeForMode(img, crop, innerW, innerH, mode, opts.RecoverPixelGrid, opts.PreferNearestReduction, opts.SpritePixelPipeline, opts.PreserveCanvasGeometry)
	} else if mode == RasterModePixel && opts.SpritePixelPipeline && !opts.PreserveCanvasGeometry {
		// The browser converter owns contain geometry itself: it fits the cropped
		// alpha content into a target-sized 4x intermediate canvas, centres it,
		// then performs the final nearest reduction. Passing a pre-contained
		// rectangle here applies contain twice and changes rounding at 32/64 px.
		out, sampling = resizeForMode(img, crop, innerW, innerH, mode, opts.RecoverPixelGrid, opts.PreferNearestReduction, opts.SpritePixelPipeline, false)
	} else {
		scaledW, scaledH := containDimensions(crop.Dx(), crop.Dy(), innerW, innerH)
		placement = image.Pt((innerW-scaledW)/2, (innerH-scaledH)/2)
		out, sampling = resizeForMode(img, crop, scaledW, scaledH, mode, opts.RecoverPixelGrid, opts.PreferNearestReduction, opts.SpritePixelPipeline, opts.PreserveCanvasGeometry)
	}

	if mode == RasterModePixel {
		if opts.SpritePixelPipeline {
			if opts.HardAlpha {
				applySpriteAIHardAlpha(out)
			}
			scrubTransparentNRGBA(out)
		} else {
			if opts.HardAlpha {
				applyHardAlpha(out, hardAlphaThreshold)
			}
			if opts.PaletteSize > 0 {
				pixelPalette := buildPalette(out, out.Bounds(), opts.PaletteSize, TransparentAlphaMax)
				remapToPalette(out, out.Bounds(), pixelPalette)
			}
		}
	} else {
		if opts.PaletteSize > 0 {
			applyPalette(out, opts.PaletteSize)
		}
		if opts.HardAlpha {
			applyHardAlpha(out, hardAlphaThreshold)
		}
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
		CroppedToContent: cropped, CoveredCanvas: opts.CoverCanvas, Margin: margin,
		PaletteSize: opts.PaletteSize, HardAlpha: opts.HardAlpha,
		Mode: mode, Sampling: sampling,
	}, nil
}

func coverCrop(src image.Rectangle, dstW, dstH int) image.Rectangle {
	if int64(src.Dx())*int64(dstH) > int64(src.Dy())*int64(dstW) {
		width := max(1, min(src.Dx(), int(int64(src.Dy())*int64(dstW)/int64(dstH))))
		left := src.Min.X + (src.Dx()-width)/2
		return image.Rect(left, src.Min.Y, left+width, src.Max.Y)
	}
	height := max(1, min(src.Dy(), int(int64(src.Dx())*int64(dstH)/int64(dstW))))
	top := src.Min.Y + (src.Dy()-height)/2
	return image.Rect(src.Min.X, top, src.Max.X, top+height)
}

func defaultAssetMargin(width, height int) int {
	margin := min(width, height) / 16
	if margin < 1 {
		return 1
	}
	return margin
}

func alphaBounds(img image.Image, threshold uint8) (image.Rectangle, bool) {
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	found := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if colorChannel8(a) <= threshold {
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

func resizeForMode(
	src *image.NRGBA,
	crop image.Rectangle,
	dstW, dstH int,
	mode RasterMode,
	recoverGrid, preferNearest, spritePipeline, preserveCanvasGeometry bool,
) (*image.NRGBA, string) {
	if mode == RasterModePixel && spritePipeline {
		return spriteAIResizeWithGeometry(src, crop, dstW, dstH, preserveCanvasGeometry), resizeSamplingNearest
	}
	if mode == RasterModePixel && recoverGrid {
		return recoverPixelGridResize(src, crop, dstW, dstH, preferNearest, spritePipeline)
	}
	if mode == RasterModePixel && preferNearest && (crop.Dx() > dstW || crop.Dy() > dstH) {
		return nearestResize(src, crop, dstW, dstH), resizeSamplingNearest
	}
	return qualityResize(src, crop, dstW, dstH, mode)
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

// qualityResize determines geometry only. Both smooth and pixel reductions use
// alpha-aware area sampling; pixel-specific colour cleanup happens afterwards
// on the final grid so it cannot shift or distort the silhouette.
func qualityResize(
	src *image.NRGBA,
	crop image.Rectangle,
	dstW, dstH int,
	mode RasterMode,
) (*image.NRGBA, string) {
	if mode == RasterModePixel {
		if crop.Dx() <= dstW && crop.Dy() <= dstH {
			return nearestResize(src, crop, dstW, dstH), resizeSamplingNearest
		}
		return areaResize(src, crop, dstW, dstH), resizeSamplingPixelArea
	}
	if crop.Dx() <= dstW && crop.Dy() <= dstH {
		return bilinearResize(src, crop, dstW, dstH), resizeSamplingBilinear
	}
	return areaResize(src, crop, dstW, dstH), resizeSamplingArea
}

func nearestResize(src *image.NRGBA, crop image.Rectangle, dstW, dstH int) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	for dy := range dstH {
		sy := crop.Min.Y + min(crop.Dy()-1, (2*dy+1)*crop.Dy()/(2*dstH))
		for dx := range dstW {
			sx := crop.Min.X + min(crop.Dx()-1, (2*dx+1)*crop.Dx()/(2*dstW))
			out.SetNRGBA(dx, dy, src.NRGBAAt(sx, sy))
		}
	}
	return out
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

// applySpriteAIHardAlpha mirrors the browser converter's strict comparison:
// alpha values must be greater than 128 to survive. The generic hard-alpha
// helper intentionally retains its historical inclusive threshold semantics.
func applySpriteAIHardAlpha(img *image.NRGBA) {
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			p := img.NRGBAAt(x, y)
			if p.A <= spriteAIAlphaThreshold {
				img.SetNRGBA(x, y, color.NRGBA{})
				continue
			}
			p.A = 255
			img.SetNRGBA(x, y, p)
		}
	}
}

type pixelColourKey uint32

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

// prequantizePixelArtSource follows the stable ordering used by dedicated
// image-to-pixel-art converters: remove translucent fringe and collapse the
// source colour field before reducing its geometry. If quantization happens
// after area reduction, neighbouring mixel blocks are averaged first and a
// new colour is created at every boundary; the palette pass can no longer
// recover the original seam or highlight.
func prequantizePixelArtSource(img *image.NRGBA, opts ResizeOptions) *image.NRGBA {
	prepared := cloneNRGBA(img)
	if opts.SpritePixelPipeline {
		applySpriteAIHardAlpha(prepared)
	} else {
		applyHardAlpha(prepared, hardAlphaThreshold)
	}

	quantizePixelArtSource(prepared, opts.PaletteSize)
	return prepared
}
