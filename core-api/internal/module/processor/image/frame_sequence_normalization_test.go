package image

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
)

func TestNormalizeAnimationImageStabilizesFramesWithSharedCrop(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 80, 40))
	// The same foreground shape is displaced differently in each source cell.
	fillRect(src, image.Rect(7, 8, 19, 34), color.NRGBA{R: 220, G: 70, B: 40, A: 255})
	fillRect(src, image.Rect(57, 4, 69, 30), color.NRGBA{R: 220, G: 70, B: 40, A: 255})
	// A moving arm changes the silhouette but must not become a per-frame crop.
	fillRect(src, image.Rect(19, 14, 28, 19), color.NRGBA{R: 220, G: 70, B: 40, A: 255})
	fillRect(src, image.Rect(48, 11, 57, 16), color.NRGBA{R: 220, G: 70, B: 40, A: 255})

	result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1,
		FrameWidth: 48, FrameHeight: 48, Margin: 3,
	})
	if err != nil {
		t.Fatalf("normalize animation: %v", err)
	}
	if len(result.Frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(result.Frames))
	}
	if result.Report.GridPolicy != "proportional_fixed_cells" {
		t.Fatalf("grid policy = %q", result.Report.GridPolicy)
	}
	if result.Report.OutputAnchorRange.X > 2 || result.Report.OutputAnchorRange.Y > 2 {
		t.Fatalf("output anchors still drift: %+v", result.Report.OutputAnchorRange)
	}
	if result.Frames[0].Translation == (AnimationOffset{}) || result.Frames[1].Translation == (AnimationOffset{}) {
		t.Fatalf("expected source displacement to be corrected: %+v %+v", result.Frames[0].Translation, result.Frames[1].Translation)
	}
	for i, frame := range result.Frames {
		decoded, err := DecodeBase64Image(frame.ImageBase64)
		if err != nil {
			t.Fatalf("decode frame %d: %v", i, err)
		}
		if got := decoded.Bounds().Size(); got != image.Pt(48, 48) {
			t.Fatalf("frame %d size = %v, want 48x48", i, got)
		}
	}
	if result.Report.SharedCrop.Width <= 0 || result.Report.SharedCrop.Height <= 0 {
		t.Fatalf("invalid shared crop: %+v", result.Report.SharedCrop)
	}
}

func TestNormalizeAnimationImageSupportsIntentionalZeroMargin(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(src, image.Rect(8, 8, 24, 24), color.NRGBA{R: 220, G: 70, B: 40, A: 255})

	legacy, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 1, Rows: 1, FrameWidth: 32, FrameHeight: 32, Margin: 0,
	})
	if err != nil {
		t.Fatalf("normalize with legacy zero margin: %v", err)
	}
	exact, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 1, Rows: 1, FrameWidth: 32, FrameHeight: 32,
		Margin: 0, UseExactMargin: true,
	})
	if err != nil {
		t.Fatalf("normalize with exact zero margin: %v", err)
	}

	if legacy.Report.Margin != defaultAssetMargin(32, 32) {
		t.Fatalf("legacy margin = %d, want default %d", legacy.Report.Margin, defaultAssetMargin(32, 32))
	}
	if exact.Report.Margin != 0 {
		t.Fatalf("exact margin = %d, want 0", exact.Report.Margin)
	}
	if exact.Report.Scale <= legacy.Report.Scale {
		t.Fatalf("exact zero margin scale = %f, want greater than legacy scale %f", exact.Report.Scale, legacy.Report.Scale)
	}
}

func TestNormalizeAnimationImagePreservesRequestedVerticalMotion(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 60, 40))
	fillRect(src, image.Rect(9, 14, 21, 36), color.NRGBA{B: 255, A: 255})
	fillRect(src, image.Rect(39, 4, 51, 26), color.NRGBA{B: 255, A: 255})

	result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1,
		FrameWidth: 40, FrameHeight: 40, PreserveVerticalMotion: true,
	})
	if err != nil {
		t.Fatalf("normalize animation: %v", err)
	}
	if result.Frames[0].Translation.Y != 0 || result.Frames[1].Translation.Y != 0 {
		t.Fatalf("vertical translations should remain disabled: %+v %+v", result.Frames[0].Translation, result.Frames[1].Translation)
	}
	if result.Report.OutputAnchorRange.Y <= 1 {
		t.Fatalf("expected vertical motion to remain, range=%+v", result.Report.OutputAnchorRange)
	}
}

func TestNormalizeAnimationImageCanRemoveGeneratedFlatBackground(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	fillRect(src, src.Bounds(), color.NRGBA{G: 255, A: 255})
	fillRect(src, image.Rect(5, 3, 15, 18), color.NRGBA{R: 230, G: 40, B: 30, A: 255})
	fillRect(src, image.Rect(25, 3, 35, 18), color.NRGBA{R: 230, G: 40, B: 30, A: 255})

	result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1,
		FrameWidth: 24, FrameHeight: 24,
		Background: &AnimationBackgroundOptions{MatteColor: "#00ff00", Material: MaterialFlatIcon},
	})
	if err != nil {
		t.Fatalf("normalize green-screen animation: %v", err)
	}
	if result.Report.BackgroundRemovalReport == nil {
		t.Fatal("expected background removal report")
	}
	for i, frame := range result.Frames {
		if frame.ContentBounds == nil {
			t.Fatalf("frame %d has no visible content", i)
		}
	}
}

func TestNormalizeAnimationImageRejectsOpaqueSourceWithoutBackgroundOptions(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	fillRect(src, src.Bounds(), color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	_, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1,
	})
	if err == nil {
		t.Fatal("expected opaque-source error")
	}
}

func TestNormalizeAnimationImagePreservesSourceCellScaleWhenActionExtendsProp(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 80, 40))
	body := color.NRGBA{R: 220, G: 70, B: 40, A: 255}
	prop := color.NRGBA{B: 220, A: 255}

	// Both source cells contain the same body at the same canonical scale. The
	// second pose additionally has a much longer held prop. A union-bounds fit
	// would scale both bodies down; source-cell scale must not.
	fillRect(src, image.Rect(12, 8, 20, 34), body)
	fillRect(src, image.Rect(52, 8, 60, 34), body)
	fillRect(src, image.Rect(20, 12, 25, 16), prop)
	fillRect(src, image.Rect(40, 10, 79, 17), prop)

	result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1, FrameWidth: 64, FrameHeight: 64,
		Anchor: AnimationAnchorFeet, PreserveSourceCellScale: true,
	})
	if err != nil {
		t.Fatalf("normalize fixed source-cell scale: %v", err)
	}
	if got, want := result.Report.Scale, 1.6; math.Abs(got-want) > 0.001 {
		t.Fatalf("scale = %f, want source-cell scale %f", got, want)
	}
	if result.Report.RegistrationPolicy != "median_root_anchor_shared_source_cell_canvas_fixed_scale_no_per_frame_recentering" {
		t.Fatalf("registration policy = %q", result.Report.RegistrationPolicy)
	}

	widths, heights := make([]int, 0, len(result.Frames)), make([]int, 0, len(result.Frames))
	for index, frame := range result.Frames {
		decoded, decodeErr := DecodeBase64Image(frame.ImageBase64)
		if decodeErr != nil {
			t.Fatalf("decode frame %d: %v", index, decodeErr)
		}
		if got := decoded.Bounds().Size(); got != image.Pt(64, 64) {
			t.Fatalf("frame %d size = %v, want 64x64", index, got)
		}
		bounds, ok := solidRedBounds(decoded)
		if !ok {
			t.Fatalf("frame %d has no body pixels", index)
		}
		widths = append(widths, bounds.Dx())
		heights = append(heights, bounds.Dy())
	}
	if widths[0] != widths[1] || heights[0] != heights[1] {
		t.Fatalf("body scale changed across prop extension: widths=%v heights=%v", widths, heights)
	}
}

func solidRedBounds(input image.Image) (image.Rectangle, bool) {
	bounds := input.Bounds()
	result := image.Rectangle{}
	found := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := color.NRGBAModel.Convert(input.At(x, y)).(color.NRGBA)
			if pixel.A < 128 || pixel.R < 150 || pixel.G > 130 || pixel.B > 130 {
				continue
			}
			point := image.Pt(x, y)
			if !found {
				result = image.Rectangle{Min: point, Max: point.Add(image.Pt(1, 1))}
				found = true
				continue
			}
			result = result.Union(image.Rectangle{Min: point, Max: point.Add(image.Pt(1, 1))})
		}
	}
	return result, found
}

func TestNormalizeAnimationImageCanNormalizeStaticDirectionContentScale(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 80, 50))
	colorValue := color.NRGBA{R: 200, G: 90, B: 40, A: 255}
	// The same static foreground appears at two visibly different heights.
	// It is also placed much higher in the second tall cell. Direction-sheet
	// normalization should correct both errors before rendering.
	fillRect(src, image.Rect(12, 10, 28, 40), colorValue)
	fillRect(src, image.Rect(54, 5, 66, 20), colorValue)

	result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1, FrameWidth: 64, FrameHeight: 64,
		Anchor: AnimationAnchorFeet, NormalizeContentScale: true,
	})
	if err != nil {
		t.Fatalf("normalize direction content scale: %v", err)
	}
	if !result.Report.ContentScaleNormalized {
		t.Fatal("content scale normalization was not reported")
	}
	if result.Report.ContentHeightMedian != 22.5 {
		t.Fatalf("median content height = %f, want 22.5", result.Report.ContentHeightMedian)
	}
	if result.Report.RegistrationPolicy != "median_content_height_per_cell_scale_median_root_anchor_shared_union_crop" {
		t.Fatalf("registration policy = %q", result.Report.RegistrationPolicy)
	}
	if result.Report.TranslationClamped != 0 {
		t.Fatalf("static direction registration was clamped: %+v", result.Report)
	}

	heights := make([]int, 0, len(result.Frames))
	bottoms := make([]int, 0, len(result.Frames))
	for index, frame := range result.Frames {
		decoded, decodeErr := DecodeBase64Image(frame.ImageBase64)
		if decodeErr != nil {
			t.Fatalf("decode frame %d: %v", index, decodeErr)
		}
		bounds, ok := alphaBoundsNRGBA(toNRGBA(decoded), defaultImageSplitAlphaThreshold)
		if !ok {
			t.Fatalf("frame %d has no visible content", index)
		}
		heights = append(heights, bounds.Dy())
		bottoms = append(bottoms, bounds.Max.Y)
	}
	if absInt(heights[0]-heights[1]) > 1 {
		t.Fatalf("normalized content heights differ: %v", heights)
	}
	if absInt(bottoms[0]-bottoms[1]) > 1 {
		t.Fatalf("normalized baselines differ: %v", bottoms)
	}
}

func TestNormalizeAnimationImageCentersStaticObjectFramesAfterSharedCrop(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	value := color.NRGBA{R: 220, G: 80, B: 30, A: 255}
	// Deliberately put equal-sized views at unrelated positions inside the
	// source cells. This mirrors a model-generated 2x2 prototype sheet and
	// catches the old failure where each returned 128x128 frame inherited the
	// source cell's off-centre placement.
	localBounds := []image.Rectangle{
		image.Rect(29, 20, 53, 43),
		image.Rect(26, 20, 50, 46),
		image.Rect(29, 5, 53, 28),
		image.Rect(15, 18, 39, 41),
	}
	for index, bounds := range localBounds {
		offset := image.Pt((index%2)*64, (index/2)*64)
		fillRect(src, bounds.Add(offset), value)
	}

	result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 2, FrameCount: 4,
		FrameWidth: 32, FrameHeight: 32, RenderScale: 4,
		Margin: animationFrameMarginForTest(32, 32),
		Anchor: AnimationAnchorCenter, NormalizeContentArea: true,
		CenterContent: true, AlphaThreshold: PixelAlphaThreshold,
	})
	if err != nil {
		t.Fatalf("normalize static object frames: %v", err)
	}
	wantCenter := image.Point{X: 64, Y: 64}
	for index, frame := range result.Frames {
		decoded, decodeErr := DecodeBase64Image(frame.ImageBase64)
		if decodeErr != nil {
			t.Fatalf("decode frame %d: %v", index, decodeErr)
		}
		bounds, ok := alphaBoundsNRGBA(toNRGBA(decoded), PixelAlphaThreshold)
		if !ok {
			t.Fatalf("frame %d has no visible content", index)
		}
		gotCenter := image.Point{
			X: (bounds.Min.X + bounds.Max.X) / 2,
			Y: (bounds.Min.Y + bounds.Max.Y) / 2,
		}
		if gotCenter != wantCenter {
			t.Fatalf("frame %d bbox=%v center=%v, want center=%v", index, bounds, gotCenter, wantCenter)
		}
	}
	if result.Report.OutputAnchorRange.X > 1 || result.Report.OutputAnchorRange.Y > 1 {
		t.Fatalf("center postcondition left excessive anchor drift: %+v", result.Report.OutputAnchorRange)
	}
}

func TestNormalizeAnimationImageCanNormalizeStaticObjectContentArea(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 80, 40))
	value := color.NRGBA{R: 180, G: 100, B: 40, A: 255}
	// Equal-area views with deliberately different aspect ratios. Height-based
	// normalization would make the 10x20 view visibly larger than the 20x10
	// view; object normalization should preserve their visual footprint instead.
	fillRect(src, image.Rect(15, 10, 25, 30), value)
	fillRect(src, image.Rect(50, 15, 70, 25), value)

	result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1, FrameWidth: 64, FrameHeight: 64,
		Anchor: AnimationAnchorCenter, NormalizeContentArea: true, CenterContent: true,
	})
	if err != nil {
		t.Fatalf("normalize object content area: %v", err)
	}
	if !result.Report.ContentAreaNormalized || result.Report.ContentAreaMedian != 200 {
		t.Fatalf("unexpected area normalization report: %+v", result.Report)
	}
	if result.Report.RegistrationPolicy != "median_content_area_per_cell_scale_median_root_anchor_shared_union_crop_per_frame_center_postcondition" {
		t.Fatalf("registration policy = %q", result.Report.RegistrationPolicy)
	}

	areas := make([]int, 0, len(result.Frames))
	for index, frame := range result.Frames {
		decoded, decodeErr := DecodeBase64Image(frame.ImageBase64)
		if decodeErr != nil {
			t.Fatalf("decode frame %d: %v", index, decodeErr)
		}
		areas = append(areas, animationOpaqueArea(toNRGBA(decoded), defaultImageSplitAlphaThreshold))
	}
	if len(areas) != 2 || absInt(areas[0]-areas[1]) > max(areas[0], areas[1])/10 {
		t.Fatalf("normalized object areas differ too much: %v", areas)
	}
}

func TestNormalizeAnimationImageRejectsTwoContentNormalizationPolicies(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	fillRect(src, image.Rect(1, 1, 9, 9), color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	fillRect(src, image.Rect(11, 1, 19, 9), color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	_, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1, FrameWidth: 32, FrameHeight: 32,
		Anchor: AnimationAnchorCenter, NormalizeContentScale: true,
		NormalizeContentArea: true,
	})
	if err == nil {
		t.Fatal("expected mutually exclusive content normalization policies to fail")
	}
}

func TestNormalizeAnimationImagePreservesBrightGreenSubjectOnGreenMatte(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 48, 24))
	fillRect(src, src.Bounds(), color.NRGBA{G: 255, A: 255})
	fillRect(src, image.Rect(6, 4, 18, 20), color.NRGBA{R: 15, G: 25, B: 20, A: 255})
	fillRect(src, image.Rect(9, 7, 15, 18), color.NRGBA{G: 255, A: 255})
	fillRect(src, image.Rect(30, 4, 42, 20), color.NRGBA{R: 15, G: 25, B: 20, A: 255})
	fillRect(src, image.Rect(33, 7, 39, 18), color.NRGBA{G: 255, A: 255})

	result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1, FrameWidth: 32, FrameHeight: 32,
		Background: &AnimationBackgroundOptions{MatteColor: "#00ff00", BorderConnectedOnly: true},
	})
	if err != nil {
		t.Fatalf("normalize green-on-green animation: %v", err)
	}
	for index, frame := range result.Frames {
		decoded, decodeErr := DecodeBase64Image(frame.ImageBase64)
		if decodeErr != nil {
			t.Fatalf("decode frame %d: %v", index, decodeErr)
		}
		foundGreen := false
		for y := decoded.Bounds().Min.Y; y < decoded.Bounds().Max.Y; y++ {
			for x := decoded.Bounds().Min.X; x < decoded.Bounds().Max.X; x++ {
				pixel := color.NRGBAModel.Convert(decoded.At(x, y)).(color.NRGBA)
				if pixel.A >= 250 && pixel.G >= 240 && pixel.R <= 10 && pixel.B <= 10 {
					foundGreen = true
				}
			}
		}
		if !foundGreen {
			t.Fatalf("frame %d lost bright green subject", index)
		}
	}
}

func TestNormalizeAnimationImageCompensatesExpandedReferenceScaleAcrossSequence(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 200, 100))
	body := color.NRGBA{R: 220, G: 70, B: 40, A: 255}
	fillRect(src, image.Rect(40, 35, 60, 65), body)
	fillRect(src, image.Rect(142, 37, 162, 67), body)

	base, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1, FrameWidth: 100, FrameHeight: 100,
		Anchor: AnimationAnchorFeet, PreserveSourceCellScale: true,
		PreserveHorizontalMotion: true, PreserveVerticalMotion: true,
	})
	if err != nil {
		t.Fatalf("normalize base source-cell scale: %v", err)
	}
	compensated, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1, FrameWidth: 100, FrameHeight: 100,
		Anchor: AnimationAnchorFeet, PreserveSourceCellScale: true,
		SourceCellScaleMultiplier: 1.875,
		PreserveHorizontalMotion:  true, PreserveVerticalMotion: true,
	})
	if err != nil {
		t.Fatalf("normalize compensated source-cell scale: %v", err)
	}
	if got, want := compensated.Report.RequestedSourceCellScaleMultiplier, 1.875; math.Abs(got-want) > 0.001 {
		t.Fatalf("requested scale multiplier = %f, want %f", got, want)
	}
	if got, want := compensated.Report.AppliedSourceCellScaleMultiplier, 1.875; math.Abs(got-want) > 0.001 {
		t.Fatalf("applied scale multiplier = %f, want %f", got, want)
	}
	if len(compensated.Report.Warnings) != 0 {
		t.Fatalf("unexpected compensation warnings: %v", compensated.Report.Warnings)
	}

	baseBounds := animationResultContentBounds(t, base)
	compensatedBounds := animationResultContentBounds(t, compensated)
	for index := range baseBounds {
		if got, want := compensatedBounds[index].Dx(), int(math.Round(float64(baseBounds[index].Dx())*1.875)); absInt(got-want) > 2 {
			t.Fatalf("frame %d compensated width = %d, want about %d (base %d)", index, got, want, baseBounds[index].Dx())
		}
		if got, want := compensatedBounds[index].Dy(), int(math.Round(float64(baseBounds[index].Dy())*1.875)); absInt(got-want) > 2 {
			t.Fatalf("frame %d compensated height = %d, want about %d (base %d)", index, got, want, baseBounds[index].Dy())
		}
	}
	baseMotion := rectangleCenterX(baseBounds[1]) - rectangleCenterX(baseBounds[0])
	compensatedMotion := rectangleCenterX(compensatedBounds[1]) - rectangleCenterX(compensatedBounds[0])
	if math.Abs(compensatedMotion-baseMotion*1.875) > 1.1 {
		t.Fatalf("compensated horizontal motion = %.2f, want about %.2f", compensatedMotion, baseMotion*1.875)
	}
}

func TestNormalizeAnimationImageClampsScaleCompensationBeforeClipping(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	fillRect(src, image.Rect(5, 15, 95, 85), color.NRGBA{R: 220, G: 70, B: 40, A: 255})

	result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 1, Rows: 1, FrameWidth: 100, FrameHeight: 100,
		Anchor: AnimationAnchorCenter, PreserveSourceCellScale: true,
		SourceCellScaleMultiplier: 1.875,
	})
	if err != nil {
		t.Fatalf("normalize clamped source-cell scale: %v", err)
	}
	if result.Report.AppliedSourceCellScaleMultiplier >= result.Report.RequestedSourceCellScaleMultiplier {
		t.Fatalf("scale compensation was not clamped: requested=%f applied=%f", result.Report.RequestedSourceCellScaleMultiplier, result.Report.AppliedSourceCellScaleMultiplier)
	}
	if len(result.Report.Warnings) == 0 {
		t.Fatal("expected a clipping-prevention warning")
	}
	bounds := animationResultContentBounds(t, result)[0]
	if bounds.Min.X < 0 || bounds.Min.Y < 0 || bounds.Max.X > 100 || bounds.Max.Y > 100 {
		t.Fatalf("clamped content escaped target canvas: %v", bounds)
	}
}

func animationResultContentBounds(t *testing.T, result *normalizedAnimation) []image.Rectangle {
	t.Helper()
	bounds := make([]image.Rectangle, len(result.Frames))
	for index, frame := range result.Frames {
		decoded, err := DecodeBase64Image(frame.ImageBase64)
		if err != nil {
			t.Fatalf("decode normalized frame %d: %v", index, err)
		}
		frameBounds, ok := alphaBoundsNRGBA(toNRGBA(decoded), defaultImageSplitAlphaThreshold)
		if !ok {
			t.Fatalf("normalized frame %d has no foreground", index)
		}
		bounds[index] = frameBounds
	}
	return bounds
}

func rectangleCenterX(bounds image.Rectangle) float64 {
	return float64(bounds.Min.X+bounds.Max.X) / 2
}

func TestNormalizeAnimationImageBackgroundOptionsCoverMatteAndChromaPaths(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	fillRect(src, src.Bounds(), color.NRGBA{G: 255, A: 255})
	fillRect(src, image.Rect(5, 3, 15, 18), color.NRGBA{R: 230, G: 40, B: 30, A: 255})
	fillRect(src, image.Rect(25, 3, 35, 18), color.NRGBA{R: 230, G: 40, B: 30, A: 255})

	tests := []struct {
		name       string
		background AnimationBackgroundOptions
	}{
		{name: "auto matte", background: AnimationBackgroundOptions{MatteColor: "  auto  "}},
		{name: "explicit matte", background: AnimationBackgroundOptions{MatteColor: "#00ff00"}},
		{
			name: "border connected",
			background: AnimationBackgroundOptions{
				MatteColor:          "#00ff00",
				BorderConnectedOnly: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
				Columns: 2, Rows: 1,
				FrameWidth: 24, FrameHeight: 24,
				Background: &test.background,
			})
			if err != nil {
				t.Fatalf("normalize animation background: %v", err)
			}
			if result.Report.BackgroundRemovalReport == nil {
				t.Fatal("expected background removal report")
			}
			for index, frame := range result.Frames {
				if frame.ContentBounds == nil {
					t.Fatalf("frame %d has no visible content", index)
				}
			}
		})
	}

	_, _, err := removeAnimationBackground(src, AnimationBackgroundOptions{MatteColor: "invalid-color"})
	if err == nil || !strings.Contains(err.Error(), "parse animation matte color") {
		t.Fatalf("expected matte parsing error, got %v", err)
	}
}
