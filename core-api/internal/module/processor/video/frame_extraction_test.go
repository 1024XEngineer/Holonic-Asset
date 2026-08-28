package video

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestQualityErrorReturnsMessage(t *testing.T) {
	err := &QualityError{Kind: "framing", Message: "unsafe frame"}
	if err.Error() != "unsafe frame" {
		t.Fatalf("unexpected quality error: %q", err.Error())
	}
}

func TestNewProcessorProvidesFFmpegExtractor(t *testing.T) {
	value, ok := NewProcessor().(*processor)
	if !ok || value.extractor == nil {
		t.Fatalf("unexpected processor: %#v", value)
	}
}

func TestProcessorPropagatesExtractionAndSelectionErrors(t *testing.T) {
	wantErr := errors.New("extract failed")
	validOptions := ProcessOptions{
		ChromaKey: testGreenChromaKey,
		Select:    func(FrameSequenceAnalysis) ([]int, error) { return []int{0}, nil },
	}
	tests := []struct {
		name      string
		processor Processor
		options   ProcessOptions
		want      string
	}{
		{name: "missing extractor", processor: newProcessor(nil), options: validOptions, want: "video: video frame extractor is required"},
		{name: "missing selector", processor: newProcessor(frameExtractorStub{}), options: ProcessOptions{ChromaKey: testGreenChromaKey}, want: "video: frame selector is required"},
		{name: "invalid chroma key", processor: newProcessor(frameExtractorStub{}), options: ProcessOptions{Select: validOptions.Select}, want: "video: valid chroma key settings are required"},
		{name: "invalid FPS", processor: newProcessor(frameExtractorStub{}), options: ProcessOptions{AnalysisFPS: -1, ChromaKey: testGreenChromaKey, Select: validOptions.Select}, want: "video: analysis FPS must be positive"},
		{name: "extract failure", processor: newProcessor(frameExtractorStub{err: wantErr}), options: validOptions, want: wantErr.Error()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.processor.Process(context.Background(), []byte("video"), test.options)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestProcessorKeepsLockedMatteInsteadOfSamplingFrameCorners(t *testing.T) {
	frame := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{R: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	drawSubject(frame, image.Rect(30, 18, 66, 88))

	key := ChromaKeyForMatte([3]uint8{0, 255, 0})
	_, err := newProcessor(frameExtractorStub{frames: []image.Image{frame}}).Process(
		context.Background(), []byte("video"), ProcessOptions{
			ChromaKey: key,
			Select:    func(FrameSequenceAnalysis) ([]int, error) { return []int{0}, nil },
		},
	)
	var qualityErr *QualityError
	if !errors.As(err, &qualityErr) || qualityErr.Kind != "framing" {
		t.Fatalf("locked green matte must not be replaced by magenta frame corners, got %v", err)
	}
}

func TestDecodeFrameAnalysesKeepsLockedMatteInsteadOfSamplingFrameCorners(t *testing.T) {
	directory := t.TempDir()
	frame := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{R: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	drawSubject(frame, image.Rect(30, 18, 66, 88))
	path := filepath.Join(directory, "analysis.png")
	writeFramePNG(t, path, frame)

	analyses, err := decodeFrameAnalyses([]string{path}, ChromaKeyForMatte([3]uint8{0, 255, 0}))
	if err != nil {
		t.Fatalf("decode locked-matte analysis: %v", err)
	}
	if len(analyses) != 1 || analyses[0].descriptor.foreground != analysisSize*analysisSize {
		got := 0
		if len(analyses) == 1 {
			got = analyses[0].descriptor.foreground
		}
		t.Fatalf("locked green matte should leave the resized magenta frame as foreground: got %d, want %d", got, analysisSize*analysisSize)
	}
}

func TestProcessorRejectsFramesWithoutForeground(t *testing.T) {
	frames := make([]image.Image, 6)
	for index := range frames {
		frames[index] = greenFrame(96, 96)
	}

	_, err := newProcessor(frameExtractorStub{frames: frames}).Process(context.Background(), []byte("video"), ProcessOptions{
		ChromaKey: testGreenChromaKey,
		Select:    func(FrameSequenceAnalysis) ([]int, error) { return []int{0}, nil },
	})
	var qualityErr *QualityError
	if !errors.As(err, &qualityErr) || qualityErr.Kind != "foreground" {
		t.Fatalf("expected foreground quality error, got %v", err)
	}
}

func TestFFmpegFrameExtractorDecodesSelectedFrames(t *testing.T) {
	directory := t.TempDir()
	framePath := filepath.Join(directory, "source.png")
	writeFramePNG(t, framePath, greenFrame(32, 24))
	script := writeFakeFFmpeg(t, directory, `
for output do :; done
cp "$TEST_FRAME_SOURCE" "$(printf "$output" 1)"
cp "$TEST_FRAME_SOURCE" "$(printf "$output" 2)"
`)
	t.Setenv("TEST_FRAME_SOURCE", framePath)

	frames, indices, err := (ffmpegFrameExtractor{path: script}).Extract(
		context.Background(),
		[]byte("video"),
		12,
		testGreenChromaKey,
		func(analysis FrameSequenceAnalysis) ([]int, error) {
			if len(analysis.Frames) != 2 {
				t.Fatalf("unexpected analysis frame count: %d", len(analysis.Frames))
			}
			return []int{0, 1}, nil
		},
	)
	if err != nil {
		t.Fatalf("extract selected frames: %v", err)
	}
	if len(indices) != 2 || len(frames) != 2 || frames[0].Bounds().Dx() != 32 || frames[0].Bounds().Dy() != 24 {
		t.Fatalf("unexpected extracted frames: %+v", frames)
	}
}

func TestProcessorStreamsHighResolutionCandidates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping FFmpeg integration test in short mode")
	}
	ffmpeg, err := resolveFFmpeg("")
	if err != nil {
		t.Skipf("FFmpeg is unavailable: %v", err)
	}
	directory := t.TempDir()
	videoPath := filepath.Join(directory, "source.mkv")
	command := exec.CommandContext( //nolint:gosec // The executable path is resolved by resolveFFmpeg.
		context.Background(),
		ffmpeg,
		"-hide_banner", "-loglevel", "error",
		"-f", "lavfi",
		"-i", "color=c=0x00ff00:s=960x960:r=12:d=4",
		"-vf", "drawbox=x=320:y=240:w=320:h=560:color=red:t=fill",
		"-frames:v", "48",
		"-c:v", "ffv1",
		videoPath,
	)
	if output, commandErr := command.CombinedOutput(); commandErr != nil {
		t.Fatalf("generate high-resolution test video: %v: %s", commandErr, strings.TrimSpace(string(output)))
	}
	video, err := os.ReadFile(videoPath) //nolint:gosec // Test path is created inside t.TempDir.
	if err != nil {
		t.Fatal(err)
	}

	result, err := newProcessor(ffmpegFrameExtractor{path: ffmpeg}).Process(
		context.Background(),
		video,
		ProcessOptions{
			AnalysisFPS: 12,
			ChromaKey:   testGreenChromaKey,
			Select: func(analysis FrameSequenceAnalysis) ([]int, error) {
				indices := make([]int, 16)
				for index := range indices {
					indices[index] = index * (len(analysis.Frames) - 1) / (len(indices) - 1)
				}
				return indices, nil
			},
		},
	)
	if err != nil {
		t.Fatalf("process high-resolution candidates: %v", err)
	}
	if len(result.Frames) != 16 {
		t.Fatalf("unexpected selected frame count: %d", len(result.Frames))
	}
	for index, frame := range result.Frames {
		if frame.Bounds().Dx() != 960 || frame.Bounds().Dy() != 960 {
			t.Fatalf("selected frame %d dimensions = %v", index, frame.Bounds().Size())
		}
	}
}

func TestFFmpegFrameExtractorReportsCommandAndOutputErrors(t *testing.T) {
	directory := t.TempDir()
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "command failure", body: `echo "decoder failed" >&2
exit 7`, want: "decoder failed"},
		{name: "no frames", body: `exit 0`, want: "only 0 decodable frame(s)"},
		{name: "invalid frame", body: `for output do :; done
printf "not-a-png" > "$(printf "$output" 1)"`, want: "decode extracted analysis frame"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			script := writeFakeFFmpeg(t, directory, test.body)
			_, _, err := (ffmpegFrameExtractor{path: script}).Extract(
				context.Background(),
				[]byte("video"),
				12,
				testGreenChromaKey,
				func(FrameSequenceAnalysis) ([]int, error) { return []int{0}, nil },
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestDecodeFrameConfigAndResolveFFmpeg(t *testing.T) {
	directory := t.TempDir()
	framePath := filepath.Join(directory, "frame.png")
	writeFramePNG(t, framePath, greenFrame(17, 19))

	config, err := decodeFrameConfig(framePath)
	if err != nil || config.Width != 17 || config.Height != 19 {
		t.Fatalf("unexpected frame config: %+v, err=%v", config, err)
	}
	if _, err := decodeFrameConfig(filepath.Join(directory, "missing.png")); err == nil {
		t.Fatal("expected missing frame metadata error")
	}

	executable := writeFakeFFmpeg(t, directory, "exit 0")
	resolved, err := resolveFFmpeg("  " + executable + "  ")
	if err != nil || resolved != executable {
		t.Fatalf("unexpected explicit ffmpeg path: %q, err=%v", resolved, err)
	}
	if _, err := resolveFFmpeg(directory); err == nil {
		t.Fatal("expected directory path to be rejected")
	}
	t.Setenv("FFMPEG_PATH", executable)
	resolved, err = resolveFFmpeg("")
	if err != nil || resolved != executable {
		t.Fatalf("unexpected environment ffmpeg path: %q, err=%v", resolved, err)
	}
	t.Setenv("FFMPEG_PATH", "")
	t.Setenv("PATH", "")
	if _, err := resolveFFmpeg(""); err == nil {
		t.Fatal("expected missing ffmpeg error")
	}
}

func TestValidateExtractedFrameConfigsRejectsInvalidDimensions(t *testing.T) {
	err := validateExtractedFrameConfigs([]image.Config{{Width: 0, Height: 24}})
	if err == nil || !strings.Contains(err.Error(), "video: extracted frame 1 has invalid dimensions") {
		t.Fatalf("expected invalid dimension error, got %v", err)
	}
	if err := validateExtractedFrameCount(maxExtractedFrames); err != nil {
		t.Fatalf("expected frame limit boundary to pass: %v", err)
	}
}

func greenFrame(width, height int) *image.NRGBA {
	frame := image.NewNRGBA(image.Rect(0, 0, width, height))
	for index := 0; index < len(frame.Pix); index += 4 {
		frame.Pix[index+1] = 255
		frame.Pix[index+3] = 255
	}
	return frame
}

func writeFramePNG(t *testing.T, path string, frame image.Image) {
	t.Helper()
	file, err := os.Create(path) //nolint:gosec // Test paths are created inside t.TempDir.
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, frame); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeFakeFFmpeg(t *testing.T, directory, body string) string {
	t.Helper()
	path := filepath.Join(directory, strings.ReplaceAll(t.Name(), "/", "_")+".sh")
	if err := os.WriteFile( //nolint:gosec // The test fixture needs an execute bit for exec.CommandContext.
		path,
		[]byte("#!/bin/sh\n"+body+"\n"),
		0o700,
	); err != nil {
		t.Fatal(err)
	}
	return path
}
