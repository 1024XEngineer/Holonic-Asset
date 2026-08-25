package image

// These fixtures keep processor tests focused on the generic resize behavior.
// The production profiles live in the generator package, where their prototype
// and animation workflow semantics belong.
func animationFrameMarginForTest(width, height int) int {
	margin := min(width, height) * 3 / 16
	if margin < 1 {
		return 1
	}
	return margin
}

func prototypePixelResizeOptionsForTest(width, height int) ResizeOptions {
	options := DefaultResizeOptions(width, height)
	options.Margin = animationFrameMarginForTest(width, height)
	options.Mode = RasterModePixel
	options.HardAlpha = true
	options.RecoverPixelGrid = true
	options.PrequantizeBeforeResize = true
	options.PreferNearestReduction = true
	options.SpritePixelPipeline = true
	options.PaletteSize = prototypePixelPaletteSizeForTest(width, height)
	return options
}

func characterPixelResizeOptionsForTest(width, height int) ResizeOptions {
	return prototypePixelResizeOptionsForTest(width, height)
}

func animationPixelResizeOptionsForTest(width, height int) ResizeOptions {
	options := prototypePixelResizeOptionsForTest(width, height)
	options.Margin = 0
	options.CropContent = false
	options.PreserveCanvasGeometry = true
	return options
}

func prototypePixelPaletteSizeForTest(width, height int) int {
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
