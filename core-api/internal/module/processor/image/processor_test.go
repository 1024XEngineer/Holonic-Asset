package image

import (
	"context"
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
	if !ok || bounds != image.Rect(6, 8, 26, 25) {
		t.Fatalf("round object bounds = %v, want aspect-preserving 20x18 content", bounds)
	}
	if bounds.Dx() == bounds.Dy() {
		t.Fatal("sprite pipeline unexpectedly regularized the source ellipse into a square")
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
	if report.Sampling != resizeSamplingNearest {
		t.Fatalf("character fixture did not recover the logical grid: %+v", report)
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
		{width: 32, height: 32, palette: 16},
		{width: 48, height: 64, palette: 16},
		{width: 64, height: 64, palette: 16},
		{width: 128, height: 128, palette: 24},
		{width: 256, height: 256, palette: 24},
	}
	for _, test := range tests {
		options := PrototypePixelResizeOptions(test.width, test.height)
		if options.Margin != AnimationFrameMargin(test.width, test.height) {
			t.Fatalf("%dx%d margin = %d, want canonical margin %d", test.width, test.height, options.Margin, AnimationFrameMargin(test.width, test.height))
		}
		if options.Mode != RasterModePixel || !options.HardAlpha || !options.RecoverPixelGrid || options.PaletteSize != test.palette ||
			!options.PrequantizeBeforeResize || !options.PreferNearestReduction || !options.SpritePixelPipeline {
			t.Fatalf("unexpected %dx%d prototype options: %+v", test.width, test.height, options)
		}
	}

	characterPalettes := map[int]int{32: 16, 64: 16, 128: 24, 256: 24}
	for dimension, wantPalette := range characterPalettes {
		character := CharacterPrototypePixelResizeOptions(dimension, dimension)
		object := PrototypePixelResizeOptions(dimension, dimension)
		if character.Margin != object.Margin || character.Mode != RasterModePixel || !character.HardAlpha || !character.RecoverPixelGrid ||
			!character.PrequantizeBeforeResize || !character.PreferNearestReduction || !character.SpritePixelPipeline {
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
	if report.Mode != RasterModePixel || report.Sampling != resizeSamplingNearest || !report.HardAlpha {
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

func TestResizeImagePrototypeRecoversSupersampledLogicalGrid(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	red := color.NRGBA{R: 220, G: 40, B: 40, A: 255}
	blue := color.NRGBA{R: 40, G: 80, B: 220, A: 255}
	for y := range 8 {
		for x := range 8 {
			pixel := red
			if x >= 4 {
				pixel = blue
			}
			source.SetNRGBA(x, y, pixel)
		}
	}

	options := DefaultResizeOptions(2, 2)
	options.Margin = 0
	options.CropContent = false
	options.Mode = RasterModePixel
	options.RecoverPixelGrid = true
	result, report, err := ResizeImage(source, options)
	if err != nil {
		t.Fatalf("resize prototype: %v", err)
	}
	if report.Sampling != pixelGridSamplingRecovered {
		t.Fatalf("sampling = %q, want %q", report.Sampling, pixelGridSamplingRecovered)
	}
	for y := range 2 {
		for x := range 2 {
			want := red
			if x == 1 {
				want = blue
			}
			if got := result.RGBAAt(x, y); got != color.RGBA(want) {
				t.Fatalf("pixel at (%d,%d) = %+v, want %+v", x, y, got, want)
			}
		}
	}
}
