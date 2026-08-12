package video

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
)

type animationFrameDescriptor struct {
	mask       [animationAnalysisSize * animationAnalysisSize]bool
	gray       [animationAnalysisSize * animationAnalysisSize]float64
	cx         float64
	cy         float64
	width      float64
	height     float64
	foreground int
}

type animationLoopCandidate struct {
	start             int
	end               int
	score             float64
	endpoint          float64
	richness          float64
	poseCoverage      float64
	spanRatio         float64
	centroidStability float64
	seamMSE           float64
}

func selectAnimationLoopFrames(
	frames []image.Image,
	count int,
	fps int,
) ([]int, AnimationLoopSelection, error) {
	if count <= 0 || len(frames) < count+1 {
		return nil, AnimationLoopSelection{}, fmt.Errorf("video: video has %d candidate frames; need at least %d", len(frames), count+1)
	}
	descriptors := make([]animationFrameDescriptor, len(frames))
	var union [animationAnalysisSize * animationAnalysisSize]bool
	for index := range frames {
		descriptors[index] = describeAnimationFrame(frames[index])
		for pixel, visible := range descriptors[index].mask {
			union[pixel] = union[pixel] || visible
		}
	}
	unionArea := 0
	for _, visible := range union {
		if visible {
			unionArea++
		}
	}
	if float64(unionArea)/float64(len(union)) < .05 {
		return nil, AnimationLoopSelection{}, &AnimationVideoQualityError{
			Kind:    "subject",
			Message: fmt.Sprintf("video: green-screen subject separation failed: foreground ratio %.3f", float64(unionArea)/float64(len(union))),
		}
	}

	mse := make([][]float64, len(descriptors))
	for i := range mse {
		mse[i] = make([]float64, len(descriptors))
		for j := range i {
			mse[i][j] = animationSubjectMSE(descriptors[i], descriptors[j], &union, unionArea)
			mse[j][i] = mse[i][j]
		}
	}

	minSpan := animationMaxInt(count, animationMaxInt(4, int(math.Ceil(float64(len(frames))*animationMinLoopSpanRatio))))
	// Framing is part of loop selection, not a post-selection rejection. The
	// selector also stays near the beginning of the source video so the output
	// preserves the provider's intended order: initial pose, preparation, main
	// action, follow-through, and recovery. Searching the whole video can find a
	// mathematically similar later interval and make the animation start in the
	// middle of the action.
	unsafePrefix := make([]int, len(frames)+1)
	for index, frame := range frames {
		unsafePrefix[index+1] = unsafePrefix[index]
		if !animationFrameInsideSafetyBand(frame) {
			unsafePrefix[index+1]++
		}
	}
	type pair struct{ start, end int }
	pairs := make([]pair, 0, len(frames)*len(frames)/2)
	endpointMSE := make([]float64, 0, cap(pairs))
	initialWindow := animationMaxInt(1, int(math.Ceil(float64(len(frames))*animationInitialWindowRatio)))
	for start := 0; start < len(frames) && start <= initialWindow; start++ {
		for end := start + minSpan; end < len(frames); end++ {
			// The source video may contain a transient clipped or blurred frame at
			// full extension. It must not be exported, but it should not invalidate
			// an otherwise complete action interval. Require only the frames that
			// will actually become sprite frames to stay inside the safety band;
			// use the full interval for action-coverage scoring below.
			sampled := sampleAnimationIndices(start, end, count)
			allSamplesSafe := true
			for _, sampleIndex := range sampled {
				if unsafePrefix[sampleIndex+1]-unsafePrefix[sampleIndex] != 0 {
					allSamplesSafe = false
					break
				}
			}
			if !allSamplesSafe {
				continue
			}
			pairs = append(pairs, pair{start: start, end: end})
			endpointMSE = append(endpointMSE, mse[start][end])
		}
	}
	if len(pairs) == 0 {
		return nil, AnimationLoopSelection{}, &AnimationVideoQualityError{
			Kind:    "framing",
			Message: fmt.Sprintf("video: no full-action interval has %d sampled frames inside the outer 2.5%% safety band; interval still needs at least %.0f%% of the source duration", count, animationMinLoopSpanRatio*100),
		}
	}
	// Frame zero is the canonical pose supplied to the video model. If it can
	// participate in a valid interval, never move the animation start into the
	// middle of preparation just because a later pose happens to be more similar
	// to a later recovery frame. Only use the small initial-window fallback when
	// frame zero cannot satisfy the safety check.
	hasInitialCandidate := false
	for _, candidate := range pairs {
		if candidate.start == 0 {
			hasInitialCandidate = true
			break
		}
	}
	if hasInitialCandidate {
		filteredPairs := make([]pair, 0, len(pairs))
		filteredEndpointMSE := make([]float64, 0, len(endpointMSE))
		for index, candidate := range pairs {
			if candidate.start != 0 {
				continue
			}
			filteredPairs = append(filteredPairs, candidate)
			filteredEndpointMSE = append(filteredEndpointMSE, endpointMSE[index])
		}
		pairs, endpointMSE = filteredPairs, filteredEndpointMSE
	}
	threshold := animationQuantile(endpointMSE, .35)
	adjacent := make([]float64, 0, len(frames)-1)
	for index := 0; index+1 < len(frames); index++ {
		adjacent = append(adjacent, mse[index][index+1])
	}
	richnessScale := math.Max(animationQuantile(adjacent, .90), 1e-6)
	motionBaseline := animationQuantile(adjacent, .25)
	activeMotion := make([]float64, len(adjacent))
	var totalActiveMotion float64
	for index, energy := range adjacent {
		// Provider video compression and chroma-key noise create small changes
		// even while the character is idle. Do not let that idle noise make the
		// trailing hold look like part of the requested action.
		activeMotion[index] = math.Max(energy-motionBaseline, 0)
		totalActiveMotion += activeMotion[index]
	}
	globalVariation := math.Max(animationDescriptorVariation(descriptors), 1e-6)

	best := animationLoopCandidate{score: math.Inf(-1)}
	for index, pair := range pairs {
		if endpointMSE[index] > threshold {
			continue
		}
		var richness float64
		var intervalActiveMotion float64
		for frame := pair.start; frame < pair.end; frame++ {
			richness += mse[frame][frame+1]
			intervalActiveMotion += activeMotion[frame]
		}
		richness /= float64(pair.end - pair.start)
		richnessNormalized := math.Min(richness/richnessScale, 1)
		motionCoverage := 1.0
		if totalActiveMotion > 1e-9 {
			motionCoverage = animationClampFloat(intervalActiveMotion/totalActiveMotion, 0, 1)
		}
		centroidStability := 1 / (1 + animationCentroidStd(descriptors[pair.start:pair.end+1]))
		translationBonus := animationStableTranslationBonus(descriptors[pair.start : pair.end+1])
		endpointSimilarity := 1 - endpointMSE[index]
		initialSimilarity := animationClampFloat(1-mse[0][pair.start], 0, 1)
		spanRatio := float64(pair.end-pair.start) / float64(len(frames)-1)
		poseCoverage := math.Min(animationDescriptorVariation(descriptors[pair.start:pair.end+1])/globalVariation, 1)
		// A complete cycle is still sampled in source order. The duration score is
		// deliberately stronger than before: once the endpoint has returned to
		// the initial pose and the interval contains the motion energy, an idle
		// tail must not receive extra frames just because it makes the interval
		// longer. This is the main protection against [start, action, idle] being
		// uniformly reduced to a few confusing poses.
		endpointMotion := 0.0
		if pair.end+1 < len(frames) {
			endpointMotion = animationClampFloat(mse[pair.end][pair.end+1]/richnessScale, 0, 1)
		}
		recoveryStability := 1 - endpointMotion
		score := endpointSimilarity + .45*richnessNormalized + .45*centroidStability + .2*translationBonus +
			animationInitialPoseWeight*initialSimilarity + animationLoopCompactnessWeight*(1-spanRatio) +
			.65*poseCoverage + animationLoopMotionWeight*motionCoverage + animationLoopRecoveryWeight*recoveryStability
		if score > best.score {
			best = animationLoopCandidate{
				start: pair.start, end: pair.end, score: score,
				endpoint: endpointSimilarity, richness: richness,
				poseCoverage: poseCoverage, spanRatio: spanRatio,
				centroidStability: centroidStability, seamMSE: endpointMSE[index],
			}
		}
	}
	if math.IsInf(best.score, -1) {
		return nil, AnimationLoopSelection{}, fmt.Errorf("video: full-action loop search produced no candidate")
	}
	warning := ""
	if best.seamMSE > .015 {
		warning = fmt.Sprintf("subject seam MSE %.4f exceeds 0.015; inspect or regenerate the video", best.seamMSE)
	}
	return sampleAnimationIndices(best.start, best.end, count), AnimationLoopSelection{
		CandidateFPS: fps, StartFrame: best.start, EndFrame: best.end,
		SpanFrames: best.end - best.start, Score: animationRoundTo(best.score, 6),
		EndpointSimilarity: animationRoundTo(best.endpoint, 6), Richness: animationRoundTo(best.richness, 6),
		PoseCoverage: animationRoundTo(best.poseCoverage, 6), SpanRatio: animationRoundTo(best.spanRatio, 6),
		CentroidStability: animationRoundTo(best.centroidStability, 6), SeamWarning: warning,
		Method: "subject_mse_full_cycle",
	}, nil
}

func describeAnimationFrame(source image.Image) animationFrameDescriptor {
	bounds := source.Bounds()
	var descriptor animationFrameDescriptor
	var sumX, sumY float64
	minX, maxX := animationAnalysisSize, -1
	minY, maxY := animationAnalysisSize, -1
	for y := range animationAnalysisSize {
		for x := range animationAnalysisSize {
			sourceX := bounds.Min.X + animationMinInt(bounds.Dx()-1, int((float64(x)+.5)*float64(bounds.Dx())/animationAnalysisSize))
			sourceY := bounds.Min.Y + animationMinInt(bounds.Dy()-1, int((float64(y)+.5)*float64(bounds.Dy())/animationAnalysisSize))
			value := color.NRGBAModel.Convert(source.At(sourceX, sourceY)).(color.NRGBA)
			if isAnimationGreen(value) {
				continue
			}
			pixel := y*animationAnalysisSize + x
			descriptor.mask[pixel] = true
			descriptor.gray[pixel] = (.299*float64(value.R) + .587*float64(value.G) + .114*float64(value.B)) / 255
			descriptor.foreground++
			sumX += float64(x)
			sumY += float64(y)
			minX, maxX = animationMinInt(minX, x), animationMaxInt(maxX, x)
			minY, maxY = animationMinInt(minY, y), animationMaxInt(maxY, y)
		}
	}
	if descriptor.foreground > 0 {
		descriptor.cx = sumX / float64(descriptor.foreground)
		descriptor.cy = sumY / float64(descriptor.foreground)
		descriptor.width = float64(maxX - minX + 1)
		descriptor.height = float64(maxY - minY + 1)
	} else {
		descriptor.cx, descriptor.cy = math.NaN(), math.NaN()
	}
	return descriptor
}

func animationSubjectMSE(
	a animationFrameDescriptor,
	b animationFrameDescriptor,
	union *[animationAnalysisSize * animationAnalysisSize]bool,
	unionArea int,
) float64 {
	if unionArea == 0 {
		return 1
	}
	var sum float64
	for index, include := range union {
		if !include {
			continue
		}
		delta := a.gray[index] - b.gray[index]
		sum += delta * delta
	}
	return sum / float64(unionArea)
}

func sampleAnimationIndices(start, end, count int) []int {
	if count <= 0 {
		return nil
	}
	if count == 1 {
		return []int{start}
	}
	span := end - start
	indices := make([]int, count)
	for index := range count {
		// Include the recovery endpoint. The old count-based phase divisor
		// stopped at roughly 87.5%% of the interval, which could omit the
		// requested return to the initial pose.
		phase := float64(index) / float64(count-1)
		target := start + int(math.Round(phase*float64(span)))
		minTarget := start + index
		maxTarget := end - (count - 1 - index)
		indices[index] = animationClampInt(target, minTarget, maxTarget)
	}
	return indices
}
