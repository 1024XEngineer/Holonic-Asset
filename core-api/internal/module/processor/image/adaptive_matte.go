package image

import (
	"image"
	"image/color"
	"slices"
)

// AnimationMatteCandidates is deliberately limited to saturated colours that
// are easy for image-to-video models and chroma keying to distinguish.
var AnimationMatteCandidates = []MatteColor{
	{0, 255, 0},   // green
	{255, 0, 255}, // magenta
	{0, 255, 255}, // cyan
	{0, 0, 255},   // blue
	{255, 255, 0}, // yellow
	{255, 0, 0},   // red
}

// MatteSafetyDistanceFloor is the minimum acceptable Euclidean RGB distance
// between any visible subject pixel and the chosen matte. A candidate whose
// closest subject pixel is nearer than this value would be incorrectly removed
// during global chroma keying, so the caller must retain conservative
// border-connected extraction instead.
const MatteSafetyDistanceFloor = 25.0

// SelectAnimationMatteColor chooses a saturated matte whose distance from the
// visible subject is largest. Transparent pixels are ignored, so the result is
// stable for both transparent prototypes and already-keyed references.
//
// The returned bool indicates whether the chosen matte is subject-safe: when
// false the caller must not enable global chroma removal because no candidate
// is far enough from every visible subject pixel.
func SelectAnimationMatteColor(subject image.Image) (MatteColor, bool) {
	if subject == nil || subject.Bounds().Empty() {
		return AnimationMatteCandidates[0], true
	}

	distances := make([][]float64, len(AnimationMatteCandidates))
	for index := range distances {
		distances[index] = make([]float64, 0, subject.Bounds().Dx()*subject.Bounds().Dy())
	}
	visible := 0
	for y := subject.Bounds().Min.Y; y < subject.Bounds().Max.Y; y++ {
		for x := subject.Bounds().Min.X; x < subject.Bounds().Max.X; x++ {
			r, g, b, a := subject.At(x, y).RGBA()
			if uint8(a>>8&0xff) < NontransparentAlphaMin {
				continue
			}
			pixel := MatteColor{uint8(r >> 8 & 0xff), uint8(g >> 8 & 0xff), uint8(b >> 8 & 0xff)}
			visible++
			for index, candidate := range AnimationMatteCandidates {
				distances[index] = append(distances[index], EuclideanColorDistance(pixel, candidate))
			}
		}
	}
	if visible == 0 {
		return AnimationMatteCandidates[0], true
	}

	best := 0
	bestScore := -1.0
	for index, values := range distances {
		slices.Sort(values)
		percentile := values[min(len(values)-1, max(0, len(values)/20))]
		mean := 0.0
		for _, value := range values {
			mean += value
		}
		mean /= float64(len(values))
		score := percentile*0.75 + mean*0.25
		if score > bestScore {
			best, bestScore = index, score
		}
	}

	safe := distances[best][0] >= MatteSafetyDistanceFloor
	return AnimationMatteCandidates[best], safe
}

// PrepareAnimationForeground converts an opaque source with a sampled matte to
// a foreground image. Existing alpha is preserved. This is used only for
// prepared references; normal prototype references are removed by Processor so
// callers can retain the regular processing/reporting path.
func PrepareAnimationForeground(source image.Image) image.Image {
	if source == nil {
		return nil
	}
	if imageHasTransparency(source) {
		return ToRGBA(source)
	}
	matte := EstimateMatteColor(source)
	foreground, _ := extractBorderConnectedChromaWithReport(source, &matte, DefaultChromaSettings())
	return foreground
}

// CompositeAnimationMatte places a foreground image on a uniform matte. It
// keeps the source alpha and RGB spill cleanup while ensuring the provider sees
// the same explicit colour that downstream chroma-keying will remove.
func CompositeAnimationMatte(foreground image.Image, matte MatteColor, canvasSize image.Point) image.Image {
	if canvasSize.X <= 0 || canvasSize.Y <= 0 {
		canvasSize = image.Pt(1, 1)
	}
	canvas := image.NewNRGBA(image.Rectangle{Max: canvasSize})
	background := color.NRGBA{R: matte[0], G: matte[1], B: matte[2], A: 255}
	for y := range canvasSize.Y {
		for x := range canvasSize.X {
			canvas.SetNRGBA(x, y, background)
		}
	}
	if foreground == nil || foreground.Bounds().Empty() {
		return canvas
	}
	placement := image.Pt(
		(canvasSize.X-foreground.Bounds().Dx())/2,
		(canvasSize.Y-foreground.Bounds().Dy())/2,
	)
	for y := foreground.Bounds().Min.Y; y < foreground.Bounds().Max.Y; y++ {
		for x := foreground.Bounds().Min.X; x < foreground.Bounds().Max.X; x++ {
			pixel := color.NRGBAModel.Convert(foreground.At(x, y)).(color.NRGBA)
			if pixel.A == 0 {
				continue
			}
			dx, dy := x-foreground.Bounds().Min.X+placement.X, y-foreground.Bounds().Min.Y+placement.Y
			if dx < 0 || dy < 0 || dx >= canvasSize.X || dy >= canvasSize.Y {
				continue
			}
			canvas.SetNRGBA(dx, dy, pixel)
		}
	}
	return canvas
}

func imageHasTransparency(source image.Image) bool {
	if source == nil {
		return false
	}
	for y := source.Bounds().Min.Y; y < source.Bounds().Max.Y; y++ {
		for x := source.Bounds().Min.X; x < source.Bounds().Max.X; x++ {
			_, _, _, alpha := source.At(x, y).RGBA()
			if alpha < 0xffff {
				return true
			}
		}
	}
	return false
}
