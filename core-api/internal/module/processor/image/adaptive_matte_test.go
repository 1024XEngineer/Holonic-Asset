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

	selected := SelectAnimationMatteColor(subject)
	if selected == (MatteColor{0, 255, 0}) {
		t.Fatalf("selected green matte for a green subject: %#v", selected)
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
