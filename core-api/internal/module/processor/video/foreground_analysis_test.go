package video

import (
	"errors"
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestValidateFrameBoundsRejectsInvalidSamples(t *testing.T) {
	valid := testGreenFrame(96, 96)
	drawSubject(valid, image.Rect(30, 20, 66, 88))
	empty := testGreenFrame(96, 96)
	clipped := testGreenFrame(96, 96)
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
		{name: "missing foreground", frames: []image.Image{empty}, indices: []int{0}, kind: "foreground"},
		{name: "clipped foreground", frames: []image.Image{clipped}, indices: []int{0}, kind: "framing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateFrameBoundsAtIndices(test.frames, test.indices, testGreenChromaKey)
			if test.kind != "" {
				var qualityErr *QualityError
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
	if err := validateFrameBoundsAtIndices([]image.Image{valid}, []int{0}, testGreenChromaKey); err != nil {
		t.Fatalf("expected safe frame to pass: %v", err)
	}
}

func TestForegroundBoundsHandlesEmptyAndOffsetImages(t *testing.T) {
	if _, ok := foregroundBounds(image.NewNRGBA(image.Rectangle{}), testGreenChromaKey); ok {
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
	bounds, ok := foregroundBounds(frame, testGreenChromaKey)
	if !ok || bounds != image.Rect(22, 30, 35, 45) {
		t.Fatalf("unexpected foreground bounds: %v, ok=%v", bounds, ok)
	}
}
