package video

import (
	"fmt"
	"math"
	"sort"
)

// FrameIntervalSelectionWeights controls how measured media properties
// contribute to interval selection. The processor does not assign defaults.
type FrameIntervalSelectionWeights struct {
	EndpointSimilarity    float64
	MeanAdjacentMSE       float64
	CentroidStability     float64
	LinearCentroidMotion  float64
	FirstFrameSimilarity  float64
	Compactness           float64
	GeometryCoverage      float64
	ChangeCoverage        float64
	PostIntervalStability float64
}

// FrameIntervalSelectionOptions describes caller-owned interval constraints.
type FrameIntervalSelectionOptions struct {
	SampleCount              int
	MinimumSpanFrames        int
	MinimumSpanRatio         float64
	MinimumStartWindowFrames int
	StartWindowRatio         float64
	PreferFirstFrame         bool
	MinimumForegroundRatio   float64
	EndpointMSEQuantile      float64
	ChangeScaleQuantile      float64
	ChangeBaselineQuantile   float64
	Weights                  FrameIntervalSelectionWeights
}

// FrameIntervalSelection reports the selected indices and measured values.
type FrameIntervalSelection struct {
	Indices            []int
	StartFrame         int
	EndFrame           int
	Score              float64
	EndpointSimilarity float64
	MeanAdjacentMSE    float64
	GeometryCoverage   float64
	SpanRatio          float64
	CentroidStability  float64
	EndpointMSE        float64
}

type frameIntervalCandidate struct {
	start              int
	end                int
	score              float64
	endpointSimilarity float64
	meanAdjacentMSE    float64
	geometryCoverage   float64
	spanRatio          float64
	centroidStability  float64
	endpointMSE        float64
}

type frameInterval struct {
	start int
	end   int
}

// SelectFrameInterval applies caller-supplied constraints and weights to
// domain-neutral frame observations.
func SelectFrameInterval(
	analysis FrameSequenceAnalysis,
	options FrameIntervalSelectionOptions,
) (FrameIntervalSelection, error) {
	if err := validateFrameIntervalSelectionInput(analysis, options); err != nil {
		return FrameIntervalSelection{}, err
	}
	frames := analysis.Frames
	if options.SampleCount <= 0 || len(frames) < options.SampleCount+1 {
		return FrameIntervalSelection{}, fmt.Errorf(
			"video: has %d candidate frames; need at least %d",
			len(frames),
			options.SampleCount+1,
		)
	}
	if analysis.ForegroundRatio < options.MinimumForegroundRatio {
		return FrameIntervalSelection{}, &QualityError{
			Kind: "foreground",
			Message: fmt.Sprintf(
				"video: foreground ratio %.3f is below required %.3f",
				analysis.ForegroundRatio,
				options.MinimumForegroundRatio,
			),
		}
	}

	minimumSpan := max(
		options.SampleCount,
		max(options.MinimumSpanFrames, int(math.Ceil(float64(len(frames))*options.MinimumSpanRatio))),
	)
	unsafePrefix := make([]int, len(frames)+1)
	for index, frame := range frames {
		unsafePrefix[index+1] = unsafePrefix[index]
		if !frame.Safe {
			unsafePrefix[index+1]++
		}
	}

	intervals := make([]frameInterval, 0, len(frames)*len(frames)/2)
	endpointMSE := make([]float64, 0, cap(intervals))
	startWindow := max(
		options.MinimumStartWindowFrames,
		int(math.Ceil(float64(len(frames))*options.StartWindowRatio)),
	)
	for start := 0; start < len(frames) && start <= startWindow; start++ {
		for end := start + minimumSpan; end < len(frames); end++ {
			indices := sampleFrameIndices(start, end, options.SampleCount)
			allSamplesSafe := true
			for _, sampleIndex := range indices {
				if unsafePrefix[sampleIndex+1]-unsafePrefix[sampleIndex] != 0 {
					allSamplesSafe = false
					break
				}
			}
			if !allSamplesSafe {
				continue
			}
			intervals = append(intervals, frameInterval{start: start, end: end})
			endpointMSE = append(endpointMSE, analysis.PairwiseMSE[start][end])
		}
	}
	if len(intervals) == 0 {
		return FrameIntervalSelection{}, &QualityError{
			Kind: "framing",
			Message: fmt.Sprintf(
				"video: no candidate interval has %d safe sampled frames and the required %.0f%% span",
				options.SampleCount,
				options.MinimumSpanRatio*100,
			),
		}
	}

	if options.PreferFirstFrame && containsIntervalStartingAtZero(intervals) {
		filteredIntervals := make([]frameInterval, 0, len(intervals))
		filteredEndpointMSE := make([]float64, 0, len(endpointMSE))
		for index, candidate := range intervals {
			if candidate.start == 0 {
				filteredIntervals = append(filteredIntervals, candidate)
				filteredEndpointMSE = append(filteredEndpointMSE, endpointMSE[index])
			}
		}
		intervals, endpointMSE = filteredIntervals, filteredEndpointMSE
	}

	endpointThreshold := frameQuantile(endpointMSE, options.EndpointMSEQuantile)
	adjacentMSE := make([]float64, 0, len(frames)-1)
	for index := 0; index+1 < len(frames); index++ {
		adjacentMSE = append(adjacentMSE, analysis.PairwiseMSE[index][index+1])
	}
	changeScale := math.Max(frameQuantile(adjacentMSE, options.ChangeScaleQuantile), 1e-6)
	changeBaseline := frameQuantile(adjacentMSE, options.ChangeBaselineQuantile)
	activeChange := make([]float64, len(adjacentMSE))
	var totalActiveChange float64
	for index, value := range adjacentMSE {
		activeChange[index] = math.Max(value-changeBaseline, 0)
		totalActiveChange += activeChange[index]
	}
	globalGeometryVariation := math.Max(frameGeometryVariation(frames), 1e-6)

	best := frameIntervalCandidate{score: math.Inf(-1)}
	for index, candidate := range intervals {
		if endpointMSE[index] > endpointThreshold {
			continue
		}
		var meanAdjacentMSE, intervalActiveChange float64
		for frame := candidate.start; frame < candidate.end; frame++ {
			meanAdjacentMSE += analysis.PairwiseMSE[frame][frame+1]
			intervalActiveChange += activeChange[frame]
		}
		meanAdjacentMSE /= float64(candidate.end - candidate.start)
		normalizedMeanChange := math.Min(meanAdjacentMSE/changeScale, 1)
		changeCoverage := 1.0
		if totalActiveChange > 1e-9 {
			changeCoverage = clampFrameFloat(intervalActiveChange/totalActiveChange, 0, 1)
		}
		intervalFrames := frames[candidate.start : candidate.end+1]
		centroidStability := 1 / (1 + frameCentroidStandardDeviation(intervalFrames))
		linearCentroidMotion := frameLinearCentroidMotionScore(intervalFrames)
		endpointSimilarity := 1 - endpointMSE[index]
		firstFrameSimilarity := clampFrameFloat(1-analysis.PairwiseMSE[0][candidate.start], 0, 1)
		spanRatio := float64(candidate.end-candidate.start) / float64(len(frames)-1)
		geometryCoverage := math.Min(frameGeometryVariation(intervalFrames)/globalGeometryVariation, 1)
		postIntervalChange := 0.0
		if candidate.end+1 < len(frames) {
			postIntervalChange = clampFrameFloat(
				analysis.PairwiseMSE[candidate.end][candidate.end+1]/changeScale,
				0,
				1,
			)
		}
		postIntervalStability := 1 - postIntervalChange
		weights := options.Weights
		score := weights.EndpointSimilarity*endpointSimilarity +
			weights.MeanAdjacentMSE*normalizedMeanChange +
			weights.CentroidStability*centroidStability +
			weights.LinearCentroidMotion*linearCentroidMotion +
			weights.FirstFrameSimilarity*firstFrameSimilarity +
			weights.Compactness*(1-spanRatio) +
			weights.GeometryCoverage*geometryCoverage +
			weights.ChangeCoverage*changeCoverage +
			weights.PostIntervalStability*postIntervalStability
		if score > best.score {
			best = frameIntervalCandidate{
				start: candidate.start, end: candidate.end, score: score,
				endpointSimilarity: endpointSimilarity, meanAdjacentMSE: meanAdjacentMSE,
				geometryCoverage: geometryCoverage, spanRatio: spanRatio,
				centroidStability: centroidStability, endpointMSE: endpointMSE[index],
			}
		}
	}
	if math.IsInf(best.score, -1) {
		return FrameIntervalSelection{}, fmt.Errorf("video: interval selection produced no candidate")
	}
	return FrameIntervalSelection{
		Indices:    sampleFrameIndices(best.start, best.end, options.SampleCount),
		StartFrame: best.start, EndFrame: best.end, Score: best.score,
		EndpointSimilarity: best.endpointSimilarity, MeanAdjacentMSE: best.meanAdjacentMSE,
		GeometryCoverage: best.geometryCoverage, SpanRatio: best.spanRatio,
		CentroidStability: best.centroidStability, EndpointMSE: best.endpointMSE,
	}, nil
}

func validateFrameIntervalSelectionInput(
	analysis FrameSequenceAnalysis,
	options FrameIntervalSelectionOptions,
) error {
	if options.MinimumSpanFrames < 0 {
		return fmt.Errorf("video: minimum span frames must not be negative")
	}
	if options.MinimumStartWindowFrames < 0 {
		return fmt.Errorf("video: minimum start window frames must not be negative")
	}
	for name, value := range map[string]float64{
		"minimum span ratio":       options.MinimumSpanRatio,
		"start window ratio":       options.StartWindowRatio,
		"minimum foreground ratio": options.MinimumForegroundRatio,
		"endpoint MSE quantile":    options.EndpointMSEQuantile,
		"change scale quantile":    options.ChangeScaleQuantile,
		"change baseline quantile": options.ChangeBaselineQuantile,
	} {
		if math.IsNaN(value) || value < 0 || value > 1 {
			return fmt.Errorf("video: %s must be between 0 and 1", name)
		}
	}
	weights := options.Weights
	for name, value := range map[string]float64{
		"endpoint similarity weight":     weights.EndpointSimilarity,
		"mean adjacent MSE weight":       weights.MeanAdjacentMSE,
		"centroid stability weight":      weights.CentroidStability,
		"linear centroid motion weight":  weights.LinearCentroidMotion,
		"first frame similarity weight":  weights.FirstFrameSimilarity,
		"compactness weight":             weights.Compactness,
		"geometry coverage weight":       weights.GeometryCoverage,
		"change coverage weight":         weights.ChangeCoverage,
		"post-interval stability weight": weights.PostIntervalStability,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("video: %s must be finite", name)
		}
	}
	if math.IsNaN(analysis.ForegroundRatio) || math.IsInf(analysis.ForegroundRatio, 0) ||
		analysis.ForegroundRatio < 0 || analysis.ForegroundRatio > 1 {
		return fmt.Errorf("video: foreground ratio must be between 0 and 1")
	}
	if len(analysis.PairwiseMSE) != len(analysis.Frames) {
		return fmt.Errorf("video: pairwise MSE matrix must match frame count")
	}
	for rowIndex, row := range analysis.PairwiseMSE {
		if len(row) != len(analysis.Frames) {
			return fmt.Errorf("video: pairwise MSE row %d must match frame count", rowIndex)
		}
		for _, value := range row {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
				return fmt.Errorf("video: pairwise MSE values must be finite and non-negative")
			}
		}
	}
	return nil
}

func containsIntervalStartingAtZero(intervals []frameInterval) bool {
	for _, candidate := range intervals {
		if candidate.start == 0 {
			return true
		}
	}
	return false
}

func sampleFrameIndices(start, end, count int) []int {
	if count <= 0 {
		return nil
	}
	if count == 1 {
		return []int{start}
	}
	span := end - start
	indices := make([]int, count)
	for index := range count {
		phase := float64(index) / float64(count-1)
		target := start + int(math.Round(phase*float64(span)))
		minimumTarget := start + index
		maximumTarget := end - (count - 1 - index)
		indices[index] = min(max(target, minimumTarget), maximumTarget)
	}
	return indices
}

func frameGeometryVariation(frames []FrameObservation) float64 {
	if len(frames) < 2 {
		return 0
	}
	widths := make([]float64, 0, len(frames))
	heights := make([]float64, 0, len(frames))
	areas := make([]float64, 0, len(frames))
	for _, frame := range frames {
		if frame.ForegroundArea == 0 {
			continue
		}
		widths = append(widths, frame.Width)
		heights = append(heights, frame.Height)
		areas = append(areas, float64(frame.ForegroundArea))
	}
	return frameStandardDeviation(widths)/analysisSize + frameStandardDeviation(heights)/analysisSize +
		frameStandardDeviation(areas)/(analysisSize*analysisSize)
}

func frameCentroidStandardDeviation(frames []FrameObservation) float64 {
	xs := make([]float64, 0, len(frames))
	ys := make([]float64, 0, len(frames))
	for _, frame := range frames {
		if !math.IsNaN(frame.CentroidX) && !math.IsNaN(frame.CentroidY) {
			xs = append(xs, frame.CentroidX)
			ys = append(ys, frame.CentroidY)
		}
	}
	return frameStandardDeviation(xs) + frameStandardDeviation(ys)
}

func frameLinearCentroidMotionScore(frames []FrameObservation) float64 {
	if len(frames) < 3 {
		return 0
	}
	var count, sumX, sumY, sumXX, sumXY float64
	for index, frame := range frames {
		if math.IsNaN(frame.CentroidX) {
			continue
		}
		x := float64(index)
		count++
		sumX += x
		sumY += frame.CentroidX
		sumXX += x * x
		sumXY += x * frame.CentroidX
	}
	denominator := count*sumXX - sumX*sumX
	if count < 3 || math.Abs(denominator) < 1e-9 {
		return 0
	}
	slope := (count*sumXY - sumX*sumY) / denominator
	intercept := (sumY - slope*sumX) / count
	var squaredResiduals float64
	for index, frame := range frames {
		if math.IsNaN(frame.CentroidX) {
			continue
		}
		residual := frame.CentroidX - (intercept + slope*float64(index))
		squaredResiduals += residual * residual
	}
	if math.Sqrt(squaredResiduals/count) < 2 {
		return 1
	}
	return 0
}

func frameStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var mean float64
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))
	var variance float64
	for _, value := range values {
		delta := value - mean
		variance += delta * delta
	}
	return math.Sqrt(variance / float64(len(values)))
}

func frameQuantile(values []float64, quantile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	position := clampFrameFloat(quantile, 0, 1) * float64(len(ordered)-1)
	low, high := int(math.Floor(position)), int(math.Ceil(position))
	if low == high {
		return ordered[low]
	}
	fraction := position - float64(low)
	return ordered[low]*(1-fraction) + ordered[high]*fraction
}

func clampFrameFloat(value, low, high float64) float64 {
	return math.Min(math.Max(value, low), high)
}
