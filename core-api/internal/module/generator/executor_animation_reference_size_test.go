package generator

import (
	"image"
	"reflect"
	"strings"
	"testing"
)

func TestAnimationReferenceCanvasUsesFixedSquareSizes(t *testing.T) {
	if got, want := animationReferenceCanvasSize(), image.Pt(1024, 1024); got != want {
		t.Fatalf("initial reference canvas = %v, want %v", got, want)
	}
	if got, want := animationReferenceCanvasSizeForLongEdge(animationExpandedReferenceSize), image.Pt(1920, 1920); got != want {
		t.Fatalf("expanded reference canvas = %v, want %v", got, want)
	}
}

func TestNormalizeAnimationGenerationRequestPreservesRequestedAspectRatio(t *testing.T) {
	result, err := normalizeAnimationGenerationRequest(&AnimationGenerationRequest{
		ReferenceImage: "prepared",
		FrameWidth:     224,
		FrameHeight:    192,
		AspectRatio:    " 16:9 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AspectRatio != "16:9" {
		t.Fatalf("normalized aspect ratio = %q, want supplied 16:9", result.AspectRatio)
	}
}

func TestAnimationReferencePrototypeCanvasSizePreservesProjectedPrototypeSize(t *testing.T) {
	tests := []struct {
		name                            string
		prototypeWidth, prototypeHeight int
		frameWidth, frameHeight         int
		want                            image.Point
	}{
		{
			name:            "square output",
			prototypeWidth:  32,
			prototypeHeight: 32,
			frameWidth:      64,
			frameHeight:     64,
			want:            image.Pt(512, 512),
		},
		{
			name:            "rectangular output",
			prototypeWidth:  128,
			prototypeHeight: 128,
			frameWidth:      224,
			frameHeight:     192,
			want:            image.Pt(683, 683),
		},
	}

	canvas := animationReferenceCanvasSize()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := animationReferencePrototypeCanvasSize(
				canvas,
				test.prototypeWidth,
				test.prototypeHeight,
				test.frameWidth,
				test.frameHeight,
			)
			if got != test.want {
				t.Fatalf("reference prototype canvas = %v, want %v", got, test.want)
			}

			outputScale := min(
				float64(test.frameWidth)/float64(canvas.X),
				float64(test.frameHeight)/float64(canvas.Y),
			)
			if difference := absAnimationReferenceFloat(float64(got.X)*outputScale - float64(test.prototypeWidth)); difference > 0.5 {
				t.Errorf("projected width differs from prototype by %.3f pixels", difference)
			}
			if difference := absAnimationReferenceFloat(float64(got.Y)*outputScale - float64(test.prototypeHeight)); difference > 0.5 {
				t.Errorf("projected height differs from prototype by %.3f pixels", difference)
			}
		})
	}
}

func TestAnimationReferencePrototypeCanvasSizeClampsUniformly(t *testing.T) {
	canvas := animationReferenceCanvasSize()
	got := animationReferencePrototypeCanvasSize(canvas, 290, 190, 298, 192)

	if got.X > canvas.X || got.Y > canvas.Y {
		t.Fatalf("reference prototype canvas %v exceeds provider canvas %v", got, canvas)
	}
	if got.X != canvas.X {
		t.Fatalf("clamped width = %d, want provider width %d", got.X, canvas.X)
	}
	if aspectError := absAnimationReferenceFloat(
		float64(got.X)/float64(got.Y) - float64(290)/float64(190),
	); aspectError > 0.002 {
		t.Fatalf("clamped reference changed aspect ratio: got %v, error %.6f", got, aspectError)
	}
}

func absAnimationReferenceFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func TestAnimationReferenceCanvasAndPrototypeSizeHandleInvalidInputs(t *testing.T) {
	if got, want := animationReferenceCanvasSizeForLongEdge(0), image.Pt(animationReferenceSize, animationReferenceSize); got != want {
		t.Fatalf("fallback reference canvas = %v, want %v", got, want)
	}

	if got := animationReferencePrototypeCanvasSize(image.Pt(0, 512), 32, 32, 64, 64); got != (image.Pt(1, 1)) {
		t.Fatalf("invalid canvas = %v, want 1x1", got)
	}
	canvas := animationReferenceCanvasSize()
	if got := animationReferencePrototypeCanvasSize(canvas, 0, 0, 0, 0); got != canvas {
		t.Fatalf("invalid prototype/frame dimensions = %v, want unchanged canvas %v", got, canvas)
	}
}

func TestNormalizeAnimationGenerationRequestValidatesFrameEditContext(t *testing.T) {
	valid := AnimationGenerationRequest{
		ReferenceImage:         "start",
		EndReferenceImage:      "end",
		ReferenceImageContext:  true,
		TargetFrameIndices:     []int{1},
		ContextReferenceImages: []string{"frame-0", "frame-1"},
		FrameCount:             2,
		Columns:                2,
		FrameWidth:             64,
		FrameHeight:            64,
		FPS:                    10,
		Duration:               5,
	}

	result, err := normalizeAnimationGenerationRequest(&valid)
	if err != nil {
		t.Fatalf("normalize valid frame edit request: %v", err)
	}
	for index := range result.ContextReferenceImages {
		result.ContextReferenceImages[index] += " "
	}
	if !reflect.DeepEqual(normalizedContext(result.ContextReferenceImages), []string{"frame-0", "frame-1"}) {
		t.Fatalf("context references were not trimmed: %v", result.ContextReferenceImages)
	}

	tests := []struct {
		name   string
		mutate func(*AnimationGenerationRequest)
		want   string
	}{
		{
			name: "missing end reference",
			mutate: func(value *AnimationGenerationRequest) {
				value.EndReferenceImage = " "
			},
			want: "end reference image is required",
		},
		{
			name: "wrong context count",
			mutate: func(value *AnimationGenerationRequest) {
				value.ContextReferenceImages = value.ContextReferenceImages[:1]
			},
			want: "requires 2 context reference images; got 1",
		},
		{
			name: "empty context reference",
			mutate: func(value *AnimationGenerationRequest) {
				value.ContextReferenceImages[1] = " "
			},
			want: "context reference image 2 is required",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := valid
			test.mutate(&value)
			_, err := normalizeAnimationGenerationRequest(&value)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func normalizedContext(values []string) []string {
	trimmed := make([]string, len(values))
	for index, value := range values {
		trimmed[index] = strings.TrimSpace(value)
	}
	return trimmed
}

func TestAnimationReferencePrototypeCanvasSizeRejectsNonPositiveOutputScale(t *testing.T) {
	canvas := image.Pt(1024, 1024)
	if got, want := animationReferencePrototypeCanvasSize(canvas, 32, 32, -64, 64), canvas; got != want {
		t.Fatalf("non-positive output scale returned %v, want unchanged canvas %v", got, want)
	}
}
