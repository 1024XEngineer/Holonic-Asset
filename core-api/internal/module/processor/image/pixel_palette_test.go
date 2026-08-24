package image

import (
	"image"
	"image/color"
	"testing"
)

func TestSplitPaletteBoxUsesLargestChannelAndWeightedMedian(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		points []palettePoint
		left   palettePoint
		right  palettePoint
	}{
		{
			name: "red",
			points: []palettePoint{
				{r: 220, g: 20, b: 20, weight: 1},
				{r: 10, g: 30, b: 30, weight: 3},
			},
			left:  palettePoint{r: 10, g: 30, b: 30, weight: 3},
			right: palettePoint{r: 220, g: 20, b: 20, weight: 1},
		},
		{
			name: "green",
			points: []palettePoint{
				{r: 30, g: 230, b: 30, weight: 1},
				{r: 20, g: 10, b: 20, weight: 3},
			},
			left:  palettePoint{r: 20, g: 10, b: 20, weight: 3},
			right: palettePoint{r: 30, g: 230, b: 30, weight: 1},
		},
		{
			name: "blue",
			points: []palettePoint{
				{r: 30, g: 30, b: 240, weight: 1},
				{r: 20, g: 20, b: 5, weight: 3},
			},
			left:  palettePoint{r: 20, g: 20, b: 5, weight: 3},
			right: palettePoint{r: 30, g: 30, b: 240, weight: 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			box := newPaletteBox(append([]palettePoint(nil), test.points...))
			if box.score() <= 0 {
				t.Fatalf("box score = %f, want positive", box.score())
			}
			left, right, ok := splitPaletteBox(box)
			if !ok {
				t.Fatal("expected palette box to split")
			}
			if len(left.points) != 1 || len(right.points) != 1 ||
				left.points[0] != test.left || right.points[0] != test.right {
				t.Fatalf("unexpected split: left=%+v right=%+v", left.points, right.points)
			}
			if left.totalWeight+right.totalWeight != box.totalWeight {
				t.Fatalf("split weight = %d, want %d", left.totalWeight+right.totalWeight, box.totalWeight)
			}
		})
	}

	if _, _, ok := splitPaletteBox(newPaletteBox([]palettePoint{{r: 1, weight: 1}})); ok {
		t.Fatal("single-point palette box unexpectedly split")
	}
}

func TestBuildWeightedPaletteHandlesEmptyInputsAndUsesSourceSamples(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	if got := buildWeightedPalette(img, img.Bounds(), 0, 0); got != nil {
		t.Fatalf("zero palette limit returned %+v", got)
	}
	if got := buildWeightedPalette(img, image.Rect(8, 8, 10, 10), 2, 0); got != nil {
		t.Fatalf("empty intersection returned %+v", got)
	}

	// The first two colours share one five-bit histogram bucket. Equal alpha
	// must select the lexicographically smaller real source sample.
	darker := color.NRGBA{R: 8, G: 16, B: 24, A: 200}
	lighter := color.NRGBA{R: 15, G: 23, B: 31, A: 200}
	accent := color.NRGBA{R: 224, G: 40, B: 32, A: 255}
	img.SetNRGBA(0, 0, lighter)
	img.SetNRGBA(1, 0, darker)
	img.SetNRGBA(2, 0, accent)
	img.SetNRGBA(3, 0, color.NRGBA{R: 255, A: 5})

	palette := buildWeightedPalette(img, img.Bounds(), 8, 5)
	if len(palette) != 2 {
		t.Fatalf("palette length = %d, want 2: %+v", len(palette), palette)
	}
	seen := map[color.RGBA]bool{}
	for _, entry := range palette {
		seen[entry.colour] = true
	}
	if !seen[color.RGBA{R: darker.R, G: darker.G, B: darker.B, A: 255}] ||
		!seen[color.RGBA{R: accent.R, G: accent.G, B: accent.B, A: 255}] {
		t.Fatalf("palette did not retain exact source samples: %+v", palette)
	}
}

func TestRepresentativePaletteColourUsesWeightThenStableColourOrder(t *testing.T) {
	t.Parallel()

	box := newPaletteBox([]palettePoint{
		{r: 240, g: 20, b: 10, weight: 2},
		{r: 20, g: 40, b: 60, weight: 5},
	})
	if got := representativePaletteColour(box); got != (color.RGBA{R: 20, G: 40, B: 60, A: 255}) {
		t.Fatalf("weighted representative = %+v", got)
	}

	tied := newPaletteBox([]palettePoint{
		{r: 80, g: 70, b: 60, weight: 4},
		{r: 10, g: 20, b: 30, weight: 4},
	})
	if got := representativePaletteColour(tied); got != (color.RGBA{R: 10, G: 20, B: 30, A: 255}) {
		t.Fatalf("stable tied representative = %+v", got)
	}
	if got := palettePointKey(palettePoint{r: 10, g: 20, b: 30}); got != 0x0a141e {
		t.Fatalf("palette point key = %#x", got)
	}
}

func TestApplyPaletteReducesColoursWithoutChangingTransparency(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 5, 1))
	for x, pixel := range []color.NRGBA{
		{R: 20, G: 30, B: 40, A: 255},
		{R: 45, G: 55, B: 65, A: 255},
		{R: 180, G: 70, B: 40, A: 255},
		{R: 220, G: 110, B: 60, A: 128},
		{R: 255, G: 255, B: 255, A: TransparentAlphaMax},
	} {
		img.SetNRGBA(x, 0, pixel)
	}

	unchanged := cloneNRGBA(img)
	remapToPalette(img, img.Bounds(), nil)
	for x := range 5 {
		if img.NRGBAAt(x, 0) != unchanged.NRGBAAt(x, 0) {
			t.Fatalf("empty palette changed pixel %d", x)
		}
	}

	applyPalette(img, 2)
	colours := map[color.NRGBA]struct{}{}
	for x := range 4 {
		pixel := img.NRGBAAt(x, 0)
		colours[color.NRGBA{R: pixel.R, G: pixel.G, B: pixel.B, A: 255}] = struct{}{}
	}
	if len(colours) > 2 {
		t.Fatalf("palette application left %d visible colours", len(colours))
	}
	if img.NRGBAAt(3, 0).A != 128 {
		t.Fatalf("palette application changed source alpha: %+v", img.NRGBAAt(3, 0))
	}
	if got := img.NRGBAAt(4, 0); got != unchanged.NRGBAAt(4, 0) {
		t.Fatalf("transparent pixel changed: %+v", got)
	}
}
