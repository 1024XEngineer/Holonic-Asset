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

func TestAnalyzeFramePairsMeasuresMaskAndAppearanceChanges(t *testing.T) {
	original := testGreenFrame(96, 96)
	drawSubject(original, image.Rect(30, 18, 66, 88))

	shifted := testGreenFrame(96, 96)
	drawSubject(shifted, image.Rect(36, 18, 72, 88))
	differences, err := AnalyzeFramePairs(
		[]image.Image{original, original},
		[]image.Image{original, shifted},
		testGreenChromaKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	if differences[0].ForegroundMaskDifference != 0 || differences[0].AppearanceMSE != 0 {
		t.Fatalf("identical frame difference = %+v, want zero", differences[0])
	}
	if differences[1].ForegroundMaskDifference <= 0 || differences[1].AppearanceMSE <= 0 {
		t.Fatalf("shifted frame difference = %+v, want positive mask and appearance changes", differences[1])
	}

	recoloured := testGreenFrame(96, 96)
	draw.Draw(recoloured, image.Rect(30, 18, 66, 88), &image.Uniform{C: color.NRGBA{R: 230, G: 210, B: 40, A: 255}}, image.Point{}, draw.Src)
	differences, err = AnalyzeFramePairs([]image.Image{original}, []image.Image{recoloured}, testGreenChromaKey)
	if err != nil {
		t.Fatal(err)
	}
	if differences[0].ForegroundMaskDifference != 0 || differences[0].AppearanceMSE <= 0 {
		t.Fatalf("recoloured frame difference = %+v, want appearance-only change", differences[0])
	}
}

func TestAnalyzeFramePairsValidatesSequences(t *testing.T) {
	frame := testGreenFrame(96, 96)
	if _, err := AnalyzeFramePairs(nil, nil, testGreenChromaKey); err == nil {
		t.Fatal("expected empty original sequence to be rejected")
	}
	if _, err := AnalyzeFramePairs([]image.Image{frame}, nil, testGreenChromaKey); err == nil {
		t.Fatal("expected mismatched frame pair lengths to be rejected")
	}
}

func TestAutoDetectedChromaKeyFollowsFrameMatte(t *testing.T) {
	base := ChromaKey{
		HueMin: 30, HueMax: 90,
		HighSaturationMin: 80, HighValueMin: 80,
		BrightSaturationMin: 50, BrightValueMin: 180,
		AutoDetect: true,
	}
	tests := []struct {
		name       string
		background color.NRGBA
		wantHue    uint8
		wantPixels int
	}{
		{name: "green", background: color.NRGBA{G: 255, A: 255}, wantHue: 60, wantPixels: 18 * 35},
		{name: "magenta", background: color.NRGBA{R: 255, B: 255, A: 255}, wantHue: 150, wantPixels: 18 * 35},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			frame := image.NewNRGBA(image.Rect(0, 0, 96, 96))
			draw.Draw(frame, frame.Bounds(), &image.Uniform{C: test.background}, image.Point{}, draw.Src)
			drawSubject(frame, image.Rect(30, 13, 66, 83))

			resolved := resolveAutoDetectedChromaKey([]image.Image{frame}, base)
			if resolved.AutoDetect {
				t.Fatal("auto-detect flag should be cleared after resolving the matte")
			}
			if resolved.HueMin > test.wantHue || resolved.HueMax < test.wantHue {
				t.Fatalf("resolved hue range = %d..%d, want to contain %d", resolved.HueMin, resolved.HueMax, test.wantHue)
			}
			descriptor := describeFrame(frame, resolved)
			if descriptor.foreground != test.wantPixels {
				t.Fatalf("foreground pixels = %d, want %d", descriptor.foreground, test.wantPixels)
			}
		})
	}
}

func TestAutoDetectedChromaKeyAllowsDifferentMattesInFramePairs(t *testing.T) {
	original := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	draw.Draw(original, original.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	drawSubject(original, image.Rect(30, 18, 66, 88))
	candidate := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	draw.Draw(candidate, candidate.Bounds(), &image.Uniform{C: color.NRGBA{R: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	drawSubject(candidate, image.Rect(30, 18, 66, 88))

	key := testGreenChromaKey
	key.AutoDetect = true
	differences, err := AnalyzeFramePairs([]image.Image{original}, []image.Image{candidate}, key)
	if err != nil {
		t.Fatal(err)
	}
	if differences[0].ForegroundMaskDifference != 0 || differences[0].AppearanceMSE != 0 {
		t.Fatalf("different mattes should be ignored: %+v", differences[0])
	}
}
