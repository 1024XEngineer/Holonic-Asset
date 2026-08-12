package video

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultCandidateFPS            = 12
	animationAnalysisSize          = 48
	animationMinLoopSpanRatio      = 0.50
	animationInitialWindowRatio    = 0.20
	animationInitialPoseWeight     = 0.45
	animationLoopCompactnessWeight = 1.15
	animationLoopRecoveryWeight    = 0.35
	animationLoopMotionWeight      = 0.65

	maxAnimationExtractedFrames   = 100
	maxAnimationFrameDimension    = 4096
	maxAnimationDecodedFrameBytes = 128 << 20
	animationDecodedBytesPerPixel = 4
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

// Processor extracts, validates, and selects an animation loop from a video.
type Processor interface {
	Process(context.Context, []byte, int) (*Result, error)
}

// Result contains the selected source frames and loop-selection metadata.
type Result struct {
	Frames []image.Image
	Loop   AnimationLoopSelection
}

type frameExtractor interface {
	Extract(context.Context, []byte, int) ([]image.Image, error)
}

type processor struct {
	extractor frameExtractor
}

// NewProcessor creates the deterministic video processor backed by FFmpeg.
func NewProcessor() Processor {
	return &processor{extractor: ffmpegFrameExtractor{}}
}

func newProcessor(extractor frameExtractor) Processor {
	return &processor{extractor: extractor}
}

func (p *processor) Process(ctx context.Context, source []byte, frameCount int) (*Result, error) {
	if p.extractor == nil {
		return nil, fmt.Errorf("video: video frame extractor is required")
	}
	frames, err := p.extractor.Extract(ctx, source, defaultCandidateFPS)
	if err != nil {
		return nil, err
	}
	indices, loop, err := selectAnimationLoopFrames(frames, frameCount, defaultCandidateFPS)
	if err != nil {
		return nil, err
	}
	if err := validateAnimationMotionSafeAreaAtIndices(frames, indices); err != nil {
		return nil, err
	}
	selected := make([]image.Image, 0, len(indices))
	for _, index := range indices {
		selected = append(selected, frames[index])
	}
	return &Result{Frames: selected, Loop: loop}, nil
}

type ffmpegFrameExtractor struct {
	path string
}

func (e ffmpegFrameExtractor) Extract(
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
		return nil, fmt.Errorf("video: create video frame temp directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(temp)
	}()

	input := filepath.Join(temp, "input.mp4")
	if err := os.WriteFile(input, video, 0o600); err != nil {
		return nil, fmt.Errorf("video: write temporary video: %w", err)
	}
	pattern := filepath.Join(temp, "frame_%05d.png")
	// The executable is either an explicitly configured ffmpeg binary or the
	// result of exec.LookPath; request data is passed as fixed arguments.
	command := exec.CommandContext( //nolint:gosec // Variable executable path is intentionally validated by resolveFFmpeg.
		ctx,
		ffmpeg,
		"-hide_banner", "-loglevel", "error",
		"-i", input,
		"-vf", fmt.Sprintf("fps=%d,format=rgba", fps),
		"-vsync", "0",
		// Stop ffmpeg one frame beyond the accepted limit. This bounds temp
		// disk usage while still allowing the post-extraction check to reject
		// videos that would have yielded too many candidate frames.
		"-frames:v", fmt.Sprintf("%d", maxAnimationExtractedFrames+1),
		pattern,
	)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("video: ffmpeg extract animation frames: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	paths, err := filepath.Glob(filepath.Join(temp, "frame_*.png"))
	if err != nil {
		return nil, fmt.Errorf("video: list extracted animation frames: %w", err)
	}
	sort.Strings(paths)
	if err := validateExtractedAnimationFrameCount(len(paths)); err != nil {
		return nil, err
	}
	configs := make([]image.Config, 0, len(paths))
	for _, path := range paths {
		config, configErr := decodeAnimationFrameConfig(path)
		if configErr != nil {
			return nil, configErr
		}
		configs = append(configs, config)
	}
	if err := validateExtractedAnimationFrameConfigs(configs); err != nil {
		return nil, err
	}

	frames := make([]image.Image, 0, len(paths))
	for _, path := range paths {
		// paths only contains entries produced by filepath.Glob inside temp.
		file, openErr := os.Open(path) //nolint:gosec // The path is constrained to the private temporary directory.
		if openErr != nil {
			return nil, fmt.Errorf("video: open extracted animation frame: %w", openErr)
		}
		frame, _, decodeErr := image.Decode(file)
		closeErr := file.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("video: decode extracted animation frame: %w", decodeErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("video: close extracted animation frame: %w", closeErr)
		}
		frames = append(frames, frame)
	}
	if len(frames) < 2 {
		return nil, fmt.Errorf("video: video yielded only %d decodable frame(s)", len(frames))
	}
	return frames, nil
}

var _ Processor = (*processor)(nil)

func validateExtractedAnimationFrameCount(count int) error {
	if count > maxAnimationExtractedFrames {
		return fmt.Errorf(
			"video: video yielded %d animation frames; limit is %d",
			count,
			maxAnimationExtractedFrames,
		)
	}
	return nil
}

func decodeAnimationFrameConfig(path string) (image.Config, error) {
	// path is produced by filepath.Glob inside the private temporary directory.
	file, err := os.Open(path) //nolint:gosec // The path is constrained to the private temporary directory.
	if err != nil {
		return image.Config{}, fmt.Errorf("video: open extracted animation frame metadata: %w", err)
	}
	config, _, decodeErr := image.DecodeConfig(file)
	closeErr := file.Close()
	if decodeErr != nil {
		return image.Config{}, fmt.Errorf("video: decode extracted animation frame metadata: %w", decodeErr)
	}
	if closeErr != nil {
		return image.Config{}, fmt.Errorf("video: close extracted animation frame metadata: %w", closeErr)
	}
	return config, nil
}

func validateExtractedAnimationFrameConfigs(configs []image.Config) error {
	var estimatedBytes int64
	for index, config := range configs {
		if config.Width < 1 || config.Height < 1 {
			return fmt.Errorf(
				"video: extracted animation frame %d has invalid dimensions %dx%d",
				index+1,
				config.Width,
				config.Height,
			)
		}
		if config.Width > maxAnimationFrameDimension || config.Height > maxAnimationFrameDimension {
			return fmt.Errorf(
				"video: extracted animation frame %d dimensions %dx%d exceed limit %dx%d",
				index+1,
				config.Width,
				config.Height,
				maxAnimationFrameDimension,
				maxAnimationFrameDimension,
			)
		}

		framePixels := int64(config.Width) * int64(config.Height)
		frameBytes := framePixels * animationDecodedBytesPerPixel
		if frameBytes > maxAnimationDecodedFrameBytes-estimatedBytes {
			return fmt.Errorf(
				"video: decoded animation frames exceed %d MiB memory budget at frame %d (%dx%d)",
				maxAnimationDecodedFrameBytes>>20,
				index+1,
				config.Width,
				config.Height,
			)
		}
		estimatedBytes += frameBytes
	}
	return nil
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
		return "", fmt.Errorf("video: FFMPEG_PATH does not point to a file: %s", path)
	}
	found, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("video: ffmpeg is required for video frame extraction; install it or set FFMPEG_PATH")
	}
	return found, nil
}
