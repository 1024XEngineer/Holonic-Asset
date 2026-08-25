package generator

import (
	"fmt"
	"testing"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
)

func TestAnimationFrameMarginUsesThreeSixteenthsOfShortEdge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		width, height      int
		wantMargin         int
		wantDrawableWidth  int
		wantDrawableHeight int
	}{
		{width: 32, height: 32, wantMargin: 6, wantDrawableWidth: 20, wantDrawableHeight: 20},
		{width: 48, height: 64, wantMargin: 9, wantDrawableWidth: 30, wantDrawableHeight: 46},
		{width: 64, height: 64, wantMargin: 12, wantDrawableWidth: 40, wantDrawableHeight: 40},
		{width: 128, height: 128, wantMargin: 24, wantDrawableWidth: 80, wantDrawableHeight: 80},
		{width: 256, height: 256, wantMargin: 48, wantDrawableWidth: 160, wantDrawableHeight: 160},
	}
	for _, test := range tests {
		margin := AnimationFrameMargin(test.width, test.height)
		if margin != test.wantMargin {
			t.Fatalf("%dx%d margin = %d, want %d", test.width, test.height, margin, test.wantMargin)
		}
		if got := test.width - 2*margin; got != test.wantDrawableWidth {
			t.Fatalf("%dx%d drawable width = %d, want %d", test.width, test.height, got, test.wantDrawableWidth)
		}
		if got := test.height - 2*margin; got != test.wantDrawableHeight {
			t.Fatalf("%dx%d drawable height = %d, want %d", test.width, test.height, got, test.wantDrawableHeight)
		}
	}
}

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
			}
			if object.Margin != AnimationFrameMargin(test.dimension, test.dimension) || character.Margin != object.Margin {
				t.Fatalf("prototype margins diverged: object=%d character=%d", object.Margin, character.Margin)
			}
			if animation.Margin != 0 || animation.CropContent || !animation.PreserveCanvasGeometry {
				t.Fatalf("animation profile changed fixed-canvas geometry: %+v", animation)
			}
		})
	}
}

func testNameForDimension(dimension int) string {
	return fmt.Sprintf("dimension-%d", dimension)
}
