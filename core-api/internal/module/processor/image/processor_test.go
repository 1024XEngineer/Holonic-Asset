package image

import (
	"context"
	"errors"
	"image"
	"image/color"
	"slices"
	"strings"
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

func TestProcessorRemoveBackgroundFallsBackToSampledMatte(t *testing.T) {
	t.Parallel()

	fixture := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			fixture.SetRGBA(x, y, color.RGBA{R: 245, G: 245, B: 245, A: 255})
		}
	}
	for y := 16; y < 48; y++ {
		for x := 20; x < 44; x++ {
			fixture.SetRGBA(x, y, color.RGBA{R: 200, G: 40, B: 80, A: 255})
		}
	}
	encoded, err := EncodePNGBase64(fixture)
	if err != nil {
		t.Fatal(err)
	}

	result, err := NewProcessor().RemoveBackground(context.Background(), &RemoveBackgroundRequest{
		ImageBase64:               encoded,
		MatteColor:                DefaultMatteColor,
		AllowSampledMatteFallback: true,
	})
	if err != nil {
		t.Fatalf("remove unexpected matte: %v", err)
	}
	if !result.Report.FallbackApplied || result.Report.MatteColorSource != "auto-sampled" {
		t.Fatalf("expected sampled-matte fallback, got %#v", result.Report)
	}
	report, err := NewProcessor().Verify(context.Background(), &VerifyRequest{
		ImageBase64: result.ImageBase64,
		Profile:     ProfileGeneric,
	})
	if err != nil || !report.Passed {
		t.Fatalf("fallback output failed verification: report=%+v err=%v", report, err)
	}
}

func TestOpaqueBackgroundProfileAcceptsFullCanvasImage(t *testing.T) {
	t.Parallel()

	fixture := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			fixture.SetRGBA(x, y, color.RGBA{R: 20, G: 35, B: 80, A: 255})
		}
	}
	encoded, err := EncodePNGBase64(fixture)
	if err != nil {
		t.Fatal(err)
	}
	processor := NewProcessor()
	generic, err := processor.Verify(context.Background(), &VerifyRequest{ImageBase64: encoded, Profile: ProfileGeneric})
	if err != nil {
		t.Fatal(err)
	}
	if generic.Passed {
		t.Fatal("generic transparency profile unexpectedly accepted an opaque canvas")
	}
	opaque, err := processor.Verify(context.Background(), &VerifyRequest{ImageBase64: encoded, Profile: ProfileOpaqueBackground})
	if err != nil || !opaque.Passed {
		t.Fatalf("opaque background profile rejected valid canvas: report=%+v err=%v", opaque, err)
	}
	if opaque.AlphaHealthScore != 1 || opaque.QualityScore != 1 || len(opaque.Warnings) != 0 {
		t.Fatalf("opaque background profile reported degraded health: report=%+v", opaque)
	}
	for y := range 32 {
		for x := range 32 {
			alpha := uint8(128)
			fixture.SetRGBA(x, y, color.RGBA{R: 20, G: 35, B: 80, A: alpha})
		}
	}
	translucent, err := EncodePNGBase64(fixture)
	if err != nil {
		t.Fatal(err)
	}
	translucentReport, err := processor.Verify(context.Background(), &VerifyRequest{ImageBase64: translucent, Profile: ProfileOpaqueBackground})
	if err != nil {
		t.Fatal(err)
	}
	if translucentReport.Passed || !slices.Contains(translucentReport.FailureReasons, "background_not_fully_opaque") {
		t.Fatalf("opaque background profile accepted translucent canvas: report=%+v", translucentReport)
	}
	for y := range 32 {
		for x := range 32 {
			shade := uint8(40)
			if (x/8+y/8)%2 == 0 {
				shade = 80
			}
			fixture.SetRGBA(x, y, color.RGBA{R: shade, G: shade, B: shade, A: 255})
		}
	}
	tiled, err := EncodePNGBase64(fixture)
	if err != nil {
		t.Fatal(err)
	}
	tiledReport, err := processor.Verify(context.Background(), &VerifyRequest{ImageBase64: tiled, Profile: ProfileOpaqueBackground})
	if err != nil || !tiledReport.Passed || tiledReport.CheckerboardDetected {
		t.Fatalf("opaque background profile rejected tiled artwork: report=%+v err=%v", tiledReport, err)
	}

	for y := range 32 {
		for x := range 32 {
			fixture.SetRGBA(x, y, color.RGBA{R: 20, G: 35, B: 80, A: 255})
		}
	}
	fixture.SetRGBA(0, 0, color.RGBA{})
	withHole, err := EncodePNGBase64(fixture)
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := processor.Verify(context.Background(), &VerifyRequest{ImageBase64: withHole, Profile: ProfileOpaqueBackground})
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Passed || !slices.Contains(rejected.FailureReasons, "background_not_full_canvas") {
		t.Fatalf("opaque background profile accepted a canvas hole: report=%+v", rejected)
	}
}

func TestProcessorRemoveBackgroundKeepsExplicitMatteAuthoritative(t *testing.T) {
	t.Parallel()

	fixture := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			fixture.SetRGBA(x, y, color.RGBA{R: 245, G: 245, B: 245, A: 255})
		}
	}
	for y := 8; y < 24; y++ {
		for x := 8; x < 24; x++ {
			fixture.SetRGBA(x, y, color.RGBA{R: 200, G: 40, B: 80, A: 255})
		}
	}
	encoded, err := EncodePNGBase64(fixture)
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewProcessor().RemoveBackground(context.Background(), &RemoveBackgroundRequest{
		ImageBase64: encoded,
		MatteColor:  DefaultMatteColor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Report.FallbackApplied || result.Report.MatteColorSource != "provided" {
		t.Fatalf("explicit matte was silently replaced: report=%+v", result.Report)
	}
}

func TestProcessorRemoveBackgroundKeepsPrimaryWhenFallbackIsDegenerate(t *testing.T) {
	t.Parallel()

	fixture := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			fixture.SetRGBA(x, y, color.RGBA{R: 120, G: 120, B: 120, A: 255})
		}
	}
	encoded, err := EncodePNGBase64(fixture)
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewProcessor().RemoveBackground(context.Background(), &RemoveBackgroundRequest{
		ImageBase64:               encoded,
		MatteColor:                DefaultMatteColor,
		AllowSampledMatteFallback: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Report.FallbackApplied || result.Report.MatteColorSource != "provided" {
		t.Fatalf("degenerate fallback replaced primary extraction: report=%+v", result.Report)
	}
}

func TestProcessorResizeCoverCanvasAvoidsTransparentLetterboxing(t *testing.T) {
	t.Parallel()

	fixture := image.NewRGBA(image.Rect(0, 0, 48, 32))
	for y := range 32 {
		for x := range 48 {
			fixture.SetRGBA(x, y, color.RGBA{R: uint8(x * 4), G: 40, B: 80, A: 255})
		}
	}
	encoded, err := EncodePNGBase64(fixture)
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewProcessor().Resize(context.Background(), &ResizeRequest{
		ImageBase64: encoded,
		Options: ResizeOptions{
			Width: 32, Height: 32, CoverCanvas: true, Mode: RasterModeSmooth,
		},
	})
	if err != nil {
		t.Fatalf("cover resize: %v", err)
	}
	if !result.Report.CoveredCanvas {
		t.Fatalf("cover resize report = %#v", result.Report)
	}
	resized, err := DecodeBase64Image(result.ImageBase64)
	if err != nil {
		t.Fatal(err)
	}
	for y := range 32 {
		for x := range 32 {
			if resized.RGBAAt(x, y).A != 255 {
				t.Fatalf("cover resize left transparency at (%d,%d)", x, y)
			}
		}
	}
}

func TestProcessorResizeHardAlphaCropsUsingFinalOpaqueBounds(t *testing.T) {
	t.Parallel()

	fixture := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 2; y < 14; y++ {
		for x := 2; x < 14; x++ {
			alpha := uint8(100)
			if x >= 3 && x < 13 && y >= 3 && y < 13 {
				alpha = 255
			}
			fixture.SetNRGBA(x, y, color.NRGBA{R: 150, G: 86, B: 43, A: alpha})
		}
	}
	encoded, err := EncodePNGBase64(fixture)
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewProcessor().Resize(context.Background(), &ResizeRequest{
		ImageBase64: encoded,
		Options: ResizeOptions{
			Width: 8, Height: 8, Margin: 0, CropContent: true, HardAlpha: true, Mode: RasterModePixel,
		},
	})
	if err != nil {
		t.Fatalf("hard-alpha resize: %v", err)
	}
	resized, err := DecodeBase64Image(result.ImageBase64)
	if err != nil {
		t.Fatal(err)
	}
	for y := range 8 {
		for x := range 8 {
			if resized.RGBAAt(x, y).A != 255 {
				t.Fatalf("hard-alpha crop left transparent output at (%d,%d)", x, y)
			}
		}
	}
}

func TestProcessorResizeCoverCanvasRejectsMargin(t *testing.T) {
	t.Parallel()

	fixture := image.NewRGBA(image.Rect(0, 0, 32, 32))
	encoded, err := EncodePNGBase64(fixture)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewProcessor().Resize(context.Background(), &ResizeRequest{
		ImageBase64: encoded,
		Options: ResizeOptions{
			Width: 32, Height: 32, Margin: -1, CoverCanvas: true, Mode: RasterModeSmooth,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "cover canvas requires zero margin") {
		t.Fatalf("expected cover-margin rejection, got %v", err)
	}
}

func TestProcessorResizeCoverCanvasCropsTallSource(t *testing.T) {
	t.Parallel()

	fixture := image.NewRGBA(image.Rect(0, 0, 32, 48))
	for y := range 48 {
		for x := range 32 {
			fixture.SetRGBA(x, y, color.RGBA{R: 60, G: uint8(y * 4), B: 80, A: 255})
		}
	}
	encoded, err := EncodePNGBase64(fixture)
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewProcessor().Resize(context.Background(), &ResizeRequest{
		ImageBase64: encoded,
		Options: ResizeOptions{
			Width: 32, Height: 32, Margin: 0, CoverCanvas: true, Mode: RasterModeSmooth,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	resized, err := DecodeBase64Image(result.ImageBase64)
	if err != nil {
		t.Fatal(err)
	}
	if resized.Bounds().Dx() != 32 || resized.Bounds().Dy() != 32 || resized.RGBAAt(0, 0).A != 255 {
		t.Fatalf("tall cover resize produced invalid canvas: bounds=%v", resized.Bounds())
	}
}

func TestHasUsableTransparentSubjectRejectsMissingImages(t *testing.T) {
	t.Parallel()

	if hasUsableTransparentSubject(nil) {
		t.Fatal("nil image was accepted as a transparent subject")
	}
	if hasUsableTransparentSubject(image.NewRGBA(image.Rectangle{})) {
		t.Fatal("empty image was accepted as a transparent subject")
	}
}

func TestOpaqueBackgroundGateReportsStructuralFailures(t *testing.T) {
	t.Parallel()

	passed, failures := evaluateTransparencyGate(TransparencyGateInput{
		Profile:                ProfileOpaqueBackground,
		IsPNG:                  true,
		AlphaMin:               MinOpaqueAlpha,
		CheckerboardDetected:   true,
		TransparentRGBScrubbed: false,
	})
	if passed {
		t.Fatal("invalid opaque background passed verification")
	}
	for _, reason := range []string{"checkerboard_detected", "empty_subject", "transparent_rgb_not_scrubbed"} {
		if !slices.Contains(failures, reason) {
			t.Fatalf("missing failure %q in %v", reason, failures)
		}
	}
}

func TestComputeOpaqueAlphaHealthScorePenalizesMissingContent(t *testing.T) {
	t.Parallel()

	if score := computeOpaqueAlphaHealthScore(false, 0, MinOpaqueAlpha); score != 0.25 {
		t.Fatalf("opaque alpha health score = %v, want 0.25", score)
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

func TestProcessorFlipHorizontalMirrorsImage(t *testing.T) {
	t.Parallel()

	source := image.NewRGBA(image.Rect(0, 0, 3, 2))
	source.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	source.SetRGBA(1, 0, color.RGBA{G: 255, A: 255})
	source.SetRGBA(2, 0, color.RGBA{B: 255, A: 255})
	source.SetRGBA(0, 1, color.RGBA{R: 255, G: 255, A: 255})
	source.SetRGBA(1, 1, color.RGBA{G: 255, B: 255, A: 255})
	source.SetRGBA(2, 1, color.RGBA{R: 255, B: 255, A: 255})
	encoded, err := EncodePNGBase64(source)
	if err != nil {
		t.Fatal(err)
	}

	result, err := NewProcessor().(HorizontalFlipper).FlipHorizontal(context.Background(), &FlipHorizontalRequest{
		ImageBase64: encoded,
	})
	if err != nil {
		t.Fatalf("flip horizontal: %v", err)
	}
	if result.ImageBase64 == "" || result.MIMEType != pngMIMEType {
		t.Fatalf("unexpected flip result: %#v", result)
	}
	flipped, err := DecodeBase64Image(result.ImageBase64)
	if err != nil {
		t.Fatal(err)
	}
	if flipped.Bounds() != source.Bounds() {
		t.Fatalf("flipped bounds = %v, want %v", flipped.Bounds(), source.Bounds())
	}
	for y := range 2 {
		for x := range 3 {
			if got, want := flipped.RGBAAt(x, y), source.RGBAAt(2-x, y); got != want {
				t.Fatalf("pixel (%d,%d) = %#v, want %#v", x, y, got, want)
			}
		}
	}
}

func TestProcessorFlipHorizontalRejectsInvalidRequestsAndContext(t *testing.T) {
	t.Parallel()

	flipper := NewProcessor().(HorizontalFlipper)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		ctx  context.Context
		req  *FlipHorizontalRequest
	}{
		{name: "cancelled context", ctx: cancelled, req: &FlipHorizontalRequest{}},
		{name: "nil request", ctx: context.Background()},
		{name: "invalid image data", ctx: context.Background(), req: &FlipHorizontalRequest{ImageBase64: "not-base64"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := flipper.FlipHorizontal(test.ctx, test.req); err == nil {
				t.Fatal("expected FlipHorizontal to fail")
			}
		})
	}

	source := image.NewRGBA(image.Rect(0, 0, 2, 1))
	source.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	source.SetRGBA(1, 0, color.RGBA{G: 255, A: 255})
	encoded, err := EncodePNGBase64(source)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &cancelOnSecondDoneContext{Context: context.Background(), done: make(chan struct{})}
	if _, err := flipper.FlipHorizontal(ctx, &FlipHorizontalRequest{ImageBase64: encoded}); err == nil {
		t.Fatal("expected cancellation after image processing")
	} else if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
}

type cancelOnSecondDoneContext struct {
	context.Context
	done  chan struct{}
	calls int
}

func (c *cancelOnSecondDoneContext) Done() <-chan struct{} {
	c.calls++
	if c.calls == 2 {
		close(c.done)
	}
	return c.done
}

func (c *cancelOnSecondDoneContext) Err() error {
	select {
	case <-c.done:
		return context.Canceled
	default:
		return nil
	}
}
