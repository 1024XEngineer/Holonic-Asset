package image

import (
	"image"
	"image/color"
	"testing"
)

func TestSelectAnimationMatteColorAvoidsSubstantialGreenSubject(t *testing.T) {
	subject := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 4; y < 28; y++ {
		for x := 4; x < 28; x++ {
			subject.SetNRGBA(x, y, color.NRGBA{G: 255, A: 255})
		}
	}

	selected, safe := SelectAnimationMatteColor(subject)
	if selected == (MatteColor{0, 255, 0}) {
		t.Fatalf("selected green matte for a green subject: %#v", selected)
	}
	if !safe {
		t.Fatalf("expected green subject to produce a safe matte with a non-green candidate")
	}
}

func TestCompositeAnimationMatteUsesExactSelectedColour(t *testing.T) {
	foreground := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	foreground.SetNRGBA(3, 3, color.NRGBA{R: 12, G: 34, B: 56, A: 255})
	matte := MatteColor{255, 0, 255}

	result := CompositeAnimationMatte(foreground, matte, image.Pt(16, 16))
	if got := color.NRGBAModel.Convert(result.At(0, 0)).(color.NRGBA); got != (color.NRGBA{R: 255, B: 255, A: 255}) {
		t.Fatalf("corner = %#v, want exact selected matte", got)
	}
	if got := color.NRGBAModel.Convert(result.At(7, 7)).(color.NRGBA); got != (color.NRGBA{R: 12, G: 34, B: 56, A: 255}) {
		t.Fatalf("subject pixel = %#v, want preserved foreground", got)
	}
}

func TestPrepareAnimationForegroundPreservesExactOldMatteInsideSubject(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 48, 48))
	for y := range 48 {
		for x := range 48 {
			source.SetNRGBA(x, y, color.NRGBA{G: 255, A: 255})
		}
	}
	for y := 10; y < 38; y++ {
		for x := 12; x < 36; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 20, G: 30, B: 25, A: 255})
		}
	}
	// This exact legacy-key green is enclosed by the subject outline, so it is
	// an intentional subject detail rather than external background.
	for y := 16; y < 32; y++ {
		for x := 18; x < 30; x++ {
			source.SetNRGBA(x, y, color.NRGBA{G: 255, A: 255})
		}
	}

	foreground := PrepareAnimationForeground(source).(*image.RGBA)
	if got := foreground.RGBAAt(1, 1).A; got != 0 {
		t.Fatalf("outside historical matte alpha = %d, want 0", got)
	}
	if got := foreground.RGBAAt(24, 24); got.A != 255 || got.G != 255 || got.R != 0 || got.B != 0 {
		t.Fatalf("enclosed green subject detail = %#v, want opaque preserved detail", got)
	}
}

func TestSelectAnimationMatteColorReturnsUnsafeWhenSubjectContainsAllCandidates(t *testing.T) {
	// Create a subject that has pixels of every candidate colour, so no
	// candidate is far enough from all subject pixels.
	subject := image.NewNRGBA(image.Rect(0, 0, 120, 120))
	for index, candidate := range AnimationMatteCandidates {
		x0 := index * 20
		for y := range 120 {
			for x := x0; x < x0+20; x++ {
				subject.SetNRGBA(x, y, color.NRGBA{R: candidate[0], G: candidate[1], B: candidate[2], A: 255})
			}
		}
	}

	_, safe := SelectAnimationMatteColor(subject)
	if safe {
		t.Fatal("expected unsafe result when subject contains all candidate colours")
	}
}

func TestSelectAnimationMatteColorReturnsSafeForMonoGreenSubject(t *testing.T) {
	subject := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 4; y < 28; y++ {
		for x := 4; x < 28; x++ {
			subject.SetNRGBA(x, y, color.NRGBA{G: 255, A: 255})
		}
	}

	_, safe := SelectAnimationMatteColor(subject)
	if !safe {
		t.Fatal("expected safe result for mono-green subject")
	}
}

func TestMatteSafetyDistanceFloorRejectsNearMatchCandidate(t *testing.T) {
	// Six blocks, each offset from a different candidate by (±10,±10,±10).
	// Every candidate is within MatteSafetyDistanceFloor of its own block,
	// so no matter which candidate wins, it must be reported as unsafe.
	subject := image.NewNRGBA(image.Rect(0, 0, 120, 120))
	for index, candidate := range AnimationMatteCandidates {
		r := uint8(clamp255(int(candidate[0])-10+index*3) & 0xff)
		g := uint8(clamp255(int(candidate[1])-10+index*3) & 0xff)
		b := uint8(clamp255(int(candidate[2])-10+index*3) & 0xff)
		x0 := (index % 3) * 40
		y0 := (index / 3) * 60
		for y := y0; y < y0+60; y++ {
			for x := x0; x < x0+40; x++ {
				subject.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
			}
		}
	}

	selected, safe := SelectAnimationMatteColor(subject)
	if safe {
		t.Fatalf("expected unsafe when every candidate is within floor of its own block; selected %v", selected)
	}
}

func clamp255(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func TestSelectAnimationMatteColorReturnsDefaultForNilSubject(t *testing.T) {
	selected, safe := SelectAnimationMatteColor(nil)
	if selected != AnimationMatteCandidates[0] {
		t.Fatalf("expected default candidate, got %v", selected)
	}
	if !safe {
		t.Fatal("expected safe for nil subject")
	}
}

func TestSelectAnimationMatteColorReturnsDefaultForEmptySubject(t *testing.T) {
	selected, safe := SelectAnimationMatteColor(image.NewNRGBA(image.Rect(0, 0, 0, 0)))
	if selected != AnimationMatteCandidates[0] {
		t.Fatalf("expected default candidate, got %v", selected)
	}
	if !safe {
		t.Fatal("expected safe for empty subject")
	}
}

func TestSelectAnimationMatteColorIgnoresFullyTransparentPixels(t *testing.T) {
	subject := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			subject.SetNRGBA(x, y, color.NRGBA{G: 255, A: 0})
		}
	}
	_, safe := SelectAnimationMatteColor(subject)
	if !safe {
		t.Fatal("expected safe when all pixels are transparent")
	}
}

func TestPrepareAnimationForegroundReturnsNilForNilSource(t *testing.T) {
	if got := PrepareAnimationForeground(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestCompositeAnimationMatteClampsTinyCanvas(t *testing.T) {
	foreground := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	foreground.SetNRGBA(4, 4, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	result := CompositeAnimationMatte(foreground, MatteColor{255, 0, 255}, image.Pt(0, 0))
	if result.Bounds().Dx() < 1 || result.Bounds().Dy() < 1 {
		t.Fatal("expected at least 1x1 canvas")
	}
}

func TestCompositeAnimationMatteHandlesNilForeground(t *testing.T) {
	result := CompositeAnimationMatte(nil, MatteColor{0, 255, 0}, image.Pt(16, 16))
	if result.Bounds().Dx() != 16 || result.Bounds().Dy() != 16 {
		t.Fatalf("expected 16x16 canvas, got %dx%d", result.Bounds().Dx(), result.Bounds().Dy())
	}
	got := color.NRGBAModel.Convert(result.At(0, 0)).(color.NRGBA)
	if got != (color.NRGBA{G: 255, A: 255}) {
		t.Fatalf("expected green background, got %v", got)
	}
}
