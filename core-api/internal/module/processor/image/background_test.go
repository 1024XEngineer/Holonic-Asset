package image

import (
	"image"
	"image/color"
	"testing"
)

func TestExtractChromaRemovesEnclosedMatteRegion(t *testing.T) {
	t.Parallel()

	matte := MatteColor{0, 255, 0}
	source := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	fillRect(source, source.Bounds(), color.NRGBA{G: 255, A: 255})
	fillRect(source, image.Rect(12, 8, 52, 56), color.NRGBA{R: 220, G: 55, B: 40, A: 255})
	// The subject encloses a patch of the original green screen, as happens
	// between an arm/tool and the torso in generated motion frames.
	fillRect(source, image.Rect(27, 24, 37, 36), color.NRGBA{G: 255, A: 255})

	result := ExtractChroma(source, matte, ChromaSettings{})

	if alpha := result.RGBAAt(32, 30).A; alpha != 0 {
		t.Fatalf("enclosed matte alpha = %d, want 0", alpha)
	}
	if alpha := result.RGBAAt(20, 20).A; alpha < 250 {
		t.Fatalf("foreground alpha = %d, want opaque subject preserved", alpha)
	}
}

func TestExtractChromaRemovesDirtyEnclosedMatteConnectedToStrongSeed(t *testing.T) {
	t.Parallel()

	matte := MatteColor{0, 255, 0}
	source := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	fillRect(source, source.Bounds(), color.NRGBA{G: 255, A: 255})
	fillRect(source, image.Rect(10, 8, 54, 56), color.NRGBA{R: 220, G: 55, B: 40, A: 255})
	fillRect(source, image.Rect(26, 22, 38, 38), color.NRGBA{R: 9, G: 245, B: 7, A: 255})
	fillRect(source, image.Rect(28, 24, 36, 36), color.NRGBA{R: 24, G: 221, B: 20, A: 255})
	fillRect(source, image.Rect(30, 27, 34, 33), color.NRGBA{R: 28, G: 127, B: 19, A: 255})
	// Keep a high-confidence matte seed in the same enclosed component.
	fillRect(source, image.Rect(26, 22, 29, 25), color.NRGBA{G: 255, A: 255})

	result := ExtractChroma(source, matte, ChromaSettings{})

	if alpha := result.RGBAAt(27, 23).A; alpha != 0 {
		t.Fatalf("strong enclosed matte seed alpha = %d, want 0", alpha)
	}
	if alpha := result.RGBAAt(32, 30).A; alpha >= 245 {
		t.Fatalf("dirty enclosed matte alpha = %d, want background attenuation", alpha)
	}
}

func TestExtractChromaPreservesEnclosedGreenForegroundWithoutMatteSeed(t *testing.T) {
	t.Parallel()

	matte := MatteColor{0, 255, 0}
	source := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	fillRect(source, source.Bounds(), color.NRGBA{G: 255, A: 255})
	fillRect(source, image.Rect(10, 8, 54, 56), color.NRGBA{R: 220, G: 55, B: 40, A: 255})
	fillRect(source, image.Rect(26, 22, 38, 38), color.NRGBA{R: 20, G: 150, B: 35, A: 255})

	result := ExtractChroma(source, matte, ChromaSettings{})

	pixel := result.RGBAAt(32, 30)
	if pixel.A != 255 {
		t.Fatalf("green foreground alpha = %d, want 255", pixel.A)
	}
	if pixel.R != 20 || pixel.G != 150 || pixel.B != 35 {
		t.Fatalf("green foreground RGB = (%d,%d,%d), want source colour preserved", pixel.R, pixel.G, pixel.B)
	}
}

func TestExtractChromaStillRemovesBorderConnectedMatte(t *testing.T) {
	t.Parallel()

	matte := MatteColor{0, 255, 0}
	source := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(source, source.Bounds(), color.NRGBA{G: 255, A: 255})
	fillRect(source, image.Rect(10, 8, 22, 28), color.NRGBA{R: 220, G: 55, B: 40, A: 255})

	result := ExtractChroma(source, matte, ChromaSettings{})

	if alpha := result.RGBAAt(0, 0).A; alpha != 0 {
		t.Fatalf("border matte alpha = %d, want 0", alpha)
	}
	if alpha := result.RGBAAt(16, 16).A; alpha < 250 {
		t.Fatalf("subject alpha = %d, want opaque subject preserved", alpha)
	}
}
