package image

import (
	"image"
	"image/color"
)

const (
	pixelGridSamplingRecovered = "recovered-pixel-grid"
	pixelGridSamplingFallback  = "recovered-pixel-grid-area"
	pixelGridMinScale          = 2
	pixelGridMaxScale          = 16
	pixelGridPhaseConfidence   = 0.015
)

// recoverPixelGridResize converts a supersampled prototype frame to its final
// logical canvas without averaging across neighbouring logical pixels. The
// prototype splitter deliberately emits a frame at PrototypeRenderScale, so
// the final reduction has a known candidate grid. We still recover the phase
// from hard-neighbour edges instead of assuming that the provider aligned its
// soft mixel blocks to the frame origin.
//
// If the source geometry does not contain an integral candidate grid, the
// reducer falls back to an alpha-aware area sample. This is intentionally
// conservative: a grid detector must not invent a grid for ordinary artwork.
func recoverPixelGridResize(src *image.NRGBA, crop image.Rectangle, dstW, dstH int, preferNearest, spritePipeline bool) (*image.NRGBA, string) {
	if src == nil || crop.Empty() || dstW <= 0 || dstH <= 0 {
		return qualityResize(src, crop, dstW, dstH, RasterModePixel)
	}
	scaleX, okX := integralGridScale(crop.Dx(), dstW)
	scaleY, okY := integralGridScale(crop.Dy(), dstH)
	if !okX || !okY || scaleX != scaleY || scaleX < pixelGridMinScale || scaleX > pixelGridMaxScale {
		if preferNearest {
			return nearestResize(src, crop, dstW, dstH), pixelGridSamplingNearestFallback
		}
		return gridAwareAreaResize(src, crop, dstW, dstH), pixelGridSamplingFallback
	}

	phaseX := recoverGridPhase(src, crop, scaleX, true)
	phaseY := recoverGridPhase(src, crop, scaleY, false)
	if spritePipeline {
		return sampleRecoveredGridNearest(src, crop, dstW, dstH, scaleX, phaseX, phaseY), pixelGridSamplingRecovered
	}
	return sampleRecoveredGrid(src, crop, dstW, dstH, scaleX, phaseX, phaseY), pixelGridSamplingRecovered
}

func integralGridScale(source, target int) (int, bool) {
	if source <= 0 || target <= 0 || source%target != 0 {
		return 0, false
	}
	return source / target, true
}

// recoverGridPhase scores candidate boundaries by the amount of hard colour or
// alpha change that lands on them. The score is compared with the strongest
// competing phase; low-confidence images use phase zero, avoiding arbitrary
// shifts on smooth or noisy input.
func recoverGridPhase(src *image.NRGBA, crop image.Rectangle, scale int, horizontal bool) int {
	if scale <= 1 {
		return 0
	}
	scores := make([]float64, scale)
	for phase := range scale {
		var boundary, interior float64
		if horizontal {
			for x := crop.Min.X + phase + scale - 1; x < crop.Max.X-1; x += scale {
				boundary += verticalEdgeEnergy(src, crop, x)
			}
			for x := crop.Min.X + phase; x < crop.Max.X-1; x++ {
				if (x-crop.Min.X-phase+1)%scale != 0 {
					interior += verticalEdgeEnergy(src, crop, x)
				}
			}
		} else {
			for y := crop.Min.Y + phase + scale - 1; y < crop.Max.Y-1; y += scale {
				boundary += horizontalEdgeEnergy(src, crop, y)
			}
			for y := crop.Min.Y + phase; y < crop.Max.Y-1; y++ {
				if (y-crop.Min.Y-phase+1)%scale != 0 {
					interior += horizontalEdgeEnergy(src, crop, y)
				}
			}
		}
		scores[phase] = boundary - interior*0.2
	}
	best, second := 0, 0.0
	for phase := 1; phase < len(scores); phase++ {
		if scores[phase] > scores[best] {
			second = scores[best]
			best = phase
		} else if scores[phase] > second {
			second = scores[phase]
		}
	}
	if scores[best] <= 0 || scores[best]-second < pixelGridPhaseConfidence {
		return 0
	}
	return best
}

func verticalEdgeEnergy(src *image.NRGBA, crop image.Rectangle, x int) float64 {
	if x < crop.Min.X || x+1 >= crop.Max.X {
		return 0
	}
	energy := 0.0
	for y := crop.Min.Y; y < crop.Max.Y; y++ {
		energy += hardNeighbourEnergy(src.NRGBAAt(x, y), src.NRGBAAt(x+1, y))
	}
	return energy
}

func horizontalEdgeEnergy(src *image.NRGBA, crop image.Rectangle, y int) float64 {
	if y < crop.Min.Y || y+1 >= crop.Max.Y {
		return 0
	}
	energy := 0.0
	for x := crop.Min.X; x < crop.Max.X; x++ {
		energy += hardNeighbourEnergy(src.NRGBAAt(x, y), src.NRGBAAt(x, y+1))
	}
	return energy
}

func hardNeighbourEnergy(left, right color.NRGBA) float64 {
	alpha := absInt(int(left.A) - int(right.A))
	colour := absInt(int(left.R)-int(right.R)) + absInt(int(left.G)-int(right.G)) + absInt(int(left.B)-int(right.B))
	return float64(alpha)/255 + float64(colour)/765
}

func sampleRecoveredGridNearest(
	src *image.NRGBA,
	crop image.Rectangle,
	dstW, dstH, scale, phaseX, phaseY int,
) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	for dy := range dstH {
		for dx := range dstW {
			x := min(crop.Max.X-1, crop.Min.X+phaseX+dx*scale+scale/2)
			y := min(crop.Max.Y-1, crop.Min.Y+phaseY+dy*scale+scale/2)
			if x < crop.Min.X || y < crop.Min.Y {
				continue
			}
			out.SetNRGBA(dx, dy, src.NRGBAAt(x, y))
		}
	}
	return out
}

func sampleRecoveredGrid(
	src *image.NRGBA,
	crop image.Rectangle,
	dstW, dstH, scale, phaseX, phaseY int,
) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	for dy := range dstH {
		for dx := range dstW {
			startX := crop.Min.X + phaseX + dx*scale
			startY := crop.Min.Y + phaseY + dy*scale
			out.SetNRGBA(dx, dy, voteRecoveredGridPixel(src, crop, startX, startY, scale))
		}
	}
	return out
}

// voteRecoveredGridPixel reads one source block as one logical pixel. The
// model's anti-aliased fringe can create several nearby RGB shades inside that
// block, so a quantized colour vote is more stable than an arithmetic average
// and does not invent a colour at a boundary. The strongest source sample is
// retained as the representative for the winning bucket.
func voteRecoveredGridPixel(src *image.NRGBA, crop image.Rectangle, startX, startY, scale int) color.NRGBA {
	type vote struct {
		support        float64
		representative color.NRGBA
		distance       int
	}
	votes := make(map[uint16]vote)
	visibleSupport := 0.0
	centerX, centerY := startX+scale/2, startY+scale/2
	for y := startY; y < startY+scale; y++ {
		for x := startX; x < startX+scale; x++ {
			if !image.Pt(x, y).In(crop) {
				continue
			}
			pixel := src.NRGBAAt(x, y)
			if pixel.A < hardAlphaThreshold {
				continue
			}
			weight := float64(pixel.A) / 255
			visibleSupport += weight
			key := uint16(pixel.R>>3)<<10 | uint16(pixel.G>>3)<<5 | uint16(pixel.B>>3)
			distance := absInt(x-centerX) + absInt(y-centerY)
			current := votes[key]
			current.support += weight
			if current.representative.A == 0 || pixel.A > current.representative.A ||
				(pixel.A == current.representative.A && distance < current.distance) {
				current.representative = pixel
				current.distance = distance
			}
			votes[key] = current
		}
	}
	if visibleSupport < float64(scale*scale)*float64(hardAlphaThreshold)/255 {
		return color.NRGBA{}
	}
	var best vote
	for _, candidate := range votes {
		if candidate.support > best.support ||
			(candidate.support == best.support && candidate.distance < best.distance) {
			best = candidate
		}
	}
	if best.representative.A == 0 {
		return strongestNearbyPixel(src, crop, centerX, centerY, scale)
	}
	best.representative.A = 255
	return best.representative
}

func strongestNearbyPixel(src *image.NRGBA, crop image.Rectangle, centerX, centerY, scale int) color.NRGBA {
	best := src.NRGBAAt(centerX, centerY)
	bestDistance := int(^uint(0) >> 1)
	radius := max(1, scale/2)
	for y := max(crop.Min.Y, centerY-radius); y <= min(crop.Max.Y-1, centerY+radius); y++ {
		for x := max(crop.Min.X, centerX-radius); x <= min(crop.Max.X-1, centerX+radius); x++ {
			pixel := src.NRGBAAt(x, y)
			distance := absInt(x-centerX) + absInt(y-centerY)
			if pixel.A > best.A || (pixel.A == best.A && distance < bestDistance) {
				best, bestDistance = pixel, distance
			}
		}
	}
	return best
}

// gridAwareAreaResize is the safe path for non-integral dimensions. It keeps
// the geometric area coverage used by the existing reducer. Palette mapping
// still runs afterwards and can only choose colours already present in the
// source. A non-integral geometry is deliberately not forced onto a guessed
// grid.
func gridAwareAreaResize(src *image.NRGBA, crop image.Rectangle, dstW, dstH int) *image.NRGBA {
	return areaResize(src, crop, dstW, dstH)
}
