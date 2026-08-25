package generator

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"testing"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	videoprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/video"
)

func TestValidateEditFrameContinuityAcceptsComparableNeighborMotion(t *testing.T) {
	original := []image.Image{
		animationContinuityFrame(28, 36, color.NRGBA{R: 35, B: 10, A: 255}),
		animationContinuityFrame(29, 36, color.NRGBA{R: 35, B: 10, A: 255}),
		animationContinuityFrame(30, 36, color.NRGBA{R: 35, B: 10, A: 255}),
	}
	generated := []image.Image{
		original[0],
		animationContinuityFrame(30, 36, color.NRGBA{R: 35, B: 10, A: 255}),
		original[2],
	}
	request := AnimationGenerationRequest{
		ReferenceImageContext:     true,
		TargetFrameIndices:        []int{1},
		FrameCount:                3,
		FPS:                       10,
		continuityReferenceFrames: original,
	}

	if err := validateEditFrameContinuity(request, generated); err != nil {
		t.Fatalf("expected smooth local edit to pass: %v", err)
	}
}

func TestValidateEditFrameContinuityRejectsAbruptPoseAndRootChanges(t *testing.T) {
	original := []image.Image{
		animationContinuityFrame(28, 36, color.NRGBA{R: 35, B: 10, A: 255}),
		animationContinuityFrame(29, 36, color.NRGBA{R: 35, B: 10, A: 255}),
		animationContinuityFrame(30, 36, color.NRGBA{R: 35, B: 10, A: 255}),
	}
	tests := []struct {
		name      string
		candidate image.Image
		want      string
	}{
		{
			name:      "pose appearance pop",
			candidate: animationContinuityFrame(29, 36, color.NRGBA{R: 255, G: 255, B: 255, A: 255}),
			want:      "pose appearance",
		},
		{
			name:      "root displacement jump",
			candidate: animationContinuityFrame(60, 24, color.NRGBA{R: 35, B: 10, A: 255}),
			want:      "root displacement",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			generated := []image.Image{original[0], test.candidate, original[2]}
			err := validateEditFrameContinuity(AnimationGenerationRequest{
				ReferenceImageContext:     true,
				TargetFrameIndices:        []int{1},
				FrameCount:                3,
				FPS:                       10,
				continuityReferenceFrames: original,
			}, generated)
			var qualityError *videoprocessor.QualityError
			if !errors.As(err, &qualityError) || qualityError.Kind != "continuity" || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %s continuity rejection, got %v", test.want, err)
			}
		})
	}
}

func TestValidateEditFrameContinuityRejectsCollapsedOriginalMotion(t *testing.T) {
	foreground := color.NRGBA{R: 35, B: 10, A: 255}
	original := []image.Image{
		animationContinuitySizedFrame(28, 36, 50, foreground),
		animationContinuitySizedFrame(28, 36, 70, foreground),
		animationContinuitySizedFrame(28, 36, 50, foreground),
	}
	generated := []image.Image{
		original[0],
		animationContinuitySizedFrame(28, 36, 50, foreground),
		original[2],
	}

	err := validateEditFrameContinuity(AnimationGenerationRequest{
		ReferenceImageContext:     true,
		TargetFrameIndices:        []int{1},
		FrameCount:                3,
		FPS:                       10,
		continuityReferenceFrames: original,
	}, generated)
	var qualityError *videoprocessor.QualityError
	if !errors.As(err, &qualityError) || qualityError.Kind != "motion_preservation" ||
		!strings.Contains(err.Error(), "overall motion excursion") {
		t.Fatalf("expected collapsed original motion rejection, got %v", err)
	}
}

func TestValidateEditFrameContinuityRejectsNoOpEdit(t *testing.T) {
	foreground := color.NRGBA{R: 35, B: 10, A: 255}
	heights := []int{50, 60, 70, 60, 50}
	original := make([]image.Image, len(heights))
	for index, height := range heights {
		original[index] = animationContinuitySizedFrame(28, 36, height, foreground)
	}

	err := validateEditFrameContinuity(AnimationGenerationRequest{
		ReferenceImageContext:     true,
		TargetFrameIndices:        []int{1, 2, 3},
		FrameCount:                len(original),
		FPS:                       10,
		continuityReferenceFrames: original,
	}, append([]image.Image(nil), original...))
	var qualityError *videoprocessor.QualityError
	if !errors.As(err, &qualityError) || qualityError.Kind != "edit_application" ||
		!strings.Contains(err.Error(), "not visibly applied") {
		t.Fatalf("expected no-op frame edit to be rejected, got %v", err)
	}
}

func TestValidateEditFrameContinuityAcceptsVisibleAdditiveGesture(t *testing.T) {
	foreground := color.NRGBA{R: 35, B: 10, A: 255}
	heights := []int{50, 60, 70, 60, 50}
	original := make([]image.Image, len(heights))
	generated := make([]image.Image, len(heights))
	for index, height := range heights {
		original[index] = animationContinuitySizedFrame(28, 36, height, foreground)
		generated[index] = original[index]
		if index >= 1 && index <= 3 {
			generated[index] = animationContinuityGestureFrame(28, 36, height, foreground)
		}
	}

	err := validateEditFrameContinuity(AnimationGenerationRequest{
		ReferenceImageContext:     true,
		TargetFrameIndices:        []int{1, 2, 3},
		FrameCount:                len(original),
		FPS:                       10,
		continuityReferenceFrames: original,
	}, generated)
	if err != nil {
		t.Fatalf("expected visible additive gesture to pass: %v", err)
	}
}

func TestValidateEditFrameContinuityAllowsLargeMotionInsideTargetInterval(t *testing.T) {
	foreground := color.NRGBA{R: 35, B: 10, A: 255}
	original := []image.Image{
		animationContinuityFrame(28, 36, foreground),
		animationContinuityFrame(28, 36, foreground),
		animationContinuityFrame(28, 36, foreground),
		animationContinuityFrame(28, 36, foreground),
		animationContinuityFrame(28, 36, foreground),
	}
	generated := []image.Image{
		original[0],
		animationContinuityGestureFrame(28, 40, 70, foreground),
		animationContinuityGestureFrame(28, 60, 70, foreground),
		animationContinuityGestureFrame(28, 40, 70, foreground),
		original[4],
	}

	err := validateEditFrameContinuity(AnimationGenerationRequest{
		ReferenceImageContext:     true,
		TargetFrameIndices:        []int{1, 2, 3},
		FrameCount:                len(original),
		FPS:                       10,
		continuityReferenceFrames: original,
	}, generated)
	if err != nil {
		t.Fatalf("expected clear internal target motion with smooth seams to pass: %v", err)
	}
}

func TestValidateEditFrameApplicationRejectsSparseTokenChange(t *testing.T) {
	targets := []bool{false, true, true, true, false}
	differences := make([]videoprocessor.FramePairDifference, len(targets))
	differences[2] = videoprocessor.FramePairDifference{ForegroundMaskDifference: 0.2, AppearanceMSE: 0.1}

	err := validateEditFrameApplication(targets, differences)
	var qualityError *videoprocessor.QualityError
	if !errors.As(err, &qualityError) || qualityError.Kind != "edit_application" ||
		!strings.Contains(err.Error(), "only 1 of 3 target frames") || !strings.Contains(err.Error(), "need 2") {
		t.Fatalf("expected one-token-frame edit to be rejected, got %v", err)
	}
}

func TestValidateEditFrameContinuityRejectsDisorderedMotionPhases(t *testing.T) {
	foreground := color.NRGBA{R: 35, B: 10, A: 255}
	originalHeights := []int{50, 55, 60, 65, 60, 55, 50}
	candidateHeights := []int{50, 65, 52, 64, 51, 62, 50}
	original := make([]image.Image, len(originalHeights))
	generated := make([]image.Image, len(candidateHeights))
	targets := make([]int, len(candidateHeights))
	for index := range originalHeights {
		original[index] = animationContinuitySizedFrame(28, 36, originalHeights[index], foreground)
		generated[index] = animationContinuitySizedFrame(28, 36, candidateHeights[index], foreground)
		targets[index] = index
	}

	err := validateEditFrameContinuity(AnimationGenerationRequest{
		ReferenceImageContext:     true,
		TargetFrameIndices:        targets,
		FrameCount:                len(targets),
		FPS:                       10,
		continuityReferenceFrames: original,
	}, generated)
	var qualityError *videoprocessor.QualityError
	if !errors.As(err, &qualityError) || qualityError.Kind != "temporal_coherence" ||
		!strings.Contains(err.Error(), "vertical pose extent changes direction") {
		t.Fatalf("expected disordered motion phases to be rejected, got %v", err)
	}
}

func TestValidateEditFrameTemporalCoherenceAcceptsSingleOrderedMotionInterval(t *testing.T) {
	foreground := color.NRGBA{R: 35, B: 10, A: 255}
	heights := []int{50, 55, 60, 65, 60, 55, 50}
	frames := make([]image.Image, len(heights))
	targets := make([]bool, len(heights))
	for index, height := range heights {
		frames[index] = animationContinuitySizedFrame(28, 36, height, foreground)
		targets[index] = true
	}
	analysis, err := videoprocessor.AnalyzeFrameSequence(frames, 10, animationVideoChromaKey())
	if err != nil {
		t.Fatal(err)
	}
	if err := validateEditFrameTemporalCoherence(targets, analysis, analysis); err != nil {
		t.Fatalf("expected one ordered motion interval to pass: %v", err)
	}
}

func TestValidateEditFrameTemporalCoherenceAllowsLayeredHorizontalMotion(t *testing.T) {
	original := animationTemporalTestAnalysis([]float64{40, 42, 44, 46, 48, 50, 52, 52, 52, 52})
	candidate := animationTemporalTestAnalysis([]float64{40, 50, 42, 51, 43, 52, 52, 52, 52, 52})
	targets := make([]bool, len(candidate.Frames))
	for index := range targets {
		targets[index] = true
	}

	if err := validateEditFrameTemporalCoherence(targets, original, candidate); err != nil {
		t.Fatalf("expected four horizontal reversals from layered arm motion to pass: %v", err)
	}
}

func TestValidateEditFrameTemporalCoherenceRejectsExcessiveHorizontalRestarts(t *testing.T) {
	original := animationTemporalTestAnalysis([]float64{40, 42, 44, 46, 48, 50, 52, 54, 56})
	candidate := animationTemporalTestAnalysis([]float64{40, 50, 42, 51, 43, 52, 44, 53, 45})
	targets := make([]bool, len(candidate.Frames))
	for index := range targets {
		targets[index] = true
	}

	err := validateEditFrameTemporalCoherence(targets, original, candidate)
	var qualityError *videoprocessor.QualityError
	if !errors.As(err, &qualityError) || qualityError.Kind != "temporal_coherence" ||
		!strings.Contains(err.Error(), "horizontal pose extent changes direction 7 times; expected at most 6") {
		t.Fatalf("expected excessive horizontal restarts to be rejected, got %v", err)
	}
}

func TestValidateEditFrameTemporalCoherenceAllowsModeratePoseRecurrence(t *testing.T) {
	original := animationTemporalRecurrenceAnalysis(0.0100, 0.0038)
	candidate := animationTemporalRecurrenceAnalysis(0.0400, 0.0072)
	targets := []bool{true, true, true, true}

	if got := animationTemporalNeighborDisorder(candidate, 0, 3); math.Abs(got-0.0328) > 1e-9 {
		t.Fatalf("test fixture has recurrence %.4f; expected 0.0328", got)
	}
	if err := validateEditFrameTemporalCoherence(targets, original, candidate); err != nil {
		t.Fatalf("expected moderate repeated-pose similarity to pass: %v", err)
	}
}

func TestValidateEditFrameTemporalCoherenceRejectsSeverePoseRecurrence(t *testing.T) {
	original := animationTemporalRecurrenceAnalysis(0.0100, 0.0038)
	candidate := animationTemporalRecurrenceAnalysis(0.1000, 0.0050)
	targets := []bool{true, true, true, true}

	err := validateEditFrameTemporalCoherence(targets, original, candidate)
	var qualityError *videoprocessor.QualityError
	if !errors.As(err, &qualityError) || qualityError.Kind != "temporal_coherence" ||
		!strings.Contains(err.Error(), "non-neighbor pose recurrence 0.0950 exceeds 0.0500") {
		t.Fatalf("expected severe repeated-pose disorder rejection, got %v", err)
	}
}

func animationTemporalRecurrenceAnalysis(adjacent, remote float64) videoprocessor.FrameSequenceAnalysis {
	analysis := animationTemporalTestAnalysis([]float64{40, 40, 40, 40})
	for left := range analysis.PairwiseMSE {
		for right := range left {
			value := remote
			if left-right == 1 {
				value = adjacent
			}
			analysis.PairwiseMSE[left][right] = value
			analysis.PairwiseMSE[right][left] = value
		}
	}
	return analysis
}

func TestAnimationGenerationRetriesMissingRequestedEdit(t *testing.T) {
	foreground := color.NRGBA{R: 35, B: 10, A: 255}
	original := []image.Image{
		animationContinuityFrame(28, 36, foreground),
		animationContinuityFrame(29, 36, foreground),
		animationContinuityFrame(30, 36, foreground),
	}
	contextReferences := make([]string, len(original))
	for index, frame := range original {
		contextReferences[index] = animationTestImageDataURL(t, frame)
	}
	visibleEdit := append([]image.Image(nil), original...)
	visibleEdit[1] = animationContinuityGestureFrame(29, 36, 70, foreground)
	videos := &animationVideoServiceStub{}
	processorForeground := animationTestForeground(t)
	processor := &animationProcessorStub{
		foregroundBase64: processorForeground,
		splitResult: &imageprocessor.SplitImageResult{
			ImageBase64: "sheet", MIMEType: "image/png",
			Regions: []imageprocessor.ImageRegion{
				{Index: 0, ImageBase64: processorForeground},
				{Index: 1, ImageBase64: processorForeground},
				{Index: 2, ImageBase64: processorForeground},
			},
		},
	}
	videoProcessor := &animationVideoProcessorStub{results: []*videoprocessor.Result{
		{Frames: append([]image.Image(nil), original...)},
		{Frames: visibleEdit},
	}}
	service := newAnimationGenerationService(videos, processor, videoProcessor)

	result, err := service.Generate(context.Background(), &AnimationGenerationRequest{
		Description:            "greeter",
		Action:                 "put the other hand near the mouth in a shush gesture",
		OriginalAction:         "raise the hat in greeting",
		ReferenceImage:         contextReferences[0],
		EndReferenceImage:      contextReferences[2],
		ReferenceImageContext:  true,
		TargetFrameIndices:     []int{1},
		ContextReferenceImages: contextReferences,
		FrameCount:             3,
		Columns:                3,
		FrameWidth:             64,
		FrameHeight:            64,
		FPS:                    10,
	})
	if err != nil {
		t.Fatalf("generate edited frame segment: %v", err)
	}
	if result.VideoAttempts != 2 || len(videos.requests) != 2 {
		t.Fatalf("missing edit did not retry: result=%+v requests=%d", result, len(videos.requests))
	}
	if !strings.Contains(videos.requests[1].Prompt, "failed to visibly perform the requested addition") ||
		!strings.Contains(videos.requests[1].Prompt, "do not return the unedited original motion") {
		t.Fatalf("edit application retry prompt was not applied: %s", videos.requests[1].Prompt)
	}
}

func TestAnimationGenerationRetriesContinuityFailure(t *testing.T) {
	original := []image.Image{
		animationContinuityFrame(28, 36, color.NRGBA{R: 35, B: 10, A: 255}),
		animationContinuityFrame(29, 36, color.NRGBA{R: 35, B: 10, A: 255}),
		animationContinuityFrame(30, 36, color.NRGBA{R: 35, B: 10, A: 255}),
	}
	contextReferences := make([]string, len(original))
	for index, frame := range original {
		contextReferences[index] = animationTestImageDataURL(t, frame)
	}
	abrupt := []image.Image{
		original[0],
		animationContinuityFrame(29, 36, color.NRGBA{R: 255, G: 255, B: 255, A: 255}),
		original[2],
	}
	smooth := []image.Image{
		original[0],
		animationContinuityFrame(30, 36, color.NRGBA{R: 35, B: 10, A: 255}),
		original[2],
	}
	videos := &animationVideoServiceStub{}
	processorForeground := animationTestForeground(t)
	processor := &animationProcessorStub{
		foregroundBase64: processorForeground,
		splitResult: &imageprocessor.SplitImageResult{
			ImageBase64: "sheet", MIMEType: "image/png",
			Regions: []imageprocessor.ImageRegion{
				{Index: 0, ImageBase64: processorForeground},
				{Index: 1, ImageBase64: processorForeground},
				{Index: 2, ImageBase64: processorForeground},
			},
		},
	}
	videoProcessor := &animationVideoProcessorStub{results: []*videoprocessor.Result{
		{Frames: abrupt},
		{Frames: smooth},
	}}
	service := newAnimationGenerationService(videos, processor, videoProcessor)

	result, err := service.Generate(context.Background(), &AnimationGenerationRequest{
		Description:            "knight",
		Action:                 "raise the sword slightly",
		ReferenceImage:         contextReferences[0],
		EndReferenceImage:      contextReferences[2],
		ReferenceImageContext:  true,
		TargetFrameIndices:     []int{1},
		ContextReferenceImages: contextReferences,
		FrameCount:             3,
		Columns:                3,
		FrameWidth:             64,
		FrameHeight:            64,
		FPS:                    10,
	})
	if err != nil {
		t.Fatalf("generate edited frame segment: %v", err)
	}
	if result.VideoAttempts != 2 || len(videos.requests) != 2 {
		t.Fatalf("continuity failure did not retry: result=%+v requests=%d", result, len(videos.requests))
	}
	if !strings.Contains(videos.requests[1].Prompt, "abrupt motion discontinuity") ||
		!strings.Contains(videos.requests[1].Prompt, "no sudden root displacement") {
		t.Fatalf("continuity retry prompt was not applied: %s", videos.requests[1].Prompt)
	}
}

func animationTemporalTestAnalysis(widths []float64) videoprocessor.FrameSequenceAnalysis {
	frames := make([]videoprocessor.FrameObservation, len(widths))
	pairwise := make([][]float64, len(widths))
	for index, width := range widths {
		frames[index] = videoprocessor.FrameObservation{
			Safe: true, CentroidX: 32, CentroidY: 32,
			Width: width, Height: 50, ForegroundArea: 1000,
		}
		pairwise[index] = make([]float64, len(widths))
	}
	return videoprocessor.FrameSequenceAnalysis{
		FPS: 10, Frames: frames, PairwiseMSE: pairwise, ForegroundRatio: .25,
	}
}

func animationContinuityGestureFrame(x, width, height int, foreground color.NRGBA) image.Image {
	frame := animationContinuitySizedFrame(x, width, height, foreground).(*image.NRGBA)
	draw.Draw(frame, image.Rect(x+width, 44, x+width+10, 54), &image.Uniform{C: foreground}, image.Point{}, draw.Src)
	return frame
}

func animationContinuityFrame(x, width int, foreground color.NRGBA) image.Image {
	return animationContinuitySizedFrame(x, width, 70, foreground)
}

func animationContinuitySizedFrame(x, width, height int, foreground color.NRGBA) image.Image {
	frame := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(frame, image.Rect(x, 88-height, x+width, 88), &image.Uniform{C: foreground}, image.Point{}, draw.Src)
	return frame
}
