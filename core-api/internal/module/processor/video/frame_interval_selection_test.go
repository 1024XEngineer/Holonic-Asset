package video

import (
	"image"
	"math"
	"strings"
	"testing"
)

func TestSelectFrameIntervalPrefersFirstFrameWhenValid(t *testing.T) {
	analysis := uniformFrameSequenceAnalysis(30)
	selected, err := SelectFrameInterval(analysis, testFrameIntervalSelectionOptions(8))
	if err != nil {
		t.Fatal(err)
	}
	if selected.StartFrame != 0 || selected.Indices[0] != 0 {
		t.Fatalf("valid first frame was not preferred: %+v", selected)
	}
	if selected.EndFrame != 15 {
		t.Fatalf("unexpected compact interval: %+v", selected)
	}
}

func TestSelectFrameIntervalLimitsStartToConfiguredWindow(t *testing.T) {
	analysis := uniformFrameSequenceAnalysis(30)
	analysis.Frames[0].Safe = false
	options := testFrameIntervalSelectionOptions(8)
	selected, err := SelectFrameInterval(analysis, options)
	if err != nil {
		t.Fatal(err)
	}
	maxStart := 6 // ceil(30 * options.StartWindowRatio)
	if selected.StartFrame < 1 || selected.StartFrame > maxStart || selected.Indices[0] != selected.StartFrame {
		t.Fatalf("selected start escaped configured window: %+v", selected)
	}
}

func TestSelectFrameIntervalRejectsInvalidInput(t *testing.T) {
	analysis := uniformFrameSequenceAnalysis(10)
	tests := []struct {
		name   string
		mutate func(*FrameSequenceAnalysis, *FrameIntervalSelectionOptions)
		want   string
	}{
		{
			name: "matrix shape",
			mutate: func(value *FrameSequenceAnalysis, _ *FrameIntervalSelectionOptions) {
				value.PairwiseMSE = value.PairwiseMSE[:len(value.PairwiseMSE)-1]
			},
			want: "matrix must match frame count",
		},
		{
			name: "invalid ratio",
			mutate: func(_ *FrameSequenceAnalysis, options *FrameIntervalSelectionOptions) {
				options.StartWindowRatio = 1.1
			},
			want: "start window ratio must be between 0 and 1",
		},
		{
			name: "invalid weight",
			mutate: func(_ *FrameSequenceAnalysis, options *FrameIntervalSelectionOptions) {
				options.Weights.EndpointSimilarity = math.NaN()
			},
			want: "endpoint similarity weight must be finite",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := analysis
			options := testFrameIntervalSelectionOptions(4)
			test.mutate(&input, &options)
			_, err := SelectFrameInterval(input, options)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestSelectFrameIntervalCoversConfiguredLongSpan(t *testing.T) {
	frames := make([]image.Image, 48)
	for index := range frames {
		frame := newFrameIntervalTestFrame()
		var movingRegion image.Rectangle
		switch {
		case index < 16:
			x := 26 + min(index%8, 8-index%8)*3
			movingRegion = image.Rect(x, 42, x+42, 49)
		case index < 30:
			y := 42 + (index-16)*3
			movingRegion = image.Rect(35, y, 88, y+7)
		default:
			y := 84 - (index-30)*2
			movingRegion = image.Rect(35, y, 88, y+7)
		}
		drawSubject(frame, movingRegion)
		frames[index] = frame
	}

	options := testFrameIntervalSelectionOptions(16)
	selected, err := SelectFrameInterval(frameSequenceAnalysisFromImages(frames), options)
	if err != nil {
		t.Fatal(err)
	}
	if selected.SpanRatio < options.MinimumSpanRatio ||
		selected.StartFrame > int(math.Ceil(float64(len(frames))*options.StartWindowRatio)) {
		t.Fatalf("selected interval omitted configured coverage: %+v", selected)
	}
	if len(selected.Indices) != options.SampleCount {
		t.Fatalf("selected %d frames; want %d", len(selected.Indices), options.SampleCount)
	}
}

func TestSelectFrameIntervalSkipsUnsafeSampledFrames(t *testing.T) {
	analysis := uniformFrameSequenceAnalysis(48)
	for _, index := range []int{10, 30} {
		analysis.Frames[index].Safe = false
	}
	options := testFrameIntervalSelectionOptions(8)
	selected, err := SelectFrameInterval(analysis, options)
	if err != nil {
		t.Fatal(err)
	}
	if selected.SpanRatio < options.MinimumSpanRatio {
		t.Fatalf("selected interval is too short: %+v", selected)
	}
	for _, index := range selected.Indices {
		if !analysis.Frames[index].Safe {
			t.Fatalf("unsafe frame %d was selected: %+v", index, selected)
		}
	}
}

func TestSelectFrameIntervalExcludesStableTail(t *testing.T) {
	const total = 60
	frames := make([]image.Image, total)
	for index := range frames {
		frame := newFrameIntervalTestFrame()
		regionX := 35
		var regionY int
		switch {
		case index < 10:
			regionY = 42 + index*2
		case index < 26:
			regionX = 35 + (index-10)*2
			regionY = 60 + (index-10)*2
		case index < 35:
			regionX = 65 - (index-26)*3
			regionY = 90 - (index-26)*4
		case index < 43:
			regionX = 38 - (index - 35)
			regionY = 54 - (index - 35)
		default:
			regionX, regionY = 35, 42
		}
		drawSubject(frame, image.Rect(regionX, regionY, regionX+42, regionY+7))
		frames[index] = frame
	}

	selected, err := SelectFrameInterval(
		frameSequenceAnalysisFromImages(frames),
		testFrameIntervalSelectionOptions(8),
	)
	if err != nil {
		t.Fatal(err)
	}
	if selected.StartFrame != 0 || selected.EndFrame < 40 || selected.EndFrame > 45 {
		t.Fatalf("selector did not exclude stable tail: %+v", selected)
	}
}

func testFrameIntervalSelectionOptions(sampleCount int) FrameIntervalSelectionOptions {
	return FrameIntervalSelectionOptions{
		SampleCount: sampleCount, MinimumSpanFrames: 4, MinimumSpanRatio: .50,
		MinimumStartWindowFrames: 1, StartWindowRatio: .20,
		PreferFirstFrame: true, MinimumForegroundRatio: .05,
		EndpointMSEQuantile: .35, ChangeScaleQuantile: .90, ChangeBaselineQuantile: .25,
		Weights: FrameIntervalSelectionWeights{
			EndpointSimilarity: 1, MeanAdjacentMSE: .45, CentroidStability: .45,
			LinearCentroidMotion: .20, FirstFrameSimilarity: .45, Compactness: 1.15,
			GeometryCoverage: .65, ChangeCoverage: .65, PostIntervalStability: .35,
		},
	}
}

func frameSequenceAnalysisFromImages(frames []image.Image) FrameSequenceAnalysis {
	analyses := make([]frameAnalysis, 0, len(frames))
	for _, frame := range frames {
		analyses = append(analyses, frameAnalysis{
			descriptor: describeFrame(frame, testGreenChromaKey),
			safe:       frameInsideSafetyBand(frame, testGreenChromaKey),
		})
	}
	return buildFrameSequenceAnalysis(analyses, 12)
}

func newFrameIntervalTestFrame() *image.NRGBA {
	frame := testGreenFrame(120, 120)
	drawSubject(frame, image.Rect(49, 24, 69, 104))
	return frame
}

func uniformFrameSequenceAnalysis(count int) FrameSequenceAnalysis {
	frames := make([]FrameObservation, count)
	pairwise := make([][]float64, count)
	for index := range frames {
		frames[index] = FrameObservation{
			Safe: true, CentroidX: 24, CentroidY: 24,
			Width: 20, Height: 32, ForegroundArea: 640,
		}
		pairwise[index] = make([]float64, count)
	}
	return FrameSequenceAnalysis{
		FPS: 12, Frames: frames, PairwiseMSE: pairwise, ForegroundRatio: .25,
	}
}
