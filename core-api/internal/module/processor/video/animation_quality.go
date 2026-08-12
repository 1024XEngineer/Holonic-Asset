package video

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"sort"
)

func validateAnimationMotionSafeAreaAtIndices(frames []image.Image, indices []int) error {
	for _, sourceIndex := range indices {
		if sourceIndex < 0 || sourceIndex >= len(frames) {
			return fmt.Errorf("video: sampled animation frame index %d is out of range", sourceIndex)
		}
		frame := frames[sourceIndex]
		bounds, ok := animationRawForegroundBounds(frame)
		if !ok {
			return &AnimationVideoQualityError{Kind: "subject", Message: fmt.Sprintf("video: video frame %d has no detectable subject on the green screen", sourceIndex)}
		}
		if !animationBoundsInsideSafetyBand(frame, bounds) {
			return &AnimationVideoQualityError{Kind: "framing", Message: fmt.Sprintf("video: character, limb, or held object enters the outer 2.5%% safety band in source frame %d", sourceIndex)}
		}
	}
	return nil
}

func animationFrameInsideSafetyBand(frame image.Image) bool {
	bounds, ok := animationRawForegroundBounds(frame)
	return ok && animationBoundsInsideSafetyBand(frame, bounds)
}

func animationBoundsInsideSafetyBand(frame image.Image, foreground image.Rectangle) bool {
	frameBounds := frame.Bounds()
	margin := animationMaxInt(4, int(math.Round(float64(animationMinInt(frameBounds.Dx(), frameBounds.Dy()))*.025)))
	return foreground.Min.X > frameBounds.Min.X+margin &&
		foreground.Min.Y > frameBounds.Min.Y+margin &&
		foreground.Max.X < frameBounds.Max.X-margin &&
		foreground.Max.Y < frameBounds.Max.Y-margin
}

func animationRawForegroundBounds(source image.Image) (image.Rectangle, bool) {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width == 0 || height == 0 {
		return image.Rectangle{}, false
	}
	columns := make([]int, width)
	rows := make([]int, height)
	for y := range height {
		for x := range width {
			if isAnimationGreen(color.NRGBAModel.Convert(source.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.NRGBA)) {
				continue
			}
			columns[x]++
			rows[y]++
		}
	}
	lineThreshold := animationMaxInt(2, animationMinInt(width, height)/320)
	minX, maxX := width, -1
	minY, maxY := height, -1
	for x, count := range columns {
		if count >= lineThreshold {
			minX, maxX = animationMinInt(minX, x), animationMaxInt(maxX, x)
		}
	}
	for y, count := range rows {
		if count >= lineThreshold {
			minY, maxY = animationMinInt(minY, y), animationMaxInt(maxY, y)
		}
	}
	if maxX < minX || maxY < minY {
		return image.Rectangle{}, false
	}
	return image.Rect(bounds.Min.X+minX, bounds.Min.Y+minY, bounds.Min.X+maxX+1, bounds.Min.Y+maxY+1), true
}

func isAnimationGreen(value color.NRGBA) bool {
	hue, saturation, brightness := animationRGBToOpenCVHSV(value.R, value.G, value.B)
	return hue >= 30 && hue <= 90 && ((saturation >= 80 && brightness >= 80) || (saturation >= 50 && brightness >= 180))
}

func animationRGBToOpenCVHSV(red8, green8, blue8 uint8) (uint8, uint8, uint8) {
	red, green, blue := float64(red8)/255, float64(green8)/255, float64(blue8)/255
	maximum, minimum := math.Max(red, math.Max(green, blue)), math.Min(red, math.Min(green, blue))
	delta := maximum - minimum
	var hue float64
	if delta != 0 {
		switch maximum {
		case red:
			hue = 60 * math.Mod((green-blue)/delta, 6)
		case green:
			hue = 60 * ((blue-red)/delta + 2)
		default:
			hue = 60 * ((red-green)/delta + 4)
		}
	}
	if hue < 0 {
		hue += 360
	}
	saturation := 0.0
	if maximum > 0 {
		saturation = delta / maximum
	}
	return uint8(math.Round(hue / 2)), uint8(math.Round(saturation * 255)), uint8(math.Round(maximum * 255))
}

func animationDescriptorVariation(frames []animationFrameDescriptor) float64 {
	if len(frames) < 2 {
		return 0
	}
	widths := make([]float64, 0, len(frames))
	heights := make([]float64, 0, len(frames))
	areas := make([]float64, 0, len(frames))
	for _, frame := range frames {
		if frame.foreground == 0 {
			continue
		}
		widths = append(widths, frame.width)
		heights = append(heights, frame.height)
		areas = append(areas, float64(frame.foreground))
	}
	return animationStandardDeviation(widths)/animationAnalysisSize +
		animationStandardDeviation(heights)/animationAnalysisSize +
		animationStandardDeviation(areas)/(animationAnalysisSize*animationAnalysisSize)
}

func animationCentroidStd(frames []animationFrameDescriptor) float64 {
	xs := make([]float64, 0, len(frames))
	ys := make([]float64, 0, len(frames))
	for _, frame := range frames {
		if !math.IsNaN(frame.cx) && !math.IsNaN(frame.cy) {
			xs = append(xs, frame.cx)
			ys = append(ys, frame.cy)
		}
	}
	return animationStandardDeviation(xs) + animationStandardDeviation(ys)
}

func animationStableTranslationBonus(frames []animationFrameDescriptor) float64 {
	if len(frames) < 3 {
		return 0
	}
	var count, sumX, sumY, sumXX, sumXY float64
	for index, frame := range frames {
		if math.IsNaN(frame.cx) {
			continue
		}
		x := float64(index)
		count++
		sumX += x
		sumY += frame.cx
		sumXX += x * x
		sumXY += x * frame.cx
	}
	denominator := count*sumXX - sumX*sumX
	if count < 3 || math.Abs(denominator) < 1e-9 {
		return 0
	}
	slope := (count*sumXY - sumX*sumY) / denominator
	intercept := (sumY - slope*sumX) / count
	var squared float64
	for index, frame := range frames {
		if math.IsNaN(frame.cx) {
			continue
		}
		residual := frame.cx - (intercept + slope*float64(index))
		squared += residual * residual
	}
	if math.Sqrt(squared/count) < 2 {
		return 1
	}
	return 0
}

func animationStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var mean float64
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))
	var variance float64
	for _, value := range values {
		delta := value - mean
		variance += delta * delta
	}
	return math.Sqrt(variance / float64(len(values)))
}

func animationQuantile(values []float64, quantile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	position := animationClampFloat(quantile, 0, 1) * float64(len(ordered)-1)
	low, high := int(math.Floor(position)), int(math.Ceil(position))
	if low == high {
		return ordered[low]
	}
	fraction := position - float64(low)
	return ordered[low]*(1-fraction) + ordered[high]*fraction
}

func animationMinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func animationMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func animationClampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func animationClampFloat(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func animationRoundTo(value float64, places int) float64 {
	factor := math.Pow10(places)
	return math.Round(value*factor) / factor
}
