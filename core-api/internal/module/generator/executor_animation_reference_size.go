package generator

import (
	"image"
	"math"
)

func animationReferenceCanvasSize() image.Point {
	return image.Pt(animationReferenceSize, animationReferenceSize)
}

func animationReferenceCanvasSizeForLongEdge(longEdge int) image.Point {
	if longEdge <= 0 {
		longEdge = animationReferenceSize
	}
	return image.Pt(longEdge, longEdge)
}

// animationReferencePrototypeCanvasSize maps the prototype canvas onto the
// square provider reference with the inverse of the contain transform used when
// video frames are normalized to the requested output size. This keeps one
// uniform scale and makes the subject project back to its prototype pixel size.
func animationReferencePrototypeCanvasSize(
	canvas image.Point,
	prototypeWidth, prototypeHeight int,
	frameWidth, frameHeight int,
) image.Point {
	if canvas.X <= 0 || canvas.Y <= 0 {
		return image.Pt(1, 1)
	}
	if prototypeWidth <= 0 || prototypeHeight <= 0 || frameWidth <= 0 || frameHeight <= 0 {
		return canvas
	}

	outputScale := math.Min(
		float64(frameWidth)/float64(canvas.X),
		float64(frameHeight)/float64(canvas.Y),
	)
	if outputScale <= 0 {
		return canvas
	}
	providerScale := 1 / outputScale
	providerScale = math.Min(providerScale, math.Min(
		float64(canvas.X)/float64(prototypeWidth),
		float64(canvas.Y)/float64(prototypeHeight),
	))
	width := int(math.Round(float64(prototypeWidth) * providerScale))
	height := int(math.Round(float64(prototypeHeight) * providerScale))
	return image.Pt(
		max(1, min(canvas.X, width)),
		max(1, min(canvas.Y, height)),
	)
}
