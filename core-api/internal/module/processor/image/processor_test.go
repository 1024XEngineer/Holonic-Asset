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

func TestResizeImagePixelModeLocksReductionToAreaSampling(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			source.SetNRGBA(x, y, color.NRGBA{R: 240, G: 30, B: 30, A: 255})
		}
	}
	// A minority blue sample exists in every 2x2 destination footprint. The
	// old structure-aware reducer promoted a source colour and shifted apparent
	// features. The new reducer must retain the exact area blend first.
	for _, point := range []image.Point{{0, 0}, {2, 0}, {0, 2}, {2, 2}} {
		source.SetNRGBA(point.X, point.Y, color.NRGBA{R: 20, G: 50, B: 240, A: 255})
	}

	pixelOptions := DefaultResizeOptions(2, 2)
	pixelOptions.Margin = 0
	pixelOptions.CropContent = false
	pixelOptions.Mode = RasterModePixel
	pixelOptions.PaletteSize = 2
	pixelImage, pixelReport, err := ResizeImage(source, pixelOptions)
	if err != nil {
		t.Fatalf("pixel resize: %v", err)
	}
	if pixelReport.Sampling != resizeSamplingPixelArea {
		t.Fatalf("pixel sampling = %q, want %q", pixelReport.Sampling, resizeSamplingPixelArea)
	}

	smoothOptions := pixelOptions
	smoothOptions.Mode = RasterModeSmooth
	smoothOptions.PaletteSize = 0
	smoothImage, smoothReport, err := ResizeImage(source, smoothOptions)
	if err != nil {
		t.Fatalf("smooth resize: %v", err)
	}
	if smoothReport.Sampling != resizeSamplingArea {
		t.Fatalf("smooth sampling = %q, want %q", smoothReport.Sampling, resizeSamplingArea)
	}
	for y := range 2 {
		for x := range 2 {
			pixel := pixelImage.RGBAAt(x, y)
			smooth := smoothImage.RGBAAt(x, y)
			if pixel != smooth {
				t.Fatalf("pixel reduction moved away from area result at (%d,%d): got %+v, want %+v", x, y, pixel, smooth)
			}
			if pixel.R >= 220 || pixel.B <= 50 {
				t.Fatalf("area blend was replaced by a source minority/majority colour: %+v", pixel)
			}
		}
	}
}

func TestRepairPixelColourBlocksMergesAmbiguousSingleton(t *testing.T) {
	t.Parallel()

	base := color.NRGBA{R: 80, G: 80, B: 80, A: 255}
	ambiguous := color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	quantized := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	reference := image.NewNRGBA(quantized.Bounds())
	for y := range 3 {
		for x := range 3 {
			quantized.SetNRGBA(x, y, base)
			reference.SetNRGBA(x, y, base)
		}
	}
	quantized.SetNRGBA(1, 1, ambiguous)
	// The smooth pixel lies between the singleton and its surrounding block;
	// either palette colour is plausible, so spatial coherence should win.
	reference.SetNRGBA(1, 1, color.NRGBA{R: 95, G: 95, B: 95, A: 255})

	repairPixelColourBlocks(quantized, reference, false)
	if got := quantized.NRGBAAt(1, 1); got != base {
		t.Fatalf("ambiguous singleton was not merged into its colour block: %+v", got)
	}
}

func TestRepairPixelColourBlocksPreservesTrueHighContrastDetail(t *testing.T) {
	t.Parallel()

	base := color.NRGBA{R: 220, G: 130, B: 55, A: 255}
	detail := color.NRGBA{R: 18, G: 14, B: 12, A: 255}
	quantized := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	reference := image.NewNRGBA(quantized.Bounds())
	for y := range 3 {
		for x := range 3 {
			quantized.SetNRGBA(x, y, base)
			reference.SetNRGBA(x, y, base)
		}
	}
	quantized.SetNRGBA(1, 1, detail)
	reference.SetNRGBA(1, 1, color.NRGBA{R: 20, G: 16, B: 13, A: 255})

	repairPixelColourBlocks(quantized, reference, false)
	if got := quantized.NRGBAAt(1, 1); got != detail {
		t.Fatalf("real high-contrast detail was erased: %+v", got)
	}
}

func TestRepairPixelColourBlocksPreservesTwoBlendedSinglePixelEyes(t *testing.T) {
	t.Parallel()

	skin := color.NRGBA{R: 184, G: 116, B: 82, A: 255}
	eye := color.NRGBA{R: 24, G: 20, B: 22, A: 255}
	quantized := image.NewNRGBA(image.Rect(0, 0, 8, 5))
	reference := image.NewNRGBA(quantized.Bounds())
	for y := range 5 {
		for x := range 8 {
			quantized.SetNRGBA(x, y, skin)
			reference.SetNRGBA(x, y, skin)
		}
	}
	for _, point := range []image.Point{{2, 2}, {5, 2}} {
		quantized.SetNRGBA(point.X, point.Y, eye)
		// Area reduction can blend a tiny dark eye with surrounding skin. The
		// repair pass must still recognize the mapped high-contrast mark as a
		// deliberate detail rather than replacing it with the face colour.
		reference.SetNRGBA(point.X, point.Y, color.NRGBA{R: 92, G: 64, B: 55, A: 255})
	}

	repairPixelColourBlocks(quantized, reference, false)
	for _, point := range []image.Point{{2, 2}, {5, 2}} {
		if got := quantized.NRGBAAt(point.X, point.Y); got != eye {
			t.Fatalf("eye at %v was erased: %+v", point, got)
		}
	}
}

func TestRepairPixelColourBlocksConsolidatesObjectNoiseWhenSmoothSourceSupportsFill(t *testing.T) {
	t.Parallel()

	base := color.NRGBA{R: 220, G: 130, B: 55, A: 255}
	noise := color.NRGBA{R: 18, G: 14, B: 12, A: 255}
	quantized := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	reference := image.NewNRGBA(quantized.Bounds())
	for y := range 4 {
		for x := range 4 {
			quantized.SetNRGBA(x, y, base)
			reference.SetNRGBA(x, y, base)
		}
	}
	for _, point := range []image.Point{{X: 1, Y: 1}, {X: 1, Y: 2}} {
		quantized.SetNRGBA(point.X, point.Y, noise)
		// The high-resolution reduction does not support a truly dark mark at
		// either pixel; this is a palette island rather than a real prop detail.
		reference.SetNRGBA(point.X, point.Y, color.NRGBA{R: 150, G: 95, B: 42, A: 255})
	}

	repairPixelColourBlocks(quantized, reference, true)
	for _, point := range []image.Point{{X: 1, Y: 1}, {X: 1, Y: 2}} {
		if got := quantized.NRGBAAt(point.X, point.Y); got != base {
			t.Fatalf("object colour noise at %v was not consolidated: %+v", point, got)
		}
	}
}

func TestRepairPixelColourBlocksObjectModeKeepsSourceSupportedDetail(t *testing.T) {
	t.Parallel()

	base := color.NRGBA{R: 220, G: 130, B: 55, A: 255}
	detail := color.NRGBA{R: 18, G: 14, B: 12, A: 255}
	quantized := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	reference := image.NewNRGBA(quantized.Bounds())
	for y := range 3 {
		for x := range 3 {
			quantized.SetNRGBA(x, y, base)
			reference.SetNRGBA(x, y, base)
		}
	}
	quantized.SetNRGBA(1, 1, detail)
	reference.SetNRGBA(1, 1, color.NRGBA{R: 20, G: 16, B: 13, A: 255})

	repairPixelColourBlocks(quantized, reference, true)
	if got := quantized.NRGBAAt(1, 1); got != detail {
		t.Fatalf("source-supported object detail was erased: %+v", got)
	}
}

func TestRemapToPalettePreservesIsolatedHighContrastAccent(t *testing.T) {
	skin := color.NRGBA{R: 184, G: 116, B: 82, A: 255}
	eye := color.NRGBA{R: 24, G: 20, B: 22, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 5, 5))
	for y := range 5 {
		for x := range 5 {
			img.SetNRGBA(x, y, skin)
		}
	}
	img.SetNRGBA(2, 2, eye)

	remapToPalettePreservingAccents(img, img.Bounds(), []color.RGBA{{R: skin.R, G: skin.G, B: skin.B, A: 255}})
	if got := img.NRGBAAt(2, 2); got != eye {
		t.Fatalf("isolated high-contrast accent was remapped into the fill: %+v", got)
	}
}

func TestRepairPixelAlphaGapsFillsCoveredBridgeOnly(t *testing.T) {
	t.Parallel()

	green := color.NRGBA{R: 40, G: 170, B: 75, A: 255}
	quantized := image.NewNRGBA(image.Rect(0, 0, 5, 3))
	reference := image.NewNRGBA(quantized.Bounds())
	quantized.SetNRGBA(1, 1, green)
	quantized.SetNRGBA(3, 1, green)
	reference.SetNRGBA(2, 1, color.NRGBA{R: 45, G: 165, B: 78, A: 80})
	// This second apparent gap has matching neighbours but no meaningful source
	// coverage, so it must remain transparent rather than growing the shape.
	quantized.SetNRGBA(0, 0, green)
	quantized.SetNRGBA(2, 0, green)
	reference.SetNRGBA(1, 0, color.NRGBA{R: 45, G: 165, B: 78, A: 20})

	repairPixelAlphaGaps(quantized, reference, pixelAlphaRepairFloor)
	if got := quantized.NRGBAAt(2, 1); got != green {
		t.Fatalf("covered one-pixel bridge was not filled: %+v", got)
	}
	if got := quantized.NRGBAAt(1, 0); got.A != 0 {
		t.Fatalf("low-coverage background was incorrectly filled: %+v", got)
	}
}

func TestRepairPixelAlphaGapsFillsCoveredEnclosedHoleAcrossColours(t *testing.T) {
	t.Parallel()

	red := color.NRGBA{R: 190, G: 55, B: 35, A: 255}
	blue := color.NRGBA{R: 35, G: 70, B: 180, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	reference := image.NewNRGBA(img.Bounds())
	img.SetNRGBA(0, 1, red)
	img.SetNRGBA(2, 1, blue)
	img.SetNRGBA(1, 0, red)
	img.SetNRGBA(1, 2, blue)
	reference.SetNRGBA(1, 1, color.NRGBA{R: 42, G: 72, B: 170, A: 80})

	repairPixelAlphaGaps(img, reference, pixelAlphaRepairFloor)
	if got := img.NRGBAAt(1, 1); got != blue {
		t.Fatalf("covered enclosed hole was not filled from the nearest existing colour: %+v", got)
	}
}

func TestRepairPixelAlphaGapsDoesNotInferFromNeighbourMajority(t *testing.T) {
	green := color.NRGBA{R: 40, G: 170, B: 75, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	reference := image.NewNRGBA(img.Bounds())
	for _, point := range []image.Point{{X: 0, Y: 1}, {X: 1, Y: 0}, {X: 2, Y: 0}} {
		img.SetNRGBA(point.X, point.Y, green)
	}
	reference.SetNRGBA(1, 1, color.NRGBA{R: 40, G: 170, B: 75, A: 100})

	repairPixelAlphaGaps(img, reference, pixelAlphaRepairFloor)
	if got := img.NRGBAAt(1, 1); got.A != 0 {
		t.Fatalf("neighbour majority created an unsupported block: %+v", got)
	}
}

func TestRegularizeNearEllipticalSilhouetteSnapsSymmetricRoundProp(t *testing.T) {
	t.Parallel()

	dark := color.NRGBA{R: 65, G: 28, B: 10, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	// This is the distorted 16x16 width profile observed after hard-alpha
	// thresholding: symmetric and recognisably circular, but pinched at both
	// ends compared with the ellipse implied by the same bounds.
	widths := []int{6, 8, 10, 12, 14, 16, 16, 16, 16, 16, 16, 14, 12, 10, 8, 6}
	for y, width := range widths {
		left := (16 - width) / 2
		for x := left; x < left+width; x++ {
			img.SetNRGBA(x, y, dark)
		}
	}
	reference := cloneNRGBA(img)
	regularizeNearEllipticalSilhouette(
		img,
		reference,
		[]color.RGBA{{R: dark.R, G: dark.G, B: dark.B, A: 255}},
	)

	changed := 0
	for y := range 16 {
		for x := range 16 {
			dx := (float64(x) + 0.5 - 8) / 8
			dy := (float64(y) + 0.5 - 8) / 8
			wantOpaque := dx*dx+dy*dy <= 1
			gotOpaque := img.NRGBAAt(x, y).A == 255
			if gotOpaque != wantOpaque {
				t.Fatalf("ellipse mismatch at (%d,%d): opaque=%v, want %v", x, y, gotOpaque, wantOpaque)
			}
			if gotOpaque != (reference.NRGBAAt(x, y).A == 255) {
				changed++
			}
		}
	}
	if changed == 0 {
		t.Fatal("near-circular silhouette was not regularized")
	}
}

func TestRegularizeNearEllipticalSilhouettePreservesAsymmetricShape(t *testing.T) {
	t.Parallel()

	fill := color.NRGBA{R: 90, G: 55, B: 30, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			dx := (float64(x) + 0.5 - 8) / 8
			dy := (float64(y) + 0.5 - 8) / 8
			if dx*dx+dy*dy <= 1 {
				img.SetNRGBA(x, y, fill)
			}
		}
	}
	for _, point := range []image.Point{
		{X: 2, Y: 5}, {X: 1, Y: 6}, {X: 0, Y: 7}, {X: 0, Y: 8},
		{X: 1, Y: 9}, {X: 2, Y: 10},
	} {
		img.SetNRGBA(point.X, point.Y, color.NRGBA{})
	}
	original := cloneNRGBA(img)
	regularizeNearEllipticalSilhouette(
		img,
		cloneNRGBA(img),
		[]color.RGBA{{R: fill.R, G: fill.G, B: fill.B, A: 255}},
	)
	for y := range 16 {
		for x := range 16 {
			if got, want := img.NRGBAAt(x, y), original.NRGBAAt(x, y); got != want {
				t.Fatalf("asymmetric shape changed at (%d,%d): got %+v, want %+v", x, y, got, want)
			}
		}
	}
}

func TestRegularizeNearCircularObjectSilhouetteRepairsPinchedRoundObject(t *testing.T) {
	t.Parallel()

	outline := color.NRGBA{R: 72, G: 31, B: 12, A: 255}
	fill := color.NRGBA{R: 196, G: 91, B: 28, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	widths := []int{4, 8, 10, 12, 14, 14, 16, 16, 16, 16, 14, 14, 12, 10, 8, 4}
	for y, width := range widths {
		left := (16 - width) / 2
		for x := left; x < left+width; x++ {
			value := fill
			if x == left || x == left+width-1 {
				value = outline
			}
			img.SetNRGBA(x, y, value)
		}
	}
	palette := []color.RGBA{
		{R: outline.R, G: outline.G, B: outline.B, A: 255},
		{R: fill.R, G: fill.G, B: fill.B, A: 255},
	}
	regularizeNearCircularObjectSilhouette(img, cloneNRGBA(img), palette)

	assertOpaqueRowWidths(t, img, []int{6, 10, 12, 14, 14, 16, 16, 16, 16, 16, 16, 14, 14, 12, 10, 6})
	assertImageUsesOnlyPalette(t, img, palette)
}

func TestRegularizeNearCircularObjectSilhouetteExpandsSquashedRoundObject(t *testing.T) {
	t.Parallel()

	orange := color.NRGBA{R: 201, G: 93, B: 30, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	widths := []int{6, 10, 12, 14, 15, 16, 16, 16, 16, 16, 14, 12, 10, 6}
	for row, width := range widths {
		left := (16 - width) / 2
		for x := left; x < left+width; x++ {
			img.SetNRGBA(x, row+1, orange)
		}
	}
	palette := []color.RGBA{{R: orange.R, G: orange.G, B: orange.B, A: 255}}
	regularizeNearCircularObjectSilhouette(img, cloneNRGBA(img), palette)

	bounds, ok := alphaBounds(img, TransparentAlphaMax)
	if !ok || bounds != image.Rect(0, 0, 16, 16) {
		t.Fatalf("squashed round object bounds = %v, want 16x16 square", bounds)
	}
	assertOpaqueRowWidths(t, img, []int{6, 10, 12, 14, 14, 16, 16, 16, 16, 16, 16, 14, 14, 12, 10, 6})
	assertImageUsesOnlyPalette(t, img, palette)
}

func TestRegularizeNearCircularObjectSilhouetteExpandsSymmetricPinchedEllipse(t *testing.T) {
	orange := color.NRGBA{R: 201, G: 93, B: 30, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := 2; x < 14; x++ {
			dx := (float64(x) + 0.5 - 8) / 6
			dy := (float64(y) + 0.5 - 8) / 8
			if dx*dx+dy*dy <= 1 {
				img.SetNRGBA(x, y, orange)
			}
		}
	}
	regularizeNearCircularObjectSilhouette(
		img,
		cloneNRGBA(img),
		[]color.RGBA{{R: orange.R, G: orange.G, B: orange.B, A: 255}},
	)

	bounds, ok := alphaBounds(img, TransparentAlphaMax)
	if !ok || bounds.Dx() != bounds.Dy() {
		t.Fatalf("symmetric pinched ellipse was not expanded to a square footprint: %v", bounds)
	}
}

func TestResizeImageObjectPipelineExpandsSquashedRoundObjectInsideCanonicalMargin(t *testing.T) {
	t.Parallel()

	orange := color.NRGBA{R: 204, G: 92, B: 27, A: 255}
	source := image.NewNRGBA(image.Rect(0, 0, 16, 14))
	widths := []int{6, 10, 12, 14, 15, 16, 16, 16, 16, 16, 14, 12, 10, 6}
	for y, width := range widths {
		left := (16 - width) / 2
		for x := left; x < left+width; x++ {
			source.SetNRGBA(x, y, orange)
		}
	}

	result, _, err := ResizeImage(source, PrototypePixelResizeOptions(32, 32))
	if err != nil {
		t.Fatalf("resize squashed round object: %v", err)
	}
	bounds, ok := alphaBounds(toNRGBA(result), TransparentAlphaMax)
	if !ok || bounds != image.Rect(6, 6, 26, 26) {
		t.Fatalf("round object bounds = %v, want canonical 20x20 drawable area", bounds)
	}
	assertOpaqueRowWidths(t, toNRGBA(result).SubImage(bounds).(*image.NRGBA), []int{
		6, 10, 14, 16, 16, 18, 18, 20, 20, 20, 20, 20, 20, 18, 18, 16, 16, 14, 10, 6,
	})
}

func TestRegularizeNearCircularObjectSilhouettePreservesSquareObject(t *testing.T) {
	t.Parallel()

	fill := color.NRGBA{R: 118, G: 79, B: 42, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			img.SetNRGBA(x, y, fill)
		}
	}
	original := cloneNRGBA(img)
	regularizeNearCircularObjectSilhouette(
		img,
		cloneNRGBA(img),
		[]color.RGBA{{R: fill.R, G: fill.G, B: fill.B, A: 255}},
	)
	assertImagesEqual(t, img, original)
}

func TestRegularizeNearCircularObjectSilhouettePreservesAsymmetricObject(t *testing.T) {
	t.Parallel()

	fill := color.NRGBA{R: 101, G: 67, B: 38, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			dx := (float64(x) + 0.5 - 8) / 8
			dy := (float64(y) + 0.5 - 8) / 8
			if dx*dx+dy*dy <= 1 {
				img.SetNRGBA(x, y, fill)
			}
		}
	}
	for y := 5; y <= 10; y++ {
		img.SetNRGBA(0, y, color.NRGBA{})
	}
	original := cloneNRGBA(img)
	regularizeNearCircularObjectSilhouette(
		img,
		cloneNRGBA(img),
		[]color.RGBA{{R: fill.R, G: fill.G, B: fill.B, A: 255}},
	)
	assertImagesEqual(t, img, original)
}

func assertOpaqueRowWidths(t *testing.T, img *image.NRGBA, want []int) {
	t.Helper()
	if img.Bounds().Dy() != len(want) {
		t.Fatalf("image height = %d, want %d", img.Bounds().Dy(), len(want))
	}
	for y, wantWidth := range want {
		gotWidth := 0
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.NRGBAAt(x, img.Bounds().Min.Y+y).A > TransparentAlphaMax {
				gotWidth++
			}
		}
		if gotWidth != wantWidth {
			t.Fatalf("opaque width at row %d = %d, want %d", y, gotWidth, wantWidth)
		}
	}
}

func assertImageUsesOnlyPalette(t *testing.T, img *image.NRGBA, palette []color.RGBA) {
	t.Helper()
	allowed := make(map[color.NRGBA]struct{}, len(palette))
	for _, value := range palette {
		allowed[color.NRGBA{R: value.R, G: value.G, B: value.B, A: 255}] = struct{}{}
	}
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			pixel := img.NRGBAAt(x, y)
			if pixel.A <= TransparentAlphaMax {
				continue
			}
			if _, ok := allowed[pixel]; !ok {
				t.Fatalf("image invented colour %+v at (%d,%d)", pixel, x, y)
			}
		}
	}
}

func assertImagesEqual(t *testing.T, got, want *image.NRGBA) {
	t.Helper()
	if got.Bounds() != want.Bounds() {
		t.Fatalf("image bounds = %v, want %v", got.Bounds(), want.Bounds())
	}
	for y := got.Bounds().Min.Y; y < got.Bounds().Max.Y; y++ {
		for x := got.Bounds().Min.X; x < got.Bounds().Max.X; x++ {
			if gotPixel, wantPixel := got.NRGBAAt(x, y), want.NRGBAAt(x, y); gotPixel != wantPixel {
				t.Fatalf("image changed at (%d,%d): got %+v, want %+v", x, y, gotPixel, wantPixel)
			}
		}
	}
}

func TestResizeImagePixelModeUsesNearestNeighbourForEnlargement(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	source.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	source.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 255})
	options := DefaultResizeOptions(4, 2)
	options.Margin = 0
	options.CropContent = false
	options.Mode = RasterModePixel

	result, report, err := ResizeImage(source, options)
	if err != nil {
		t.Fatalf("enlarge pixel image: %v", err)
	}
	if report.Sampling != resizeSamplingNearest {
		t.Fatalf("sampling = %q, want %q", report.Sampling, resizeSamplingNearest)
	}
	for y := range 2 {
		if got := result.RGBAAt(1, y); got.R != 255 || got.B != 0 {
			t.Fatalf("left block was interpolated at y=%d: %+v", y, got)
		}
		if got := result.RGBAAt(2, y); got.R != 0 || got.B != 255 {
			t.Fatalf("right block was interpolated at y=%d: %+v", y, got)
		}
	}
}

func TestRemapToPaletteUsesPerceptualColourDistance(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	source.SetNRGBA(0, 0, color.NRGBA{R: 192, G: 32, B: 224, A: 255})
	perceptualMatch := color.RGBA{R: 192, G: 224, B: 96, A: 255}
	rgbArithmeticMatch := color.RGBA{R: 0, G: 0, B: 64, A: 255}
	remapToPalette(source, source.Bounds(), []color.RGBA{perceptualMatch, rgbArithmeticMatch})
	if got := source.NRGBAAt(0, 0); got != (color.NRGBA{
		R: perceptualMatch.R,
		G: perceptualMatch.G,
		B: perceptualMatch.B,
		A: 255,
	}) {
		t.Fatalf("perceptual remap chose %+v, want %+v", got, perceptualMatch)
	}
}

func TestCharacterPrototypePixelPipelinePreservesSourceColoursAndBothEyes(t *testing.T) {
	t.Parallel()

	hair := color.NRGBA{R: 42, G: 25, B: 22, A: 255}
	skin := color.NRGBA{R: 178, G: 110, B: 78, A: 255}
	eye := color.NRGBA{R: 18, G: 17, B: 21, A: 255}
	shirt := color.NRGBA{R: 34, G: 78, B: 148, A: 255}
	highlight := color.NRGBA{R: 211, G: 151, B: 112, A: 255}
	sourceColours := map[color.RGBA]struct{}{}
	for _, value := range []color.NRGBA{hair, skin, eye, shirt, highlight} {
		sourceColours[color.RGBA{R: value.R, G: value.G, B: value.B, A: 255}] = struct{}{}
	}

	logical := image.NewNRGBA(image.Rect(0, 0, 20, 20))
	for y := range 20 {
		for x := range 20 {
			value := skin
			switch {
			case y < 5:
				value = hair
			case y >= 14:
				value = shirt
			case y == 12 && x >= 8 && x <= 11:
				value = highlight
			}
			logical.SetNRGBA(x, y, value)
		}
	}
	eyes := []image.Point{{7, 8}, {12, 8}}
	for _, point := range eyes {
		logical.SetNRGBA(point.X, point.Y, eye)
	}
	// Scale the logical fixture up so the production path must perform its
	// alpha-aware area reduction before palette mapping. Each 4x4 eye becomes
	// one final logical pixel in the canonical 20x20 drawable area.
	source := image.NewNRGBA(image.Rect(0, 0, 80, 80))
	for y := range 80 {
		for x := range 80 {
			source.SetNRGBA(x, y, logical.NRGBAAt(x/4, y/4))
		}
	}

	options := CharacterPrototypePixelResizeOptions(32, 32)
	result, report, err := ResizeImage(source, options)
	if err != nil {
		t.Fatalf("resize character prototype: %v", err)
	}
	if report.Sampling != resizeSamplingPixelArea {
		t.Fatalf("character fixture did not exercise area reduction: %+v", report)
	}
	margin := AnimationFrameMargin(32, 32)
	for _, point := range eyes {
		got := result.RGBAAt(margin+point.X, margin+point.Y)
		if got != (color.RGBA{R: eye.R, G: eye.G, B: eye.B, A: 255}) {
			t.Fatalf("eye at %v was not preserved through the full pipeline: %+v", point, got)
		}
	}
	for y := range result.Bounds().Dy() {
		for x := range result.Bounds().Dx() {
			pixel := result.RGBAAt(x, y)
			if pixel.A == 0 {
				continue
			}
			if _, ok := sourceColours[pixel]; !ok {
				t.Fatalf("pipeline invented colour %+v at (%d,%d)", pixel, x, y)
			}
		}
	}
}

func TestAnimationFrameMarginUsesThreeSixteenthsOfShortEdge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		width, height      int
		wantMargin         int
		wantDrawableWidth  int
		wantDrawableHeight int
	}{
		{width: 32, height: 32, wantMargin: 6, wantDrawableWidth: 20, wantDrawableHeight: 20},
		{width: 48, height: 64, wantMargin: 9, wantDrawableWidth: 30, wantDrawableHeight: 46},
		{width: 64, height: 64, wantMargin: 12, wantDrawableWidth: 40, wantDrawableHeight: 40},
		{width: 128, height: 128, wantMargin: 24, wantDrawableWidth: 80, wantDrawableHeight: 80},
		{width: 256, height: 256, wantMargin: 48, wantDrawableWidth: 160, wantDrawableHeight: 160},
	}
	for _, test := range tests {
		margin := AnimationFrameMargin(test.width, test.height)
		if margin != test.wantMargin {
			t.Fatalf("%dx%d margin = %d, want %d", test.width, test.height, margin, test.wantMargin)
		}
		if got := test.width - 2*margin; got != test.wantDrawableWidth {
			t.Fatalf("%dx%d drawable width = %d, want %d", test.width, test.height, got, test.wantDrawableWidth)
		}
		if got := test.height - 2*margin; got != test.wantDrawableHeight {
			t.Fatalf("%dx%d drawable height = %d, want %d", test.width, test.height, got, test.wantDrawableHeight)
		}
	}
}

func TestPrototypePixelResizeOptionsKeepCanonicalMarginAndBoundOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		width, height int
		palette       int
	}{
		{width: 32, height: 32, palette: 10},
		{width: 48, height: 64, palette: 14},
		{width: 64, height: 64, palette: 14},
		{width: 128, height: 128, palette: 18},
		{width: 256, height: 256, palette: 24},
	}
	for _, test := range tests {
		options := PrototypePixelResizeOptions(test.width, test.height)
		if options.Margin != AnimationFrameMargin(test.width, test.height) {
			t.Fatalf("%dx%d margin = %d, want canonical margin %d", test.width, test.height, options.Margin, AnimationFrameMargin(test.width, test.height))
		}
		if options.Mode != RasterModePixel || !options.HardAlpha || options.PaletteSize != test.palette ||
			!options.NormalizeNearRound || !options.RemoveIsolatedComponents ||
			options.RemoveWeakEdgePixels || options.ConsolidateColourIslands || options.PreserveColourAccents ||
			!options.PreserveInternalEdges || !options.RegularizeContour {
			t.Fatalf("unexpected %dx%d prototype options: %+v", test.width, test.height, options)
		}
	}

	characterPalettes := map[int]int{32: 10, 64: 14, 128: 18, 256: 24}
	for dimension, wantPalette := range characterPalettes {
		character := CharacterPrototypePixelResizeOptions(dimension, dimension)
		object := PrototypePixelResizeOptions(dimension, dimension)
		if character.Margin != object.Margin || character.Mode != RasterModePixel || !character.HardAlpha ||
			character.NormalizeNearRound || character.RemoveIsolatedComponents ||
			character.RemoveWeakEdgePixels || character.ConsolidateColourIslands || !character.PreserveColourAccents ||
			character.PreserveInternalEdges {
			t.Fatalf("character %dx%d changed canonical geometry or enabled object-only rounding: character=%+v object=%+v", dimension, dimension, character, object)
		}
		if character.PaletteSize != wantPalette || character.PaletteSize != object.PaletteSize {
			t.Fatalf("%dx%d palette budgets diverged: character=%d object=%d, want both %d", dimension, dimension, character.PaletteSize, object.PaletteSize, wantPalette)
		}
	}

	source := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			source.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 4), G: uint8(y * 4), B: uint8((x + y) * 2),
				A: uint8(96 + (x+y)%160),
			})
		}
	}
	options := PrototypePixelResizeOptions(32, 32)
	result, report, err := ResizeImage(source, options)
	if err != nil {
		t.Fatalf("resize prototype: %v", err)
	}
	if report.Mode != RasterModePixel || report.Sampling != resizeSamplingPixelArea || !report.HardAlpha {
		t.Fatalf("unexpected prototype report: %+v", report)
	}
	colours := map[color.RGBA]struct{}{}
	for y := range result.Bounds().Dy() {
		for x := range result.Bounds().Dx() {
			pixel := result.RGBAAt(x, y)
			if pixel.A != 0 && pixel.A != 255 {
				t.Fatalf("prototype retained partial alpha at (%d,%d): %d", x, y, pixel.A)
			}
			if pixel.A == 255 {
				colours[color.RGBA{R: pixel.R, G: pixel.G, B: pixel.B, A: 255}] = struct{}{}
			}
		}
	}
	if len(colours) > options.PaletteSize {
		t.Fatalf("visible colour count = %d, palette limit = %d", len(colours), options.PaletteSize)
	}
}

func TestRegularizeNearCircularObjectSilhouetteRepairsStronglyPinchedRoundProp(t *testing.T) {
	t.Parallel()

	orange := color.NRGBA{R: 204, G: 92, B: 27, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 24, 24))
	for y := 2; y < 22; y++ {
		for x := 5; x < 18; x++ {
			dx := (float64(x) + 0.5 - 11.5) / 6.5
			dy := (float64(y) + 0.5 - 12) / 10
			if dx*dx+dy*dy <= 1 {
				img.SetNRGBA(x, y, orange)
			}
		}
	}

	regularizeNearCircularObjectSilhouette(
		img,
		cloneNRGBA(img),
		[]color.RGBA{{R: orange.R, G: orange.G, B: orange.B, A: 255}},
	)

	bounds, ok := alphaBounds(img, TransparentAlphaMax)
	if !ok || bounds.Dx() != bounds.Dy() {
		t.Fatalf("strongly pinched round prop was not restored to a square footprint: %v", bounds)
	}
}

func TestRemoveIsolatedAlphaComponentsRemovesSmallObjectSpecks(t *testing.T) {
	t.Parallel()

	fill := color.NRGBA{R: 118, G: 79, B: 42, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 5; y < 11; y++ {
		for x := 5; x < 11; x++ {
			img.SetNRGBA(x, y, fill)
		}
	}
	img.SetNRGBA(1, 1, fill)
	img.SetNRGBA(14, 13, fill)
	img.SetNRGBA(14, 14, fill)

	removeIsolatedAlphaComponents(img, 2)

	if img.NRGBAAt(1, 1).A != 0 || img.NRGBAAt(14, 13).A != 0 || img.NRGBAAt(14, 14).A != 0 {
		t.Fatal("detached object specks were not removed")
	}
	if img.NRGBAAt(7, 7).A == 0 {
		t.Fatal("main object component was removed")
	}
}

func TestRemoveIsolatedAlphaComponentsPreservesSourceSupportedDetachedDetail(t *testing.T) {
	t.Parallel()

	fill := color.NRGBA{R: 118, G: 79, B: 42, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	reference := image.NewNRGBA(img.Bounds())
	for y := 2; y < 6; y++ {
		for x := 2; x < 6; x++ {
			img.SetNRGBA(x, y, fill)
			reference.SetNRGBA(x, y, fill)
		}
	}
	// This detached two-pixel part is small, but its supersampled source
	// coverage is strong enough to be an intentional prop attachment rather
	// than a thresholding speck.
	for _, point := range []image.Point{{X: 0, Y: 0}, {X: 0, Y: 1}} {
		img.SetNRGBA(point.X, point.Y, fill)
		reference.SetNRGBA(point.X, point.Y, color.NRGBA{R: fill.R, G: fill.G, B: fill.B, A: 220})
	}

	removeIsolatedAlphaComponents(img, 2, reference)
	for _, point := range []image.Point{{X: 0, Y: 0}, {X: 0, Y: 1}} {
		if got := img.NRGBAAt(point.X, point.Y); got != fill {
			t.Fatalf("source-supported detached detail at %v was removed: %+v", point, got)
		}
	}
}

func TestRemoveWeakAlphaEdgePixelsRemovesOnlyWeakTerminalTips(t *testing.T) {
	t.Parallel()

	fill := color.NRGBA{R: 105, G: 75, B: 42, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 7, 5))
	reference := image.NewNRGBA(img.Bounds())
	for y := 1; y <= 3; y++ {
		for x := 1; x <= 3; x++ {
			img.SetNRGBA(x, y, fill)
			reference.SetNRGBA(x, y, fill)
		}
	}
	// Two equally shaped protrusions differ only in source coverage.
	img.SetNRGBA(4, 1, fill)
	reference.SetNRGBA(4, 1, color.NRGBA{R: fill.R, G: fill.G, B: fill.B, A: 120})
	img.SetNRGBA(4, 3, fill)
	reference.SetNRGBA(4, 3, color.NRGBA{R: fill.R, G: fill.G, B: fill.B, A: 220})

	removeWeakAlphaEdgePixels(img, reference)

	if got := img.NRGBAAt(4, 1); got.A != 0 {
		t.Fatalf("weak terminal antialias tip was retained: %+v", got)
	}
	if got := img.NRGBAAt(4, 3); got != fill {
		t.Fatalf("strong source-supported tip was removed: %+v", got)
	}
	if got := img.NRGBAAt(2, 2); got != fill {
		t.Fatalf("main silhouette was changed: %+v", got)
	}
}

func TestRegularizePixelContourRepairsEvidenceSupportedCornerNotch(t *testing.T) {
	t.Parallel()

	fill := color.NRGBA{R: 150, G: 90, B: 45, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 7, 7))
	reference := image.NewNRGBA(img.Bounds())
	palette := []color.RGBA{{R: fill.R, G: fill.G, B: fill.B, A: 255}}
	for y := 2; y <= 4; y++ {
		for x := 2; x <= 4; x++ {
			img.SetNRGBA(x, y, fill)
			reference.SetNRGBA(x, y, fill)
		}
	}
	// A transparent corner notch with three cardinal foreground neighbours.
	reference.SetNRGBA(3, 2, color.NRGBA{R: fill.R, G: fill.G, B: fill.B, A: 100})
	img.SetNRGBA(3, 2, color.NRGBA{})

	regularizePixelContour(img, reference, palette)

	if got := img.NRGBAAt(3, 2); got != fill {
		t.Fatalf("supported contour notch was not filled: %+v", got)
	}
}

func TestRegularizePixelContourRemovesWeakContourTooth(t *testing.T) {
	t.Parallel()

	fill := color.NRGBA{R: 90, G: 120, B: 170, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 7, 7))
	reference := image.NewNRGBA(img.Bounds())
	palette := []color.RGBA{{R: fill.R, G: fill.G, B: fill.B, A: 255}}
	for y := 2; y <= 4; y++ {
		for x := 2; x <= 3; x++ {
			img.SetNRGBA(x, y, fill)
			reference.SetNRGBA(x, y, fill)
		}
	}
	// The pixel is cardinally attached to the body but has only weak source
	// coverage, which is the signature of a one-pixel antialias tooth.
	img.SetNRGBA(4, 2, fill)
	reference.SetNRGBA(4, 2, color.NRGBA{R: fill.R, G: fill.G, B: fill.B, A: 150})

	regularizePixelContour(img, reference, palette)

	if got := img.NRGBAAt(4, 2); got.A != 0 {
		t.Fatalf("weak contour tooth was retained: %+v", got)
	}
}

func TestRegularizePixelContourPreservesEnclosedHoleAndStrongTip(t *testing.T) {
	t.Parallel()

	fill := color.NRGBA{R: 180, G: 70, B: 100, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	reference := image.NewNRGBA(img.Bounds())
	palette := []color.RGBA{{R: fill.R, G: fill.G, B: fill.B, A: 255}}
	for y := 2; y <= 5; y++ {
		for x := 2; x <= 5; x++ {
			img.SetNRGBA(x, y, fill)
			reference.SetNRGBA(x, y, fill)
		}
	}
	// This hole is not connected to the exterior background, so local three-
	// neighbour evidence must not fill it.
	img.SetNRGBA(3, 3, color.NRGBA{})
	reference.SetNRGBA(3, 3, color.NRGBA{R: fill.R, G: fill.G, B: fill.B, A: 100})
	// Strong source evidence protects a small deliberate terminal detail.
	img.SetNRGBA(6, 2, fill)
	reference.SetNRGBA(6, 2, color.NRGBA{R: fill.R, G: fill.G, B: fill.B, A: 230})

	regularizePixelContour(img, reference, palette)

	if got := img.NRGBAAt(3, 3); got.A != 0 {
		t.Fatalf("enclosed hole was filled: %+v", got)
	}
	if got := img.NRGBAAt(6, 2); got != fill {
		t.Fatalf("strong contour detail was removed: %+v", got)
	}
}

func TestRegularizeBoundaryRunsRemovesOnePixelBoundaryJitter(t *testing.T) {
	t.Parallel()

	fill := color.NRGBA{R: 120, G: 130, B: 80, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 9, 7))
	reference := image.NewNRGBA(img.Bounds())
	palette := []color.RGBA{{R: fill.R, G: fill.G, B: fill.B, A: 255}}
	for y := 1; y <= 5; y++ {
		left, right := 2, 6
		if y == 3 {
			left = 3
		}
		for x := left; x <= right; x++ {
			img.SetNRGBA(x, y, fill)
			reference.SetNRGBA(x, y, fill)
		}
	}
	// The missing boundary pixel is still supported by the supersampled source.
	reference.SetNRGBA(2, 3, color.NRGBA{R: fill.R, G: fill.G, B: fill.B, A: 100})

	regularizeBoundaryRuns(img, reference, palette)

	if got := img.NRGBAAt(2, 3); got != fill {
		t.Fatalf("one-pixel boundary jitter was not filled: %+v", got)
	}
}

func TestRegularizeBoundaryRunsPreservesLargeBoundaryChange(t *testing.T) {
	t.Parallel()

	fill := color.NRGBA{R: 70, G: 150, B: 120, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 10, 7))
	reference := image.NewNRGBA(img.Bounds())
	palette := []color.RGBA{{R: fill.R, G: fill.G, B: fill.B, A: 255}}
	for y := 1; y <= 5; y++ {
		left, right := 2, 7
		if y == 3 {
			left = 4
		}
		for x := left; x <= right; x++ {
			img.SetNRGBA(x, y, fill)
			reference.SetNRGBA(x, y, fill)
		}
	}
	// A two-pixel change is a structural feature, not a one-pixel contour tooth.

	regularizeBoundaryRuns(img, reference, palette)

	if got := img.NRGBAAt(4, 3); got != fill || img.NRGBAAt(3, 3).A != 0 {
		t.Fatalf("large boundary change was altered: edge=%+v preceding=%+v", got, img.NRGBAAt(3, 3))
	}
}

func TestStabilizeInternalHardEdgesMakesThinSourceLineCoherent(t *testing.T) {
	t.Parallel()

	base := color.NRGBA{R: 226, G: 104, B: 31, A: 255}
	blendA := color.NRGBA{R: 151, G: 70, B: 27, A: 255}
	blendB := color.NRGBA{R: 111, G: 52, B: 24, A: 255}
	line := color.RGBA{R: 42, G: 27, B: 22, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 11, 11))
	reference := image.NewNRGBA(img.Bounds())
	for y := range 11 {
		for x := range 11 {
			img.SetNRGBA(x, y, base)
			reference.SetNRGBA(x, y, base)
		}
	}
	for y := 2; y <= 8; y++ {
		quantized := blendA
		if y%2 == 0 {
			quantized = blendB
		}
		img.SetNRGBA(5, y, quantized)
		reference.SetNRGBA(5, y, color.NRGBA{R: 73, G: 38, B: 24, A: 255})
	}

	stabilizeInternalHardEdges(img, reference, []color.RGBA{
		{R: base.R, G: base.G, B: base.B, A: 255},
		line,
	})

	for y := 2; y <= 8; y++ {
		if got := img.NRGBAAt(5, y); got != (color.NRGBA{R: line.R, G: line.G, B: line.B, A: 255}) {
			t.Fatalf("internal line pixel at y=%d remained inconsistent: %+v", y, got)
		}
	}
	if got := img.NRGBAAt(4, 5); got != base {
		t.Fatalf("line stabilization changed surrounding fill: %+v", got)
	}
}

func TestStabilizeInternalHardEdgesDoesNotTraceBroadShadingBoundary(t *testing.T) {
	t.Parallel()

	light := color.NRGBA{R: 220, G: 130, B: 70, A: 255}
	shadow := color.NRGBA{R: 132, G: 76, B: 43, A: 255}
	line := color.RGBA{R: 31, G: 24, B: 20, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 12, 10))
	reference := image.NewNRGBA(img.Bounds())
	for y := range 10 {
		for x := range 12 {
			pixel := light
			if x >= 6 {
				pixel = shadow
			}
			img.SetNRGBA(x, y, pixel)
			reference.SetNRGBA(x, y, pixel)
		}
	}

	stabilizeInternalHardEdges(img, reference, []color.RGBA{
		{R: light.R, G: light.G, B: light.B, A: 255},
		{R: shadow.R, G: shadow.G, B: shadow.B, A: 255},
		line,
	})

	for y := range 10 {
		for x := range 12 {
			want := light
			if x >= 6 {
				want = shadow
			}
			if got := img.NRGBAAt(x, y); got != want {
				t.Fatalf("broad shading boundary was incorrectly outlined at (%d,%d): %+v", x, y, got)
			}
		}
	}
}

func TestStabilizeInternalHardEdgesThinsDoubledInternalStripe(t *testing.T) {
	t.Parallel()

	base := color.NRGBA{R: 225, G: 108, B: 34, A: 255}
	blendedLine := color.NRGBA{R: 79, G: 42, B: 25, A: 255}
	line := color.RGBA{R: 38, G: 26, B: 21, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 12, 12))
	reference := image.NewNRGBA(img.Bounds())
	for y := range 12 {
		for x := range 12 {
			img.SetNRGBA(x, y, base)
			reference.SetNRGBA(x, y, base)
		}
	}
	for y := 2; y <= 9; y++ {
		for x := 5; x <= 6; x++ {
			img.SetNRGBA(x, y, blendedLine)
			reference.SetNRGBA(x, y, blendedLine)
		}
	}

	stabilizeInternalHardEdges(img, reference, []color.RGBA{
		{R: base.R, G: base.G, B: base.B, A: 255},
		line,
	})

	for y := 2; y <= 9; y++ {
		left := img.NRGBAAt(5, y)
		right := img.NRGBAAt(6, y)
		linePixels := 0
		if left.R == line.R && left.G == line.G && left.B == line.B {
			linePixels++
		}
		if right.R == line.R && right.G == line.G && right.B == line.B {
			linePixels++
		}
		if linePixels != 1 {
			t.Fatalf("doubled stripe row %d retained %d stabilized line pixels: left=%+v right=%+v", y, linePixels, left, right)
		}
		if left.A != 255 || right.A != 255 {
			t.Fatalf("line thinning changed alpha at row %d: left=%+v right=%+v", y, left, right)
		}
	}
}

func TestPrototypeObjectUsesSparsePaletteForTinySilhouettes(t *testing.T) {
	t.Parallel()

	object := PrototypePixelResizeOptions(32, 32)
	if !object.AdaptiveSparsePalette {
		t.Fatal("object prototype should enable sparse palette adaptation")
	}
	character := CharacterPrototypePixelResizeOptions(32, 32)
	if character.AdaptiveSparsePalette {
		t.Fatal("character prototype should not enable sparse palette adaptation")
	}

	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for i := range 29 {
		x, y := i%5, i/5
		img.SetNRGBA(x, y, color.NRGBA{R: uint8(20 + i*7), G: uint8(40 + i*3), B: uint8(80 + i*2), A: 255})
	}
	if got := sparseSilhouettePaletteSize(img, 10); got != 4 {
		t.Fatalf("sparse palette size for 29 visible pixels = %d, want 4", got)
	}
	img = image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for i := range 40 {
		x, y := 10+i%8, 10+i/8
		img.SetNRGBA(x, y, color.NRGBA{R: 90, G: 100, B: 110, A: 255})
	}
	if got := sparseSilhouettePaletteSize(img, 10); got != 6 {
		t.Fatalf("sparse palette size for 40 visible pixels = %d, want 6", got)
	}
}
