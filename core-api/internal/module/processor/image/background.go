package image

import (
	"image"
	"image/color"
	"math"
	"slices"
)

// Chroma math in this file is adapted from OpenAI Codex's Apache-2.0-licensed
// remove_chroma_key.py sample:
// https://github.com/openai/codex/blob/main/codex-rs/skills/src/assets/samples/imagegen/scripts/remove_chroma_key.py
// The adaptive routing and border-connected candidate mask are repository-
// specific additions. Images whose foreground substantially overlaps the key
// hue retain the repository's original distance-only extraction; other images
// use the imagegen-inspired soft matte and edge cleanup.
const (
	chromaKeyDominanceThreshold = 16.0
	chromaAlphaNoiseFloor       = uint8(8)
	chromaOverlapDistanceRatio  = 0.5
	chromaOverlapDistanceMax    = 24.0
	chromaKeyOverlapRouteRatio  = 0.05
	chromaEdgeContractPixels    = 1
	chromaEdgeFeatherRadius     = 0.6
)

type chromaCandidatePixel struct {
	rgb         MatteColor
	sourceAlpha uint8
	outputAlpha uint8
	candidate   bool
	keyLike     bool
}

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
	if chromaHasHighForegroundKeyOverlap(input, matte, settings) {
		return extractGlobalDistanceChroma(input, matte, settings)
	}
	return extractBorderConnectedChroma(input, matte, settings)
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
			out := color.NRGBA{}
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
			output.Set(x, y, out)
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

// chromaHasHighForegroundKeyOverlap distinguishes a subject that deliberately
// uses the key hue from the ordinary antialiased key spill around a silhouette.
// Near-flat matte pixels are excluded first; among the remaining image content,
// a meaningful share of key-like transition pixels selects the distance-only
// path so dominance alpha cannot hollow out same-coloured foreground details.
func chromaHasHighForegroundKeyOverlap(
	input image.Image,
	matte MatteColor,
	settings ChromaSettings,
) bool {
	bounds := input.Bounds()
	nearMatteThreshold := math.Min(
		chromaOverlapDistanceMax,
		settings.Threshold*chromaOverlapDistanceRatio,
	)
	opaqueThreshold := math.Max(
		settings.Threshold+1,
		settings.Threshold+settings.Softness,
	)
	var keyTransitionPixels, foregroundPixels uint64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := input.At(x, y).RGBA()
			rgb := MatteColor{colorChannel8(r), colorChannel8(g), colorChannel8(b)}
			distance := chromaChannelDistance(rgb, matte)
			if distance <= nearMatteThreshold {
				continue
			}
			if distance < opaqueThreshold && chromaLooksKeyColored(rgb, matte, distance) {
				keyTransitionPixels++
				continue
			}
			foregroundPixels++
		}
	}
	totalSubjectPixels := keyTransitionPixels + foregroundPixels
	if totalSubjectPixels == 0 {
		return false
	}
	return float64(keyTransitionPixels)/float64(totalSubjectPixels) >= chromaKeyOverlapRouteRatio
}

// extractGlobalDistanceChroma retains the repository's original Euclidean
// distance extraction for subjects with substantial key-coloured content. It
// removes enclosed matte regions while avoiding dominance alpha, which would
// otherwise treat the subject's own key hue as spill.
func extractGlobalDistanceChroma(
	input image.Image,
	matte MatteColor,
	settings ChromaSettings,
) *image.RGBA {
	bounds := input.Bounds()
	output := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	low := settings.Threshold
	high := math.Max(low+1, low+settings.Softness)
	for y := range bounds.Dy() {
		for x := range bounds.Dx() {
			r, g, b, a := input.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			source := [4]uint8{
				colorChannel8(r),
				colorChannel8(g),
				colorChannel8(b),
				colorChannel8(a),
			}
			distance := EuclideanColorDistance(
				MatteColor{source[0], source[1], source[2]},
				matte,
			)
			t := clamp((distance-low)/(high-low), 0, 1)
			smoothed := t * t * (3 - 2*t)
			alpha := uint8(math.Round(clamp(smoothed*255, 0, 255)))
			output.Set(
				x,
				y,
				decontaminatePixel(source, matte, alpha, settings.SpillSuppression),
			)
		}
	}
	return output
}

func decontaminatePixel(
	source [4]uint8,
	matte MatteColor,
	alpha uint8,
	spillSuppression float64,
) color.NRGBA {
	if source[3] < alpha {
		alpha = source[3]
	}
	if alpha <= chromaAlphaNoiseFloor {
		return color.NRGBA{}
	}
	alphaF := float64(alpha) / 255
	decontaminationStrength := alphaF * alphaF
	out := color.NRGBA{}
	for channel := range 3 {
		recovered := (float64(source[channel]) - float64(matte[channel])*(1-alphaF)) /
			math.Max(alphaF, 0.001)
		// Straight matte inversion is unstable near transparent edges. Blend the
		// recovered foreground by alpha so small matte deviations cannot explode
		// into saturated colours that were absent from the source image.
		value := float64(source[channel]) +
			(recovered-float64(source[channel]))*decontaminationStrength
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
	preserveSourceKeyChannelOrder(&out, source, matte)
	out.A = alpha
	return out
}

func preserveSourceKeyChannelOrder(
	pixel *color.NRGBA,
	source [4]uint8,
	matte MatteColor,
) {
	spillChannels := chromaSpillChannels(matte)
	if len(spillChannels) == 0 {
		return
	}
	sourceRGB := MatteColor{source[0], source[1], source[2]}
	corrected := MatteColor{pixel.R, pixel.G, pixel.B}
	sourceNonKey := chromaNonSpillStrength(sourceRGB, spillChannels)
	correctedNonKey := chromaNonSpillStrength(corrected, spillChannels)
	for _, channel := range spillChannels {
		if float64(sourceRGB[channel]) < sourceNonKey ||
			float64(corrected[channel]) >= correctedNonKey {
			continue
		}
		corrected[channel] = uint8(math.Round(correctedNonKey))
	}
	pixel.R = corrected[0]
	pixel.G = corrected[1]
	pixel.B = corrected[2]
}

func suppressMatteSpill(
	pixel *color.NRGBA,
	matte MatteColor,
	alpha uint8,
	amount float64,
) {
	amount = clamp(amount, 0, 1)
	if amount <= 0 || alpha <= TransparentAlphaMax {
		return
	}
	maxMatte, minMatte := matte[0], matte[0]
	for _, value := range matte[1:] {
		maxMatte = max(maxMatte, value)
		minMatte = min(minMatte, value)
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
		reference = max(reference, value)
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

func extractBorderConnectedChroma(
	input image.Image,
	matte MatteColor,
	settings ChromaSettings,
) *image.RGBA {
	bounds := input.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	output := image.NewRGBA(image.Rect(0, 0, width, height))
	if width == 0 || height == 0 {
		return output
	}

	transparentThreshold := settings.Threshold
	opaqueThreshold := math.Max(
		transparentThreshold+1,
		transparentThreshold+settings.Softness,
	)
	pixels := make([]chromaCandidatePixel, width*height)
	for y := range height {
		for x := range width {
			r, g, b, a := input.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			rgb := MatteColor{colorChannel8(r), colorChannel8(g), colorChannel8(b)}
			sourceAlpha := colorChannel8(a)
			distance := chromaChannelDistance(rgb, matte)
			keyLike := chromaLooksKeyColored(rgb, matte, distance)
			distanceAlpha := chromaSoftAlpha(
				distance,
				transparentThreshold,
				opaqueThreshold,
			)
			outputAlpha := sourceAlpha
			if keyLike {
				outputAlpha = min(
					distanceAlpha,
					chromaDominanceAlpha(rgb, matte),
				)
				outputAlpha = uint8(math.Round(
					float64(outputAlpha) * float64(sourceAlpha) / 255,
				))
				if outputAlpha > 0 && outputAlpha <= chromaAlphaNoiseFloor {
					outputAlpha = 0
				}
			}
			pixels[y*width+x] = chromaCandidatePixel{
				rgb:         rgb,
				sourceAlpha: sourceAlpha,
				outputAlpha: outputAlpha,
				candidate:   keyLike && outputAlpha < sourceAlpha,
				keyLike:     keyLike,
			}
		}
	}

	removable := borderConnectedChromaMask(pixels, width, height)
	for y := range height {
		for x := range width {
			index := y*width + x
			candidate := pixels[index]
			alpha := candidate.sourceAlpha
			rgb := candidate.rgb
			if removable[index] {
				alpha = candidate.outputAlpha
				if alpha == 0 {
					output.SetRGBA(x, y, color.RGBA{})
					continue
				}
				if candidate.keyLike {
					rgb = chromaCleanupSpill(
						rgb,
						matte,
						alpha,
						settings.SpillSuppression,
					)
				}
			}
			output.Set(x, y, color.NRGBA{
				R: rgb[0],
				G: rgb[1],
				B: rgb[2],
				A: alpha,
			})
		}
	}

	contractChromaAlpha(output, chromaEdgeContractPixels)
	featherChromaAlpha(output, chromaEdgeFeatherRadius)
	ScrubTransparentRGB(output)
	return output
}

func borderConnectedChromaMask(
	pixels []chromaCandidatePixel,
	width, height int,
) []bool {
	connected := make([]bool, len(pixels))
	queue := make([]int, 0, 2*(width+height))
	enqueue := func(x, y int) {
		index := y*width + x
		if connected[index] || !pixels[index].candidate {
			return
		}
		connected[index] = true
		queue = append(queue, index)
	}

	for x := range width {
		enqueue(x, 0)
		if height > 1 {
			enqueue(x, height-1)
		}
	}
	for y := 1; y < height-1; y++ {
		enqueue(0, y)
		if width > 1 {
			enqueue(width-1, y)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		x, y := current%width, current/width
		for nearbyY := max(0, y-1); nearbyY <= min(height-1, y+1); nearbyY++ {
			for nearbyX := max(0, x-1); nearbyX <= min(width-1, x+1); nearbyX++ {
				if nearbyX == x && nearbyY == y {
					continue
				}
				enqueue(nearbyX, nearbyY)
			}
		}
	}
	return connected
}

func chromaChannelDistance(left, right MatteColor) float64 {
	return float64(max(
		absInt(int(left[0])-int(right[0])),
		absInt(int(left[1])-int(right[1])),
		absInt(int(left[2])-int(right[2])),
	))
}

func chromaSoftAlpha(distance, transparentThreshold, opaqueThreshold float64) uint8 {
	if distance <= transparentThreshold {
		return 0
	}
	if distance >= opaqueThreshold {
		return 255
	}
	ratio := (distance - transparentThreshold) / (opaqueThreshold - transparentThreshold)
	smoothed := ratio * ratio * (3 - 2*ratio)
	return uint8(math.Round(clamp(255*smoothed, 0, 255)))
}

func chromaDominanceAlpha(rgb, key MatteColor) uint8 {
	spillChannels := chromaSpillChannels(key)
	if len(spillChannels) == 0 {
		return 255
	}

	keyStrength := float64(rgb[spillChannels[0]])
	if len(spillChannels) > 1 {
		for _, channel := range spillChannels[1:] {
			keyStrength = math.Min(keyStrength, float64(rgb[channel]))
		}
	}
	nonKeyStrength := chromaNonSpillStrength(rgb, spillChannels)
	dominance := keyStrength - nonKeyStrength
	if dominance <= 0 {
		return 255
	}
	denominator := math.Max(1, float64(max(key[0], key[1], key[2]))-nonKeyStrength)
	alpha := 1 - math.Min(1, dominance/denominator)
	return uint8(math.Round(clamp(alpha*255, 0, 255)))
}

func chromaLooksKeyColored(rgb, key MatteColor, distance float64) bool {
	if distance <= 32 {
		return true
	}
	if len(chromaSpillChannels(key)) == 0 {
		return true
	}
	return chromaKeyChannelDominance(rgb, key) >= chromaKeyDominanceThreshold
}

func chromaKeyChannelDominance(rgb, key MatteColor) float64 {
	spillChannels := chromaSpillChannels(key)
	if len(spillChannels) == 0 {
		return 0
	}
	keyStrength := float64(rgb[spillChannels[0]])
	if len(spillChannels) > 1 {
		for _, channel := range spillChannels[1:] {
			keyStrength = math.Min(keyStrength, float64(rgb[channel]))
		}
	}
	return keyStrength - chromaNonSpillStrength(rgb, spillChannels)
}

func chromaSpillChannels(key MatteColor) []int {
	keyMax := max(key[0], key[1], key[2])
	if keyMax < 128 {
		return nil
	}
	channels := make([]int, 0, 2)
	for channel, value := range key {
		if value >= 128 && int(value) >= int(keyMax)-16 {
			channels = append(channels, channel)
		}
	}
	return channels
}

func chromaNonSpillStrength(rgb MatteColor, spillChannels []int) float64 {
	strength := 0.0
	for channel, value := range rgb {
		if containsChannel(spillChannels, channel) {
			continue
		}
		strength = math.Max(strength, float64(value))
	}
	return strength
}

func containsChannel(channels []int, wanted int) bool {
	return slices.Contains(channels, wanted)
}

func chromaCleanupSpill(
	rgb, key MatteColor,
	alpha uint8,
	amount float64,
) MatteColor {
	amount = clamp(amount, 0, 1)
	if amount <= 0 || alpha >= 252 {
		return rgb
	}
	spillChannels := chromaSpillChannels(key)
	if len(spillChannels) == 0 {
		return rgb
	}
	anchor := chromaNonSpillStrength(rgb, spillChannels)
	capValue := math.Max(0, anchor-1)
	cleaned := rgb
	for _, channel := range spillChannels {
		current := float64(cleaned[channel])
		if current <= capValue {
			continue
		}
		cleaned[channel] = uint8(math.Round(clamp(
			current+(capValue-current)*amount,
			0,
			255,
		)))
	}
	return cleaned
}

func contractChromaAlpha(img *image.RGBA, iterations int) {
	if iterations <= 0 {
		return
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	alpha := chromaAlphaChannel(img)
	for range iterations {
		contracted := make([]uint8, len(alpha))
		for y := range height {
			for x := range width {
				minimum := uint8(255)
				for nearbyY := max(0, y-1); nearbyY <= min(height-1, y+1); nearbyY++ {
					for nearbyX := max(0, x-1); nearbyX <= min(width-1, x+1); nearbyX++ {
						minimum = min(minimum, alpha[nearbyY*width+nearbyX])
					}
				}
				contracted[y*width+x] = minimum
			}
		}
		alpha = contracted
	}
	setChromaAlphaChannel(img, alpha)
}

func featherChromaAlpha(img *image.RGBA, radius float64) {
	if radius <= 0 {
		return
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	alpha := chromaAlphaChannel(img)
	kernel, halfWidth := chromaGaussianKernel(radius)
	horizontal := make([]float64, len(alpha))
	for y := range height {
		for x := range width {
			value := 0.0
			for offset := -halfWidth; offset <= halfWidth; offset++ {
				sampleX := min(width-1, max(0, x+offset))
				value += float64(alpha[y*width+sampleX]) * kernel[offset+halfWidth]
			}
			horizontal[y*width+x] = value
		}
	}
	blurred := make([]uint8, len(alpha))
	for y := range height {
		for x := range width {
			value := 0.0
			for offset := -halfWidth; offset <= halfWidth; offset++ {
				sampleY := min(height-1, max(0, y+offset))
				value += horizontal[sampleY*width+x] * kernel[offset+halfWidth]
			}
			blurred[y*width+x] = uint8(math.Round(clamp(value, 0, 255)))
		}
	}
	setChromaAlphaChannel(img, blurred)
}

func chromaGaussianKernel(radius float64) ([]float64, int) {
	sigma := math.Max(radius, 0.1)
	halfWidth := max(1, int(math.Ceil(3*sigma)))
	kernel := make([]float64, halfWidth*2+1)
	total := 0.0
	for offset := -halfWidth; offset <= halfWidth; offset++ {
		weight := math.Exp(-float64(offset*offset) / (2 * sigma * sigma))
		kernel[offset+halfWidth] = weight
		total += weight
	}
	for index := range kernel {
		kernel[index] /= total
	}
	return kernel, halfWidth
}

func chromaAlphaChannel(img *image.RGBA) []uint8 {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	alpha := make([]uint8, width*height)
	for y := range height {
		for x := range width {
			alpha[y*width+x] = img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y).A
		}
	}
	return alpha
}

func setChromaAlphaChannel(img *image.RGBA, alpha []uint8) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	for y := range height {
		for x := range width {
			pixel := color.NRGBAModel.Convert(
				img.At(bounds.Min.X+x, bounds.Min.Y+y),
			).(color.NRGBA)
			pixel.A = alpha[y*width+x]
			img.Set(bounds.Min.X+x, bounds.Min.Y+y, pixel)
		}
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
