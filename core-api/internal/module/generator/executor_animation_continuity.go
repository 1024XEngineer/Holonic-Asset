package generator

import (
	"fmt"
	"image"
	"math"
	"slices"

	videoprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/video"
)

const (
	animationEditVisualChangeFloor              = 0.07
	animationEditCentroidChangeFloor            = 0.30
	animationEditDimensionChangeFloor           = 0.40
	animationEditAreaChangeFloor                = 0.65
	animationEditBaselineMultiplier             = 2.5
	animationEditBaselineMADMultiplier          = 6.0
	animationEditMotionRetentionRatio           = 0.20
	animationEditApplicationTargetRatio         = 0.60
	animationEditApplicationMaskFloor           = 0.025
	animationEditApplicationAppearanceFloor     = 0.0002
	animationEditApplicationMaskMargin          = 0.008
	animationEditApplicationAppearanceMargin    = 0.00004
	animationEditApplicationBaselineMultiplier  = 1.25
	animationEditVisualExcursionFloor           = 0.01
	animationEditDimensionExcursionFloor        = 0.08
	animationEditAreaExcursionFloor             = 0.12
	animationEditTemporalDeltaFloor             = 0.04
	animationEditTemporalAreaDeltaFloor         = 0.06
	animationEditTemporalReversalAllowance      = 2
	animationEditTemporalMinimumReversals       = 4
	animationEditTemporalWidthReversalAllowance = 3
	animationEditTemporalWidthMinimumReversals  = 6
	animationEditTemporalDisorderFloor          = 0.05
	animationEditTemporalDisorderMultiplier     = 5.0
	animationEditMetricComparisonEpsilon        = 1e-9
)

type animationTransitionMetrics struct {
	visual    float64
	centroid  float64
	width     float64
	height    float64
	dimension float64
	area      float64
}

type animationMotionExcursion struct {
	visual float64
	width  float64
	height float64
	area   float64
}

func animationVideoChromaKey() videoprocessor.ChromaKey {
	return videoprocessor.ChromaKey{
		HueMin: animationChromaHueMin, HueMax: animationChromaHueMax,
		HighSaturationMin: animationChromaHighSaturationMin, HighValueMin: animationChromaHighValueMin,
		BrightSaturationMin: animationChromaBrightSaturationMin, BrightValueMin: animationChromaBrightValueMin,
		AutoDetect: true,
	}
}

func animationVideoChromaKeyForFrame(width, height int) videoprocessor.ChromaKey {
	key := animationVideoChromaKey()
	key.SafetyMarginRatio = animationFrameSafetyMarginRatio(width, height)
	return key
}

func animationFrameSafetyMarginRatio(width, height int) float64 {
	shortEdge := min(width, height)
	if shortEdge <= 0 {
		return 0
	}
	// Keep approximately one final logical pixel clear at the edge. Cap the
	// ratio at the legacy 2.5% so small frames do not become more restrictive.
	return min(.025, 1/float64(shortEdge))
}

func validateEditFrameContinuity(request AnimationGenerationRequest, generated []image.Image) error {
	original := request.continuityReferenceFrames
	if len(original) != request.FrameCount {
		return fmt.Errorf("generator: edit frame continuity references contain %d frames; expected %d", len(original), request.FrameCount)
	}
	if len(generated) != request.FrameCount {
		return fmt.Errorf("generator: edit frame continuity candidate contains %d frames; expected %d", len(generated), request.FrameCount)
	}

	targets := make([]bool, request.FrameCount)
	if len(request.TargetFrameIndices) == 0 {
		for index := range targets {
			targets[index] = true
		}
	} else {
		for _, index := range request.TargetFrameIndices {
			targets[index] = true
		}
	}
	finalFrames := append([]image.Image(nil), original...)
	for index, target := range targets {
		if target {
			finalFrames[index] = generated[index]
		}
	}

	originalAnalysis, err := videoprocessor.AnalyzeFrameSequence(original, request.FPS, animationVideoChromaKeyForFrame(request.FrameWidth, request.FrameHeight))
	if err != nil {
		return fmt.Errorf("generator: analyze original edit frame context: %w", err)
	}
	finalAnalysis, err := videoprocessor.AnalyzeFrameSequence(finalFrames, request.FPS, animationVideoChromaKeyForFrame(request.FrameWidth, request.FrameHeight))
	if err != nil {
		return fmt.Errorf("generator: analyze edited frame context: %w", err)
	}
	editDifferences, err := videoprocessor.AnalyzeFramePairs(original, generated, animationVideoChromaKey())
	if err != nil {
		return fmt.Errorf("generator: compare original and edited frame context: %w", err)
	}

	if request.FrameCount < 2 {
		return nil
	}

	baseline := animationTransitionSequence(originalAnalysis)
	limits := animationTransitionMetrics{
		visual: animationEditMetricLimit(
			animationMetricValues(baseline, func(value animationTransitionMetrics) float64 { return value.visual }),
			animationEditVisualChangeFloor,
		),
		centroid: animationEditMetricLimit(
			animationMetricValues(baseline, func(value animationTransitionMetrics) float64 { return value.centroid }),
			animationEditCentroidChangeFloor,
		),
		dimension: animationEditMetricLimit(
			animationMetricValues(baseline, func(value animationTransitionMetrics) float64 { return value.dimension }),
			animationEditDimensionChangeFloor,
		),
		area: animationEditMetricLimit(
			animationMetricValues(baseline, func(value animationTransitionMetrics) float64 { return value.area }),
			animationEditAreaChangeFloor,
		),
	}
	candidate := animationTransitionSequence(finalAnalysis)
	for index, metrics := range candidate {
		// Only gate the seams where generated target samples join untouched
		// original frames. Internal target-to-target motion is allowed to depart
		// substantially from the original; constraining every internal transition
		// caused meaningful requested edits to be selected out in favour of near
		// copies of the source animation.
		if targets[index] == targets[index+1] {
			continue
		}
		if metrics.centroid > limits.centroid+animationEditMetricComparisonEpsilon {
			return editFrameContinuityError(index, "root displacement", metrics.centroid, limits.centroid)
		}
		if metrics.dimension > limits.dimension+animationEditMetricComparisonEpsilon {
			return editFrameContinuityError(index, "subject scale", metrics.dimension, limits.dimension)
		}
		if metrics.area > limits.area+animationEditMetricComparisonEpsilon {
			return editFrameContinuityError(index, "foreground area", metrics.area, limits.area)
		}
		if metrics.visual > limits.visual+animationEditMetricComparisonEpsilon {
			return editFrameContinuityError(index, "pose appearance", metrics.visual, limits.visual)
		}
	}
	if err := validateEditFrameMotionPreservation(targets, originalAnalysis, finalAnalysis); err != nil {
		return err
	}
	if err := validateEditFrameTemporalCoherence(targets, originalAnalysis, finalAnalysis); err != nil {
		return err
	}
	return validateEditFrameApplication(targets, editDifferences)
}

func animationTransitionSequence(analysis videoprocessor.FrameSequenceAnalysis) []animationTransitionMetrics {
	metrics := make([]animationTransitionMetrics, max(0, len(analysis.Frames)-1))
	for index := range metrics {
		left, right := analysis.Frames[index], analysis.Frames[index+1]
		width := animationLogRatio(left.Width, right.Width)
		height := animationLogRatio(left.Height, right.Height)
		metrics[index] = animationTransitionMetrics{
			visual:    analysis.PairwiseMSE[index][index+1],
			centroid:  animationCentroidChange(left, right),
			width:     width,
			height:    height,
			dimension: math.Max(width, height),
			area:      animationAreaChange(left, right),
		}
	}
	return metrics
}

func validateEditFrameMotionPreservation(
	targets []bool,
	original, candidate videoprocessor.FrameSequenceAnalysis,
) error {
	start, end, ok := animationEditedContextSpan(targets)
	if !ok || end <= start {
		return nil
	}
	baseline := animationSequenceMotionExcursion(original, start, end)
	edited := animationSequenceMotionExcursion(candidate, start, end)
	// Use one coarse anti-freeze signal instead of requiring every width,
	// height, area, and appearance trajectory to resemble the source. A clear
	// additive edit may legitimately redistribute those individual metrics.
	originalMotion := math.Max(baseline.visual, math.Max(baseline.width, math.Max(baseline.height, baseline.area)))
	editedMotion := math.Max(edited.visual, math.Max(edited.width, math.Max(edited.height, edited.area)))
	if originalMotion < animationEditDimensionExcursionFloor {
		return nil
	}
	minimum := originalMotion * animationEditMotionRetentionRatio
	if editedMotion+animationEditMetricComparisonEpsilon >= minimum {
		return nil
	}
	return &videoprocessor.QualityError{
		Kind: "motion_preservation",
		Message: fmt.Sprintf(
			"generator: edit frame motion preservation failed across context frames %d-%d: overall motion excursion %.4f retained less than %.4f from original %.4f",
			start+1,
			end+1,
			editedMotion,
			minimum,
			originalMotion,
		),
	}
}

func validateEditFrameApplication(
	targets []bool,
	differences []videoprocessor.FramePairDifference,
) error {
	if len(differences) != len(targets) {
		return fmt.Errorf("generator: edit application comparison contains %d frames; expected %d", len(differences), len(targets))
	}
	backgroundMaskChanges := make([]float64, 0, len(targets))
	backgroundAppearanceChanges := make([]float64, 0, len(targets))
	targetCount := 0
	for index, target := range targets {
		if target {
			targetCount++
			continue
		}
		backgroundMaskChanges = append(backgroundMaskChanges, differences[index].ForegroundMaskDifference)
		backgroundAppearanceChanges = append(backgroundAppearanceChanges, differences[index].AppearanceMSE)
	}
	if targetCount == 0 {
		return nil
	}
	maskBaseline := animationEditDifferenceMedian(backgroundMaskChanges)
	appearanceBaseline := animationEditDifferenceMedian(backgroundAppearanceChanges)
	maskThreshold := math.Max(
		animationEditApplicationMaskFloor,
		maskBaseline*animationEditApplicationBaselineMultiplier+animationEditApplicationMaskMargin,
	)
	appearanceThreshold := math.Max(
		animationEditApplicationAppearanceFloor,
		appearanceBaseline*animationEditApplicationBaselineMultiplier+animationEditApplicationAppearanceMargin,
	)
	changedTargets := 0
	maximumMaskChange := 0.0
	maximumAppearanceChange := 0.0
	for index, target := range targets {
		if !target {
			continue
		}
		difference := differences[index]
		maximumMaskChange = math.Max(maximumMaskChange, difference.ForegroundMaskDifference)
		maximumAppearanceChange = math.Max(maximumAppearanceChange, difference.AppearanceMSE)
		if difference.ForegroundMaskDifference >= maskThreshold || difference.AppearanceMSE >= appearanceThreshold {
			changedTargets++
		}
	}
	requiredTargets := max(1, int(math.Ceil(float64(targetCount)*animationEditApplicationTargetRatio)))
	if changedTargets < requiredTargets {
		return &videoprocessor.QualityError{
			Kind: "edit_application",
			Message: fmt.Sprintf(
				"generator: requested frame edit is not visibly applied: only %d of %d target frames differ from the original (need %d; maximum mask change %.4f, appearance change %.6f)",
				changedTargets,
				targetCount,
				requiredTargets,
				maximumMaskChange,
				maximumAppearanceChange,
			),
		}
	}
	return nil
}

func animationEditDifferenceMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	slices.Sort(sorted)
	return animationMedian(sorted)
}

func validateEditFrameTemporalCoherence(
	targets []bool,
	original, candidate videoprocessor.FrameSequenceAnalysis,
) error {
	start, end, ok := animationEditedContextSpan(targets)
	if !ok || end-start < 3 {
		return nil
	}
	type temporalSignal struct {
		name              string
		original          []float64
		candidate         []float64
		tolerance         float64
		minimumReversals  int
		reversalAllowance int
	}
	signals := []temporalSignal{
		{
			name: "horizontal pose extent",
			original: animationFrameSignal(original, start, end, func(frame videoprocessor.FrameObservation) float64 {
				return math.Log(frame.Width)
			}),
			candidate: animationFrameSignal(candidate, start, end, func(frame videoprocessor.FrameObservation) float64 {
				return math.Log(frame.Width)
			}),
			tolerance:         animationEditTemporalDeltaFloor,
			minimumReversals:  animationEditTemporalWidthMinimumReversals,
			reversalAllowance: animationEditTemporalWidthReversalAllowance,
		},
		{
			name: "vertical pose extent",
			original: animationFrameSignal(original, start, end, func(frame videoprocessor.FrameObservation) float64 {
				return math.Log(frame.Height)
			}),
			candidate: animationFrameSignal(candidate, start, end, func(frame videoprocessor.FrameObservation) float64 {
				return math.Log(frame.Height)
			}),
			tolerance:         animationEditTemporalDeltaFloor,
			minimumReversals:  animationEditTemporalMinimumReversals,
			reversalAllowance: animationEditTemporalReversalAllowance,
		},
		{
			name: "foreground area",
			original: animationFrameSignal(original, start, end, func(frame videoprocessor.FrameObservation) float64 {
				return math.Log(float64(frame.ForegroundArea))
			}),
			candidate: animationFrameSignal(candidate, start, end, func(frame videoprocessor.FrameObservation) float64 {
				return math.Log(float64(frame.ForegroundArea))
			}),
			tolerance:         animationEditTemporalAreaDeltaFloor,
			minimumReversals:  animationEditTemporalMinimumReversals,
			reversalAllowance: animationEditTemporalReversalAllowance,
		},
	}
	for _, signal := range signals {
		originalReversals := animationSignalDirectionChanges(signal.original, signal.tolerance)
		candidateReversals := animationSignalDirectionChanges(signal.candidate, signal.tolerance)
		limit := max(signal.minimumReversals, originalReversals+signal.reversalAllowance)
		if candidateReversals > limit {
			return editFrameTemporalCoherenceError(
				start,
				end,
				fmt.Sprintf("%s changes direction %d times; expected at most %d", signal.name, candidateReversals, limit),
			)
		}
	}
	originalDisorder := animationTemporalNeighborDisorder(original, start, end)
	candidateDisorder := animationTemporalNeighborDisorder(candidate, start, end)
	limit := math.Max(
		animationEditTemporalDisorderFloor,
		originalDisorder*animationEditTemporalDisorderMultiplier,
	)
	if candidateDisorder > limit+animationEditMetricComparisonEpsilon {
		return editFrameTemporalCoherenceError(
			start,
			end,
			fmt.Sprintf("non-neighbor pose recurrence %.4f exceeds %.4f", candidateDisorder, limit),
		)
	}
	return nil
}

func animationFrameSignal(
	analysis videoprocessor.FrameSequenceAnalysis,
	start, end int,
	value func(videoprocessor.FrameObservation) float64,
) []float64 {
	result := make([]float64, end-start+1)
	for index := range result {
		result[index] = value(analysis.Frames[start+index])
	}
	return result
}

func animationSignalDirectionChanges(values []float64, tolerance float64) int {
	previousDirection := 0
	changes := 0
	for index := 1; index < len(values); index++ {
		delta := values[index] - values[index-1]
		direction := 0
		if delta > tolerance {
			direction = 1
		} else if delta < -tolerance {
			direction = -1
		}
		if direction == 0 {
			continue
		}
		if previousDirection != 0 && direction != previousDirection {
			changes++
		}
		previousDirection = direction
	}
	return changes
}

func animationTemporalNeighborDisorder(
	analysis videoprocessor.FrameSequenceAnalysis,
	start, end int,
) float64 {
	disorder := 0.0
	for index := start; index <= end; index++ {
		adjacent := math.Inf(1)
		if index > start {
			adjacent = math.Min(adjacent, analysis.PairwiseMSE[index][index-1])
		}
		if index < end {
			adjacent = math.Min(adjacent, analysis.PairwiseMSE[index][index+1])
		}
		remote := math.Inf(1)
		for other := start; other <= end; other++ {
			distance := index - other
			if distance > -2 && distance < 2 {
				continue
			}
			remote = math.Min(remote, analysis.PairwiseMSE[index][other])
		}
		if !math.IsInf(remote, 1) {
			disorder = math.Max(disorder, adjacent-remote)
		}
	}
	return math.Max(0, disorder)
}

func editFrameTemporalCoherenceError(start, end int, reason string) error {
	return &videoprocessor.QualityError{
		Kind: "temporal_coherence",
		Message: fmt.Sprintf(
			"generator: edit frame temporal coherence failed across context frames %d-%d: %s",
			start+1,
			end+1,
			reason,
		),
	}
}

func animationEditedContextSpan(targets []bool) (int, int, bool) {
	first, last := -1, -1
	for index, target := range targets {
		if !target {
			continue
		}
		if first < 0 {
			first = index
		}
		last = index
	}
	if first < 0 {
		return 0, 0, false
	}
	return max(0, first-1), min(len(targets)-1, last+1), true
}

func animationSequenceMotionExcursion(
	analysis videoprocessor.FrameSequenceAnalysis,
	start, end int,
) animationMotionExcursion {
	minimumWidth, maximumWidth := math.Inf(1), 0.0
	minimumHeight, maximumHeight := math.Inf(1), 0.0
	minimumArea, maximumArea := math.Inf(1), 0.0
	visual := 0.0
	for index := start; index <= end; index++ {
		frame := analysis.Frames[index]
		minimumWidth, maximumWidth = math.Min(minimumWidth, frame.Width), math.Max(maximumWidth, frame.Width)
		minimumHeight, maximumHeight = math.Min(minimumHeight, frame.Height), math.Max(maximumHeight, frame.Height)
		area := float64(frame.ForegroundArea)
		minimumArea, maximumArea = math.Min(minimumArea, area), math.Max(maximumArea, area)
		for other := start; other < index; other++ {
			visual = math.Max(visual, analysis.PairwiseMSE[other][index])
		}
	}
	return animationMotionExcursion{
		visual: visual,
		width:  animationLogRatio(minimumWidth, maximumWidth),
		height: animationLogRatio(minimumHeight, maximumHeight),
		area:   animationLogRatio(minimumArea, maximumArea),
	}
}

func animationCentroidChange(left, right videoprocessor.FrameObservation) float64 {
	normalizer := math.Max(1, (left.Height+right.Height)/2)
	return math.Hypot(right.CentroidX-left.CentroidX, right.CentroidY-left.CentroidY) / normalizer
}

func animationAreaChange(left, right videoprocessor.FrameObservation) float64 {
	return animationLogRatio(float64(left.ForegroundArea), float64(right.ForegroundArea))
}

func animationLogRatio(left, right float64) float64 {
	if left <= 0 || right <= 0 {
		return math.Inf(1)
	}
	return math.Abs(math.Log(right / left))
}

func animationMetricValues(
	metrics []animationTransitionMetrics,
	selectValue func(animationTransitionMetrics) float64,
) []float64 {
	values := make([]float64, len(metrics))
	for index, metric := range metrics {
		values[index] = selectValue(metric)
	}
	return values
}

func animationEditMetricLimit(values []float64, floor float64) float64 {
	if len(values) == 0 {
		return floor
	}
	sorted := append([]float64(nil), values...)
	slices.Sort(sorted)
	median := animationMedian(sorted)
	deviations := make([]float64, len(sorted))
	maximum := 0.0
	for index, value := range sorted {
		deviations[index] = math.Abs(value - median)
		maximum = math.Max(maximum, value)
	}
	slices.Sort(deviations)
	return math.Max(
		floor,
		math.Max(
			maximum*animationEditBaselineMultiplier,
			median+animationEditBaselineMADMultiplier*animationMedian(deviations),
		),
	)
}

func animationMedian(sorted []float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	middle := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[middle]
	}
	return (sorted[middle-1] + sorted[middle]) / 2
}

func editFrameContinuityError(index int, metric string, actual, limit float64) error {
	return &videoprocessor.QualityError{
		Kind: "continuity",
		Message: fmt.Sprintf(
			"generator: edit frame continuity failed between context frames %d and %d: %s %.4f exceeds %.4f",
			index+1,
			index+2,
			metric,
			actual,
			limit,
		),
	}
}
