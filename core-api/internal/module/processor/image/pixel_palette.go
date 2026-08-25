package image

import (
	"image"
	"image/color"
	"math"
	"sort"
)

type palettePoint struct {
	r, g, b float64
	weight  uint64
}

type weightedPaletteColour struct {
	colour color.RGBA
	weight uint64
}

type oklabColour struct {
	l, a, b float64
}

type paletteBox struct {
	points      []palettePoint
	minR, maxR  float64
	minG, maxG  float64
	minB, maxB  float64
	totalWeight uint64
}

func newPaletteBox(points []palettePoint) paletteBox {
	box := paletteBox{points: points, minR: 255, minG: 255, minB: 255}
	for _, p := range points {
		box.minR, box.maxR = math.Min(box.minR, p.r), math.Max(box.maxR, p.r)
		box.minG, box.maxG = math.Min(box.minG, p.g), math.Max(box.maxG, p.g)
		box.minB, box.maxB = math.Min(box.minB, p.b), math.Max(box.maxB, p.b)
		box.totalWeight += p.weight
	}
	return box
}

func (box paletteBox) score() float64 {
	rangeMax := math.Max(box.maxR-box.minR, math.Max(box.maxG-box.minG, box.maxB-box.minB))
	return rangeMax * rangeMax * math.Sqrt(float64(box.totalWeight))
}

func splitPaletteBox(box paletteBox) (paletteBox, paletteBox, bool) {
	if len(box.points) < 2 {
		return paletteBox{}, paletteBox{}, false
	}
	rRange, gRange, bRange := box.maxR-box.minR, box.maxG-box.minG, box.maxB-box.minB
	channel := 0
	if gRange >= rRange && gRange >= bRange {
		channel = 1
	} else if bRange >= rRange && bRange >= gRange {
		channel = 2
	}
	sort.Slice(box.points, func(i, j int) bool {
		a, b := box.points[i], box.points[j]
		var primaryA, primaryB float64
		switch channel {
		case 1:
			primaryA, primaryB = a.g, b.g
		case 2:
			primaryA, primaryB = a.b, b.b
		default:
			primaryA, primaryB = a.r, b.r
		}
		if primaryA != primaryB {
			return primaryA < primaryB
		}
		return palettePointKey(a) < palettePointKey(b)
	})
	half, accumulated, split := box.totalWeight/2, uint64(0), 1
	for i, point := range box.points[:len(box.points)-1] {
		accumulated += point.weight
		if accumulated >= half {
			split = i + 1
			break
		}
	}
	return newPaletteBox(box.points[:split]), newPaletteBox(box.points[split:]), true
}

// buildWeightedPalette uses weighted median-cut rather than retaining only the
// most frequent histogram buckets. Each palette entry keeps its represented
// pixel weight so the pixel-art pass can distinguish a dominant colour ramp
// from a small intentional accent.
func buildWeightedPalette(
	img *image.NRGBA,
	bounds image.Rectangle,
	limit int,
	alphaThreshold uint8,
) []weightedPaletteColour {
	if limit <= 0 {
		return nil
	}
	bounds = bounds.Intersect(img.Bounds())
	type accumulator struct {
		weight       uint64
		sampleKey    uint32
		sampleWeight uint64
		hasSample    bool
	}
	hist := make(map[int]*accumulator)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			p := img.NRGBAAt(x, y)
			if p.A <= alphaThreshold {
				continue
			}
			key := int(p.R>>3)<<10 | int(p.G>>3)<<5 | int(p.B>>3)
			entry := hist[key]
			if entry == nil {
				entry = &accumulator{}
				hist[key] = entry
			}
			weight := uint64(p.A)
			entry.weight += weight
			sampleKey := rgbaKey(color.RGBA{R: p.R, G: p.G, B: p.B})
			if !entry.hasSample || weight > entry.sampleWeight ||
				(weight == entry.sampleWeight && sampleKey < entry.sampleKey) {
				entry.sampleKey = sampleKey
				entry.sampleWeight = weight
				entry.hasSample = true
			}
		}
	}
	points := make([]palettePoint, 0, len(hist))
	for _, entry := range hist {
		if entry.weight == 0 {
			continue
		}
		// Keep a real colour from the reduced source rather than the weighted
		// average of a histogram bucket. The latter can create a new saturated
		// colour that was never present in the generated art.
		points = append(points, palettePoint{
			r:      float64((entry.sampleKey >> 16) & 0xff),
			g:      float64((entry.sampleKey >> 8) & 0xff),
			b:      float64(entry.sampleKey & 0xff),
			weight: entry.weight,
		})
	}
	if len(points) == 0 {
		return nil
	}
	// Map iteration order must not decide the palette or the generated asset.
	sort.Slice(points, func(i, j int) bool {
		return palettePointKey(points[i]) < palettePointKey(points[j])
	})
	boxes := []paletteBox{newPaletteBox(points)}
	for len(boxes) < limit {
		best := -1
		bestScore := -1.0
		for i, box := range boxes {
			if len(box.points) > 1 && box.score() > bestScore {
				best, bestScore = i, box.score()
			}
		}
		if best < 0 {
			break
		}
		left, right, ok := splitPaletteBox(boxes[best])
		if !ok {
			break
		}
		boxes[best] = left
		boxes = append(boxes, right)
	}
	palette := make([]weightedPaletteColour, 0, len(boxes))
	for _, box := range boxes {
		palette = append(palette, weightedPaletteColour{
			colour: representativePaletteColour(box),
			weight: box.totalWeight,
		})
	}
	return palette
}

func buildPalette(img *image.NRGBA, bounds image.Rectangle, limit int, alphaThreshold uint8) []color.RGBA {
	weighted := buildWeightedPalette(img, bounds, limit, alphaThreshold)
	palette := make([]color.RGBA, 0, len(weighted))
	for _, entry := range weighted {
		palette = append(palette, entry.colour)
	}
	return palette
}

// representativePaletteColour returns an exact source colour from a median-cut
// box. Choosing a real sample rather than the box average ensures palette
// reduction never invents a new hue, saturation, or lightness.
func representativePaletteColour(box paletteBox) color.RGBA {
	best := box.points[0]
	for _, point := range box.points[1:] {
		if point.weight > best.weight ||
			(point.weight == best.weight && palettePointKey(point) < palettePointKey(best)) {
			best = point
		}
	}
	return color.RGBA{R: clampByte(best.r), G: clampByte(best.g), B: clampByte(best.b), A: 255}
}

func palettePointKey(point palettePoint) uint32 {
	return uint32(clampByte(point.r))<<16 | uint32(clampByte(point.g))<<8 | uint32(clampByte(point.b))
}

func remapToPalette(img *image.NRGBA, bounds image.Rectangle, palette []color.RGBA) {
	if len(palette) == 0 {
		return
	}
	bounds = bounds.Intersect(img.Bounds())
	snapshot := cloneNRGBA(img)
	paletteLabs := make([]oklabColour, len(palette))
	for index, candidate := range palette {
		paletteLabs[index] = rgbaToOKLab(candidate)
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			p := snapshot.NRGBAAt(x, y)
			if p.A <= TransparentAlphaMax {
				continue
			}
			sourceLab := nrgbaToOKLab(p)
			best := palette[0]
			bestDistance := perceptualColourDistance(sourceLab, paletteLabs[0])
			for index, candidate := range palette[1:] {
				distance := perceptualColourDistance(sourceLab, paletteLabs[index+1])
				if distance < bestDistance ||
					(distance == bestDistance && rgbaKey(candidate) < rgbaKey(best)) {
					best, bestDistance = candidate, distance
				}
			}
			img.SetNRGBA(x, y, color.NRGBA{R: best.R, G: best.G, B: best.B, A: p.A})
		}
	}
}

func perceptualColourDistance(left, right oklabColour) float64 {
	dl := left.l - right.l
	da := left.a - right.a
	db := left.b - right.b
	// Readable value grouping is slightly more important than tiny hue changes
	// at sprite scale, while a/b remain perceptually uniform chroma axes.
	return 1.6*dl*dl + da*da + db*db
}

func rgbaToOKLab(value color.RGBA) oklabColour {
	return srgbToOKLab(value.R, value.G, value.B)
}

func nrgbaToOKLab(value color.NRGBA) oklabColour {
	return srgbToOKLab(value.R, value.G, value.B)
}

func srgbToOKLab(red, green, blue uint8) oklabColour {
	r := srgbChannelToLinear(float64(red) / 255)
	g := srgbChannelToLinear(float64(green) / 255)
	b := srgbChannelToLinear(float64(blue) / 255)
	l := 0.4122214708*r + 0.5363325363*g + 0.0514459929*b
	m := 0.2119034982*r + 0.6806995451*g + 0.1073969566*b
	s := 0.0883024619*r + 0.2817188376*g + 0.6299787005*b
	lc, mc, sc := math.Cbrt(l), math.Cbrt(m), math.Cbrt(s)
	return oklabColour{
		l: 0.2104542553*lc + 0.793617785*mc - 0.0040720468*sc,
		a: 1.9779984951*lc - 2.428592205*mc + 0.4505937099*sc,
		b: 0.0259040371*lc + 0.7827717662*mc - 0.808675766*sc,
	}
}

func srgbChannelToLinear(value float64) float64 {
	if value <= 0.04045 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}

func rgbaKey(value color.RGBA) uint32 {
	return uint32(value.R)<<16 | uint32(value.G)<<8 | uint32(value.B)
}

func applyPalette(img *image.NRGBA, limit int) {
	palette := buildPalette(img, img.Bounds(), limit, TransparentAlphaMax)
	remapToPalette(img, img.Bounds(), palette)
}
