package generator

import (
	"fmt"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func defaultAnimationFrameDimensions(prototype assetdomain.Size) (int, int) {
	return defaultAnimationFrameDimension(int(prototype.Width)), defaultAnimationFrameDimension(int(prototype.Height))
}

func defaultAnimationFrameDimension(value int) int {
	// ceil(value * 1.5), using integer arithmetic shared with the frontend.
	return (value*3 + 1) / 2
}

func resolveAnimationFrameDimensions(
	prototype assetdomain.Size,
	width, height int,
) (int, int, error) {
	if (width == 0) != (height == 0) {
		return 0, 0, fmt.Errorf("generator: animation frame width and height must both be provided or both be omitted")
	}
	if width == 0 {
		width, height = defaultAnimationFrameDimensions(prototype)
	}
	if width < int(prototype.Width) || height < int(prototype.Height) {
		return 0, 0, fmt.Errorf(
			"generator: animation frame dimensions %dx%d must not be smaller than prototype dimensions %dx%d",
			width, height, prototype.Width, prototype.Height,
		)
	}
	if width < 32 || width > 1024 || height < 32 || height > 1024 {
		return 0, 0, fmt.Errorf("generator: animation frame dimensions must be between 32 and 1024 pixels")
	}
	return width, height, nil
}
