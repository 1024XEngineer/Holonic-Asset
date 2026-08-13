package generator

import (
	"encoding/json"
	"strings"
	"testing"

	videoprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/video"
)

func TestAnimationFrameSelectionOptionsOwnBusinessPolicy(t *testing.T) {
	options := animationFrameSelectionOptions(16)
	if options.SampleCount != 16 ||
		options.MinimumSpanFrames != animationMinLoopSpanFrames ||
		options.MinimumSpanRatio != animationMinLoopSpanRatio ||
		options.MinimumStartWindowFrames != animationMinStartWindow ||
		options.StartWindowRatio != animationInitialWindowRatio ||
		!options.PreferFirstFrame ||
		options.MinimumForegroundRatio != animationMinForegroundRatio {
		t.Fatalf("unexpected animation frame selection constraints: %+v", options)
	}
	weights := options.Weights
	if weights.EndpointSimilarity != animationEndpointWeight ||
		weights.MeanAdjacentMSE != animationRichnessWeight ||
		weights.CentroidStability != animationCentroidStabilityWeight ||
		weights.LinearCentroidMotion != animationTranslationWeight ||
		weights.FirstFrameSimilarity != animationInitialFrameWeight ||
		weights.Compactness != animationLoopCompactnessWeight ||
		weights.GeometryCoverage != animationPoseCoverageWeight ||
		weights.ChangeCoverage != animationMotionCoverageWeight ||
		weights.PostIntervalStability != animationRecoveryWeight {
		t.Fatalf("unexpected animation frame selection weights: %+v", weights)
	}
}

func TestAnimationLoopSelectionMapsMediaMeasurements(t *testing.T) {
	loop := animationLoopSelection(videoprocessor.FrameIntervalSelection{
		StartFrame: 2, EndFrame: 18, Score: 1.1234567,
		EndpointSimilarity: .9876543, MeanAdjacentMSE: .1234567,
		GeometryCoverage: .7654321, SpanRatio: .5000004,
		CentroidStability: .8765432, EndpointMSE: animationSeamWarningMSE + .001,
	}, 12)
	if loop.CandidateFPS != 12 || loop.StartFrame != 2 || loop.EndFrame != 18 || loop.SpanFrames != 16 {
		t.Fatalf("unexpected animation loop boundaries: %+v", loop)
	}
	if loop.Score != 1.123457 || loop.Richness != .123457 || loop.PoseCoverage != .765432 {
		t.Fatalf("unexpected mapped animation metrics: %+v", loop)
	}
	if loop.Method != "subject_mse_full_cycle" || loop.SeamWarning == "" {
		t.Fatalf("unexpected animation loop metadata: %+v", loop)
	}
}

func TestAnimationDirectionIndexUsesAssetDirectionLayout(t *testing.T) {
	tests := []struct {
		name           string
		direction      string
		directionCount uint
		want           int
	}{
		{name: "two left", direction: AnimationDirectionLeft, directionCount: 2, want: 0},
		{name: "two right", direction: AnimationDirectionRight, directionCount: 2, want: 1},
		{name: "four right", direction: AnimationDirectionRight, directionCount: 4, want: 1},
		{name: "four left", direction: AnimationDirectionLeft, directionCount: 4, want: 3},
		{name: "eight front right", direction: AnimationDirectionFrontRight, directionCount: 8, want: 1},
		{name: "eight back right", direction: AnimationDirectionBackRight, directionCount: 8, want: 3},
		{name: "eight front left", direction: AnimationDirectionFrontLeft, directionCount: 8, want: 7},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := animationDirectionIndex(test.direction, test.directionCount)
			if err != nil {
				t.Fatalf("resolve direction: %v", err)
			}
			if got != test.want {
				t.Fatalf("index = %d, want %d", got, test.want)
			}
		})
	}
}

func TestAnimationDirectionIndexRejectsInvalidLayoutsAndNames(t *testing.T) {
	tests := []struct {
		name           string
		direction      string
		directionCount uint
		want           string
	}{
		{name: "missing multi direction", directionCount: 8, want: "direction is required"},
		{name: "diagonal unavailable in four directions", direction: AnimationDirectionFrontRight, directionCount: 4, want: "is unavailable"},
		{name: "unknown direction", direction: "up", directionCount: 8, want: "is unavailable"},
		{name: "more than eight", direction: AnimationDirectionFront, directionCount: 9, want: "at most 8 directions"},
		{name: "single direction unsupported", direction: AnimationDirectionFront, directionCount: 1, want: "must be one of 2, 4, or 8"},
		{name: "unsupported count", direction: AnimationDirectionFront, directionCount: 3, want: "must be one of 2, 4, or 8"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := animationDirectionIndex(test.direction, test.directionCount)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestCreateAnimationPayloadRejectsNumericDirection(t *testing.T) {
	var payload CreateAnimationPayload
	err := json.Unmarshal([]byte(`{"direction":3}`), &payload)
	if err == nil {
		t.Fatal("numeric animation direction should be rejected")
	}
}

func TestAnimationUnprocessedImageURLAddsSuffixWithoutChangingReferenceSemantics(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "URL with query",
			value: "https://cdn.example.com/hero/front.png?version=7",
			want:  "https://cdn.example.com/hero/front-unprocessed.png?version=7",
		},
		{name: "object key", value: "uploads/hero/front.png", want: "uploads/hero/front-unprocessed.png"},
		{name: "no extension", value: "uploads/hero/front", want: "uploads/hero/front-unprocessed"},
		{name: "data URL", value: "data:image/png;base64,parent", want: "data:image/png;base64,parent"},
		{name: "blank", value: "  ", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := animationUnprocessedImageURL(test.value); got != test.want {
				t.Fatalf("unprocessed URL = %q, want %q", got, test.want)
			}
		})
	}
}
