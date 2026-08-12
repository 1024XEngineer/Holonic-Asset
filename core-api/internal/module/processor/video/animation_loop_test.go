package video

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"math"
	"testing"
)

type frameExtractorStub struct {
	frames []image.Image
	err    error
}

func (s frameExtractorStub) Extract(context.Context, []byte, int) ([]image.Image, error) {
	return s.frames, s.err
}

func TestProcessorSelectsCompleteLongCycle(t *testing.T) {
	frames := make([]image.Image, 48)
	for index := range frames {
		frame := newAnimationTestFrame()
		var prop image.Rectangle
		switch {
		case index < 16:
			x := 26 + animationMinInt(index%8, 8-index%8)*3
			prop = image.Rect(x, 42, x+42, 49)
		case index < 30:
			y := 42 + (index-16)*3
			prop = image.Rect(35, y, 88, y+7)
		default:
			y := 84 - (index-30)*2
			prop = image.Rect(35, y, 88, y+7)
		}
		drawSubject(frame, prop)
		frames[index] = frame
	}

	result, err := newProcessor(frameExtractorStub{frames: frames}).Process(context.Background(), []byte("video"), 16)
	if err != nil {
		t.Fatal(err)
	}
	loop := result.Loop
	if loop.SpanRatio < animationMinLoopSpanRatio || loop.StartFrame > int(math.Ceil(float64(len(frames))*animationInitialWindowRatio)) {
		t.Fatalf("selector omitted part of the complete cycle: loop=%+v", loop)
	}
	if loop.Method != "subject_mse_full_cycle" || len(result.Frames) != 16 {
		t.Fatalf("unexpected processed result: %+v", result)
	}
}

func TestProcessorSkipsTransientUnsafeSourceFrames(t *testing.T) {
	frames := animationTestVideoFrames(48)
	unsafeFrames := map[image.Image]bool{}
	for _, index := range []int{10, 30} {
		frame := frames[index].(*image.NRGBA)
		drawSubject(frame, image.Rect(0, 40, 12, 48))
		unsafeFrames[frame] = true
	}

	result, err := newProcessor(frameExtractorStub{frames: frames}).Process(context.Background(), []byte("video"), 8)
	if err != nil {
		t.Fatalf("process video with transient unsafe source frames: %v", err)
	}
	if result.Loop.SpanRatio < animationMinLoopSpanRatio {
		t.Fatalf("selected interval is too short: %+v", result.Loop)
	}
	for _, frame := range result.Frames {
		if unsafeFrames[frame] {
			t.Fatal("unsafe source frame was selected")
		}
	}
}

func TestProcessorDoesNotSampleIdleTail(t *testing.T) {
	const total = 60
	frames := make([]image.Image, total)
	for index := range frames {
		frame := newAnimationTestFrame()
		propX, propY := 35, 42
		switch {
		case index < 10:
			propY = 42 + index*2
		case index < 26:
			propX = 35 + (index-10)*2
			propY = 60 + (index-10)*2
		case index < 35:
			propX = 65 - (index-26)*3
			propY = 90 - (index-26)*4
		case index < 43:
			propX = 38 - (index - 35)
			propY = 54 - (index - 35)
		}
		if index >= 43 {
			propX, propY = 35, 42
		}
		drawSubject(frame, image.Rect(propX, propY, propX+42, propY+7))
		frames[index] = frame
	}

	result, err := newProcessor(frameExtractorStub{frames: frames}).Process(context.Background(), []byte("video"), 8)
	if err != nil {
		t.Fatalf("process video: %v", err)
	}
	if result.Loop.StartFrame != 0 || result.Loop.EndFrame < 40 || result.Loop.EndFrame > 45 {
		t.Fatalf("selector should include recovery and exclude idle tail: %+v", result.Loop)
	}
}

func animationTestVideoFrames(count int) []image.Image {
	frames := make([]image.Image, count)
	for index := range frames {
		frame := image.NewNRGBA(image.Rect(0, 0, 96, 96))
		draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
		offset := index % 3
		drawSubject(frame, image.Rect(30+offset, 18, 66+offset, 88))
		frames[index] = frame
	}
	return frames
}

func newAnimationTestFrame() *image.NRGBA {
	frame := image.NewNRGBA(image.Rect(0, 0, 120, 120))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	drawSubject(frame, image.Rect(49, 24, 69, 104))
	return frame
}

func drawSubject(frame draw.Image, bounds image.Rectangle) {
	draw.Draw(frame, bounds, &image.Uniform{C: color.NRGBA{R: 105, G: 50, B: 32, A: 255}}, image.Point{}, draw.Src)
}

var _ frameExtractor = frameExtractorStub{}
