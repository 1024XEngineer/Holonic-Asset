package video

import (
	"errors"
	"math"
	"testing"
)

func TestValidateFrameIntervalSelectionInputValidationErrors(t *testing.T) {
	t.Parallel()

	analysis := FrameSequenceAnalysis{
		FPS:             12,
		ForegroundRatio: 0.5,
		Frames:          make([]FrameObservation, 4),
		PairwiseMSE: [][]float64{
			{0, 0.1, 0.2, 0.3},
			{0.1, 0, 0.1, 0.2},
			{0.2, 0.1, 0, 0.1},
			{0.3, 0.2, 0.1, 0},
		},
	}
	opts := FrameIntervalSelectionOptions{
		SampleCount:              2,
		MinimumSpanFrames:        1,
		MinimumSpanRatio:         0.2,
		MinimumStartWindowFrames: 1,
		StartWindowRatio:         0.5,
		MinimumForegroundRatio:   0.1,
		EndpointMSEQuantile:      0.8,
		ChangeScaleQuantile:      0.8,
		ChangeBaselineQuantile:   0.2,
	}

	// 1. Negative MinimumSpanFrames / MinimumStartWindowFrames
	negSpanOpts := opts
	negSpanOpts.MinimumSpanFrames = -1
	if err := validateFrameIntervalSelectionInput(analysis, negSpanOpts); err == nil {
		t.Fatal("expected error on negative MinimumSpanFrames")
	}

	negStartOpts := opts
	negStartOpts.MinimumStartWindowFrames = -1
	if err := validateFrameIntervalSelectionInput(analysis, negStartOpts); err == nil {
		t.Fatal("expected error on negative MinimumStartWindowFrames")
	}

	// 2. Invalid float options (NaN, <0, >1)
	for _, invalid := range []float64{-0.1, 1.1, math.NaN()} {
		badRatioOpts := opts
		badRatioOpts.MinimumSpanRatio = invalid
		if err := validateFrameIntervalSelectionInput(analysis, badRatioOpts); err == nil {
			t.Fatalf("expected error for MinimumSpanRatio=%f", invalid)
		}
	}

	// 3. Invalid weights (NaN, Inf)
	badWeightOpts := opts
	badWeightOpts.Weights.EndpointSimilarity = math.NaN()
	if err := validateFrameIntervalSelectionInput(analysis, badWeightOpts); err == nil {
		t.Fatal("expected error on NaN weight")
	}
	badWeightOpts.Weights.EndpointSimilarity = math.Inf(1)
	if err := validateFrameIntervalSelectionInput(analysis, badWeightOpts); err == nil {
		t.Fatal("expected error on Inf weight")
	}

	// 4. Invalid analysis.ForegroundRatio
	badRatioAnalysis := analysis
	badRatioAnalysis.ForegroundRatio = 1.5
	if err := validateFrameIntervalSelectionInput(badRatioAnalysis, opts); err == nil {
		t.Fatal("expected error on ForegroundRatio > 1")
	}

	// 5. Mismatched PairwiseMSE rows/columns/values
	badRowsAnalysis := analysis
	badRowsAnalysis.PairwiseMSE = [][]float64{{0}}
	if err := validateFrameIntervalSelectionInput(badRowsAnalysis, opts); err == nil {
		t.Fatal("expected error on PairwiseMSE row count mismatch")
	}

	badColsAnalysis := analysis
	badColsAnalysis.PairwiseMSE = [][]float64{
		{0, 0.1},
		{0.1, 0},
		{0.2, 0.1},
		{0.3, 0.2},
	}
	if err := validateFrameIntervalSelectionInput(badColsAnalysis, opts); err == nil {
		t.Fatal("expected error on PairwiseMSE col count mismatch")
	}

	badValAnalysis := analysis
	badValAnalysis.PairwiseMSE = [][]float64{
		{0, -0.1, 0.2, 0.3},
		{0.1, 0, 0.1, 0.2},
		{0.2, 0.1, 0, 0.1},
		{0.3, 0.2, 0.1, 0},
	}
	if err := validateFrameIntervalSelectionInput(badValAnalysis, opts); err == nil {
		t.Fatal("expected error on negative PairwiseMSE value")
	}
}

func TestSelectFrameIntervalEdgeCasesAndErrors(t *testing.T) {
	t.Parallel()

	frames := []FrameObservation{
		{Safe: true, CentroidX: 10, CentroidY: 10, Width: 20, Height: 20, ForegroundArea: 400},
		{Safe: true, CentroidX: 11, CentroidY: 10, Width: 20, Height: 20, ForegroundArea: 400},
		{Safe: false, CentroidX: 12, CentroidY: 10, Width: 20, Height: 20, ForegroundArea: 400}, // unsafe
		{Safe: true, CentroidX: 13, CentroidY: 10, Width: 20, Height: 20, ForegroundArea: 400},
	}
	pairwise := [][]float64{
		{0, 0.1, 0.2, 0.3},
		{0.1, 0, 0.1, 0.2},
		{0.2, 0.1, 0, 0.1},
		{0.3, 0.2, 0.1, 0},
	}
	analysis := FrameSequenceAnalysis{
		FPS:             12,
		ForegroundRatio: 0.5,
		Frames:          frames,
		PairwiseMSE:     pairwise,
	}

	// 1. SampleCount too large
	opts := FrameIntervalSelectionOptions{
		SampleCount:            10,
		MinimumSpanRatio:       0.1,
		StartWindowRatio:       0.5,
		MinimumForegroundRatio: 0.1,
		EndpointMSEQuantile:    0.8,
		ChangeScaleQuantile:    0.8,
		ChangeBaselineQuantile: 0.2,
	}
	if _, err := SelectFrameInterval(analysis, opts); err == nil {
		t.Fatal("expected error on sample count > frame count")
	}

	// 2. ForegroundRatio too low
	lowRatioOpts := opts
	lowRatioOpts.SampleCount = 2
	lowRatioOpts.MinimumForegroundRatio = 0.9
	_, errLow := SelectFrameInterval(analysis, lowRatioOpts)
	var qErrForeground *QualityError
	if !errors.As(errLow, &qErrForeground) || qErrForeground.Kind != "foreground" {
		t.Fatalf("expected foreground quality error, got %v", errLow)
	}

	// 3. No safe intervals when unsafe frames interrupt all candidates
	allUnsafeFrames := []FrameObservation{
		{Safe: false}, {Safe: false}, {Safe: false}, {Safe: false},
	}
	allUnsafeAnalysis := analysis
	allUnsafeAnalysis.Frames = allUnsafeFrames
	validOpts := opts
	validOpts.SampleCount = 2
	validOpts.MinimumForegroundRatio = 0.1
	_, errUnsafe := SelectFrameInterval(allUnsafeAnalysis, validOpts)
	var qErrFraming *QualityError
	if !errors.As(errUnsafe, &qErrFraming) || qErrFraming.Kind != "framing" {
		t.Fatalf("expected framing quality error, got %v", errUnsafe)
	}

	// 4. All intervals filtered by low endpoint threshold
	filterAllOpts := opts
	filterAllOpts.SampleCount = 2
	filterAllOpts.MinimumSpanRatio = 0.1
	filterAllOpts.StartWindowRatio = 1.0
	filterAllOpts.EndpointMSEQuantile = 0.0 // forces threshold to lowest value
	// Make pairwise MSE for first pair higher than threshold
	badMSEAnalysis := analysis
	badMSEAnalysis.PairwiseMSE = [][]float64{
		{0, 0.9, 0.9, 0.9},
		{0.9, 0, 0.9, 0.9},
		{0.9, 0.9, 0, 0.9},
		{0.9, 0.9, 0.9, 0},
	}
	// With threshold = 0.0, any endpointMSE > 0 gets skipped
	_, _ = SelectFrameInterval(badMSEAnalysis, filterAllOpts)
}

func TestFrameIntervalMathHelpers(t *testing.T) {
	t.Parallel()

	// 1. sampleFrameIndices
	if idx := sampleFrameIndices(0, 10, 0); idx != nil {
		t.Fatalf("expected nil for count <= 0, got %v", idx)
	}
	if idx := sampleFrameIndices(5, 10, 1); len(idx) != 1 || idx[0] != 5 {
		t.Fatalf("expected [5] for count=1, got %v", idx)
	}

	// 2. frameGeometryVariation
	if v := frameGeometryVariation(nil); v != 0 {
		t.Fatalf("expected 0 for empty frames, got %f", v)
	}
	zeroAreaFrames := []FrameObservation{
		{ForegroundArea: 0},
		{ForegroundArea: 0},
	}
	if v := frameGeometryVariation(zeroAreaFrames); v != 0 {
		t.Fatalf("expected 0 for zero area frames, got %f", v)
	}

	// 3. frameLinearCentroidMotionScore
	if s := frameLinearCentroidMotionScore([]FrameObservation{{}, {}}); s != 0 {
		t.Fatalf("expected 0 for <3 frames, got %f", s)
	}
	// Stationary centroid (denominator will be large, residuals 0 -> score 1)
	stationary := []FrameObservation{
		{CentroidX: 10},
		{CentroidX: 10},
		{CentroidX: 10},
	}
	if s := frameLinearCentroidMotionScore(stationary); s != 1 {
		t.Fatalf("expected score 1 for stationary centroids, got %f", s)
	}

	// NaN and non-linear motion
	nonLinear := []FrameObservation{
		{CentroidX: math.NaN()},
		{CentroidX: 0},
		{CentroidX: 100},
		{CentroidX: 0},
	}
	if s := frameLinearCentroidMotionScore(nonLinear); s != 0 {
		t.Fatalf("expected score 0 for highly non-linear motion, got %f", s)
	}

	// Denominator near 0 with identical X coordinates (all index=0? Not possible with distinct indices, but count < 3)
	fewValid := []FrameObservation{
		{CentroidX: math.NaN()},
		{CentroidX: 10},
		{CentroidX: math.NaN()},
	}
	if s := frameLinearCentroidMotionScore(fewValid); s != 0 {
		t.Fatalf("expected score 0 for <3 valid centroids, got %f", s)
	}

	// 4. frameStandardDeviation
	if sd := frameStandardDeviation(nil); sd != 0 {
		t.Fatalf("expected 0 for empty slice, got %f", sd)
	}

	// 5. frameQuantile
	if q := frameQuantile(nil, 0.5); q != 0 {
		t.Fatalf("expected 0 for empty slice, got %f", q)
	}
	if q := frameQuantile([]float64{5.0}, 0.5); q != 5.0 {
		t.Fatalf("expected 5.0 for single element, got %f", q)
	}
	if q := frameQuantile([]float64{0.0, 10.0}, 0.5); q != 5.0 {
		t.Fatalf("expected 5.0 for median of [0, 10], got %f", q)
	}
}
