package generator

import (
	"context"
	"errors"
	"image"
	"image/color"
	"strings"
	"testing"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
)

type animationFrameProcessorStub struct {
	resizeResults []*imageprocessor.ResizeResult
	resizeErrors  []error
	requests      []*imageprocessor.ResizeRequest
}

func (*animationFrameProcessorStub) RemoveBackground(context.Context, *imageprocessor.RemoveBackgroundRequest) (*imageprocessor.RemoveBackgroundResult, error) {
	return nil, errors.New("unexpected RemoveBackground call")
}

func (*animationFrameProcessorStub) NormalizeReference(context.Context, *imageprocessor.NormalizeReferenceRequest) (*imageprocessor.NormalizeReferenceResult, error) {
	return nil, errors.New("unexpected NormalizeReference call")
}

func (s *animationFrameProcessorStub) Resize(_ context.Context, request *imageprocessor.ResizeRequest) (*imageprocessor.ResizeResult, error) {
	copy := *request
	s.requests = append(s.requests, &copy)
	index := len(s.requests) - 1
	if index < len(s.resizeErrors) && s.resizeErrors[index] != nil {
		return nil, s.resizeErrors[index]
	}
	if index >= len(s.resizeResults) {
		return nil, errors.New("unexpected Resize call")
	}
	return s.resizeResults[index], nil
}

func (*animationFrameProcessorStub) Verify(context.Context, *imageprocessor.VerifyRequest) (*imageprocessor.VerificationReport, error) {
	return nil, errors.New("unexpected Verify call")
}

func (*animationFrameProcessorStub) SplitImage(context.Context, *imageprocessor.SplitImageRequest) (*imageprocessor.SplitImageResult, error) {
	return nil, errors.New("unexpected SplitImage call")
}

func TestPixelProcessAnimationFramesBuildsSheetAndDefaultsMIMEType(t *testing.T) {
	red := encodedAnimationFrame(t, 2, 2, color.NRGBA{R: 220, A: 255})
	blue := encodedAnimationFrame(t, 2, 2, color.NRGBA{B: 210, A: 255})
	processor := &animationFrameProcessorStub{resizeResults: []*imageprocessor.ResizeResult{
		{ImageBase64: red},
		{ImageBase64: blue, MIMEType: "image/custom"},
	}}
	service := &animationGenerationService{processor: processor}
	input := []imageprocessor.ImageRegion{
		{Index: 3, ImageBase64: "source-red", MIMEType: "image/jpeg"},
		{Index: 4, ImageBase64: "source-blue", MIMEType: "image/jpeg"},
	}

	regions, encodedSheet, err := service.pixelProcessAnimationFrames(context.Background(), input, 2, 2, 2)
	if err != nil {
		t.Fatalf("pixel process animation frames: %v", err)
	}
	if len(regions) != 2 || regions[0].Index != 3 || regions[1].Index != 4 {
		t.Fatalf("processed regions did not preserve metadata: %+v", regions)
	}
	if regions[0].MIMEType != "image/png" || regions[1].MIMEType != "image/custom" {
		t.Fatalf("processed MIME types = %q, %q", regions[0].MIMEType, regions[1].MIMEType)
	}
	if regions[0].ImageBase64 != red || regions[1].ImageBase64 != blue {
		t.Fatal("processed regions did not use resized frame payloads")
	}
	if len(processor.requests) != 2 {
		t.Fatalf("Resize calls = %d, want 2", len(processor.requests))
	}
	for index, request := range processor.requests {
		if request.ImageBase64 != input[index].ImageBase64 {
			t.Fatalf("Resize request %d input = %q", index, request.ImageBase64)
		}
		options := request.Options
		if options.Width != 2 || options.Height != 2 || !options.SpritePixelPipeline || !options.PreserveCanvasGeometry {
			t.Fatalf("Resize request %d options = %+v", index, options)
		}
	}

	sheet, err := imageprocessor.DecodeBase64Image(encodedSheet)
	if err != nil {
		t.Fatalf("decode animation sheet: %v", err)
	}
	if got := sheet.Bounds().Size(); got != image.Pt(4, 2) {
		t.Fatalf("sheet size = %v, want 4x2", got)
	}
	if got := sheet.RGBAAt(0, 0); got.R != 220 || got.A != 255 {
		t.Fatalf("first sheet frame pixel = %+v", got)
	}
	if got := sheet.RGBAAt(3, 1); got.B != 210 || got.A != 255 {
		t.Fatalf("second sheet frame pixel = %+v", got)
	}
}

func TestPixelProcessAnimationFramesRejectsInvalidStages(t *testing.T) {
	valid := encodedAnimationFrame(t, 2, 2, color.NRGBA{G: 180, A: 255})
	wrongSize := encodedAnimationFrame(t, 3, 2, color.NRGBA{G: 180, A: 255})
	resizeErr := errors.New("resize failed")

	tests := []struct {
		name       string
		regions    []imageprocessor.ImageRegion
		columns    int
		results    []*imageprocessor.ResizeResult
		errors     []error
		wantError  string
		wantResize int
	}{
		{
			name:      "empty input",
			regions:   []imageprocessor.ImageRegion{{ImageBase64: "  "}},
			columns:   1,
			wantError: "frame 1: empty input",
		},
		{
			name:       "resize error",
			regions:    []imageprocessor.ImageRegion{{ImageBase64: "source"}},
			columns:    1,
			errors:     []error{resizeErr},
			wantError:  "frame 1: resize failed",
			wantResize: 1,
		},
		{
			name:       "nil resize result",
			regions:    []imageprocessor.ImageRegion{{ImageBase64: "source"}},
			columns:    1,
			results:    []*imageprocessor.ResizeResult{nil},
			wantError:  "frame 1: empty result",
			wantResize: 1,
		},
		{
			name:       "blank resize result",
			regions:    []imageprocessor.ImageRegion{{ImageBase64: "source"}},
			columns:    1,
			results:    []*imageprocessor.ResizeResult{{ImageBase64: "\t"}},
			wantError:  "frame 1: empty result",
			wantResize: 1,
		},
		{
			name:       "invalid encoded result",
			regions:    []imageprocessor.ImageRegion{{ImageBase64: "source"}},
			columns:    1,
			results:    []*imageprocessor.ResizeResult{{ImageBase64: "not-base64"}},
			wantError:  "decode pixel-processed animation frame 1",
			wantResize: 1,
		},
		{
			name:       "wrong result dimensions",
			regions:    []imageprocessor.ImageRegion{{ImageBase64: "source"}},
			columns:    1,
			results:    []*imageprocessor.ResizeResult{{ImageBase64: wrongSize}},
			wantError:  "dimensions 3x2; want 2x2",
			wantResize: 1,
		},
		{
			name:       "invalid sheet columns",
			regions:    []imageprocessor.ImageRegion{{ImageBase64: "source"}},
			columns:    0,
			results:    []*imageprocessor.ResizeResult{{ImageBase64: valid}},
			wantError:  "pack pixel-processed animation frames: generator: animation columns must be positive",
			wantResize: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			processor := &animationFrameProcessorStub{resizeResults: test.results, resizeErrors: test.errors}
			service := &animationGenerationService{processor: processor}
			regions, sheet, err := service.pixelProcessAnimationFrames(
				context.Background(), test.regions, test.columns, 2, 2,
			)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want containing %q", err, test.wantError)
			}
			if regions != nil || sheet != "" {
				t.Fatalf("failure returned regions=%+v sheet=%q", regions, sheet)
			}
			if len(processor.requests) != test.wantResize {
				t.Fatalf("Resize calls = %d, want %d", len(processor.requests), test.wantResize)
			}
		})
	}
}

func TestPackAnimationFramesValidatesInputAndPlacement(t *testing.T) {
	t.Run("requires frames", func(t *testing.T) {
		if _, err := packTransparentAnimationFrames(nil, 1); err == nil || !strings.Contains(err.Error(), "frames are required") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("requires positive columns", func(t *testing.T) {
		if _, err := packTransparentAnimationFrames([]image.Image{image.NewNRGBA(image.Rect(0, 0, 1, 1))}, 0); err == nil || !strings.Contains(err.Error(), "columns must be positive") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("requires positive frame dimensions", func(t *testing.T) {
		if _, err := packTransparentAnimationFrames([]image.Image{image.NewNRGBA(image.Rectangle{})}, 1); err == nil || !strings.Contains(err.Error(), "dimensions must be positive") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("requires matching frame dimensions", func(t *testing.T) {
		frames := []image.Image{
			image.NewNRGBA(image.Rect(0, 0, 2, 2)),
			image.NewNRGBA(image.Rect(0, 0, 1, 2)),
		}
		if _, err := packTransparentAnimationFrames(frames, 2); err == nil || !strings.Contains(err.Error(), "frame 2 dimensions differ") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("packs non-zero bounds and fills video matte", func(t *testing.T) {
		first := image.NewNRGBA(image.Rect(5, 7, 7, 9))
		first.SetNRGBA(5, 7, color.NRGBA{R: 255, A: 255})
		second := image.NewNRGBA(image.Rect(3, 4, 5, 6))
		second.SetNRGBA(4, 5, color.NRGBA{B: 255, A: 255})
		sheet, err := packAnimationVideoFrames([]image.Image{first, second}, 3)
		if err != nil {
			t.Fatalf("pack video frames: %v", err)
		}
		if got := sheet.Bounds().Size(); got != image.Pt(6, 2) {
			t.Fatalf("sheet size = %v, want 6x2", got)
		}
		if got := sheet.NRGBAAt(0, 0); got.R != 255 || got.A != 255 {
			t.Fatalf("first frame placement = %+v", got)
		}
		if got := sheet.NRGBAAt(3, 1); got.B != 255 || got.A != 255 {
			t.Fatalf("second frame placement = %+v", got)
		}
		if got := sheet.NRGBAAt(4, 0); got != (color.NRGBA{G: 255, A: 255}) {
			t.Fatalf("video matte = %+v", got)
		}
		if got := animationRows(5, 2); got != 3 {
			t.Fatalf("animationRows(5, 2) = %d, want 3", got)
		}
	})
}

func encodedAnimationFrame(t *testing.T, width, height int, fill color.NRGBA) string {
	t.Helper()
	frame := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			frame.SetNRGBA(x, y, fill)
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(frame)
	if err != nil {
		t.Fatalf("encode animation frame: %v", err)
	}
	return encoded
}

var _ imageprocessor.Processor = (*animationFrameProcessorStub)(nil)
