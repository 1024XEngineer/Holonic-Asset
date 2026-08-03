package image

import (
	"fmt"
	"image"
	"math"
	"slices"
	"strconv"
	"strings"
)

func ParseMatteColor(value string) (MatteColor, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "black":
		return MatteColor{0, 0, 0}, nil
	case "white":
		return MatteColor{255, 255, 255}, nil
	case "green", "chroma-green":
		return MatteColor{0, 255, 0}, nil
	case "magenta":
		return MatteColor{255, 0, 255}, nil
	case "cyan":
		return MatteColor{0, 255, 255}, nil
	case "blue":
		return MatteColor{0, 0, 255}, nil
	}
	hex := strings.TrimPrefix(normalized, "#")
	if len(hex) != 6 {
		return MatteColor{}, fmt.Errorf("matte color must be a named color or #RRGGBB: %q", value)
	}
	var out MatteColor
	for i := range 3 {
		parsed, err := strconv.ParseUint(hex[i*2:i*2+2], 16, 8)
		if err != nil {
			return MatteColor{}, fmt.Errorf("invalid matte color %q: %w", value, err)
		}
		out[i] = uint8(parsed)
	}
	return out, nil
}

func ParseMatteColorOrAuto(value string) (MatteColor, bool, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "auto", "sample", "auto-sample", "auto_sample":
		return MatteColor{}, true, nil
	default:
		matte, err := ParseMatteColor(value)
		if err != nil {
			return MatteColor{}, false, err
		}
		return matte, false, nil
	}
}

func ColorToHex(color MatteColor) string {
	return fmt.Sprintf("#%02x%02x%02x", color[0], color[1], color[2])
}

func ColorDistance(a, b MatteColor) float64 {
	red := float64(a[0]) - float64(b[0])
	green := float64(a[1]) - float64(b[1])
	blue := float64(a[2]) - float64(b[2])
	return (red*red + green*green + blue*blue) / 1
}

func EuclideanColorDistance(a, b MatteColor) float64 {
	return sqrt(ColorDistance(a, b))
}

func EstimateMatteColor(img image.Image) MatteColor {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return MatteColor{}
	}
	sample := min(min(width, height), 32)
	reds := make([]uint8, 0, sample*sample*4)
	greens := make([]uint8, 0, sample*sample*4)
	blues := make([]uint8, 0, sample*sample*4)
	for y := range sample {
		for x := range sample {
			pushRGB(img, bounds.Min.X+x, bounds.Min.Y+y, &reds, &greens, &blues)
			pushRGB(img, bounds.Max.X-1-x, bounds.Min.Y+y, &reds, &greens, &blues)
			pushRGB(img, bounds.Min.X+x, bounds.Max.Y-1-y, &reds, &greens, &blues)
			pushRGB(img, bounds.Max.X-1-x, bounds.Max.Y-1-y, &reds, &greens, &blues)
		}
	}
	return MatteColor{median(reds), median(greens), median(blues)}
}

func pushRGB(img image.Image, x, y int, red, green, blue *[]uint8) {
	r, g, b, _ := img.At(x, y).RGBA()
	*red = append(*red, colorChannel8(r))
	*green = append(*green, colorChannel8(g))
	*blue = append(*blue, colorChannel8(b))
}

func colorChannel8(value uint32) uint8 {
	value >>= 8
	if value > 255 {
		return 255
	}
	return uint8(value)
}

func median(values []uint8) uint8 {
	if len(values) == 0 {
		return 0
	}
	slices.Sort(values)
	return values[len(values)/2]
}

func ratio(count, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(count) / float64(total)
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func sqrt(v float64) float64 {
	// Kept as a tiny wrapper so color math remains easy to scan beside the Rust port.
	return math.Sqrt(v)
}
