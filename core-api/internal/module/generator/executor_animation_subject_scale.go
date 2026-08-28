package generator

import (
	"fmt"
	"image"
	"math"
	"sort"

	videoprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/video"
)

const (
	animationSubjectScaleMinMultiplier = 0.25
	animationSubjectScaleMaxMultiplier = 4.00
	animationSubjectScaleDeadband      = 0.05
	maxAspectDivergenceThreshold       = 0.50
)

// animationSubjectScaleMultiplier estimates how much the video model shrank or
// enlarged the reference subject. Effect-heavy actions (spray, bubbles, lasers,
// slashes) can cause the model to reduce the character or cause bounding boxes
// across frames to inflate with effect particles. We filter out effect-heavy
// frames by measuring geometric similarity against the clean reference.
func animationSubjectScaleMultiplier(
	reference image.Image,
	frames []image.Image,
	chromaKey videoprocessor.ChromaKey,
) (float64, error) {
	if reference == nil || reference.Bounds().Empty() {
		return 1, fmt.Errorf("generator: animation scale reference is empty")
	}
	if len(frames) == 0 {
		return 1, fmt.Errorf("generator: animation frames are required to measure subject scale")
	}
	referenceKey := chromaKey
	referenceKey.AutoDetect = false
	frameKey := chromaKey
	if frameKey.AutoDetect {
		frameKey = videoprocessor.ResolveChromaKey(frames, frameKey)
	}
	referenceBounds, ok := videoprocessor.ForegroundBounds(reference, referenceKey)
	if !ok {
		return 1, fmt.Errorf("generator: animation scale reference has no detectable foreground")
	}

	refWidth := float64(referenceBounds.Dx())
	refHeight := float64(referenceBounds.Dy())
	refAspectRatio := refWidth / refHeight

	type frameMetric struct {
		relativeHeight   float64
		aspectDivergence float64
	}

	metrics := make([]frameMetric, 0, len(frames))
	for index, frame := range frames {
		if frame == nil || frame.Bounds().Empty() {
			return 1, fmt.Errorf("generator: selected animation frame %d is empty", index+1)
		}
		bounds, ok := videoprocessor.ForegroundBounds(frame, frameKey)
		if !ok {
			continue
		}
		frameHeight := float64(frame.Bounds().Dy())
		aspectRatio := float64(bounds.Dx()) / float64(bounds.Dy())
		divergence := math.Abs(math.Log(aspectRatio / refAspectRatio))
		metrics = append(metrics, frameMetric{
			relativeHeight:   float64(bounds.Dy()) / frameHeight,
			aspectDivergence: divergence,
		})
	}
	if len(metrics) == 0 {
		return 1, fmt.Errorf("generator: generated animation has no measurable foreground for subject-scale compensation")
	}

	// Effect particles usually distort the foreground bounds' aspect ratio, so
	// keep the frames whose silhouette is closest to the clean reference.
	sort.SliceStable(metrics, func(i, j int) bool {
		return metrics[i].aspectDivergence < metrics[j].aspectDivergence
	})

	candidateRelativeHeights := make([]float64, 0, len(metrics))
	minDivergence := metrics[0].aspectDivergence
	threshold := math.Max(maxAspectDivergenceThreshold, minDivergence+0.15)

	for _, m := range metrics {
		if m.aspectDivergence <= threshold {
			candidateRelativeHeights = append(candidateRelativeHeights, m.relativeHeight)
		}
	}

	sort.Float64s(candidateRelativeHeights)
	medianRelativeHeight := candidateRelativeHeights[len(candidateRelativeHeights)/2]
	if len(candidateRelativeHeights)%2 == 0 {
		medianRelativeHeight = (candidateRelativeHeights[len(candidateRelativeHeights)/2-1] + candidateRelativeHeights[len(candidateRelativeHeights)/2]) / 2
	}

	// The reference image (green canvas) and the video frames have different
	// absolute resolutions (e.g. 1920px vs 720px). Comparing raw pixel heights
	// directly produces a spurious scale factor. We normalise both heights
	// relative to their respective image heights so we compare the fraction of
	// the canvas each subject occupies, not absolute pixels.
	refImgHeight := float64(reference.Bounds().Dy())
	refRelativeHeight := refHeight / refImgHeight
	multiplier := refRelativeHeight / medianRelativeHeight

	// Minor variations within deadband (+-5%) are treated as 1.0 to avoid
	// unnecessary fractional resampling.
	if math.Abs(multiplier-1.0) < animationSubjectScaleDeadband {
		return 1, nil
	}

	return math.Max(animationSubjectScaleMinMultiplier, math.Min(multiplier, animationSubjectScaleMaxMultiplier)), nil
}
