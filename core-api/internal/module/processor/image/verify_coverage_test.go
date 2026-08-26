package image

import (
	"context"
	"image"
	"image/color"
	"testing"
)

func TestEvaluateTransparencyGateAllProfiles(t *testing.T) {
	t.Parallel()

	// 1. Not PNG
	pass, reasons := evaluateTransparencyGate(TransparencyGateInput{
		IsPNG: false,
	})
	if pass || len(reasons) == 0 || reasons[0] != "not_png" {
		t.Fatalf("expected not_png failure, got %v", reasons)
	}

	// 2. ProfileOpaqueBackground failures and success
	passOpaque, reasonsOpaque := evaluateTransparencyGate(TransparencyGateInput{
		IsPNG:                  true,
		Profile:                ProfileOpaqueBackground,
		CheckerboardDetected:   true,
		NontransparentPixels:   0,
		AlphaMin:               100,
		TransparentRatio:       0.1,
		TransparentRGBScrubbed: false,
	})
	if passOpaque || len(reasonsOpaque) < 5 {
		t.Fatalf("expected multiple opaque failures, got %v", reasonsOpaque)
	}

	// Valid opaque background
	passOpaqueValid, _ := evaluateTransparencyGate(TransparencyGateInput{
		IsPNG:                  true,
		Profile:                ProfileOpaqueBackground,
		NontransparentPixels:   100,
		AlphaMin:               255,
		TransparentRatio:       0,
		TransparentRGBScrubbed: true,
	})
	if !passOpaqueValid {
		t.Fatal("expected valid opaque background to pass")
	}

	// 3. Transparent Gate Common Failures
	passCommon, reasonsCommon := evaluateTransparencyGate(TransparencyGateInput{
		IsPNG:                  true,
		Profile:                ProfileGeneric,
		HasAlpha:               false,
		CheckerboardDetected:   true,
		NontransparentPixels:   0,
		AlphaMin:               20,
		AlphaMax:               10,
		TransparentRatio:       0.001,
		TransparentRGBScrubbed: false,
	})
	if passCommon || len(reasonsCommon) < 6 {
		t.Fatalf("expected common transparency failures, got %v", reasonsCommon)
	}

	// 4. ProfileSticker
	residueHigh := 0.30
	passSticker, reasonsSticker := evaluateTransparencyGate(TransparencyGateInput{
		IsPNG:                  true,
		Profile:                ProfileSticker,
		HasAlpha:               true,
		NontransparentPixels:   100,
		AlphaMin:               0,
		AlphaMax:               240,  // < 250
		TransparentRatio:       0.01, // < 0.05
		TouchesEdge:            true,
		LargestComponentRatio:  0.50,         // < 0.75
		AlphaNoiseScore:        0.30,         // > 0.25
		MatteResidueScore:      &residueHigh, // > 0.22
		TransparentRGBScrubbed: true,
	})
	if passSticker || len(reasonsSticker) < 5 {
		t.Fatalf("expected sticker failures, got %v", reasonsSticker)
	}

	// 5. ProfileSeal
	residueSeal := 0.26
	passSeal, reasonsSeal := evaluateTransparencyGate(TransparencyGateInput{
		IsPNG:                  true,
		Profile:                ProfileSeal,
		HasAlpha:               true,
		NontransparentPixels:   100,
		AlphaMin:               0,
		AlphaMax:               255,
		TransparentRatio:       0.1,
		TouchesEdge:            true,
		AlphaNoiseScore:        0.70,         // > 0.60
		MatteResidueScore:      &residueSeal, // > 0.24
		TransparentRGBScrubbed: true,
	})
	if passSeal || len(reasonsSeal) < 3 {
		t.Fatalf("expected seal failures, got %v", reasonsSeal)
	}

	// 6. ProfileEffect
	passEffect, reasonsEffect := evaluateTransparencyGate(TransparencyGateInput{
		IsPNG:                  true,
		Profile:                ProfileEffect,
		HasAlpha:               true,
		NontransparentPixels:   100,
		AlphaMin:               0,
		AlphaMax:               255,
		TransparentRatio:       0.01, // < 0.02
		TouchesEdge:            true,
		TransparentRGBScrubbed: true,
	})
	if passEffect || len(reasonsEffect) < 2 {
		t.Fatalf("expected effect failures, got %v", reasonsEffect)
	}

	// 7. ProfileTranslucent, ProfileGlow, ProfileShadow
	for _, prof := range []Profile{ProfileTranslucent, ProfileGlow, ProfileShadow} {
		passTrans, reasonsTrans := evaluateTransparencyGate(TransparencyGateInput{
			IsPNG:                  true,
			Profile:                prof,
			HasAlpha:               true,
			NontransparentPixels:   100,
			AlphaMin:               0,
			AlphaMax:               255,
			PartialPixels:          0, // requires partial alpha
			TransparentRatio:       0.01,
			TouchesEdge:            true,
			TransparentRGBScrubbed: true,
		})
		if passTrans || len(reasonsTrans) < 3 {
			t.Fatalf("expected %s failures, got %v", prof, reasonsTrans)
		}
	}
}

func TestVerifyImageHelperFunctions(t *testing.T) {
	t.Parallel()

	// 1. edgePixelCount with various dimensions
	if c := edgePixelCount(0, 0); c != 0 {
		t.Fatalf("expected 0 for 0x0, got %d", c)
	}
	if c := edgePixelCount(1, 1); c != 1 {
		t.Fatalf("expected 1 for 1x1, got %d", c)
	}
	if c := edgePixelCount(1, 5); c != 5 {
		t.Fatalf("expected 5 for 1x5, got %d", c)
	}
	if c := edgePixelCount(5, 1); c != 5 {
		t.Fatalf("expected 5 for 5x1, got %d", c)
	}
	if c := edgePixelCount(10, 10); c != 36 {
		t.Fatalf("expected 36 for 10x10, got %d", c)
	}

	// 2. edgeMarginPx
	if m := edgeMarginPx(nil, 10, 10); m != 0 {
		t.Fatalf("expected 0 for nil bbox, got %d", m)
	}
	bbox := &AlphaBoundingBox{X: 2, Y: 3, Width: 4, Height: 4}
	if m := edgeMarginPx(bbox, 10, 10); m != 2 {
		t.Fatalf("expected edge margin 2, got %d", m)
	}

	// 3. detectCheckerboard
	smallImg := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	if detected := detectCheckerboard(smallImg); detected {
		t.Fatal("expected false for image smaller than checkerboard min size")
	}

	// 4. verifyImage with ProfileOpaqueBackground and empty image
	opaqueImg := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(opaqueImg, opaqueImg.Bounds(), color.NRGBA{R: 100, G: 150, B: 200, A: 255})
	report := verifyImage(opaqueImg, true, "rgb", false, VerificationOptions{
		Profile: ProfileOpaqueBackground,
	})
	if !report.Passed {
		t.Fatalf("opaque verify failed: report: %#v", report)
	}

	emptyReport := verifyImage(image.NewNRGBA(image.Rect(0, 0, 0, 0)), true, "rgba", true, VerificationOptions{})
	if emptyReport.Passed || len(emptyReport.FailureReasons) == 0 {
		t.Fatalf("expected empty image failure, got %#v", emptyReport)
	}

	// 5. processor Verify API error handling
	p := NewProcessor()
	cancCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := p.Verify(cancCtx, &VerifyRequest{})
	if err == nil {
		t.Fatal("expected error on canceled context")
	}

	_, err = p.Verify(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error on nil request")
	}

	// 6. matteResidueScoreFor with non-saturated and saturated matte
	residueImg := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			residueImg.SetNRGBA(x, y, color.NRGBA{R: 100, G: 100, B: 100, A: 128})
		}
	}
	scoreNonSat := matteResidueScoreFor(residueImg, MatteColor{100, 100, 100})
	if scoreNonSat <= 0 {
		t.Fatalf("expected positive residue score for matching gray matte, got %f", scoreNonSat)
	}

	// Saturated matte
	for y := range 16 {
		for x := range 16 {
			residueImg.SetNRGBA(x, y, color.NRGBA{R: 10, G: 240, B: 10, A: 128})
		}
	}
	scoreSat := matteResidueScoreFor(residueImg, MatteColor{0, 255, 0})
	if scoreSat <= 0 {
		t.Fatalf("expected positive saturated residue score, got %f", scoreSat)
	}

	// 7. haloScore
	hScore := haloScore(residueImg)
	if hScore < 0 {
		t.Fatalf("invalid halo score: %f", hScore)
	}

	// 8. detectCheckerboard with alternating 16x16 blocks
	cbImg := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			cell := (x / 16) + (y / 16)
			if cell%2 == 0 {
				cbImg.SetNRGBA(x, y, color.NRGBA{R: 200, G: 200, B: 200, A: 255})
			} else {
				cbImg.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}
	if cbDetected := detectCheckerboard(cbImg); !cbDetected {
		t.Fatal("expected checkerboard to be detected for synthetic checkerboard pattern")
	}
}
