package image

import "image"

func integerNearestNeighborScale(source image.Image, scale int) *image.NRGBA {
	input := toNRGBA(source)
	width, height := input.Bounds().Dx(), input.Bounds().Dy()
	if scale <= 1 {
		return input
	}
	output := image.NewNRGBA(image.Rect(0, 0, width*scale, height*scale))
	for y := range output.Bounds().Dy() {
		for x := range output.Bounds().Dx() {
			output.SetNRGBA(x, y, input.NRGBAAt(x/scale, y/scale))
		}
	}
	return output
}
