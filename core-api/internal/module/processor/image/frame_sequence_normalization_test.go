package image

import (
	"image"
	"image/color"
	"math"
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
		Anchor: AnimationAnchorCenter, NormalizeContentArea: true,
	})
	if err != nil {
		t.Fatalf("normalize object content area: %v", err)
	}
	if !result.Report.ContentAreaNormalized || result.Report.ContentAreaMedian != 200 {
		t.Fatalf("unexpected area normalization report: %+v", result.Report)
	}
	if result.Report.RegistrationPolicy != "median_content_area_per_cell_scale_median_root_anchor_shared_union_crop" {
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
