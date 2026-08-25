package image

import (
	"image"
	"image/color"
	"testing"
)

func TestRecoverPixelGridResizeSelectsRecoveredAndFallbackSamplers(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 6, 4))
	redGradient := [...]uint8{0, 30, 60, 90, 120, 150}
	greenGradient := [...]uint8{0, 50, 100, 150}
	for y := range 4 {
		for x := range 6 {
			source.SetNRGBA(x, y, color.NRGBA{R: redGradient[x], G: greenGradient[y], A: 255})
		}
	}

	nearest, sampling := recoverPixelGridResize(source, source.Bounds(), 2, 2, true, true)
	if sampling != pixelGridSamplingNearestFallback || nearest.Bounds() != image.Rect(0, 0, 2, 2) {
		t.Fatalf("nearest fallback: sampling=%q bounds=%v", sampling, nearest.Bounds())
	}
	area, sampling := recoverPixelGridResize(source, source.Bounds(), 2, 2, false, false)
	if sampling != pixelGridSamplingFallback || area.Bounds() != image.Rect(0, 0, 2, 2) {
		t.Fatalf("area fallback: sampling=%q bounds=%v", sampling, area.Bounds())
	}

	integral := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	red := color.NRGBA{R: 220, G: 30, B: 20, A: 255}
	blue := color.NRGBA{R: 20, G: 40, B: 220, A: 255}
	for y := range 4 {
		for x := range 4 {
			pixel := red
			if x >= 2 {
				pixel = blue
			}
			integral.SetNRGBA(x, y, pixel)
		}
	}
	for _, spritePipeline := range []bool{false, true} {
		got, gotSampling := recoverPixelGridResize(integral, integral.Bounds(), 2, 2, false, spritePipeline)
		if gotSampling != pixelGridSamplingRecovered {
			t.Fatalf("spritePipeline=%v sampling=%q", spritePipeline, gotSampling)
		}
		if got.NRGBAAt(0, 0) != red || got.NRGBAAt(1, 0) != blue {
			t.Fatalf("spritePipeline=%v recovered pixels=%+v %+v", spritePipeline, got.NRGBAAt(0, 0), got.NRGBAAt(1, 0))
		}
	}
}

func TestIntegralGridScaleRejectsInvalidGeometry(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		source, target int
		wantScale      int
		wantOK         bool
	}{
		{source: 0, target: 2},
		{source: 4, target: 0},
		{source: 5, target: 2},
		{source: 8, target: 4, wantScale: 2, wantOK: true},
	} {
		gotScale, gotOK := integralGridScale(test.source, test.target)
		if gotScale != test.wantScale || gotOK != test.wantOK {
			t.Fatalf("integralGridScale(%d,%d)=(%d,%v), want (%d,%v)", test.source, test.target, gotScale, gotOK, test.wantScale, test.wantOK)
		}
	}
}

func TestRecoverGridPhaseFindsShiftedHardEdgesAndHandlesFlatInput(t *testing.T) {
	t.Parallel()

	horizontal := image.NewNRGBA(image.Rect(0, 0, 9, 3))
	vertical := image.NewNRGBA(image.Rect(0, 0, 3, 9))
	colours := []color.NRGBA{{R: 230, A: 255}, {G: 230, A: 255}, {B: 230, A: 255}}
	for y := range 3 {
		for x := range 9 {
			bucket := 0
			if x >= 4 {
				bucket = 1
			}
			if x >= 7 {
				bucket = 2
			}
			horizontal.SetNRGBA(x, y, colours[bucket])
			vertical.SetNRGBA(y, x, colours[bucket])
		}
	}
	if got := recoverGridPhase(horizontal, horizontal.Bounds(), 3, true); got != 1 {
		t.Fatalf("horizontal phase = %d, want 1", got)
	}
	if got := recoverGridPhase(vertical, vertical.Bounds(), 3, false); got != 1 {
		t.Fatalf("vertical phase = %d, want 1", got)
	}
	if got := recoverGridPhase(horizontal, horizontal.Bounds(), 1, true); got != 0 {
		t.Fatalf("scale-one phase = %d", got)
	}

	flat := image.NewNRGBA(image.Rect(0, 0, 6, 6))
	for y := range 6 {
		for x := range 6 {
			flat.SetNRGBA(x, y, color.NRGBA{R: 80, G: 90, B: 100, A: 255})
		}
	}
	if got := recoverGridPhase(flat, flat.Bounds(), 3, true); got != 0 {
		t.Fatalf("flat input phase = %d", got)
	}
	if verticalEdgeEnergy(flat, flat.Bounds(), -1) != 0 || verticalEdgeEnergy(flat, flat.Bounds(), 5) != 0 {
		t.Fatal("vertical edge energy accepted an out-of-bounds edge")
	}
	if horizontalEdgeEnergy(flat, flat.Bounds(), -1) != 0 || horizontalEdgeEnergy(flat, flat.Bounds(), 5) != 0 {
		t.Fatal("horizontal edge energy accepted an out-of-bounds edge")
	}
}

func TestRecoveredGridHelpersRespectCropAndChooseStrongestPixel(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 6, 6))
	weak := color.NRGBA{R: 40, G: 50, B: 60, A: 140}
	strong := color.NRGBA{R: 210, G: 100, B: 30, A: 250}
	source.SetNRGBA(2, 2, weak)
	source.SetNRGBA(3, 2, strong)

	if got := strongestNearbyPixel(source, image.Rect(1, 1, 5, 5), 2, 2, 4); got != strong {
		t.Fatalf("strongest nearby pixel = %+v, want %+v", got, strong)
	}

	nearest := sampleRecoveredGridNearest(source, image.Rect(1, 1, 5, 5), 2, 2, 2, 0, 0)
	if nearest.Bounds() != image.Rect(0, 0, 2, 2) {
		t.Fatalf("nearest recovered bounds = %v", nearest.Bounds())
	}

	area := gridAwareAreaResize(source, image.Rect(1, 1, 5, 5), 2, 2)
	if area.Bounds() != image.Rect(0, 0, 2, 2) {
		t.Fatalf("grid-aware area bounds = %v", area.Bounds())
	}
}
