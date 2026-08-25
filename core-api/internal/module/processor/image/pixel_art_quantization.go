package image

import (
	"fmt"
	"image"
	"image/color"
	"sort"
)

type pixelArtColourSample struct {
	r, g, b, a uint8
	count      float64
}

type pixelArtColourBox struct {
	entries []pixelArtColourSample
}

// QuantizePixelArtSources applies one deterministic palette to a complete set
// of source frames. It remains available for callers that explicitly require a
// shared material palette; the prototype generator intentionally quantizes each
// direction independently so a detail present in one view cannot consume the
// palette budget of another view.
func QuantizePixelArtSources(inputs []image.Image, limit int) ([]*image.RGBA, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	frames := make([]*image.NRGBA, len(inputs))
	for index, input := range inputs {
		if input == nil {
			return nil, fmt.Errorf("pixel-art source %d is nil", index)
		}
		frame := toNRGBA(input)
		if frame.Bounds().Empty() {
			return nil, fmt.Errorf("pixel-art source %d is empty", index)
		}
		applySpriteAIHardAlpha(frame)
		frames[index] = frame
	}
	quantizePixelArtSourcesNRGBA(frames, limit)

	outputs := make([]*image.RGBA, len(frames))
	for index, frame := range frames {
		outputs[index] = ToRGBA(frame)
	}
	return outputs, nil
}

// quantizePixelArtSource implements the single-frame form used by ResizeImage.
// The shared implementation also makes the ordering and tie-breaking identical
// for one frame and for a complete direction set.
func quantizePixelArtSource(img *image.NRGBA, limit int) {
	if img == nil {
		return
	}
	quantizePixelArtSourcesNRGBA([]*image.NRGBA{img}, limit)
}

// quantizePixelArtSourcesNRGBA implements the intentionally small, deterministic
// quantization stage used by the pixel-art converter profile. Unlike the
// general palette reducer, it computes centroids and performs a few k-means
// refinement passes. This matters before downsampling: a palette built from
// already mixed destination pixels has lost narrow seam/highlight colours.
func quantizePixelArtSourcesNRGBA(frames []*image.NRGBA, limit int) {
	if len(frames) == 0 || limit <= 0 {
		return
	}
	histogram := make(map[uint32]pixelArtColourSample)
	for _, img := range frames {
		if img == nil {
			continue
		}
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				pixel := img.NRGBAAt(x, y)
				if pixel.A == 0 {
					continue
				}
				key := uint32(pixel.R)<<24 | uint32(pixel.G)<<16 | uint32(pixel.B)<<8 | uint32(pixel.A)
				entry := histogram[key]
				entry.r, entry.g, entry.b, entry.a = pixel.R, pixel.G, pixel.B, pixel.A
				entry.count++
				histogram[key] = entry
			}
		}
	}
	if len(histogram) == 0 || len(histogram) <= limit {
		return
	}

	entries := make([]pixelArtColourSample, 0, len(histogram))
	for _, entry := range histogram {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return pixelArtColourKey(entries[i]) < pixelArtColourKey(entries[j]) })
	boxes := []pixelArtColourBox{{entries: entries}}
	for len(boxes) < limit {
		best := -1
		bestScore := 0.0
		for index, box := range boxes {
			score := pixelArtBoxScore(box.entries)
			if len(box.entries) > 1 && score > bestScore {
				best, bestScore = index, score
			}
		}
		if best < 0 {
			break
		}
		left, right := splitPixelArtBox(boxes[best].entries)
		if len(left) == 0 || len(right) == 0 {
			break
		}
		boxes[best] = pixelArtColourBox{entries: left}
		boxes = append(boxes, pixelArtColourBox{entries: right})
	}

	palette := make([][4]uint8, 0, len(boxes))
	for _, box := range boxes {
		palette = append(palette, pixelArtBoxCentroid(box.entries))
	}
	for range 8 {
		changed := false
		sumR := make([]float64, len(palette))
		sumG := make([]float64, len(palette))
		sumB := make([]float64, len(palette))
		sumA := make([]float64, len(palette))
		weights := make([]float64, len(palette))
		for _, entry := range entries {
			best := 0
			bestDistance := pixelArtColourDistance(entry, palette[0])
			for index := 1; index < len(palette); index++ {
				distance := pixelArtColourDistance(entry, palette[index])
				if distance < bestDistance {
					best, bestDistance = index, distance
				}
			}
			weight := entry.count
			sumR[best] += float64(entry.r) * weight
			sumG[best] += float64(entry.g) * weight
			sumB[best] += float64(entry.b) * weight
			sumA[best] += float64(entry.a) * weight
			weights[best] += weight
		}
		for index := range palette {
			if weights[index] == 0 {
				continue
			}
			updated := [4]uint8{
				clampByte(sumR[index] / weights[index]),
				clampByte(sumG[index] / weights[index]),
				clampByte(sumB[index] / weights[index]),
				clampByte(sumA[index] / weights[index]),
			}
			if updated != palette[index] {
				changed = true
				palette[index] = updated
			}
		}
		if !changed {
			break
		}
	}

	for _, img := range frames {
		if img == nil {
			continue
		}
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				pixel := img.NRGBAAt(x, y)
				if pixel.A == 0 {
					continue
				}
				entry := pixelArtColourSample{r: pixel.R, g: pixel.G, b: pixel.B, a: pixel.A}
				best := 0
				bestDistance := pixelArtColourDistance(entry, palette[0])
				for index := 1; index < len(palette); index++ {
					distance := pixelArtColourDistance(entry, palette[index])
					if distance < bestDistance {
						best, bestDistance = index, distance
					}
				}
				value := palette[best]
				img.SetNRGBA(x, y, color.NRGBA{R: value[0], G: value[1], B: value[2], A: value[3]})
			}
		}
	}
}

func pixelArtBoxScore(entries []pixelArtColourSample) float64 {
	if len(entries) == 0 {
		return 0
	}
	var total, r, g, b float64
	for _, entry := range entries {
		total += entry.count
		r += float64(entry.r) * entry.count
		g += float64(entry.g) * entry.count
		b += float64(entry.b) * entry.count
	}
	if total == 0 {
		return 0
	}
	r, g, b = r/total, g/total, b/total
	var vr, vg, vb float64
	for _, entry := range entries {
		weight := entry.count
		vr += (float64(entry.r) - r) * (float64(entry.r) - r) * weight
		vg += (float64(entry.g) - g) * (float64(entry.g) - g) * weight
		vb += (float64(entry.b) - b) * (float64(entry.b) - b) * weight
	}
	return total * maxPixelArtFloat(0.4375*vr, maxPixelArtFloat(0.5625*vg, 0.3125*vb))
}

func splitPixelArtBox(entries []pixelArtColourSample) ([]pixelArtColourSample, []pixelArtColourSample) {
	if len(entries) < 2 {
		return nil, nil
	}
	var total, r, g, b float64
	for _, entry := range entries {
		total += entry.count
		r += float64(entry.r) * entry.count
		g += float64(entry.g) * entry.count
		b += float64(entry.b) * entry.count
	}
	r, g, b = r/total, g/total, b/total
	var vr, vg, vb float64
	for _, entry := range entries {
		weight := entry.count
		vr += (float64(entry.r) - r) * (float64(entry.r) - r) * weight
		vg += (float64(entry.g) - g) * (float64(entry.g) - g) * weight
		vb += (float64(entry.b) - b) * (float64(entry.b) - b) * weight
	}
	dimension := 0
	if 0.5625*vg >= 0.4375*vr && 0.5625*vg >= 0.3125*vb {
		dimension = 1
	} else if 0.4375*vr < 0.3125*vb {
		dimension = 2
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return pixelArtColourChannel(entries[i], dimension) < pixelArtColourChannel(entries[j], dimension)
	})
	half := total / 2
	weight := 0.0
	cut := 1
	for index, entry := range entries {
		weight += entry.count
		if weight >= half {
			cut = max(1, min(index+1, len(entries)-1))
			break
		}
	}
	return append([]pixelArtColourSample(nil), entries[:cut]...), append([]pixelArtColourSample(nil), entries[cut:]...)
}

func pixelArtBoxCentroid(entries []pixelArtColourSample) [4]uint8 {
	var r, g, b, a, total float64
	for _, entry := range entries {
		r += float64(entry.r) * entry.count
		g += float64(entry.g) * entry.count
		b += float64(entry.b) * entry.count
		a += float64(entry.a) * entry.count
		total += entry.count
	}
	return [4]uint8{clampByte(r / total), clampByte(g / total), clampByte(b / total), clampByte(a / total)}
}

func pixelArtColourDistance(entry pixelArtColourSample, palette [4]uint8) float64 {
	r := (float64(entry.r) - float64(palette[0])) * 0.4375
	g := (float64(entry.g) - float64(palette[1])) * 0.5625
	b := (float64(entry.b) - float64(palette[2])) * 0.3125
	a := (float64(entry.a) - float64(palette[3])) * 0.25
	return r*r + g*g + b*b + a*a
}

func pixelArtColourKey(entry pixelArtColourSample) uint32 {
	return uint32(entry.r)<<24 | uint32(entry.g)<<16 | uint32(entry.b)<<8 | uint32(entry.a)
}

func pixelArtColourChannel(entry pixelArtColourSample, dimension int) uint8 {
	switch dimension {
	case 0:
		return entry.r
	case 1:
		return entry.g
	default:
		return entry.b
	}
}

func maxPixelArtFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
