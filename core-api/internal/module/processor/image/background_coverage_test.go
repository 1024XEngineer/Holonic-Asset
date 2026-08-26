package image

import (
	"image"
	"image/color"
	"testing"
)

func TestResolveChromaSettingsBranches(t *testing.T) {
	t.Parallel()

	th := 35.0
	soft := 45.0
	spill := 0.65

	settings := ResolveChromaSettings(MaterialFlatIcon, &th, &soft, &spill)
	if settings.Threshold != 35.0 || settings.Softness != 45.0 || settings.SpillSuppression != 0.65 {
		t.Fatalf("unexpected settings: %#v", settings)
	}

	// Test zero defaults and normalization
	zeroTh := 0.0
	zeroSoft := 0.0
	negSpill := -0.5
	s2 := ResolveChromaSettings(MaterialStandard, &zeroTh, &zeroSoft, &negSpill)
	if s2.Threshold != DefaultChromaThreshold || s2.Softness != DefaultChromaSoftness || s2.SpillSuppression != 0 {
		t.Fatalf("unexpected normalized settings: %#v", s2)
	}
}

func TestExtractDualAndDualAlignmentReport(t *testing.T) {
	t.Parallel()

	// 1. Matching dark/light background images
	w, h := 16, 16
	dark := image.NewRGBA(image.Rect(0, 0, w, h))
	light := image.NewRGBA(image.Rect(0, 0, w, h))

	// Background: dark is (0,0,0), light is (255,255,255)
	for y := range h {
		for x := range w {
			dark.Set(x, y, color.RGBA{0, 0, 0, 255})
			light.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	// Subject in center (4,4) to (12,12): opaque red (200, 50, 50) on both
	for y := 4; y < 12; y++ {
		for x := 4; x < 12; x++ {
			dark.Set(x, y, color.RGBA{200, 50, 50, 255})
			light.Set(x, y, color.RGBA{200, 50, 50, 255})
		}
	}

	result := ExtractDual(dark, light)
	if result.Bounds().Dx() != w || result.Bounds().Dy() != h {
		t.Fatalf("unexpected bounds: %v", result.Bounds())
	}
	// Background should be transparent
	if a := result.RGBAAt(0, 0).A; a != 0 {
		t.Fatalf("background alpha = %d, want 0", a)
	}
	// Subject should be opaque
	if a := result.RGBAAt(8, 8).A; a < 250 {
		t.Fatalf("subject alpha = %d, want >= 250", a)
	}

	// Alignment report
	report := DualAlignmentReportFor(dark, light)
	if !report.Passed || report.Score < 0.55 {
		t.Fatalf("expected alignment pass, got %#v", report)
	}

	// 2. Mismatched images with noise and negative deltas
	noisyLight := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			// light is darker than dark (negative delta)
			noisyLight.Set(x, y, color.RGBA{0, 0, 0, 255})
			dark.Set(x, y, color.RGBA{255, 100, 50, 255})
		}
	}
	noisyReport := DualAlignmentReportFor(dark, noisyLight)
	if noisyReport.Passed || noisyReport.Score >= 0.55 {
		t.Fatalf("expected alignment fail for inverted delta, got %#v", noisyReport)
	}

	// 3. Zero pixels image bounds
	emptyDark := image.NewRGBA(image.Rect(0, 0, 0, 0))
	emptyLight := image.NewRGBA(image.Rect(0, 0, 0, 0))
	emptyReport := DualAlignmentReportFor(emptyDark, emptyLight)
	if emptyReport.Score != 1.0 {
		t.Fatalf("expected score 1.0 for empty images, got %f", emptyReport.Score)
	}
}

func TestExtractChromaWithHighForegroundOverlap(t *testing.T) {
	t.Parallel()

	matte := MatteColor{0, 255, 0}
	source := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	// Fill background with green
	fillRect(source, source.Bounds(), color.NRGBA{G: 255, A: 255})

	// Fill center with key-like transition green (distance > nearMatteThreshold and < opaqueThreshold)
	// Matte is (0, 255, 0). Transition pixel: (10, 240, 10)
	fillRect(source, image.Rect(4, 4, 28, 28), color.NRGBA{R: 15, G: 235, B: 15, A: 255})

	// ExtractChroma will evaluate chromaHasHighForegroundKeyOverlap -> true and use extractGlobalDistanceChroma
	settings := ChromaSettings{Threshold: 10, Softness: 40, SpillSuppression: 0.5}
	out := ExtractChroma(source, matte, settings)
	if out == nil {
		t.Fatal("expected non-nil result from ExtractChroma")
	}
}

func TestRemoveSmallEdgeComponentsCases(t *testing.T) {
	t.Parallel()

	// 1. Zero size
	emptyImg := image.NewRGBA(image.Rect(0, 0, 0, 0))
	if removed := RemoveSmallEdgeComponents(emptyImg); removed != 0 {
		t.Fatalf("expected 0 removed for empty image, got %d", removed)
	}

	// 2. All transparent
	allTrans := image.NewRGBA(image.Rect(0, 0, 16, 16))
	if removed := RemoveSmallEdgeComponents(allTrans); removed != 0 {
		t.Fatalf("expected 0 removed for all transparent, got %d", removed)
	}

	// 3. Image with large central subject and small 2-pixel edge noise
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	// Large subject in center (10x10 = 100 pixels)
	for y := 10; y < 20; y++ {
		for x := 10; x < 20; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	// Small edge noise at border (x=0, y=0) and (x=1, y=0)
	img.SetRGBA(0, 0, color.RGBA{R: 255, A: 50})
	img.SetRGBA(1, 0, color.RGBA{R: 255, A: 50})

	removed := RemoveSmallEdgeComponents(img)
	if removed != 2 {
		t.Fatalf("expected 2 edge noise pixels removed, got %d", removed)
	}
	if img.RGBAAt(0, 0).A != 0 || img.RGBAAt(1, 0).A != 0 {
		t.Fatal("edge noise pixels were not scrubbed")
	}
	if img.RGBAAt(15, 15).A != 255 {
		t.Fatal("central subject should be preserved")
	}

	// 4. Component touching edge that is equal to largest (e.g. only 1 component)
	onlyEdge := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 5 {
		for x := range 5 {
			onlyEdge.SetRGBA(x, y, color.RGBA{R: 200, A: 255})
		}
	}
	// Since it's the largest (and only) component, it must not be removed
	if r := RemoveSmallEdgeComponents(onlyEdge); r != 0 {
		t.Fatalf("largest component touching edge must not be removed, got %d", r)
	}
}

func TestExtractChromaHelperBranches(t *testing.T) {
	t.Parallel()

	// 1. extractBorderConnectedChromaMode with 0 size
	emptyImg := image.NewNRGBA(image.Rect(0, 0, 0, 0))
	out := extractBorderConnectedChroma(emptyImg, MatteColor{0, 255, 0}, ChromaSettings{})
	if out.Bounds().Dx() != 0 {
		t.Fatalf("expected empty bounds, got %v", out.Bounds())
	}

	// 2. chromaDominanceAlpha with dark matte (no spill channels)
	darkMatte := MatteColor{50, 50, 50}
	if a := chromaDominanceAlpha(MatteColor{100, 100, 100}, darkMatte); a != 255 {
		t.Fatalf("expected 255 for matte without spill channels, got %d", a)
	}

	// 3. chromaDominanceAlpha with multi-spill channels (e.g. cyan key [0, 200, 200])
	cyanKey := MatteColor{0, 200, 200}
	aCyan := chromaDominanceAlpha(MatteColor{10, 220, 220}, cyanKey)
	if aCyan == 255 {
		t.Fatalf("expected dominance attenuation for cyan key, got %d", aCyan)
	}

	// 4. chromaLooksKeyColored with distance <= 32 or no spill channels
	if !chromaLooksKeyColored(MatteColor{10, 10, 10}, darkMatte, 10) {
		t.Fatal("expected true for distance <= 32")
	}
	if !chromaLooksKeyColored(MatteColor{200, 200, 200}, darkMatte, 100) {
		t.Fatal("expected true for key with no spill channels")
	}

	// 5. chromaCleanupSpill with amount <= 0, alpha >= 252, or dark key
	rgb := MatteColor{200, 100, 50}
	if res := chromaCleanupSpill(rgb, cyanKey, 100, 0); res != rgb {
		t.Fatal("expected unchanged for amount <= 0")
	}
	if res := chromaCleanupSpill(rgb, cyanKey, 253, 0.8); res != rgb {
		t.Fatal("expected unchanged for alpha >= 252")
	}
	if res := chromaCleanupSpill(rgb, darkMatte, 100, 0.8); res != rgb {
		t.Fatal("expected unchanged for key without spill channels")
	}

	// 6. contractChromaAlpha & featherChromaAlpha with invalid values
	testRGBA := image.NewRGBA(image.Rect(0, 0, 8, 8))
	testRGBA.SetRGBA(4, 4, color.RGBA{R: 255, A: 255})
	contractChromaAlpha(testRGBA, 0)
	contractChromaAlpha(testRGBA, -1)
	featherChromaAlpha(testRGBA, 0)
	featherChromaAlpha(testRGBA, -0.5)

	// Valid contraction and feathering
	contractChromaAlpha(testRGBA, 1)
	featherChromaAlpha(testRGBA, 1.0)

	// 7. suppressMatteSpill
	p := color.NRGBA{R: 10, G: 200, B: 10, A: 200}
	suppressMatteSpill(&p, MatteColor{0, 255, 0}, 200, 0.8)
	if p.G > 200 {
		t.Fatal("suppressMatteSpill should suppress excess green")
	}
	// suppressMatteSpill with low amount or alpha
	pZero := color.NRGBA{R: 10, G: 200, B: 10, A: 0}
	suppressMatteSpill(&pZero, MatteColor{0, 255, 0}, 0, 0.8)
	suppressMatteSpill(&p, MatteColor{50, 50, 50}, 200, 0.8) // dark matte

	// 8. preserveSourceKeyChannelOrder
	source := [4]uint8{20, 200, 20, 255}
	preserveSourceKeyChannelOrder(&p, source, MatteColor{0, 255, 0})
	preserveSourceKeyChannelOrder(&p, source, darkMatte)
}

func TestIncludeEnclosedChromaComponentsLowConfidence(t *testing.T) {
	t.Parallel()

	// Create an image where an enclosed component has candidate pixels but:
	// strongSeeds == 0 or meanConfidence < chromaEnclosedMatteMinMeanConfidence
	w, h := 32, 32
	pixels := make([]chromaCandidatePixel, w*h)
	for i := range pixels {
		pixels[i] = chromaCandidatePixel{
			sourceAlpha: 255,
			outputAlpha: 255,
			candidate:   false,
		}
	}

	// Create enclosed component at center (12,12) to (20,20)
	// with candidate = true, but outputAlpha is moderate (e.g. 200 > chromaAlphaNoiseFloor 8)
	// so strongSeeds remains 0
	for y := 12; y < 20; y++ {
		for x := 12; x < 20; x++ {
			idx := y*w + x
			pixels[idx] = chromaCandidatePixel{
				rgb:         MatteColor{0, 200, 0},
				sourceAlpha: 255,
				outputAlpha: 200,
				candidate:   true,
				keyLike:     true,
			}
		}
	}

	connected := make([]bool, len(pixels))
	includeEnclosedChromaComponents(pixels, connected, w, h)

	// Since strongSeeds == 0, center pixels should NOT be marked connected
	if connected[15*w+15] {
		t.Fatal("expected enclosed component with zero strong seeds to remain unconnected")
	}
}

func TestExtractChromaWithReportBranches(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	fillRect(source, source.Bounds(), color.NRGBA{G: 255, A: 255})
	fillRect(source, image.Rect(4, 4, 12, 12), color.NRGBA{R: 200, G: 50, B: 50, A: 255})

	// 1. ExtractChromaWithReport with nil matte (auto-sampled)
	out, report := ExtractChromaWithReport(source, nil, DefaultChromaSettings())
	if out == nil || report.MatteColorSource != "auto-sampled" {
		t.Fatalf("unexpected auto-sampled report: %#v", report)
	}

	// 2. extractBorderConnectedChromaWithReport with nil matte
	out2, report2 := extractBorderConnectedChromaWithReport(source, nil, DefaultChromaSettings())
	if out2 == nil || report2.MatteColorSource != "auto-sampled" {
		t.Fatalf("unexpected auto-sampled report2: %#v", report2)
	}

	// 3. extractBorderConnectedChromaWithReport with provided matte
	matte := MatteColor{0, 255, 0}
	out3, report3 := extractBorderConnectedChromaWithReport(source, &matte, DefaultChromaSettings())
	if out3 == nil || report3.MatteColorSource != "provided" {
		t.Fatalf("unexpected provided report3: %#v", report3)
	}
}
