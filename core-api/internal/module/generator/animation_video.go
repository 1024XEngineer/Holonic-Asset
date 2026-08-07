package generator

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	animationCandidateFPS          = 12
	animationAnalysisSize          = 48
	animationMinLoopSpanRatio      = 0.50
	animationInitialWindowRatio    = 0.20
	animationInitialPoseWeight     = 0.45
	animationLoopCompactnessWeight = 1.15
	animationLoopRecoveryWeight    = 0.35
	animationLoopMotionWeight      = 0.65
)

type AnimationLoopSelection struct {
	CandidateFPS       int     `json:"candidate_fps"`
	StartFrame         int     `json:"start_frame"`
	EndFrame           int     `json:"end_frame"`
	SpanFrames         int     `json:"span_frames"`
	Score              float64 `json:"score"`
	EndpointSimilarity float64 `json:"endpoint_similarity"`
	Richness           float64 `json:"richness"`
	PoseCoverage       float64 `json:"pose_coverage"`
	SpanRatio          float64 `json:"span_ratio"`
	CentroidStability  float64 `json:"centroid_stability"`
	SeamWarning        string  `json:"seam_warning,omitempty"`
	Method             string  `json:"method"`
}

type AnimationVideoQualityError struct {
	Kind    string
	Message string
}

func (e *AnimationVideoQualityError) Error() string { return e.Message }

type animationFrameExtractor interface {
	Extract(context.Context, []byte, int) ([]image.Image, error)
}

type ffmpegAnimationFrameExtractor struct {
	path string
}

func (e ffmpegAnimationFrameExtractor) Extract(
	ctx context.Context,
	video []byte,
	fps int,
) ([]image.Image, error) {
	ffmpeg, err := resolveFFmpeg(e.path)
	if err != nil {
		return nil, err
	}
	temp, err := os.MkdirTemp("", "holonic-animation-video-*")
	if err != nil {
		return nil, fmt.Errorf("generator: create video frame temp directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(temp)
	}()

	input := filepath.Join(temp, "input.mp4")
	if err := os.WriteFile(input, video, 0o600); err != nil {
		return nil, fmt.Errorf("generator: write temporary video: %w", err)
	}
	pattern := filepath.Join(temp, "frame_%05d.png")
	// The executable is either an explicitly configured ffmpeg binary or the
	// result of exec.LookPath; request data is passed as fixed arguments.
	command := exec.CommandContext( //nolint:gosec // Variable executable path is intentionally validated by resolveFFmpeg.
		ctx,
		ffmpeg,
		"-hide_banner", "-loglevel", "error",
		"-i", input,
		"-vf", fmt.Sprintf("fps=%d", fps),
		"-vsync", "0",
		pattern,
	)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("generator: ffmpeg extract animation frames: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	paths, err := filepath.Glob(filepath.Join(temp, "frame_*.png"))
	if err != nil {
		return nil, fmt.Errorf("generator: list extracted animation frames: %w", err)
	}
	sort.Strings(paths)
	frames := make([]image.Image, 0, len(paths))
	for _, path := range paths {
		// paths only contains entries produced by filepath.Glob inside temp.
		file, openErr := os.Open(path) //nolint:gosec // The path is constrained to the private temporary directory.
		if openErr != nil {
			return nil, fmt.Errorf("generator: open extracted animation frame: %w", openErr)
		}
		frame, _, decodeErr := image.Decode(file)
		closeErr := file.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("generator: decode extracted animation frame: %w", decodeErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("generator: close extracted animation frame: %w", closeErr)
		}
		frames = append(frames, frame)
	}
	if len(frames) < 2 {
		return nil, fmt.Errorf("generator: video yielded only %d decodable frame(s)", len(frames))
	}
	return frames, nil
}

func resolveFFmpeg(configured string) (string, error) {
	path := strings.TrimSpace(configured)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("FFMPEG_PATH"))
	}
	if path != "" {
		// A caller may intentionally configure an ffmpeg binary outside PATH.
		info, err := os.Stat(path) //nolint:gosec // This is an operator-supplied executable path, not request input.
		if err == nil && !info.IsDir() {
			return path, nil
		}
		return "", fmt.Errorf("generator: FFMPEG_PATH does not point to a file: %s", path)
	}
	found, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("generator: ffmpeg is required for video frame extraction; install it or set FFMPEG_PATH")
	}
	return found, nil
}

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
		return nil, AnimationLoopSelection{}, fmt.Errorf("generator: video has %d candidate frames; need at least %d", len(frames), count+1)
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
			Message: fmt.Sprintf("generator: green-screen subject separation failed: foreground ratio %.3f", float64(unionArea)/float64(len(union))),
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
			Message: fmt.Sprintf("generator: no full-action interval has %d sampled frames inside the outer 2.5%% safety band; interval still needs at least %.0f%% of the source duration", count, animationMinLoopSpanRatio*100),
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
		return nil, AnimationLoopSelection{}, fmt.Errorf("generator: full-action loop search produced no candidate")
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

func validateAnimationMotionSafeAreaAtIndices(frames []image.Image, indices []int) error {
	for _, sourceIndex := range indices {
		if sourceIndex < 0 || sourceIndex >= len(frames) {
			return fmt.Errorf("generator: sampled animation frame index %d is out of range", sourceIndex)
		}
		frame := frames[sourceIndex]
		bounds, ok := animationRawForegroundBounds(frame)
		if !ok {
			return &AnimationVideoQualityError{Kind: "subject", Message: fmt.Sprintf("generator: video frame %d has no detectable subject on the green screen", sourceIndex)}
		}
		if !animationBoundsInsideSafetyBand(frame, bounds) {
			return &AnimationVideoQualityError{Kind: "framing", Message: fmt.Sprintf("generator: character, limb, or held object enters the outer 2.5%% safety band in source frame %d", sourceIndex)}
		}
	}
	return nil
}

func animationFrameInsideSafetyBand(frame image.Image) bool {
	bounds, ok := animationRawForegroundBounds(frame)
	return ok && animationBoundsInsideSafetyBand(frame, bounds)
}

func animationBoundsInsideSafetyBand(frame image.Image, foreground image.Rectangle) bool {
	frameBounds := frame.Bounds()
	margin := animationMaxInt(4, int(math.Round(float64(animationMinInt(frameBounds.Dx(), frameBounds.Dy()))*.025)))
	return foreground.Min.X > frameBounds.Min.X+margin &&
		foreground.Min.Y > frameBounds.Min.Y+margin &&
		foreground.Max.X < frameBounds.Max.X-margin &&
		foreground.Max.Y < frameBounds.Max.Y-margin
}

func animationRawForegroundBounds(source image.Image) (image.Rectangle, bool) {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width == 0 || height == 0 {
		return image.Rectangle{}, false
	}
	columns := make([]int, width)
	rows := make([]int, height)
	for y := range height {
		for x := range width {
			if isAnimationGreen(color.NRGBAModel.Convert(source.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.NRGBA)) {
				continue
			}
			columns[x]++
			rows[y]++
		}
	}
	lineThreshold := animationMaxInt(2, animationMinInt(width, height)/320)
	minX, maxX := width, -1
	minY, maxY := height, -1
	for x, count := range columns {
		if count >= lineThreshold {
			minX, maxX = animationMinInt(minX, x), animationMaxInt(maxX, x)
		}
	}
	for y, count := range rows {
		if count >= lineThreshold {
			minY, maxY = animationMinInt(minY, y), animationMaxInt(maxY, y)
		}
	}
	if maxX < minX || maxY < minY {
		return image.Rectangle{}, false
	}
	return image.Rect(bounds.Min.X+minX, bounds.Min.Y+minY, bounds.Min.X+maxX+1, bounds.Min.Y+maxY+1), true
}

func isAnimationGreen(value color.NRGBA) bool {
	hue, saturation, brightness := animationRGBToOpenCVHSV(value.R, value.G, value.B)
	return hue >= 30 && hue <= 90 && ((saturation >= 80 && brightness >= 80) || (saturation >= 50 && brightness >= 180))
}

func animationRGBToOpenCVHSV(red8, green8, blue8 uint8) (uint8, uint8, uint8) {
	red, green, blue := float64(red8)/255, float64(green8)/255, float64(blue8)/255
	maximum, minimum := math.Max(red, math.Max(green, blue)), math.Min(red, math.Min(green, blue))
	delta := maximum - minimum
	var hue float64
	if delta != 0 {
		switch maximum {
		case red:
			hue = 60 * math.Mod((green-blue)/delta, 6)
		case green:
			hue = 60 * ((blue-red)/delta + 2)
		default:
			hue = 60 * ((red-green)/delta + 4)
		}
	}
	if hue < 0 {
		hue += 360
	}
	saturation := 0.0
	if maximum > 0 {
		saturation = delta / maximum
	}
	return uint8(math.Round(hue / 2)), uint8(math.Round(saturation * 255)), uint8(math.Round(maximum * 255))
}

func animationDescriptorVariation(frames []animationFrameDescriptor) float64 {
	if len(frames) < 2 {
		return 0
	}
	widths := make([]float64, 0, len(frames))
	heights := make([]float64, 0, len(frames))
	areas := make([]float64, 0, len(frames))
	for _, frame := range frames {
		if frame.foreground == 0 {
			continue
		}
		widths = append(widths, frame.width)
		heights = append(heights, frame.height)
		areas = append(areas, float64(frame.foreground))
	}
	return animationStandardDeviation(widths)/animationAnalysisSize +
		animationStandardDeviation(heights)/animationAnalysisSize +
		animationStandardDeviation(areas)/(animationAnalysisSize*animationAnalysisSize)
}

func animationCentroidStd(frames []animationFrameDescriptor) float64 {
	xs := make([]float64, 0, len(frames))
	ys := make([]float64, 0, len(frames))
	for _, frame := range frames {
		if !math.IsNaN(frame.cx) && !math.IsNaN(frame.cy) {
			xs = append(xs, frame.cx)
			ys = append(ys, frame.cy)
		}
	}
	return animationStandardDeviation(xs) + animationStandardDeviation(ys)
}

func animationStableTranslationBonus(frames []animationFrameDescriptor) float64 {
	if len(frames) < 3 {
		return 0
	}
	var count, sumX, sumY, sumXX, sumXY float64
	for index, frame := range frames {
		if math.IsNaN(frame.cx) {
			continue
		}
		x := float64(index)
		count++
		sumX += x
		sumY += frame.cx
		sumXX += x * x
		sumXY += x * frame.cx
	}
	denominator := count*sumXX - sumX*sumX
	if count < 3 || math.Abs(denominator) < 1e-9 {
		return 0
	}
	slope := (count*sumXY - sumX*sumY) / denominator
	intercept := (sumY - slope*sumX) / count
	var squared float64
	for index, frame := range frames {
		if math.IsNaN(frame.cx) {
			continue
		}
		residual := frame.cx - (intercept + slope*float64(index))
		squared += residual * residual
	}
	if math.Sqrt(squared/count) < 2 {
		return 1
	}
	return 0
}

func animationStandardDeviation(values []float64) float64 {
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

func animationQuantile(values []float64, quantile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	position := animationClampFloat(quantile, 0, 1) * float64(len(ordered)-1)
	low, high := int(math.Floor(position)), int(math.Ceil(position))
	if low == high {
		return ordered[low]
	}
	fraction := position - float64(low)
	return ordered[low]*(1-fraction) + ordered[high]*fraction
}

func animationMinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func animationMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func animationClampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func animationClampFloat(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func animationRoundTo(value float64, places int) float64 {
	factor := math.Pow10(places)
	return math.Round(value*factor) / factor
}
