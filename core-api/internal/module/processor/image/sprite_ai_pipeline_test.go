package image

import (
	"image"
	"image/color"
	"testing"
)

func TestSpriteAINearestResizeUsesBrowserFloorSampling(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	colours := []color.NRGBA{
		{R: 255, A: 255},
		{G: 255, A: 255},
		{B: 255, A: 255},
		{R: 255, G: 255, A: 255},
	}
	for x, pixel := range colours {
		source.SetNRGBA(x, 0, pixel)
	}

	got := spriteAINearestResize(source, 2, 1)
	if got.NRGBAAt(0, 0) != colours[0] || got.NRGBAAt(1, 0) != colours[2] {
		t.Fatalf("nearest samples = %+v, %+v; want source indices 0 and 2", got.NRGBAAt(0, 0), got.NRGBAAt(1, 0))
	}
}

func TestSpriteAIResizeProducesOpaqueLogicalPixelsAfterHardAlpha(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			alpha := uint8(255)
			if x == 0 || y == 0 {
				alpha = 127
			}
			source.SetNRGBA(x, y, color.NRGBA{R: 220, G: 80, B: 40, A: alpha})
		}
	}
	applySpriteAIHardAlpha(source)
	result := spriteAIResize(source, image.Rect(0, 0, 8, 8), 4, 4)

	for y := range 4 {
		for x := range 4 {
			pixel := result.NRGBAAt(x, y)
			if pixel.A != 0 && pixel.A != 255 {
				t.Fatalf("partial alpha at (%d,%d): %+v", x, y, pixel)
			}
		}
	}
}

func TestQuantizePixelArtSourcesUsesSpriteAlphaThreshold(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	source.SetNRGBA(0, 0, color.NRGBA{R: 255, A: spriteAIAlphaThreshold - 1})
	source.SetNRGBA(1, 0, color.NRGBA{G: 255, A: spriteAIAlphaThreshold})
	source.SetNRGBA(2, 0, color.NRGBA{B: 255, A: spriteAIAlphaThreshold + 1})

	frames, err := QuantizePixelArtSources([]image.Image{source}, 2)
	if err != nil {
		t.Fatalf("quantize source: %v", err)
	}
	if got := frames[0].RGBAAt(0, 0).A; got != 0 {
		t.Fatalf("below-threshold alpha = %d, want 0", got)
	}
	if got := frames[0].RGBAAt(1, 0).A; got != 0 {
		t.Fatalf("threshold alpha = %d, want 0 for strict greater-than comparison", got)
	}
	if got := frames[0].RGBAAt(2, 0).A; got != 255 {
		t.Fatalf("above-threshold alpha = %d, want 255", got)
	}
}

func TestSpriteAIResizePreservesFixedCanvasGeometry(t *testing.T) {
	t.Parallel()

	// This models a supersampled animation frame: the subject is already
	// centred inside the frame and deliberately leaves an action/safety margin.
	source := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	for y := 28; y < 100; y++ {
		for x := 28; x < 100; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 220, G: 80, B: 40, A: 255})
		}
	}

	got := spriteAIResizeWithGeometry(source, source.Bounds(), 32, 32, true)
	bounds, visible := alphaBounds(got, spriteAIAlphaThreshold)
	if !visible {
		t.Fatal("fixed-frame resize removed the subject")
	}
	if bounds.Min.X == 0 || bounds.Max.X == 32 || bounds.Min.Y == 0 || bounds.Max.Y == 32 {
		t.Fatalf("fixed-frame resize refit content to canvas: bounds=%v", bounds)
	}
	if bounds.Min.X < 6 || bounds.Max.X > 26 || bounds.Min.Y < 6 || bounds.Max.Y > 26 {
		t.Fatalf("fixed-frame resize changed the safety margin too much: bounds=%v", bounds)
	}
}

func TestSpriteAIResizeWithoutFixedCanvasGeometryStillFitsContent(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	for y := 28; y < 100; y++ {
		for x := 28; x < 100; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 220, G: 80, B: 40, A: 255})
		}
	}

	got := spriteAIResize(source, source.Bounds(), 32, 32)
	bounds, visible := alphaBounds(got, spriteAIAlphaThreshold)
	if !visible {
		t.Fatal("content-fit resize removed the subject")
	}
	if bounds.Min.X != 0 || bounds.Max.X != 32 || bounds.Min.Y != 0 || bounds.Max.Y != 32 {
		t.Fatalf("content-fit resize should fill the target in standalone mode: bounds=%v", bounds)
	}
}

func TestObjectSpritePipelineFillsInnerCanvasThenAddsSafetyMargin(t *testing.T) {
	t.Parallel()

	// The visible object is deliberately off-centre in a larger generated frame.
	// Object conversion must ignore that source placement, fit the alpha bbox into
	// the inner drawable canvas, and only then add the final animation margin.
	source := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	for y := 40; y < 104; y++ {
		for x := 80; x < 112; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 220, G: 96, B: 40, A: 255})
		}
	}

	options := PrototypePixelResizeOptions(32, 32)
	options.Margin = AnimationFrameMargin(32, 32)
	options.CropContent = true
	options.PreserveCanvasGeometry = false

	got, report, err := ResizeImage(source, options)
	if err != nil {
		t.Fatalf("resize object sprite: %v", err)
	}
	if got.Bounds() != image.Rect(0, 0, 32, 32) {
		t.Fatalf("output bounds = %v, want 32x32", got.Bounds())
	}
	if !report.CroppedToContent {
		t.Fatal("off-centre object was not cropped to its alpha bounds")
	}

	bounds, visible := alphaBounds(toNRGBA(got), spriteAIAlphaThreshold-1)
	if !visible {
		t.Fatal("object disappeared during conversion")
	}
	margin := AnimationFrameMargin(32, 32)
	if bounds.Min.X < margin || bounds.Min.Y < margin || bounds.Max.X > 32-margin || bounds.Max.Y > 32-margin {
		t.Fatalf("object escaped animation safety area: bounds=%v margin=%d", bounds, margin)
	}
	if bounds.Dx() < 9 || bounds.Dy() < 19 {
		t.Fatalf("object did not substantially fill the 20x20 inner canvas: bounds=%v", bounds)
	}
	if gotCenterX, gotCenterY := bounds.Min.X+bounds.Dx()/2, bounds.Min.Y+bounds.Dy()/2; gotCenterX != 16 || gotCenterY != 16 {
		t.Fatalf("object was not centred after alpha crop: bounds=%v center=(%d,%d)", bounds, gotCenterX, gotCenterY)
	}
}

func TestObjectSpritePipelinePreservesLongThinAspectRatioInsideSafetyMargin(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 160, 96))
	for y := 60; y < 70; y++ {
		for x := 20; x < 140; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 196, G: 204, B: 216, A: 255})
		}
	}

	options := PrototypePixelResizeOptions(32, 32)
	options.Margin = AnimationFrameMargin(32, 32)
	options.CropContent = true
	options.PreserveCanvasGeometry = false
	got, _, err := ResizeImage(source, options)
	if err != nil {
		t.Fatalf("resize long object sprite: %v", err)
	}

	bounds, visible := alphaBounds(toNRGBA(got), spriteAIAlphaThreshold-1)
	if !visible {
		t.Fatal("long object disappeared during conversion")
	}
	margin := AnimationFrameMargin(32, 32)
	if bounds.Min.X < margin || bounds.Min.Y < margin || bounds.Max.X > 32-margin || bounds.Max.Y > 32-margin {
		t.Fatalf("long object escaped animation safety area: bounds=%v margin=%d", bounds, margin)
	}
	if bounds.Dx() < 19 {
		t.Fatalf("long object did not fill the inner canvas width: bounds=%v", bounds)
	}
	if bounds.Dy() > 3 {
		t.Fatalf("long object aspect ratio was distorted: bounds=%v", bounds)
	}
}

func TestAnimationPixelResizePreservesFixedFramePositionAndHardensEdges(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	// A deliberately off-centre subject models intentional animation motion.
	// The translucent outer row models the smooth fringe produced by animation
	// normalization before the final pixel-art conversion.
	for y := 8; y < 25; y++ {
		for x := 3; x < 13; x++ {
			alpha := uint8(255)
			if x == 3 || x == 12 || y == 8 || y == 24 {
				alpha = 96
			}
			source.SetNRGBA(x, y, color.NRGBA{
				R: uint8(160 + (x % 4 * 12)),
				G: uint8(48 + (y % 3 * 9)),
				B: 32,
				A: alpha,
			})
		}
	}

	options := AnimationPixelResizeOptions(32, 32)
	got, report, err := ResizeImage(source, options)
	if err != nil {
		t.Fatalf("pixel-process normalized animation frame: %v", err)
	}
	if report.CroppedToContent || report.Margin != 0 || report.Sampling != resizeSamplingNearest {
		t.Fatalf("animation pixel report changed fixed-canvas geometry: %+v", report)
	}

	bounds, visible := alphaBounds(toNRGBA(got), 0)
	if !visible {
		t.Fatal("animation subject disappeared during pixel conversion")
	}
	wantBounds := image.Rect(4, 9, 12, 24)
	if bounds != wantBounds {
		t.Fatalf("animation subject moved or was refit: bounds=%v, want %v", bounds, wantBounds)
	}
	for y := range 32 {
		for x := range 32 {
			alpha := got.RGBAAt(x, y).A
			if alpha != 0 && alpha != 255 {
				t.Fatalf("animation frame retained partial alpha at (%d,%d): %d", x, y, alpha)
			}
		}
	}
}
