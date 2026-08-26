package generator

import imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"

// PrototypePixelResizeOptions returns the pixel-art profile for generated
// object prototypes.
func PrototypePixelResizeOptions(width, height int) imageprocessor.ResizeOptions {
	options := prototypePixelResizeOptions(width, height)
	options.PaletteSize = prototypePixelPaletteSize(width, height)
	return options
}

// CharacterPrototypePixelResizeOptions returns the pixel-art profile for
// generated character prototypes. Character and object prototypes share the
// same deterministic pixel pipeline and palette budget.
func CharacterPrototypePixelResizeOptions(width, height int) imageprocessor.ResizeOptions {
	options := prototypePixelResizeOptions(width, height)
	options.PaletteSize = prototypePixelPaletteSize(width, height)
	return options
}

// AnimationPixelResizeOptions returns the pixel-art profile for an animation
// frame that has already been normalized onto its final canvas. The complete
// canvas geometry is preserved instead of cropping and refitting the frame.
func AnimationPixelResizeOptions(width, height int) imageprocessor.ResizeOptions {
	options := prototypePixelResizeOptions(width, height)
	options.PaletteSize = prototypePixelPaletteSize(width, height)
	options.Margin = 0
	options.CropContent = false
	options.PreserveCanvasGeometry = true
	return options
}

func prototypePixelPaletteSize(width, height int) int {
	targetShortEdge := min(width, height)
	switch {
	case targetShortEdge <= 16:
		return 8
	case targetShortEdge <= 64:
		return 16
	default:
		return 24
	}
}

func prototypePixelResizeOptions(width, height int) imageprocessor.ResizeOptions {
	options := imageprocessor.DefaultResizeOptions(width, height)
	options.Margin = 0
	options.CropContent = false
	options.PreserveCanvasGeometry = true
	options.Mode = imageprocessor.RasterModePixel
	options.HardAlpha = true
	options.RecoverPixelGrid = true
	options.PrequantizeBeforeResize = true
	options.PreferNearestReduction = true
	options.SpritePixelPipeline = true
	return options
}
