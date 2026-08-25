package image

import (
	"image"
	"image/color"
	"testing"
)

func TestHarmonizePrototypeDirectionColoursUsesOnePaletteAcrossDirections(t *testing.T) {
	first := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	second := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 1; y < 3; y++ {
		for x := 1; x < 3; x++ {
			first.SetNRGBA(x, y, color.NRGBA{R: 210, G: 78, B: 42, A: 255})
			second.SetNRGBA(x, y, color.NRGBA{R: 218, G: 84, B: 46, A: 255})
		}
	}

	outputs, err := HarmonizePrototypeDirectionColours(
		[]image.Image{first, second},
		PrototypeDirectionPaletteOptions{PaletteSize: 1},
	)
	if err != nil {
		t.Fatalf("harmonize direction colours: %v", err)
	}
	if len(outputs) != 2 {
		t.Fatalf("output frame count = %d, want 2", len(outputs))
	}
	if got, want := outputs[0].RGBAAt(1, 1), outputs[1].RGBAAt(1, 1); got != want {
		t.Fatalf("shared material colour differs across directions: first=%v second=%v", got, want)
	}
	if got := outputs[0].RGBAAt(0, 0).A; got != 0 {
		t.Fatalf("transparent background changed alpha: %d", got)
	}
}

func TestHarmonizePrototypeDirectionColoursDoesNotFlattenDistinctDetails(t *testing.T) {
	first := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	second := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	for y := range 2 {
		for x := range 2 {
			first.SetNRGBA(x, y, color.NRGBA{R: 180, G: 40, B: 35, A: 255})
			second.SetNRGBA(x, y, color.NRGBA{R: 42, G: 74, B: 180, A: 255})
		}
		for x := 2; x < 4; x++ {
			first.SetNRGBA(x, y, color.NRGBA{R: 38, G: 42, B: 48, A: 255})
			second.SetNRGBA(x, y, color.NRGBA{R: 190, G: 46, B: 38, A: 255})
		}
	}

	outputs, err := HarmonizePrototypeDirectionColours(
		[]image.Image{first, second},
		PrototypeDirectionPaletteOptions{PaletteSize: 10},
	)
	if err != nil {
		t.Fatalf("harmonize distinct details: %v", err)
	}
	if got := outputs[0].RGBAAt(0, 0); got.R < 150 || got.B > 80 {
		t.Fatalf("red detail was flattened: %v", got)
	}
	if got := outputs[1].RGBAAt(0, 0); got.B < 130 || got.R > 90 {
		t.Fatalf("blue detail was flattened: %v", got)
	}
}

func TestHarmonizePrototypeDirectionColoursKeepsStrictSparseObjectBudget(t *testing.T) {
	frames := make([]image.Image, 2)
	for index := range frames {
		frame := image.NewNRGBA(image.Rect(0, 0, 32, 32))
		colours := []color.NRGBA{
			{R: 100, G: 80, B: 40, A: 255},
			{R: 140, G: 80, B: 40, A: 255},
		}
		for x := 4; x < 8; x++ {
			frame.SetNRGBA(x, 16+index, colours[index])
		}
		frames[index] = frame
	}

	outputs, err := HarmonizePrototypeDirectionColours(
		frames,
		PrototypeDirectionPaletteOptions{PaletteSize: 10},
	)
	if err != nil {
		t.Fatalf("harmonize sparse directions: %v", err)
	}
	for index, output := range outputs {
		if got := output.RGBAAt(4, 16+index); got.R != 100+40*uint8(index) || got.G != 80 || got.B != 40 {
			t.Fatalf("direction %d changed a distinct source colour: %v", index, got)
		}
	}
}
