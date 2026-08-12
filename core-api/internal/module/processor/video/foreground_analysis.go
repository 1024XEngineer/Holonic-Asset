package video

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

func (key ChromaKey) valid() bool {
	return key.HueMax >= key.HueMin && key.HighSaturationMin > 0 && key.HighValueMin > 0 &&
		key.BrightSaturationMin > 0 && key.BrightValueMin > 0
}

func (key ChromaKey) matches(value color.NRGBA) bool {
	hue, saturation, brightness := rgbToOpenCVHSV(value.R, value.G, value.B)
	return hue >= key.HueMin && hue <= key.HueMax &&
		((saturation >= key.HighSaturationMin && brightness >= key.HighValueMin) ||
			(saturation >= key.BrightSaturationMin && brightness >= key.BrightValueMin))
}

func validateSelectedFrameBounds(frames []image.Image, sourceIndices []int, chromaKey ChromaKey) error {
	if len(frames) != len(sourceIndices) {
		return fmt.Errorf("video: decoded %d selected frames; expected %d", len(frames), len(sourceIndices))
	}
	for index, frame := range frames {
		sourceIndex := sourceIndices[index]
		bounds, ok := foregroundBounds(frame, chromaKey)
		if !ok {
			return &QualityError{Kind: "foreground", Message: fmt.Sprintf("video: frame %d has no detectable foreground outside the configured chroma key", sourceIndex)}
		}
		if !boundsInsideSafetyBand(frame, bounds) {
			return &QualityError{Kind: "framing", Message: fmt.Sprintf("video: foreground content enters the outer 2.5%% safety band in source frame %d", sourceIndex)}
		}
	}
	return nil
}

func validateFrameBoundsAtIndices(frames []image.Image, indices []int, chromaKey ChromaKey) error {
	for _, sourceIndex := range indices {
		if sourceIndex < 0 || sourceIndex >= len(frames) {
			return fmt.Errorf("video: sampled frame index %d is out of range", sourceIndex)
		}
		bounds, ok := foregroundBounds(frames[sourceIndex], chromaKey)
		if !ok {
			return &QualityError{Kind: "foreground", Message: fmt.Sprintf("video: frame %d has no detectable foreground outside the configured chroma key", sourceIndex)}
		}
		if !boundsInsideSafetyBand(frames[sourceIndex], bounds) {
			return &QualityError{Kind: "framing", Message: fmt.Sprintf("video: foreground content enters the outer 2.5%% safety band in source frame %d", sourceIndex)}
		}
	}
	return nil
}

func frameInsideSafetyBand(frame image.Image, chromaKey ChromaKey) bool {
	bounds, ok := foregroundBounds(frame, chromaKey)
	return ok && boundsInsideSafetyBand(frame, bounds)
}

func boundsInsideSafetyBand(frame image.Image, foreground image.Rectangle) bool {
	frameBounds := frame.Bounds()
	margin := maxInt(4, int(math.Round(float64(minInt(frameBounds.Dx(), frameBounds.Dy()))*.025)))
	return foreground.Min.X > frameBounds.Min.X+margin &&
		foreground.Min.Y > frameBounds.Min.Y+margin &&
		foreground.Max.X < frameBounds.Max.X-margin &&
		foreground.Max.Y < frameBounds.Max.Y-margin
}

func foregroundBounds(source image.Image, chromaKey ChromaKey) (image.Rectangle, bool) {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width == 0 || height == 0 {
		return image.Rectangle{}, false
	}
	columns := make([]int, width)
	rows := make([]int, height)
	for y := range height {
		for x := range width {
			if chromaKey.matches(color.NRGBAModel.Convert(source.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.NRGBA)) {
				continue
			}
			columns[x]++
			rows[y]++
		}
	}
	lineThreshold := maxInt(2, minInt(width, height)/320)
	minX, maxX := width, -1
	minY, maxY := height, -1
	for x, count := range columns {
		if count >= lineThreshold {
			minX, maxX = minInt(minX, x), maxInt(maxX, x)
		}
	}
	for y, count := range rows {
		if count >= lineThreshold {
			minY, maxY = minInt(minY, y), maxInt(maxY, y)
		}
	}
	if maxX < minX || maxY < minY {
		return image.Rectangle{}, false
	}
	return image.Rect(bounds.Min.X+minX, bounds.Min.Y+minY, bounds.Min.X+maxX+1, bounds.Min.Y+maxY+1), true
}

func rgbToOpenCVHSV(red8, green8, blue8 uint8) (uint8, uint8, uint8) {
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
