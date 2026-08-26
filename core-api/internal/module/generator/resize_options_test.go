package generator

import (
	"fmt"
	"testing"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
)

func TestPixelResizeProfilesAreOwnedByGenerator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		dimension int
		palette   int
	}{
		{dimension: 16, palette: 8},
		{dimension: 32, palette: 16},
		{dimension: 64, palette: 16},
		{dimension: 128, palette: 24},
		{dimension: 256, palette: 24},
	}
	for _, test := range tests {
		t.Run(testNameForDimension(test.dimension), func(t *testing.T) {
			object := PrototypePixelResizeOptions(test.dimension, test.dimension)
			character := CharacterPrototypePixelResizeOptions(test.dimension, test.dimension)
			animation := AnimationPixelResizeOptions(test.dimension, test.dimension)
			for name, options := range map[string]imageprocessor.ResizeOptions{
				"object prototype":    object,
				"character prototype": character,
				"animation":           animation,
			} {
				if options.Width != test.dimension || options.Height != test.dimension || options.Mode != imageprocessor.RasterModePixel ||
					!options.HardAlpha || !options.RecoverPixelGrid || !options.PrequantizeBeforeResize ||
					!options.PreferNearestReduction || !options.SpritePixelPipeline || options.PaletteSize != test.palette {
					t.Fatalf("unexpected %s options: %+v", name, options)
				}
				if options.Margin != 0 || options.CropContent || !options.PreserveCanvasGeometry {
					t.Fatalf("%s profile changed fixed full-canvas geometry: %+v", name, options)
				}
			}
		})
	}
}

func testNameForDimension(dimension int) string {
	return fmt.Sprintf("dimension-%d", dimension)
}
