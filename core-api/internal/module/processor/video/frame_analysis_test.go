package video

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"testing"
)

var testGreenChromaKey = ChromaKey{
	HueMin: 30, HueMax: 90,
	HighSaturationMin: 80, HighValueMin: 80,
	BrightSaturationMin: 50, BrightValueMin: 180,
}

type frameExtractorStub struct {
	frames []image.Image
	err    error
}

func (s frameExtractorStub) Extract(
	_ context.Context,
	_ []byte,
	fps int,
	chromaKey ChromaKey,
	selectFrames FrameSelector,
) ([]image.Image, []int, error) {
	if s.err != nil {
		return nil, nil, s.err
	}
	analyses := make([]frameAnalysis, 0, len(s.frames))
	for _, frame := range s.frames {
		analyses = append(analyses, frameAnalysis{
			descriptor: describeFrame(frame, chromaKey),
			safe:       frameInsideSafetyBand(frame, chromaKey),
		})
	}
	indices, err := selectFrames(buildFrameSequenceAnalysis(analyses, fps))
	if err != nil {
		return nil, nil, err
	}
	selected := make([]image.Image, 0, len(indices))
	for _, index := range indices {
		selected = append(selected, s.frames[index])
	}
	return selected, indices, nil
}

func TestProcessorSuppliesMediaAnalysisToCallerSelector(t *testing.T) {
	frames := testVideoFrames(6)
	var received FrameSequenceAnalysis
	result, err := newProcessor(frameExtractorStub{frames: frames}).Process(
		context.Background(),
		[]byte("video"),
		ProcessOptions{
			AnalysisFPS: 12,
			ChromaKey:   testGreenChromaKey,
			Select: func(analysis FrameSequenceAnalysis) ([]int, error) {
				received = analysis
				return []int{1, 4}, nil
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if received.FPS != 12 || len(received.Frames) != len(frames) || len(received.PairwiseMSE) != len(frames) {
		t.Fatalf("unexpected analysis: %+v", received)
	}
	if received.ForegroundRatio <= 0 || received.PairwiseMSE[0][1] < 0 {
		t.Fatalf("missing foreground measurements: %+v", received)
	}
	if len(result.Frames) != 2 || result.SourceIndices[0] != 1 || result.SourceIndices[1] != 4 {
		t.Fatalf("unexpected selected result: %+v", result)
	}
}

func testVideoFrames(count int) []image.Image {
	frames := make([]image.Image, count)
	for index := range frames {
		frame := testGreenFrame(96, 96)
		drawSubject(frame, image.Rect(30+index%3, 18, 66+index%3, 88))
		frames[index] = frame
	}
	return frames
}

func testGreenFrame(width, height int) *image.NRGBA {
	frame := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	return frame
}

func drawSubject(frame draw.Image, bounds image.Rectangle) {
	draw.Draw(frame, bounds, &image.Uniform{C: color.NRGBA{R: 105, G: 50, B: 32, A: 255}}, image.Point{}, draw.Src)
}

var _ frameExtractor = frameExtractorStub{}
