package generator

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"testing"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	videoprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/video"
)

func subjectScaleTestFrame(size image.Point, effectWidth int) image.Image {
	return subjectScaleTestFrameOnCanvas(image.Pt(192, 192), size, effectWidth)
}

func subjectScaleTestFrameOnCanvas(canvasSize, size image.Point, effectWidth int) image.Image {
	frame := image.NewNRGBA(image.Rectangle{Max: canvasSize})
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	min := image.Pt((canvasSize.X-size.X)/2, (canvasSize.Y-size.Y)/2)
	draw.Draw(frame, image.Rectangle{Min: min, Max: min.Add(size)}, &image.Uniform{C: color.NRGBA{R: 140, G: 50, B: 35, A: 255}}, image.Point{}, draw.Src)
	if effectWidth > 0 {
		left := image.Pt(min.X-effectWidth, min.Y+size.Y/2)
		right := image.Pt(max(left.X, min.X+size.X), left.Y)
		effect := image.Rect(left.X, left.Y-3, right.X+effectWidth+8, left.Y+3)
		draw.Draw(frame, effect, &image.Uniform{C: color.NRGBA{R: 30, G: 180, B: 240, A: 255}}, image.Point{}, draw.Src)
	}
	return frame
}

func subjectScaleTestFrameWithVerticalEffect(size image.Point, effectWidth, effectHeight int) image.Image {
	frame := image.NewNRGBA(image.Rect(0, 0, 192, 192))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	min := image.Pt((192-size.X)/2, (192-size.Y)/2)
	draw.Draw(frame, image.Rectangle{Min: min, Max: min.Add(size)}, &image.Uniform{C: color.NRGBA{R: 140, G: 50, B: 35, A: 255}}, image.Point{}, draw.Src)
	if effectWidth > 0 || effectHeight > 0 {
		left := image.Pt(min.X-effectWidth, min.Y-effectHeight)
		right := image.Pt(max(left.X, min.X+size.X+effectWidth), min.Y+size.Y+effectHeight)
		effect := image.Rect(left.X, left.Y, right.X, right.Y)
		draw.Draw(frame, effect, &image.Uniform{C: color.NRGBA{R: 30, G: 180, B: 240, A: 255}}, image.Point{}, draw.Src)
	}
	return frame
}

func TestAnimationSubjectScaleMultiplierIgnoresNormalAndWideEffects(t *testing.T) {
	key := animationVideoChromaKey()
	key.AutoDetect = false
	reference := subjectScaleTestFrame(image.Pt(60, 80), 0)

	tests := []struct {
		name   string
		frames []image.Image
		want   float64
	}{
		{name: "matching height", frames: []image.Image{
			subjectScaleTestFrame(image.Pt(60, 79), 0),
			subjectScaleTestFrame(image.Pt(90, 81), 20),
		}, want: 1},
		{name: "uniform shrink", frames: []image.Image{
			subjectScaleTestFrame(image.Pt(45, 60), 50),
			subjectScaleTestFrame(image.Pt(55, 60), 35),
		}, want: 80.0 / 60.0},
		{name: "2x shrink from prompt", frames: []image.Image{
			subjectScaleTestFrame(image.Pt(30, 40), 0),
			subjectScaleTestFrame(image.Pt(30, 40), 10),
			subjectScaleTestFrame(image.Pt(30, 40), 20),
		}, want: 2.0},
		{name: "spray and floating bubbles do not pollute subject height", frames: []image.Image{
			subjectScaleTestFrame(image.Pt(30, 40), 0), // clean initial frame
			subjectScaleTestFrameWithVerticalEffect(image.Pt(30, 40), 40, 20),
			subjectScaleTestFrameWithVerticalEffect(image.Pt(30, 40), 50, 30),
		}, want: 2.0},
		{name: "character enlarged in video", frames: []image.Image{
			subjectScaleTestFrame(image.Pt(90, 120), 0),
			subjectScaleTestFrame(image.Pt(90, 120), 0),
		}, want: 80.0 / 120.0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := animationSubjectScaleMultiplier(reference, test.frames, key)
			if err != nil {
				t.Fatalf("measure subject scale: %v", err)
			}
			if math.Abs(got-test.want) > .01 {
				t.Fatalf("multiplier = %f, want %f", got, test.want)
			}
		})
	}
}

func TestAnimationSubjectScaleMultiplierClampsExtremeShrink(t *testing.T) {
	key := animationVideoChromaKey()
	key.AutoDetect = false
	reference := subjectScaleTestFrame(image.Pt(60, 100), 0)
	got, err := animationSubjectScaleMultiplier(reference, []image.Image{
		subjectScaleTestFrame(image.Pt(10, 15), 0),
		subjectScaleTestFrame(image.Pt(10, 15), 0),
	}, key)
	if err != nil {
		t.Fatalf("measure subject scale: %v", err)
	}
	if math.Abs(got-animationSubjectScaleMaxMultiplier) > .0001 {
		t.Fatalf("multiplier = %f, want clamped %f", got, animationSubjectScaleMaxMultiplier)
	}
}

func TestAnimationSubjectScaleMultiplierNormalizesDifferentCanvasSizes(t *testing.T) {
	key := animationVideoChromaKey()
	key.AutoDetect = false
	reference := subjectScaleTestFrameOnCanvas(image.Pt(1920, 1920), image.Pt(535, 741), 0)
	frames := []image.Image{
		subjectScaleTestFrameOnCanvas(image.Pt(720, 720), image.Pt(268, 372), 0),
		subjectScaleTestFrameOnCanvas(image.Pt(720, 720), image.Pt(268, 371), 0),
		subjectScaleTestFrameOnCanvas(image.Pt(720, 720), image.Pt(490, 371), 80),
	}

	got, err := animationSubjectScaleMultiplier(reference, frames, key)
	if err != nil {
		t.Fatalf("measure subject scale: %v", err)
	}
	want := (741.0 / 1920.0) / (371.5 / 720.0)
	if math.Abs(got-want) > .01 {
		t.Fatalf("multiplier = %f, want normalized %f", got, want)
	}
}

func TestProcessAnimationVideoCompensatesUniformSubjectShrink(t *testing.T) {
	videoProcessor := &animationVideoProcessorStub{results: []*videoprocessor.Result{{Frames: []image.Image{
		subjectScaleTestFrame(image.Pt(45, 60), 0),
		subjectScaleTestFrame(image.Pt(45, 60), 0),
		subjectScaleTestFrame(image.Pt(45, 61), 0),
	}}}}
	service := &animationGenerationService{
		processor:      imageprocessor.NewProcessor(),
		videoProcessor: videoProcessor,
	}
	referenceData, err := imageprocessor.EncodePNGBase64(subjectScaleTestFrame(image.Pt(60, 80), 0))
	if err != nil {
		t.Fatalf("encode subject-scale reference: %v", err)
	}
	result, err := service.processVideoWithSourceCellScale(
		context.Background(), []byte("video"), "data:image/png;base64,"+referenceData, AnimationGenerationRequest{
			Action: "spray detergent forward attack", FrameCount: 3, Columns: 3,
			FrameWidth: 64, FrameHeight: 64,
		}, 1,
	)
	if err != nil {
		t.Fatalf("process shrunk video: %v", err)
	}
	report := result.Normalization
	if report == nil || math.Abs(report.RequestedSourceCellScaleMultiplier-80.0/60.0) > .0001 ||
		math.Abs(report.AppliedSourceCellScaleMultiplier-80.0/60.0) > .02 {
		t.Fatalf("shrink compensation not applied: requested=%f applied=%f warnings=%v",
			report.RequestedSourceCellScaleMultiplier, report.AppliedSourceCellScaleMultiplier, report.Warnings)
	}
}

func TestAnimationSubjectScaleMultiplierRejectsInvalidInputs(t *testing.T) {
	key := animationVideoChromaKey()
	key.AutoDetect = false
	validReference := subjectScaleTestFrame(image.Pt(60, 80), 0)
	validFrame := subjectScaleTestFrame(image.Pt(45, 60), 0)
	emptyImage := image.NewNRGBA(image.Rectangle{})
	allGreen := image.NewNRGBA(image.Rect(0, 0, 192, 192))
	draw.Draw(allGreen, allGreen.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)

	tests := []struct {
		name      string
		reference image.Image
		frames    []image.Image
		want      string
	}{
		{name: "nil reference", reference: nil, frames: []image.Image{validFrame}, want: "scale reference is empty"},
		{name: "empty reference", reference: emptyImage, frames: []image.Image{validFrame}, want: "scale reference is empty"},
		{name: "no frames", reference: validReference, frames: nil, want: "frames are required"},
		{name: "reference without foreground", reference: allGreen, frames: []image.Image{validFrame}, want: "reference has no detectable foreground"},
		{name: "nil frame", reference: validReference, frames: []image.Image{nil}, want: "frame 1 is empty"},
		{name: "empty frame", reference: validReference, frames: []image.Image{emptyImage}, want: "frame 1 is empty"},
		{name: "frames without foreground", reference: validReference, frames: []image.Image{allGreen}, want: "no measurable foreground"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := animationSubjectScaleMultiplier(test.reference, test.frames, key)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestProcessAnimationVideoRejectsInvalidReferenceBase64(t *testing.T) {
	videoProcessor := &animationVideoProcessorStub{results: []*videoprocessor.Result{{Frames: []image.Image{
		subjectScaleTestFrame(image.Pt(45, 60), 0),
		subjectScaleTestFrame(image.Pt(45, 60), 0),
	}}}}
	service := &animationGenerationService{
		processor:      imageprocessor.NewProcessor(),
		videoProcessor: videoProcessor,
	}
	_, err := service.processVideoWithSourceCellScale(
		context.Background(), []byte("video"), "not-valid-base64", AnimationGenerationRequest{
			Action: "spray detergent forward attack", FrameCount: 2, Columns: 2,
			FrameWidth: 64, FrameHeight: 64,
		}, 1,
	)
	if err == nil || !strings.Contains(err.Error(), "decode animation subject-scale reference") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestProcessAnimationVideoRejectsReferenceWithoutForeground(t *testing.T) {
	videoProcessor := &animationVideoProcessorStub{results: []*videoprocessor.Result{{Frames: []image.Image{
		subjectScaleTestFrame(image.Pt(45, 60), 0),
		subjectScaleTestFrame(image.Pt(45, 60), 0),
	}}}}
	allGreen := image.NewNRGBA(image.Rect(0, 0, 192, 192))
	draw.Draw(allGreen, allGreen.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	referenceData, err := imageprocessor.EncodePNGBase64(allGreen)
	if err != nil {
		t.Fatalf("encode reference: %v", err)
	}
	service := &animationGenerationService{
		processor:      imageprocessor.NewProcessor(),
		videoProcessor: videoProcessor,
	}
	_, err = service.processVideoWithSourceCellScale(
		context.Background(), []byte("video"), "data:image/png;base64,"+referenceData, AnimationGenerationRequest{
			Action: "spray detergent forward attack", FrameCount: 2, Columns: 2,
			FrameWidth: 64, FrameHeight: 64,
		}, 1,
	)
	if err == nil || !strings.Contains(err.Error(), "no detectable foreground") {
		t.Fatalf("expected measure error, got %v", err)
	}
}

func subjectScaleTestFrameOnMatte(canvasSize, size image.Point, matte color.NRGBA) image.Image {
	frame := image.NewNRGBA(image.Rectangle{Max: canvasSize})
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: matte}, image.Point{}, draw.Src)
	min := image.Pt((canvasSize.X-size.X)/2, (canvasSize.Y-size.Y)/2)
	draw.Draw(frame, image.Rectangle{Min: min, Max: min.Add(size)}, &image.Uniform{C: color.NRGBA{R: 140, G: 50, B: 35, A: 255}}, image.Point{}, draw.Src)
	return frame
}

func TestAnimationSubjectScaleMultiplierResolvesNonGreenMatte(t *testing.T) {
	key := animationVideoChromaKey() // AutoDetect enabled
	reference := subjectScaleTestFrame(image.Pt(60, 80), 0)
	blueMatte := color.NRGBA{B: 255, A: 255}
	frames := []image.Image{
		subjectScaleTestFrameOnMatte(image.Pt(192, 192), image.Pt(60, 80), blueMatte),
		subjectScaleTestFrameOnMatte(image.Pt(192, 192), image.Pt(60, 80), blueMatte),
	}

	got, err := animationSubjectScaleMultiplier(reference, frames, key)
	if err != nil {
		t.Fatalf("measure subject scale on non-green matte: %v", err)
	}
	if math.Abs(got-1.0) > .01 {
		t.Fatalf("multiplier = %f, want ~1.0 for matching subject on non-green matte", got)
	}
}
