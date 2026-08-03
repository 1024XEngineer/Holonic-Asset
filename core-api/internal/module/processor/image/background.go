package image

import (
	"image"
	"image/color"
	"math"
)

func ResolveChromaSettings(material Material, threshold, softness, spillSuppression *float64) ChromaSettings {
	settings := ChromaSettingsForMaterial(material)
	if threshold != nil {
		settings.Threshold = *threshold
	}
	if softness != nil {
		settings.Softness = *softness
	}
	if spillSuppression != nil {
		settings.SpillSuppression = *spillSuppression
	}
	return normalizeChromaSettings(settings)
}

func normalizeChromaSettings(settings ChromaSettings) ChromaSettings {
	if settings.Threshold == 0 {
		settings.Threshold = DefaultChromaThreshold
	}
	if settings.Softness == 0 {
		settings.Softness = DefaultChromaSoftness
	}
	settings.Threshold = math.Max(0, settings.Threshold)
	settings.Softness = math.Max(1, settings.Softness)
	settings.SpillSuppression = clamp(settings.SpillSuppression, 0, 1)
	return settings
}

func ExtractChromaWithReport(input image.Image, matte *MatteColor, settings ChromaSettings) (*image.RGBA, ExtractionReport) {
	settings = normalizeChromaSettings(settings)
	var resolved MatteColor
	source := "provided"
	if matte == nil {
		resolved = EstimateMatteColor(input)
		source = "auto-sampled"
	} else {
		resolved = *matte
	}
	output := ExtractChroma(input, resolved, settings)
	edgeNoisePixelsRemoved := RemoveSmallEdgeComponents(output)
	ScrubTransparentRGB(output)
	return output, ExtractionReport{
		Method:                      MethodChroma,
		MatteColor:                  ColorToHex(resolved),
		MatteColorSource:            source,
		Threshold:                   settings.Threshold,
		Softness:                    settings.Softness,
		SpillSuppression:            settings.SpillSuppression,
		Material:                    settings.Material,
		MatteDecontaminationApplied: true,
		RGBScrubbed:                 true,
		EdgeNoisePixelsRemoved:      edgeNoisePixelsRemoved,
	}
}

func ExtractChroma(input image.Image, matte MatteColor, settings ChromaSettings) *image.RGBA {
	settings = normalizeChromaSettings(settings)
	bounds := input.Bounds()
	output := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	low := settings.Threshold
	high := math.Max(low+1, low+settings.Softness)
	for y := range bounds.Dy() {
		for x := range bounds.Dx() {
			r, g, b, a := input.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			source := [4]uint8{colorChannel8(r), colorChannel8(g), colorChannel8(b), colorChannel8(a)}
			distance := EuclideanColorDistance(MatteColor{source[0], source[1], source[2]}, matte)
			t := clamp((distance-low)/(high-low), 0, 1)
			smoothed := t * t * (3 - 2*t)
			alpha := uint8(math.Round(clamp(smoothed*255, 0, 255)))
			output.SetRGBA(x, y, decontaminatePixel(source, matte, alpha, settings.SpillSuppression))
		}
	}
	return output
}

func ExtractDual(dark, light image.Image) *image.RGBA {
	bounds := dark.Bounds()
	output := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := range bounds.Dy() {
		for x := range bounds.Dx() {
			dr, dg, db, _ := dark.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			lr, lg, lb, _ := light.At(light.Bounds().Min.X+x, light.Bounds().Min.Y+y).RGBA()
			d := [3]float64{float64(dr >> 8), float64(dg >> 8), float64(db >> 8)}
			l := [3]float64{float64(lr >> 8), float64(lg >> 8), float64(lb >> 8)}
			delta := (math.Max(0, l[0]-d[0]) + math.Max(0, l[1]-d[1]) + math.Max(0, l[2]-d[2])) / 3
			alphaF := clamp(1-delta/255, 0, 1)
			alpha := uint8(math.Round(clamp(alphaF*255, 0, 255)))
			if alpha <= TransparentAlphaMax {
				output.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
				continue
			}
			out := color.RGBA{}
			for channel, value := range d {
				set := uint8(math.Round(clamp(value/math.Max(alphaF, 0.001), 0, 255)))
				switch channel {
				case 0:
					out.R = set
				case 1:
					out.G = set
				case 2:
					out.B = set
				}
			}
			out.A = alpha
			output.SetRGBA(x, y, out)
		}
	}
	return output
}

func DualAlignmentReportFor(dark, light image.Image) DualAlignmentReport {
	bounds := dark.Bounds()
	var negativeChannels, totalChannels uint64
	var noiseSum float64
	var pixels uint64
	for y := range bounds.Dy() {
		for x := range bounds.Dx() {
			dr, dg, db, _ := dark.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			lr, lg, lb, _ := light.At(light.Bounds().Min.X+x, light.Bounds().Min.Y+y).RGBA()
			deltas := [3]float64{float64(lr>>8) - float64(dr>>8), float64(lg>>8) - float64(dg>>8), float64(lb>>8) - float64(db>>8)}
			for _, delta := range deltas {
				if delta < -2 {
					negativeChannels++
				}
				totalChannels++
			}
			mean := (deltas[0] + deltas[1] + deltas[2]) / 3
			variance := 0.0
			for _, delta := range deltas {
				centered := delta - mean
				variance += centered * centered
			}
			noiseSum += math.Sqrt(variance/3) / 255
			pixels++
		}
	}
	negativeRatio := ratio(negativeChannels, totalChannels)
	noise := 0.0
	if pixels > 0 {
		noise = noiseSum / float64(pixels)
	}
	score := clamp(1-negativeRatio*1.5-noise*1.2, 0, 1)
	return DualAlignmentReport{
		Score:              score,
		Passed:             score >= 0.55,
		NegativeDeltaRatio: negativeRatio,
		DeltaChannelNoise:  noise,
		ColorSpace:         "srgb",
	}
}

func decontaminatePixel(source [4]uint8, matte MatteColor, alpha uint8, spillSuppression float64) color.RGBA {
	if alpha <= TransparentAlphaMax {
		return color.RGBA{0, 0, 0, 0}
	}
	alphaF := float64(alpha) / 255
	out := color.RGBA{}
	for channel := range 3 {
		value := (float64(source[channel]) - float64(matte[channel])*(1-alphaF)) / math.Max(alphaF, 0.001)
		set := uint8(math.Round(clamp(value, 0, 255)))
		switch channel {
		case 0:
			out.R = set
		case 1:
			out.G = set
		case 2:
			out.B = set
		}
	}
	suppressMatteSpill(&out, matte, alpha, spillSuppression)
	if source[3] < alpha {
		alpha = source[3]
	}
	out.A = alpha
	return out
}

func suppressMatteSpill(pixel *color.RGBA, matte MatteColor, alpha uint8, amount float64) {
	amount = clamp(amount, 0, 1)
	if amount <= 0 || alpha <= TransparentAlphaMax {
		return
	}
	maxMatte, minMatte := matte[0], matte[0]
	for _, value := range matte[1:] {
		if value > maxMatte {
			maxMatte = value
		}
		if value < minMatte {
			minMatte = value
		}
	}
	if maxMatte < 192 || int(maxMatte)-int(minMatte) < 128 {
		return
	}
	dominant := make([]int, 0, 3)
	other := make([]int, 0, 3)
	for channel, value := range matte {
		if value >= maxMatte-8 {
			dominant = append(dominant, channel)
		} else {
			other = append(other, channel)
		}
	}
	if len(dominant) == 0 || len(other) == 0 {
		return
	}
	rgb := MatteColor{pixel.R, pixel.G, pixel.B}
	maxDistance := 255 * math.Sqrt(3)
	matteSimilarity := clamp(1-EuclideanColorDistance(rgb, matte)/maxDistance, 0, 1)
	alphaEdgeFactor := math.Sqrt(clamp(1-float64(alpha)/255, 0, 1))
	strength := amount * math.Max(math.Sqrt(matteSimilarity), alphaEdgeFactor)
	if strength <= 0.01 {
		return
	}
	reference := uint8(0)
	for _, channel := range other {
		var value uint8
		switch channel {
		case 0:
			value = pixel.R
		case 1:
			value = pixel.G
		case 2:
			value = pixel.B
		}
		if value > reference {
			reference = value
		}
	}
	for _, channel := range dominant {
		var current uint8
		switch channel {
		case 0:
			current = pixel.R
		case 1:
			current = pixel.G
		case 2:
			current = pixel.B
		}
		if current <= reference {
			continue
		}
		excess := float64(current - reference)
		set := uint8(math.Round(clamp(float64(current)-excess*strength, 0, 255)))
		switch channel {
		case 0:
			pixel.R = set
		case 1:
			pixel.G = set
		case 2:
			pixel.B = set
		}
	}
}

func ScrubTransparentRGB(img *image.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			pixel := img.RGBAAt(x, y)
			if pixel.A <= TransparentAlphaMax {
				img.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}
}

// RemoveSmallEdgeComponents clears tiny nontransparent components connected to
// the canvas edge. Controlled-matte generators sometimes add a few near-matte
// noise pixels at the border; after chroma extraction those pixels can have a
// small alpha and make an otherwise isolated subject appear to touch the edge.
//
// The largest component is never removed, so a genuinely edge-touching subject
// is still reported by verification instead of being silently erased.
func RemoveSmallEdgeComponents(img *image.RGBA) uint64 {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width == 0 || height == 0 {
		return 0
	}

	type component struct {
		pixels      []int
		touchesEdge bool
	}
	visited := make([]bool, width*height)
	components := make([]component, 0, 8)
	largest := 0
	index := func(x, y int) int { return y*width + x }
	alphaAtLocal := func(x, y int) uint8 {
		return img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y).A
	}

	for y := range height {
		for x := range width {
			start := index(x, y)
			if visited[start] || alphaAtLocal(x, y) <= TransparentAlphaMax {
				continue
			}
			visited[start] = true
			stack := []int{start}
			current := component{pixels: make([]int, 0, 32)}
			for len(stack) > 0 {
				last := len(stack) - 1
				pixelIndex := stack[last]
				stack = stack[:last]
				px, py := pixelIndex%width, pixelIndex/width
				current.pixels = append(current.pixels, pixelIndex)
				if px == 0 || py == 0 || px == width-1 || py == height-1 {
					current.touchesEdge = true
				}
				for ny := max(0, py-1); ny <= min(height-1, py+1); ny++ {
					for nx := max(0, px-1); nx <= min(width-1, px+1); nx++ {
						next := index(nx, ny)
						if visited[next] || alphaAtLocal(nx, ny) <= TransparentAlphaMax {
							continue
						}
						visited[next] = true
						stack = append(stack, next)
					}
				}
			}
			if len(current.pixels) > largest {
				largest = len(current.pixels)
			}
			components = append(components, current)
		}
	}
	if largest == 0 {
		return 0
	}

	// Allow enough room for scattered compression/generation noise while
	// keeping the limit far below any meaningful component relative to the
	// primary subject.
	maxNoisePixels := max(32, largest/1000)
	maxNoisePixels = min(maxNoisePixels, 1024)
	var removed uint64
	for _, current := range components {
		if !current.touchesEdge || len(current.pixels) >= largest || len(current.pixels) > maxNoisePixels {
			continue
		}
		for _, pixelIndex := range current.pixels {
			x, y := pixelIndex%width, pixelIndex/width
			img.SetRGBA(bounds.Min.X+x, bounds.Min.Y+y, color.RGBA{})
			removed++
		}
	}
	return removed
}
