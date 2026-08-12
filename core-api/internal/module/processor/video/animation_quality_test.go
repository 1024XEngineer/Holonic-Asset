package video

import (
	"errors"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
)

func TestValidateAnimationMotionSafeAreaRejectsInvalidSamples(t *testing.T) {
	valid := greenAnimationFrame(96, 96)
	drawSubject(valid, image.Rect(30, 20, 66, 88))
	empty := greenAnimationFrame(96, 96)
	clipped := greenAnimationFrame(96, 96)
	drawSubject(clipped, image.Rect(0, 20, 36, 88))

	tests := []struct {
		name    string
		frames  []image.Image
		indices []int
		kind    string
		want    string
	}{
		{name: "negative index", frames: []image.Image{valid}, indices: []int{-1}, want: "out of range"},
		{name: "large index", frames: []image.Image{valid}, indices: []int{1}, want: "out of range"},
		{name: "missing subject", frames: []image.Image{empty}, indices: []int{0}, kind: "subject"},
		{name: "clipped subject", frames: []image.Image{clipped}, indices: []int{0}, kind: "framing"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateAnimationMotionSafeAreaAtIndices(test.frames, test.indices)
			if test.kind != "" {
				var qualityErr *AnimationVideoQualityError
				if !errors.As(err, &qualityErr) || qualityErr.Kind != test.kind {
					t.Fatalf("expected %s quality error, got %v", test.kind, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}

	if err := validateAnimationMotionSafeAreaAtIndices([]image.Image{valid}, []int{0}); err != nil {
		t.Fatalf("expected safe frame to pass: %v", err)
	}
}

func TestAnimationForegroundBoundsHandlesEmptyAndOffsetImages(t *testing.T) {
	if _, ok := animationRawForegroundBounds(image.NewNRGBA(image.Rectangle{})); ok {
		t.Fatal("expected empty image to have no foreground")
	}
	frame := image.NewNRGBA(image.Rect(10, 20, 50, 60))
	for y := frame.Bounds().Min.Y; y < frame.Bounds().Max.Y; y++ {
		for x := frame.Bounds().Min.X; x < frame.Bounds().Max.X; x++ {
			frame.SetNRGBA(x, y, color.NRGBA{G: 255, A: 255})
		}
	}
	for y := 30; y < 45; y++ {
		for x := 22; x < 35; x++ {
			frame.SetNRGBA(x, y, color.NRGBA{R: 120, G: 40, B: 20, A: 255})
		}
	}
	bounds, ok := animationRawForegroundBounds(frame)
	if !ok || bounds != image.Rect(22, 30, 35, 45) {
		t.Fatalf("unexpected foreground bounds: %v, ok=%v", bounds, ok)
	}
}

func TestAnimationNumericHelpersCoverBoundaries(t *testing.T) {
	if got := animationStandardDeviation(nil); got != 0 {
		t.Fatalf("empty standard deviation = %v", got)
	}
	if got := animationQuantile(nil, .5); got != 0 {
		t.Fatalf("empty quantile = %v", got)
	}
	if got := animationQuantile([]float64{3, 1, 2}, 0); got != 1 {
		t.Fatalf("low quantile = %v", got)
	}
	if got := animationQuantile([]float64{3, 1, 2}, 1); got != 3 {
		t.Fatalf("high quantile = %v", got)
	}
	if got := animationQuantile([]float64{1, 3}, .5); got != 2 {
		t.Fatalf("interpolated quantile = %v", got)
	}
	if animationClampInt(-1, 0, 4) != 0 || animationClampInt(5, 0, 4) != 4 || animationClampInt(2, 0, 4) != 2 {
		t.Fatal("unexpected integer clamp")
	}
	if animationClampFloat(-1, 0, 4) != 0 || animationClampFloat(5, 0, 4) != 4 || animationClampFloat(2, 0, 4) != 2 {
		t.Fatal("unexpected float clamp")
	}
	if got := animationRoundTo(1.234, 2); math.Abs(got-1.23) > 1e-9 {
		t.Fatalf("rounded value = %v", got)
	}
}

func TestAnimationDescriptorMetricsHandleSparseInput(t *testing.T) {
	if got := animationDescriptorVariation(nil); got != 0 {
		t.Fatalf("empty variation = %v", got)
	}
	frames := []animationFrameDescriptor{
		{cx: math.NaN(), cy: math.NaN()},
		{cx: 10, cy: 20, width: 5, height: 8, foreground: 40},
		{cx: 11, cy: 21, width: 7, height: 10, foreground: 70},
	}
	if got := animationDescriptorVariation(frames); got <= 0 {
		t.Fatalf("expected descriptor variation, got %v", got)
	}
	if got := animationCentroidStd(frames); got <= 0 {
		t.Fatalf("expected centroid deviation, got %v", got)
	}
	if got := animationStableTranslationBonus(frames[:2]); got != 0 {
		t.Fatalf("short translation bonus = %v", got)
	}
	linear := []animationFrameDescriptor{{cx: 1}, {cx: 2}, {cx: 3}, {cx: 4}}
	if got := animationStableTranslationBonus(linear); got != 1 {
		t.Fatalf("linear translation bonus = %v", got)
	}
	unstable := []animationFrameDescriptor{{cx: 1}, {cx: 20}, {cx: -10}, {cx: 30}}
	if got := animationStableTranslationBonus(unstable); got != 0 {
		t.Fatalf("unstable translation bonus = %v", got)
	}
}
