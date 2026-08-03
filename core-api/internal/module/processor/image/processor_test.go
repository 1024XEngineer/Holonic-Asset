package image

import (
	"context"
	"image"
	"image/color"
	"testing"
)

func TestProcessorRemoveResizeAndVerify(t *testing.T) {
	t.Parallel()

	sourceBase64 := controlledMatteFixtureBase64(t)
	processor := NewProcessor()
	removed, err := processor.RemoveBackground(context.Background(), &RemoveBackgroundRequest{
		ImageBase64: sourceBase64,
		MatteColor:  "#ff00ff",
		Material:    MaterialFlatIcon,
	})
	if err != nil {
		t.Fatalf("remove background: %v", err)
	}
	if removed.ImageBase64 == "" || removed.MIMEType != pngMIMEType {
		t.Fatalf("unexpected remove result: %#v", removed)
	}
	if removed.Report.Method != MethodChroma || !removed.Report.RGBScrubbed {
		t.Fatalf("unexpected extraction report: %#v", removed.Report)
	}

	verification, err := processor.Verify(context.Background(), &VerifyRequest{
		ImageBase64:        removed.ImageBase64,
		Profile:            ProfileIcon,
		ExpectedMatteColor: "#ff00ff",
	})
	if err != nil {
		t.Fatalf("verify transparent output: %v", err)
	}
	if !verification.Passed {
		t.Fatalf("transparent verification failed: %v", verification.FailureReasons)
	}

	options := DefaultResizeOptions(32, 32)
	options.Margin = 2
	resized, err := processor.Resize(context.Background(), &ResizeRequest{
		ImageBase64: removed.ImageBase64,
		Options:     options,
	})
	if err != nil {
		t.Fatalf("resize: %v", err)
	}
	if resized.ImageBase64 == "" || resized.MIMEType != pngMIMEType {
		t.Fatalf("unexpected resize result: %#v", resized)
	}
	if resized.Report.OutputWidth != 32 || resized.Report.OutputHeight != 32 {
		t.Fatalf("resize report = %#v", resized.Report)
	}

	finalImage, err := DecodeBase64Image(resized.ImageBase64)
	if err != nil {
		t.Fatal(err)
	}
	if finalImage.Bounds().Dx() != 32 || finalImage.Bounds().Dy() != 32 {
		t.Fatalf("final bounds = %v", finalImage.Bounds())
	}
	if finalImage.RGBAAt(0, 0).A != 0 {
		t.Fatal("expected transparent final margin")
	}

	finalVerification, err := processor.Verify(context.Background(), &VerifyRequest{
		ImageBase64:        resized.ImageBase64,
		Profile:            ProfileIcon,
		ExpectedMatteColor: "#ff00ff",
	})
	if err != nil {
		t.Fatalf("verify final output: %v", err)
	}
	if !finalVerification.Passed {
		t.Fatalf("final verification failed: %v", finalVerification.FailureReasons)
	}
}

func TestProcessorRemoveBackgroundSupportsAutoMatteAndDataURL(t *testing.T) {
	t.Parallel()

	dataURL := "data:image/png;base64," + controlledMatteFixtureBase64(t)
	result, err := NewProcessor().RemoveBackground(context.Background(), &RemoveBackgroundRequest{
		ImageBase64: dataURL,
		MatteColor:  "auto",
	})
	if err != nil {
		t.Fatalf("remove background with auto matte: %v", err)
	}
	if result.Report.MatteColorSource != "auto-sampled" {
		t.Fatalf("matte source = %q", result.Report.MatteColorSource)
	}
}

func TestProcessorRejectsInvalidBase64(t *testing.T) {
	t.Parallel()

	_, err := NewProcessor().Resize(context.Background(), &ResizeRequest{
		ImageBase64: "not-base64",
		Options:     DefaultResizeOptions(32, 32),
	})
	if err == nil {
		t.Fatal("expected invalid image data error")
	}
}

func TestProcessorHonoursCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewProcessor().Verify(ctx, &VerifyRequest{ImageBase64: "unused"})
	if err == nil {
		t.Fatal("expected context cancellation")
	}
}

func controlledMatteFixtureBase64(t *testing.T) string {
	t.Helper()

	fixture := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			fixture.SetRGBA(x, y, color.RGBA{R: 255, B: 255, A: 255})
		}
	}
	for y := 16; y < 48; y++ {
		for x := 20; x < 44; x++ {
			fixture.SetRGBA(x, y, color.RGBA{G: 180, B: 30, A: 255})
		}
	}
	encoded, err := EncodePNGBase64(fixture)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
