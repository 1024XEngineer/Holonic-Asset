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
	defaultAnalysisFPS     = 12
	analysisSize           = 48
	maxExtractedFrames     = 100
	maxFrameDimension      = 4096
	maxDecodedFrameBytes   = 128 << 20
	decodedBytesPerPixel   = 4
	analysisFrameDimension = 256
)

type QualityError struct {
	Kind    string
	Message string
}

func (e *QualityError) Error() string { return e.Message }

// ChromaKey defines the background colour family excluded from foreground
// measurements. Callers must provide explicit thresholds.
type ChromaKey struct {
	HueMin              uint8
	HueMax              uint8
	HighSaturationMin   uint8
	HighValueMin        uint8
	BrightSaturationMin uint8
	BrightValueMin      uint8
}

// FrameSelector chooses source-frame indices from domain-neutral observations.
type FrameSelector func(FrameSequenceAnalysis) ([]int, error)

type ProcessOptions struct {
	AnalysisFPS int
	ChromaKey   ChromaKey
	Select      FrameSelector
}

// Processor extracts selected frames from video using caller-supplied policy.
type Processor interface {
	Process(context.Context, []byte, ProcessOptions) (*Result, error)
}

type Result struct {
	Frames        []image.Image
	SourceIndices []int
}

type frameExtractor interface {
	Extract(context.Context, []byte, int, ChromaKey, FrameSelector) ([]image.Image, []int, error)
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

func (p *processor) Process(ctx context.Context, source []byte, options ProcessOptions) (*Result, error) {
	if p.extractor == nil {
		return nil, fmt.Errorf("video: video frame extractor is required")
	}
	if options.Select == nil {
		return nil, fmt.Errorf("video: frame selector is required")
	}
	if !options.ChromaKey.valid() {
		return nil, fmt.Errorf("video: valid chroma key settings are required")
	}
	fps := options.AnalysisFPS
	if fps == 0 {
		fps = defaultAnalysisFPS
	}
	if fps < 1 {
		return nil, fmt.Errorf("video: analysis FPS must be positive")
	}
	frames, sourceIndices, err := p.extractor.Extract(ctx, source, fps, options.ChromaKey, options.Select)
	if err != nil {
		return nil, err
	}
	if err := validateSelectedFrameBounds(frames, sourceIndices, options.ChromaKey); err != nil {
		return nil, err
	}
	return &Result{Frames: frames, SourceIndices: sourceIndices}, nil
}

type ffmpegFrameExtractor struct {
	path string
}

func (e ffmpegFrameExtractor) Extract(
	ctx context.Context,
	video []byte,
	fps int,
	chromaKey ChromaKey,
	selectFrames FrameSelector,
) ([]image.Image, []int, error) {
	ffmpeg, err := resolveFFmpeg(e.path)
	if err != nil {
		return nil, nil, err
	}
	temp, err := os.MkdirTemp("", "holonic-video-frames-*")
	if err != nil {
		return nil, nil, fmt.Errorf("video: create video frame temp directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(temp)
	}()

	input := filepath.Join(temp, "input.mp4")
	if err := os.WriteFile(input, video, 0o600); err != nil {
		return nil, nil, fmt.Errorf("video: write temporary video: %w", err)
	}
	if selectFrames == nil {
		return nil, nil, fmt.Errorf("video: frame selector is required")
	}

	analysisPattern := filepath.Join(temp, "analysis_%05d.png")
	if err := runFrameExtraction(
		ctx,
		ffmpeg,
		input,
		analysisPattern,
		fmt.Sprintf(
			"fps=%d,scale=%d:%d:flags=area,format=rgba",
			fps,
			analysisFrameDimension,
			analysisFrameDimension,
		),
		maxExtractedFrames+1,
	); err != nil {
		return nil, nil, err
	}
	analysisPaths, err := filepath.Glob(filepath.Join(temp, "analysis_*.png"))
	if err != nil {
		return nil, nil, fmt.Errorf("video: list extracted analysis frames: %w", err)
	}
	sort.Strings(analysisPaths)
	if err := validateExtractedFrameCount(len(analysisPaths)); err != nil {
		return nil, nil, err
	}
	analyses, err := decodeFrameAnalyses(analysisPaths, chromaKey)
	if err != nil {
		return nil, nil, err
	}
	if len(analyses) < 2 {
		return nil, nil, fmt.Errorf("video: video yielded only %d decodable frame(s)", len(analyses))
	}

	indices, err := selectFrames(buildFrameSequenceAnalysis(analyses, fps))
	if err != nil {
		return nil, nil, err
	}
	if err := validateSelectedFrameIndices(indices, len(analyses)); err != nil {
		return nil, nil, err
	}
	selectedPattern := filepath.Join(temp, "selected_%05d.png")
	if err := runFrameExtraction(
		ctx,
		ffmpeg,
		input,
		selectedPattern,
		fmt.Sprintf("fps=%d,select=%s,format=rgba", fps, ffmpegSelectExpression(indices)),
		len(indices),
	); err != nil {
		return nil, nil, err
	}
	selectedPaths, err := filepath.Glob(filepath.Join(temp, "selected_*.png"))
	if err != nil {
		return nil, nil, fmt.Errorf("video: list selected frames: %w", err)
	}
	sort.Strings(selectedPaths)
	if len(selectedPaths) != len(indices) {
		return nil, nil, fmt.Errorf(
			"video: decoded %d selected frames; expected %d",
			len(selectedPaths),
			len(indices),
		)
	}
	configs := make([]image.Config, 0, len(selectedPaths))
	for _, path := range selectedPaths {
		config, configErr := decodeFrameConfig(path)
		if configErr != nil {
			return nil, nil, configErr
		}
		configs = append(configs, config)
	}
	if err := validateExtractedFrameConfigs(configs); err != nil {
		return nil, nil, err
	}
	frames, err := decodeFrames(selectedPaths, "")
	return frames, indices, err
}

func runFrameExtraction(
	ctx context.Context,
	ffmpeg string,
	input string,
	outputPattern string,
	filter string,
	frameLimit int,
) error {
	// The executable is either an explicitly configured ffmpeg binary or the
	// result of exec.LookPath; request data is passed as fixed arguments.
	command := exec.CommandContext( //nolint:gosec // Variable executable path is intentionally validated by resolveFFmpeg.
		ctx,
		ffmpeg,
		"-hide_banner", "-loglevel", "error",
		"-i", input,
		"-vf", filter,
		"-vsync", "0",
		"-frames:v", fmt.Sprintf("%d", frameLimit),
		outputPattern,
	)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("video: ffmpeg extract frames: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func decodeFrameAnalyses(paths []string, chromaKey ChromaKey) ([]frameAnalysis, error) {
	analyses := make([]frameAnalysis, 0, len(paths))
	for _, path := range paths {
		frames, err := decodeFrames([]string{path}, "analysis ")
		if err != nil {
			return nil, err
		}
		frame := frames[0]
		analyses = append(analyses, frameAnalysis{
			descriptor: describeFrame(frame, chromaKey),
			safe:       frameInsideSafetyBand(frame, chromaKey),
		})
	}
	return analyses, nil
}

func decodeFrames(paths []string, label string) ([]image.Image, error) {
	frames := make([]image.Image, 0, len(paths))
	for _, path := range paths {
		// paths only contains entries produced by filepath.Glob inside temp.
		file, openErr := os.Open(path) //nolint:gosec // The path is constrained to the private temporary directory.
		if openErr != nil {
			return nil, fmt.Errorf("video: open extracted %sframe: %w", label, openErr)
		}
		frame, _, decodeErr := image.Decode(file)
		closeErr := file.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("video: decode extracted %sframe: %w", label, decodeErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("video: close extracted %sframe: %w", label, closeErr)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

func validateSelectedFrameIndices(indices []int, candidateCount int) error {
	if len(indices) == 0 {
		return fmt.Errorf("video: frame selector returned no frames")
	}
	previous := -1
	for _, index := range indices {
		if index < 0 || index >= candidateCount {
			return fmt.Errorf("video: selected frame index %d is out of range", index)
		}
		if index <= previous {
			return fmt.Errorf("video: selected frame indices must be strictly increasing")
		}
		previous = index
	}
	return nil
}

func ffmpegSelectExpression(indices []int) string {
	parts := make([]string, 0, len(indices))
	for _, index := range indices {
		parts = append(parts, fmt.Sprintf("eq(n\\,%d)", index))
	}
	return strings.Join(parts, "+")
}

var _ Processor = (*processor)(nil)

func validateExtractedFrameCount(count int) error {
	if count > maxExtractedFrames {
		return fmt.Errorf(
			"video: video yielded %d frames; limit is %d",
			count,
			maxExtractedFrames,
		)
	}
	return nil
}

func decodeFrameConfig(path string) (image.Config, error) {
	// path is produced by filepath.Glob inside the private temporary directory.
	file, err := os.Open(path) //nolint:gosec // The path is constrained to the private temporary directory.
	if err != nil {
		return image.Config{}, fmt.Errorf("video: open extracted frame metadata: %w", err)
	}
	config, _, decodeErr := image.DecodeConfig(file)
	closeErr := file.Close()
	if decodeErr != nil {
		return image.Config{}, fmt.Errorf("video: decode extracted frame metadata: %w", decodeErr)
	}
	if closeErr != nil {
		return image.Config{}, fmt.Errorf("video: close extracted frame metadata: %w", closeErr)
	}
	return config, nil
}

func validateExtractedFrameConfigs(configs []image.Config) error {
	var estimatedBytes int64
	for index, config := range configs {
		if config.Width < 1 || config.Height < 1 {
			return fmt.Errorf(
				"video: extracted frame %d has invalid dimensions %dx%d",
				index+1,
				config.Width,
				config.Height,
			)
		}
		if config.Width > maxFrameDimension || config.Height > maxFrameDimension {
			return fmt.Errorf(
				"video: extracted frame %d dimensions %dx%d exceed limit %dx%d",
				index+1,
				config.Width,
				config.Height,
				maxFrameDimension,
				maxFrameDimension,
			)
		}

		framePixels := int64(config.Width) * int64(config.Height)
		frameBytes := framePixels * decodedBytesPerPixel
		if frameBytes > maxDecodedFrameBytes-estimatedBytes {
			return fmt.Errorf(
				"video: decoded frames exceed %d MiB memory budget at frame %d (%dx%d)",
				maxDecodedFrameBytes>>20,
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
