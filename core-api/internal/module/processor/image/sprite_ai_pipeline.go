package image

import (
	"image"
	"math"
)

// spriteAIAlphaThreshold mirrors the threshold used by Sprite-AI's browser
// converter. Its regular image path converts alpha values above 128 to opaque
// pixels and everything else to transparent pixels before quantization.
const spriteAIAlphaThreshold = uint8(128)

const (
	spriteAIIntermediateScale = 4
	spriteAIFitIterations     = 5
)

// spriteAIResize reproduces the browser converter's geometry stage:
//
//  1. crop to the visible source content (the caller supplies that crop),
//  2. fit the content into a 4x intermediate canvas with nearest sampling,
//  3. centre it,
//  4. reduce that intermediate canvas to the requested grid with nearest
//     sampling, and
//  5. make a few conservative fit corrections when nearest rounding leaves
//     one axis short of the target.
//
// The intermediate canvas is important. Directly sampling a large generated
// image into 32x32 makes thin seams and highlights depend on a single source
// pixel. The browser converter first creates a clean 4x nearest representation
// and only then performs the final nearest reduction.
func spriteAIResize(src *image.NRGBA, crop image.Rectangle, dstW, dstH int) *image.NRGBA {
	return spriteAIResizeWithGeometry(src, crop, dstW, dstH, false)
}

func spriteAIResizeWithGeometry(src *image.NRGBA, crop image.Rectangle, dstW, dstH int, preserveCanvasGeometry bool) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, max(1, dstW), max(1, dstH)))
	if src == nil || dstW <= 0 || dstH <= 0 || crop.Empty() {
		return out
	}

	content := cropNRGBA(src, crop)
	if content.Bounds().Empty() {
		return out
	}

	contentW, contentH := content.Bounds().Dx(), content.Bounds().Dy()
	scale := math.Min(float64(dstW)/float64(contentW), float64(dstH)/float64(contentH))
	if preserveCanvasGeometry {
		// The input crop is a complete, already-padded animation frame. Do not
		// inspect its alpha bounds and enlarge the visible subject: that would
		// turn the action/safety margin into a content-fit margin and make a
		// 32x32 prototype touch the canvas edges.
		return spriteAIResizeOnce(content, dstW, dstH, scale)
	}
	if scale <= 0 {
		return out
	}

	for range spriteAIFitIterations {
		intermediateW := max(1, mathRound(float64(contentW)*scale*float64(spriteAIIntermediateScale)))
		intermediateH := max(1, mathRound(float64(contentH)*scale*float64(spriteAIIntermediateScale)))
		intermediate := image.NewNRGBA(image.Rect(0, 0, dstW*spriteAIIntermediateScale, dstH*spriteAIIntermediateScale))
		scaled := spriteAINearestResize(content, intermediateW, intermediateH)
		pasteNRGBA(intermediate, scaled, (intermediate.Bounds().Dx()-intermediateW)/2, (intermediate.Bounds().Dy()-intermediateH)/2)

		candidate := spriteAINearestResize(intermediate, dstW, dstH)
		bounds, visible := alphaBounds(candidate, 50)
		if !visible {
			return candidate
		}

		touchesX := bounds.Min.X == 0 || bounds.Max.X == dstW
		touchesY := bounds.Min.Y == 0 || bounds.Max.Y == dstH
		if touchesX && touchesY {
			return candidate
		}

		adjustX, adjustY := 1.0, 1.0
		if !touchesX && bounds.Dx() > 0 {
			adjustX = float64(dstW) / float64(bounds.Dx())
		}
		if !touchesY && bounds.Dy() > 0 {
			adjustY = float64(dstH) / float64(bounds.Dy())
		}
		scale *= math.Min(adjustX, adjustY)
	}

	intermediateW := max(1, mathRound(float64(contentW)*scale*float64(spriteAIIntermediateScale)))
	intermediateH := max(1, mathRound(float64(contentH)*scale*float64(spriteAIIntermediateScale)))
	intermediate := image.NewNRGBA(image.Rect(0, 0, dstW*spriteAIIntermediateScale, dstH*spriteAIIntermediateScale))
	scaled := spriteAINearestResize(content, intermediateW, intermediateH)
	pasteNRGBA(intermediate, scaled, (intermediate.Bounds().Dx()-intermediateW)/2, (intermediate.Bounds().Dy()-intermediateH)/2)
	return spriteAINearestResize(intermediate, dstW, dstH)
}

func spriteAIResizeOnce(content *image.NRGBA, dstW, dstH int, scale float64) *image.NRGBA {
	contentW, contentH := content.Bounds().Dx(), content.Bounds().Dy()
	intermediateW := max(1, mathRound(float64(contentW)*scale*float64(spriteAIIntermediateScale)))
	intermediateH := max(1, mathRound(float64(contentH)*scale*float64(spriteAIIntermediateScale)))
	intermediate := image.NewNRGBA(image.Rect(0, 0, dstW*spriteAIIntermediateScale, dstH*spriteAIIntermediateScale))
	scaled := spriteAINearestResize(content, intermediateW, intermediateH)
	pasteNRGBA(intermediate, scaled, (intermediate.Bounds().Dx()-intermediateW)/2, (intermediate.Bounds().Dy()-intermediateH)/2)
	return spriteAINearestResize(intermediate, dstW, dstH)
}

// spriteAINearestResize intentionally uses floor(source coordinate), matching
// the browser's ImageData nearest sampler. It is separate from the processor's
// centre-sampling nearest resize because changing that behaviour would alter
// existing non-Sprite pixel processing.
func spriteAINearestResize(src *image.NRGBA, dstW, dstH int) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, max(1, dstW), max(1, dstH)))
	if src == nil || dstW <= 0 || dstH <= 0 || src.Bounds().Empty() {
		return out
	}
	srcW, srcH := src.Bounds().Dx(), src.Bounds().Dy()
	for y := range dstH {
		sy := min(srcH-1, int(float64(y)*float64(srcH)/float64(dstH)))
		for x := range dstW {
			sx := min(srcW-1, int(float64(x)*float64(srcW)/float64(dstW)))
			out.SetNRGBA(x, y, src.NRGBAAt(src.Bounds().Min.X+sx, src.Bounds().Min.Y+sy))
		}
	}
	return out
}

func cropNRGBA(src *image.NRGBA, crop image.Rectangle) *image.NRGBA {
	crop = crop.Intersect(src.Bounds())
	if crop.Empty() {
		return image.NewNRGBA(image.Rectangle{})
	}
	out := image.NewNRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	cropW, cropH := crop.Dx(), crop.Dy()
	for y := range cropH {
		for x := range cropW {
			out.SetNRGBA(x, y, src.NRGBAAt(crop.Min.X+x, crop.Min.Y+y))
		}
	}
	return out
}

func pasteNRGBA(dst, src *image.NRGBA, offsetX, offsetY int) {
	srcW, srcH := src.Bounds().Dx(), src.Bounds().Dy()
	dstW, dstH := dst.Bounds().Dx(), dst.Bounds().Dy()
	for y := range srcH {
		dy := offsetY + y
		if dy < 0 || dy >= dstH {
			continue
		}
		for x := range srcW {
			dx := offsetX + x
			if dx < 0 || dx >= dstW {
				continue
			}
			dst.SetNRGBA(dx, dy, src.NRGBAAt(src.Bounds().Min.X+x, src.Bounds().Min.Y+y))
		}
	}
}

func mathRound(value float64) int {
	return int(math.Floor(value + 0.5))
}
