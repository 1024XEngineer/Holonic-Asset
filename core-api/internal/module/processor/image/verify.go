package image

import (
	"image"
	"math"
)

func verifyImage(
	img image.Image,
	isPNG bool,
	colorType string,
	hasAlpha bool,
	opts VerificationOptions,
) VerificationReport {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return VerificationReport{
			Profile:        opts.Profile,
			ColorType:      colorType,
			IsPNG:          isPNG,
			HasAlpha:       hasAlpha,
			InputHasAlpha:  hasAlpha,
			Passed:         false,
			FailureReasons: []string{"empty_image"},
		}
	}

	var alphaMin, alphaMax uint8 = 255, 0
	var transparentPixels, partialPixels, opaquePixels, nontransparentPixels uint64
	var edgeNontransparentPixels uint64
	var transparentRGBScrubbed = true
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	anyNontransparent := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			r8, g8, b8, a8 := colorChannel8(r), colorChannel8(g), colorChannel8(b), colorChannel8(a)
			if a8 < alphaMin {
				alphaMin = a8
			}
			if a8 > alphaMax {
				alphaMax = a8
			}
			if a8 <= TransparentAlphaMax {
				transparentPixels++
				if r8 > 2 || g8 > 2 || b8 > 2 {
					transparentRGBScrubbed = false
				}
				continue
			}
			nontransparentPixels++
			anyNontransparent = true
			if a8 < MinOpaqueAlpha {
				partialPixels++
			} else {
				opaquePixels++
			}
			if x == bounds.Min.X || y == bounds.Min.Y || x == bounds.Max.X-1 || y == bounds.Max.Y-1 {
				edgeNontransparentPixels++
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}

	edgePixels := edgePixelCount(width, height)
	edgeNontransparentRatio := ratio(edgeNontransparentPixels, edgePixels)
	var bbox *AlphaBoundingBox
	if anyNontransparent {
		bbox = &AlphaBoundingBox{
			X:      minX - bounds.Min.X,
			Y:      minY - bounds.Min.Y,
			Width:  maxX - minX + 1,
			Height: maxY - minY + 1,
		}
	}
	var edgeMargin *int
	if bbox != nil {
		v := edgeMarginPx(bbox, width, height)
		edgeMargin = &v
	}
	touchesEdge := bbox != nil && edgeMargin != nil && *edgeMargin == 0

	stats := computeComponentStats(img)
	matteResidueChecked := opts.ExpectedMatteColor != nil
	var matteResidueScore *float64
	if opts.ExpectedMatteColor != nil {
		score := matteResidueScoreFor(img, *opts.ExpectedMatteColor)
		matteResidueScore = &score
	}
	halo := haloScore(img)
	totalPixels := uint64(width) * uint64(height)
	transparentRatio := ratio(transparentPixels, totalPixels)
	checkerboardDetected := opts.Profile != ProfileOpaqueBackground &&
		(!hasAlpha || transparentRatio < MinTransparentRatio) && detectCheckerboard(img)

	warnings := make([]string, 0, 4)
	if opts.Profile != ProfileOpaqueBackground && (touchesEdge || edgeNontransparentRatio > 0.15) {
		warnings = append(warnings, "nontransparent pixels reach the image edge; consider adding margin before extraction")
	}
	if opts.Profile != ProfileOpaqueBackground && partialPixels == 0 {
		warnings = append(warnings, "no semi-transparent pixels detected")
	}
	if checkerboardDetected {
		warnings = append(warnings, "checkerboard-like pattern detected; visual transparency is not enough")
	}
	if !transparentRGBScrubbed {
		warnings = append(warnings, "fully transparent pixels contain non-zero RGB values; scrub them to avoid compositing artifacts")
	}
	if opts.Profile != ProfileOpaqueBackground && matteResidueScore != nil && *matteResidueScore > 0.12 {
		warnings = append(warnings, "possible matte-color residue on semi-transparent edge pixels")
	}
	if !matteResidueChecked && (opts.Profile == ProfileIcon || opts.Profile == ProfileProduct || opts.Profile == ProfileSticker || opts.Profile == ProfileSeal) && partialPixels > 0 {
		warnings = append(warnings, "matte residue was not checked; pass an expected matte color when verifying chroma outputs")
	}

	passed, failureReasons := evaluateTransparencyGate(TransparencyGateInput{
		Profile:                opts.Profile,
		IsPNG:                  isPNG,
		HasAlpha:               hasAlpha,
		AlphaMin:               alphaMin,
		AlphaMax:               alphaMax,
		NontransparentPixels:   nontransparentPixels,
		TransparentRatio:       transparentRatio,
		PartialPixels:          partialPixels,
		TouchesEdge:            touchesEdge,
		LargestComponentRatio:  stats.LargestComponentRatio,
		AlphaNoiseScore:        stats.AlphaNoiseScore,
		MatteResidueScore:      matteResidueScore,
		CheckerboardDetected:   checkerboardDetected,
		TransparentRGBScrubbed: transparentRGBScrubbed,
	})

	alphaHealth := computeAlphaHealthScore(AlphaHealthInput{
		IsPNG:                  isPNG,
		HasAlpha:               hasAlpha,
		AlphaMin:               alphaMin,
		AlphaMax:               alphaMax,
		NontransparentPixels:   nontransparentPixels,
		TransparentRatio:       transparentRatio,
		CheckerboardDetected:   checkerboardDetected,
		TransparentRGBScrubbed: transparentRGBScrubbed,
	})
	scoreTouchesEdge := touchesEdge
	scoreMatteResidue := matteResidueScore
	if opts.Profile == ProfileOpaqueBackground {
		alphaHealth = computeOpaqueAlphaHealthScore(isPNG, nontransparentPixels, alphaMin)
		scoreTouchesEdge = false
		scoreMatteResidue = nil
	}
	residue := computeResidueScore(stats.AlphaNoiseScore, scoreMatteResidue, halo, scoreTouchesEdge)
	quality := computeQualityScore(passed, scoreTouchesEdge, stats.AlphaNoiseScore, scoreMatteResidue, halo, checkerboardDetected, transparentRGBScrubbed)

	return VerificationReport{
		Profile:                  opts.Profile,
		Width:                    width,
		Height:                   height,
		IsPNG:                    isPNG,
		ColorType:                colorType,
		HasAlpha:                 hasAlpha,
		InputHasAlpha:            hasAlpha,
		AlphaMin:                 alphaMin,
		AlphaMax:                 alphaMax,
		TransparentPixels:        transparentPixels,
		PartialPixels:            partialPixels,
		OpaquePixels:             opaquePixels,
		NontransparentPixels:     nontransparentPixels,
		TransparentRatio:         transparentRatio,
		PartialRatio:             ratio(partialPixels, totalPixels),
		OpaqueRatio:              ratio(opaquePixels, totalPixels),
		EdgeNontransparentPixels: edgeNontransparentPixels,
		EdgeNontransparentRatio:  edgeNontransparentRatio,
		TouchesEdge:              touchesEdge,
		EdgeMarginPx:             edgeMargin,
		ComponentCount:           stats.ComponentCount,
		LargestComponentPixels:   stats.LargestComponentPixels,
		LargestComponentRatio:    stats.LargestComponentRatio,
		StrayPixelCount:          stats.StrayPixelCount,
		AlphaNoiseScore:          stats.AlphaNoiseScore,
		MatteResidueScore:        matteResidueScore,
		MatteResidueChecked:      matteResidueChecked,
		HaloScore:                halo,
		TransparentRGBScrubbed:   transparentRGBScrubbed,
		CheckerboardDetected:     checkerboardDetected,
		AlphaHealthScore:         alphaHealth,
		ResidueScore:             residue,
		QualityScore:             quality,
		BBox:                     bbox,
		Passed:                   passed,
		FailureReasons:           failureReasons,
		Warnings:                 warnings,
	}
}

type TransparencyGateInput struct {
	Profile                Profile
	IsPNG                  bool
	HasAlpha               bool
	AlphaMin               uint8
	AlphaMax               uint8
	NontransparentPixels   uint64
	TransparentRatio       float64
	PartialPixels          uint64
	TouchesEdge            bool
	LargestComponentRatio  float64
	AlphaNoiseScore        float64
	MatteResidueScore      *float64
	CheckerboardDetected   bool
	TransparentRGBScrubbed bool
}

func evaluateTransparencyGate(input TransparencyGateInput) (bool, []string) {
	failures := make([]string, 0, 8)
	if !input.IsPNG {
		failures = append(failures, "not_png")
	}
	if input.Profile == ProfileOpaqueBackground {
		if input.CheckerboardDetected {
			failures = append(failures, "checkerboard_detected")
		}
		if input.NontransparentPixels == 0 {
			failures = append(failures, "empty_subject")
		}
		if input.AlphaMin < MinOpaqueAlpha {
			failures = append(failures, "background_not_fully_opaque")
		}
		if input.TransparentRatio > 0 {
			failures = append(failures, "background_not_full_canvas")
		}
		if !input.TransparentRGBScrubbed {
			failures = append(failures, "transparent_rgb_not_scrubbed")
		}
		return len(failures) == 0, failures
	}
	if !input.HasAlpha {
		failures = append(failures, "missing_alpha_channel")
	}
	if input.CheckerboardDetected {
		failures = append(failures, "checkerboard_detected")
	}
	if input.NontransparentPixels == 0 {
		failures = append(failures, "empty_subject")
	}
	if input.AlphaMin > TransparentAlphaMax {
		failures = append(failures, "no_fully_transparent_pixels")
	}
	if input.AlphaMax < NontransparentAlphaMin {
		failures = append(failures, "alpha_range_too_low")
	}
	if input.TransparentRatio < MinTransparentRatio {
		failures = append(failures, "transparent_area_too_small")
	}
	if !input.TransparentRGBScrubbed {
		failures = append(failures, "transparent_rgb_not_scrubbed")
	}

	switch input.Profile {
	case ProfileIcon, ProfileProduct:
		if input.AlphaMax < MinOpaqueAlpha {
			failures = append(failures, "profile_requires_opaque_pixels")
		}
		if input.TransparentRatio < StrictMinTransparentRatio {
			failures = append(failures, "profile_transparent_area_too_small")
		}
		if input.TouchesEdge {
			failures = append(failures, "subject_touches_edge")
		}
		if input.LargestComponentRatio < 0.92 || input.AlphaNoiseScore > 0.08 {
			failures = append(failures, "too_many_stray_pixels")
		}
		if input.MatteResidueScore != nil && *input.MatteResidueScore > 0.18 {
			failures = append(failures, "matte_residue_too_high")
		}
	case ProfileSticker:
		if input.AlphaMax < MinOpaqueAlpha {
			failures = append(failures, "profile_requires_opaque_pixels")
		}
		if input.TransparentRatio < StrictMinTransparentRatio {
			failures = append(failures, "profile_transparent_area_too_small")
		}
		if input.TouchesEdge {
			failures = append(failures, "subject_touches_edge")
		}
		if input.LargestComponentRatio < 0.75 || input.AlphaNoiseScore > 0.25 {
			failures = append(failures, "too_many_stray_pixels")
		}
		if input.MatteResidueScore != nil && *input.MatteResidueScore > 0.22 {
			failures = append(failures, "matte_residue_too_high")
		}
	case ProfileSeal:
		if input.AlphaMax < MinOpaqueAlpha {
			failures = append(failures, "profile_requires_opaque_pixels")
		}
		if input.TransparentRatio < StrictMinTransparentRatio {
			failures = append(failures, "profile_transparent_area_too_small")
		}
		if input.TouchesEdge {
			failures = append(failures, "subject_touches_edge")
		}
		if input.AlphaNoiseScore > 0.60 {
			failures = append(failures, "too_many_stray_pixels")
		}
		if input.MatteResidueScore != nil && *input.MatteResidueScore > 0.24 {
			failures = append(failures, "matte_residue_too_high")
		}
	case ProfileEffect:
		if input.TransparentRatio < 0.02 {
			failures = append(failures, "profile_transparent_area_too_small")
		}
		if input.TouchesEdge {
			failures = append(failures, "effect_touches_edge")
		}
	case ProfileTranslucent, ProfileGlow, ProfileShadow:
		if input.PartialPixels == 0 {
			failures = append(failures, "profile_requires_partial_alpha")
		}
		if input.TransparentRatio < 0.02 {
			failures = append(failures, "profile_transparent_area_too_small")
		}
		if input.TouchesEdge {
			failures = append(failures, "effect_touches_edge")
		}
	}
	return len(failures) == 0, failures
}

func computeOpaqueAlphaHealthScore(isPNG bool, nontransparentPixels uint64, alphaMin uint8) float64 {
	score := 1.0
	if !isPNG {
		score -= 0.25
	}
	if nontransparentPixels == 0 {
		score -= 0.5
	}
	if alphaMin < MinOpaqueAlpha {
		score -= 0.5
	}
	return clamp(score, 0, 1)
}

type componentStats struct {
	ComponentCount         uint64
	LargestComponentPixels uint64
	LargestComponentRatio  float64
	StrayPixelCount        uint64
	AlphaNoiseScore        float64
}

func computeComponentStats(img image.Image) componentStats {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width == 0 || height == 0 {
		return componentStats{}
	}
	visited := make([]bool, width*height)
	index := func(x, y int) int { return y*width + x }
	stack := make([][2]int, 0, 64)
	var componentCount, largest, nontransparent uint64
	for y := range height {
		for x := range width {
			idx := index(x, y)
			if visited[idx] || alphaAt(img, bounds.Min.X+x, bounds.Min.Y+y) <= TransparentAlphaMax {
				continue
			}
			componentCount++
			count := uint64(0)
			visited[idx] = true
			stack = append(stack, [2]int{x, y})
			for len(stack) > 0 {
				last := len(stack) - 1
				cx, cy := stack[last][0], stack[last][1]
				stack = stack[:last]
				count++
				for ny := max(0, cy-1); ny <= min(height-1, cy+1); ny++ {
					for nx := max(0, cx-1); nx <= min(width-1, cx+1); nx++ {
						nidx := index(nx, ny)
						if visited[nidx] || alphaAt(img, bounds.Min.X+nx, bounds.Min.Y+ny) <= TransparentAlphaMax {
							continue
						}
						visited[nidx] = true
						stack = append(stack, [2]int{nx, ny})
					}
				}
			}
			nontransparent += count
			if count > largest {
				largest = count
			}
		}
	}
	stray := nontransparent - largest
	return componentStats{
		ComponentCount:         componentCount,
		LargestComponentPixels: largest,
		LargestComponentRatio:  ratio(largest, nontransparent),
		StrayPixelCount:        stray,
		AlphaNoiseScore:        ratio(stray, nontransparent),
	}
}

func alphaAt(img image.Image, x, y int) uint8 {
	_, _, _, a := img.At(x, y).RGBA()
	return colorChannel8(a)
}

func edgePixelCount(width, height int) uint64 {
	if width <= 0 || height <= 0 {
		return 0
	}
	if width == 1 && height == 1 {
		return 1
	}
	if width == 1 {
		return uint64(height)
	}
	if height == 1 {
		return uint64(width)
	}
	return uint64(width)*2 + (uint64(height)-2)*2
}

func edgeMarginPx(bbox *AlphaBoundingBox, width, height int) int {
	if bbox == nil {
		return 0
	}
	right := width - (bbox.X + bbox.Width)
	bottom := height - (bbox.Y + bbox.Height)
	m := max(0, min(min(bbox.X, bbox.Y), min(right, bottom)))
	return m
}

func matteResidueScoreFor(img image.Image, matte MatteColor) float64 {
	maxMatte, minMatte := matte[0], matte[0]
	for _, v := range matte[1:] {
		if v > maxMatte {
			maxMatte = v
		}
		if v < minMatte {
			minMatte = v
		}
	}
	dominant := make([]int, 0, 3)
	other := make([]int, 0, 3)
	for i, v := range matte {
		if v >= maxMatte-8 {
			dominant = append(dominant, i)
		} else {
			other = append(other, i)
		}
	}
	if maxMatte >= 192 && int(maxMatte)-int(minMatte) >= 128 && len(other) > 0 {
		return saturatedMatteResidueScore(img, dominant, other)
	}
	var weightedScore, totalWeight float64
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, b2, a := img.At(x, y).RGBA()
			alpha := colorChannel8(a)
			if alpha <= TransparentAlphaMax || alpha >= MinOpaqueAlpha {
				continue
			}
			pixel := MatteColor{colorChannel8(r), colorChannel8(g), colorChannel8(b2)}
			alphaWeight := 1 - float64(alpha)/255
			similarity := 1 - EuclideanColorDistance(pixel, matte)/(255*math.Sqrt(3))
			weightedScore += clamp(similarity, 0, 1) * alphaWeight
			totalWeight += alphaWeight
		}
	}
	if totalWeight == 0 {
		return 0
	}
	return weightedScore / totalWeight
}

func saturatedMatteResidueScore(img image.Image, dominantChannels, otherChannels []int) float64 {
	var weightedScore, totalWeight float64
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, b2, a := img.At(x, y).RGBA()
			alpha := colorChannel8(a)
			if alpha <= TransparentAlphaMax || alpha >= MinOpaqueAlpha {
				continue
			}
			pixel := [3]uint8{colorChannel8(r), colorChannel8(g), colorChannel8(b2)}
			alphaWeight := 1 - float64(alpha)/255
			reference := uint8(0)
			for _, channel := range otherChannels {
				if pixel[channel] > reference {
					reference = pixel[channel]
				}
			}
			excess := 0.0
			for _, channel := range dominantChannels {
				// Convert before subtraction. uint8 subtraction wraps when the
				// dominant matte channel is below the reference channel, which
				// would turn a negative/non-residue value into a large false score.
				channelExcess := int(pixel[channel]) - int(reference)
				if channelExcess > 0 {
					excess += float64(channelExcess)
				}
			}
			weightedScore += (excess / float64(len(dominantChannels)) / 255) * alphaWeight
			totalWeight += alphaWeight
		}
	}
	if totalWeight == 0 {
		return 0
	}
	return weightedScore / totalWeight
}

func haloScore(img image.Image) float64 {
	var haloPixels, sampledPixels uint64
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, b2, a := img.At(x, y).RGBA()
			alpha := colorChannel8(a)
			if alpha <= TransparentAlphaMax || alpha >= MinOpaqueAlpha {
				continue
			}
			sampledPixels++
			red, green, blue := float64(r>>8)/255, float64(g>>8)/255, float64(b2>>8)/255
			luma := 0.2126*red + 0.7152*green + 0.0722*blue
			maxv := math.Max(red, math.Max(green, blue))
			minv := math.Min(red, math.Min(green, blue))
			chroma := maxv - minv
			if (luma < 0.04 || luma > 0.96) && chroma < 0.08 {
				haloPixels++
			}
		}
	}
	return ratio(haloPixels, sampledPixels)
}

func detectCheckerboard(img image.Image) bool {
	bounds := img.Bounds()
	if bounds.Dx() < 32 || bounds.Dy() < 32 {
		return false
	}
	for _, cellSize := range []int{8, 16, 32} {
		if checkerboardAtCellSize(img, cellSize) {
			return true
		}
	}
	return false
}

func checkerboardAtCellSize(img image.Image, cellSize int) bool {
	bounds := img.Bounds()
	cellsX := bounds.Dx() / cellSize
	cellsY := bounds.Dy() / cellSize
	if cellsX < 4 || cellsY < 4 {
		return false
	}
	sums := [2][3]float64{}
	counts := [2]float64{}
	cellColors := make([]struct {
		parity int
		color  MatteColor
	}, 0, cellsX*cellsY)
	for cy := range cellsY {
		for cx := range cellsX {
			color := averageCellColor(img, bounds.Min.X+cx*cellSize, bounds.Min.Y+cy*cellSize, cellSize)
			parity := (cx + cy) % 2
			for channel := range 3 {
				sums[parity][channel] += float64(color[channel])
			}
			counts[parity]++
			cellColors = append(cellColors, struct {
				parity int
				color  MatteColor
			}{parity: parity, color: color})
		}
	}
	if counts[0] == 0 || counts[1] == 0 {
		return false
	}
	means := [2][3]float64{}
	for parity := range 2 {
		for channel := range 3 {
			means[parity][channel] = sums[parity][channel] / counts[parity]
		}
	}
	if colorDistanceF64(means[0], means[1]) < 25 {
		return false
	}
	var squaredError, samples float64
	for _, item := range cellColors {
		for channel := range 3 {
			delta := float64(item.color[channel]) - means[item.parity][channel]
			squaredError += delta * delta
			samples++
		}
	}
	return math.Sqrt(squaredError/math.Max(samples, 1)) < 18
}

func averageCellColor(img image.Image, startX, startY, cellSize int) MatteColor {
	bounds := img.Bounds()
	endX := min(startX+cellSize, bounds.Max.X)
	endY := min(startY+cellSize, bounds.Max.Y)
	var sums [3]uint64
	var count uint64
	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			sums[0] += uint64(r >> 8)
			sums[1] += uint64(g >> 8)
			sums[2] += uint64(b >> 8)
			count++
		}
	}
	if count == 0 {
		return MatteColor{}
	}
	return MatteColor{uint64ToUint8(sums[0] / count), uint64ToUint8(sums[1] / count), uint64ToUint8(sums[2] / count)}
}

func uint64ToUint8(value uint64) uint8 {
	if value >= 255 {
		return 255
	}
	return uint8(value)
}

func computeAlphaHealthScore(input AlphaHealthInput) float64 {
	score := 1.0
	if !input.IsPNG {
		score -= 0.2
	}
	if !input.HasAlpha {
		score -= 0.45
	}
	if input.NontransparentPixels == 0 {
		score -= 0.35
	}
	if input.AlphaMin > TransparentAlphaMax {
		score -= 0.2
	}
	if input.AlphaMax < NontransparentAlphaMin {
		score -= 0.25
	}
	if input.TransparentRatio < MinTransparentRatio {
		score -= 0.2
	}
	if input.CheckerboardDetected {
		score -= 0.35
	}
	if !input.TransparentRGBScrubbed {
		score -= 0.12
	}
	return clamp(score, 0, 1)
}

type AlphaHealthInput struct {
	IsPNG                  bool
	HasAlpha               bool
	AlphaMin               uint8
	AlphaMax               uint8
	NontransparentPixels   uint64
	TransparentRatio       float64
	CheckerboardDetected   bool
	TransparentRGBScrubbed bool
}

func computeResidueScore(alphaNoiseScore float64, matteResidueScore *float64, haloScore float64, touchesEdge bool) float64 {
	score := 1.0
	score -= math.Min(alphaNoiseScore, 1.0) * 0.35
	if matteResidueScore != nil {
		score -= math.Min(*matteResidueScore, 1.0) * 0.35
	}
	score -= math.Min(haloScore, 1.0) * 0.15
	if touchesEdge {
		score -= 0.15
	}
	return clamp(score, 0, 1)
}

func computeQualityScore(passed bool, touchesEdge bool, alphaNoiseScore float64, matteResidueScore *float64, haloScore float64, checkerboardDetected bool, transparentRGBScrubbed bool) float64 {
	score := 0.65
	if passed {
		score = 1.0
	}
	if touchesEdge {
		score -= 0.2
	}
	if checkerboardDetected {
		score -= 0.45
	}
	if !transparentRGBScrubbed {
		score -= 0.2
	}
	score -= math.Min(alphaNoiseScore, 1.0) * 0.25
	if matteResidueScore != nil {
		score -= math.Min(*matteResidueScore, 1.0) * 0.25
	}
	score -= math.Min(haloScore, 1.0) * 0.10
	return clamp(score, 0, 1)
}

func colorDistanceF64(a, b [3]float64) float64 {
	red := a[0] - b[0]
	green := a[1] - b[1]
	blue := a[2] - b[2]
	return math.Sqrt(red*red + green*green + blue*blue)
}
