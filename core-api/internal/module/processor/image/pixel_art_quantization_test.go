package image

import (
	"image"
	"image/color"
	"slices"
	"testing"
)

func TestQuantizePixelArtSourceIsDeterministicAndBounded(t *testing.T) {
	t.Parallel()

	first := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	second := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			pixel := color.NRGBA{
				R: uint8(24 + x*5 + y%3),
				G: uint8(32 + y*6 + x%5),
				B: uint8(64 + (x+y)*3),
				A: 255,
			}
			first.SetNRGBA(x, y, pixel)
			second.SetNRGBA(x, y, pixel)
		}
	}

	quantizePixelArtSource(first, 6)
	quantizePixelArtSource(second, 6)
	colours := make(map[color.NRGBA]struct{})
	for y := range 32 {
		for x := range 32 {
			got := first.NRGBAAt(x, y)
			if got != second.NRGBAAt(x, y) {
				t.Fatalf("quantization is not deterministic at (%d,%d): %+v != %+v", x, y, got, second.NRGBAAt(x, y))
			}
			colours[got] = struct{}{}
		}
	}
	if len(colours) > 6 {
		t.Fatalf("quantized colour count = %d, want at most 6", len(colours))
	}
}

func TestQuantizePixelArtSourcePreservesSmallExistingPalette(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	palette := []color.NRGBA{
		{R: 28, G: 36, B: 48, A: 255},
		{R: 192, G: 64, B: 48, A: 255},
		{R: 48, G: 132, B: 208, A: 255},
	}
	for y := range 8 {
		for x := range 8 {
			img.SetNRGBA(x, y, palette[(x+y)%len(palette)])
		}
	}
	quantizePixelArtSource(img, len(palette))

	seen := make(map[color.NRGBA]struct{})
	for y := range 8 {
		for x := range 8 {
			seen[img.NRGBAAt(x, y)] = struct{}{}
		}
	}
	for got := range seen {
		if !slices.Contains(palette, got) {
			t.Fatalf("quantization synthesized colour %+v despite fitting source palette", got)
		}
	}
}

func TestPrequantizePixelArtSourceHardensAlphaBeforeQuantization(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 240, G: 20, B: 20, A: hardAlphaThreshold - 1})
	img.SetNRGBA(1, 0, color.NRGBA{R: 20, G: 240, B: 20, A: hardAlphaThreshold})

	prepared := prequantizePixelArtSource(img, ResizeOptions{PaletteSize: 2})
	if got := prepared.NRGBAAt(0, 0); got.A != 0 {
		t.Fatalf("below-threshold alpha survived prequantization: %+v", got)
	}
	if got := prepared.NRGBAAt(1, 0); got.A != 255 {
		t.Fatalf("threshold alpha was not promoted to opaque: %+v", got)
	}
}

func TestSpritePixelPipelineUsesNearestRepresentativeForThinLine(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	background := color.NRGBA{R: 28, G: 36, B: 48, A: 255}
	line := color.NRGBA{R: 224, G: 188, B: 48, A: 255}
	for y := range 8 {
		for x := range 8 {
			pixel := background
			if x < 4 {
				pixel = line
			}
			source.SetNRGBA(x, y, pixel)
		}
	}

	options := DefaultResizeOptions(2, 2)
	options.Margin = 0
	options.CropContent = false
	options.Mode = RasterModePixel
	options.PaletteSize = 2
	options.HardAlpha = true
	options.RecoverPixelGrid = true
	options.PrequantizeBeforeResize = true
	options.PreferNearestReduction = true
	options.SpritePixelPipeline = true
	result, report, err := ResizeImage(source, options)
	if err != nil {
		t.Fatalf("resize sprite: %v", err)
	}
	if report.Sampling != resizeSamplingNearest {
		t.Fatalf("sampling = %q, want Sprite-AI nearest sampling", report.Sampling)
	}
	for y := range 2 {
		if got := result.RGBAAt(0, y); got != color.RGBA(line) {
			t.Fatalf("thin line at (%d,%d) = %+v, want %+v", 0, y, got, line)
		}
		if got := result.RGBAAt(1, y); got != color.RGBA(background) {
			t.Fatalf("background at (%d,%d) = %+v, want %+v", 1, y, got, background)
		}
	}
}
