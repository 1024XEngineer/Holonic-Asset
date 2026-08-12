package video

import (
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnimationVideoQualityErrorReturnsMessage(t *testing.T) {
	err := &AnimationVideoQualityError{Kind: "framing", Message: "unsafe frame"}
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
	tests := []struct {
		name      string
		processor Processor
		want      string
	}{
		{name: "missing extractor", processor: newProcessor(nil), want: "video: video frame extractor is required"},
		{name: "extract failure", processor: newProcessor(frameExtractorStub{err: wantErr}), want: wantErr.Error()},
		{name: "insufficient frames", processor: newProcessor(frameExtractorStub{frames: animationTestVideoFrames(2)}), want: "need at least 5"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.processor.Process(context.Background(), []byte("video"), 4)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestProcessorRejectsFramesWithoutSubject(t *testing.T) {
	frames := make([]image.Image, 6)
	for index := range frames {
		frames[index] = greenAnimationFrame(96, 96)
	}

	_, err := newProcessor(frameExtractorStub{frames: frames}).Process(context.Background(), []byte("video"), 4)
	var qualityErr *AnimationVideoQualityError
	if !errors.As(err, &qualityErr) || qualityErr.Kind != "subject" {
		t.Fatalf("expected subject quality error, got %v", err)
	}
}

func TestFFmpegFrameExtractorDecodesGeneratedFrames(t *testing.T) {
	directory := t.TempDir()
	framePath := filepath.Join(directory, "source.png")
	writeAnimationPNG(t, framePath, greenAnimationFrame(32, 24))
	script := writeFakeFFmpeg(t, directory, `
for output do :; done
cp "$TEST_FRAME_SOURCE" "$(printf "$output" 1)"
cp "$TEST_FRAME_SOURCE" "$(printf "$output" 2)"
`)
	t.Setenv("TEST_FRAME_SOURCE", framePath)

	frames, err := (ffmpegFrameExtractor{path: script}).Extract(
		context.Background(),
		[]byte("video"),
		12,
		func(analyses []animationFrameAnalysis) ([]int, error) {
			if len(analyses) != 2 {
				t.Fatalf("unexpected analysis frame count: %d", len(analyses))
			}
			return []int{0, 1}, nil
		},
	)
	if err != nil {
		t.Fatalf("extract generated frames: %v", err)
	}
	if len(frames) != 2 || frames[0].Bounds().Dx() != 32 || frames[0].Bounds().Dy() != 24 {
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
		16,
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
		{name: "command failure", body: `echo "provider failed" >&2
exit 7`, want: "provider failed"},
		{name: "no frames", body: `exit 0`, want: "only 0 decodable frame(s)"},
		{name: "invalid frame", body: `for output do :; done
printf "not-a-png" > "$(printf "$output" 1)"`, want: "decode extracted animation analysis frame"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			script := writeFakeFFmpeg(t, directory, test.body)
			_, err := (ffmpegFrameExtractor{path: script}).Extract(
				context.Background(),
				[]byte("video"),
				12,
				func([]animationFrameAnalysis) ([]int, error) { return []int{0}, nil },
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestDecodeAnimationFrameConfigAndResolveFFmpeg(t *testing.T) {
	directory := t.TempDir()
	framePath := filepath.Join(directory, "frame.png")
	writeAnimationPNG(t, framePath, greenAnimationFrame(17, 19))

	config, err := decodeAnimationFrameConfig(framePath)
	if err != nil || config.Width != 17 || config.Height != 19 {
		t.Fatalf("unexpected frame config: %+v, err=%v", config, err)
	}
	if _, err := decodeAnimationFrameConfig(filepath.Join(directory, "missing.png")); err == nil {
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

func TestValidateExtractedAnimationFrameConfigsRejectsInvalidDimensions(t *testing.T) {
	err := validateExtractedAnimationFrameConfigs([]image.Config{{Width: 0, Height: 24}})
	if err == nil || !strings.Contains(err.Error(), "video: extracted animation frame 1 has invalid dimensions") {
		t.Fatalf("expected invalid dimension error, got %v", err)
	}
	if err := validateExtractedAnimationFrameCount(maxAnimationExtractedFrames); err != nil {
		t.Fatalf("expected frame limit boundary to pass: %v", err)
	}
}

func greenAnimationFrame(width, height int) *image.NRGBA {
	frame := image.NewNRGBA(image.Rect(0, 0, width, height))
	for index := 0; index < len(frame.Pix); index += 4 {
		frame.Pix[index+1] = 255
		frame.Pix[index+3] = 255
	}
	return frame
}

func writeAnimationPNG(t *testing.T, path string, frame image.Image) {
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
