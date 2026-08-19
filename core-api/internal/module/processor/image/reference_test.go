package image

import (
	"context"
	"image"
	"image/color"
	"testing"
)

func TestNormalizeReferenceUsesIntegerNearestNeighborScaling(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	pixels := []color.NRGBA{
		{R: 255, A: 255}, {G: 255, A: 128},
		{B: 255, A: 64}, {R: 255, G: 255, A: 255},
		{G: 255, B: 255, A: 32}, {R: 255, B: 255, A: 1},
	}
	for index, pixel := range pixels {
		source.SetNRGBA(index%2, index/2, pixel)
	}
	encoded, err := EncodePNGBase64(source)
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}

	result, err := (&processor{}).NormalizeReference(context.Background(), &NormalizeReferenceRequest{
		ImageBase64: encoded,
		MaxEdge:     8,
	})
	if err != nil {
		t.Fatalf("normalize reference: %v", err)
	}
	if result.Report != (ReferenceNormalizationReport{
		InputWidth: 2, InputHeight: 3, OutputWidth: 4, OutputHeight: 6, Scale: 2, Upscaled: true,
	}) {
		t.Fatalf("unexpected normalization report: %+v", result.Report)
	}
	normalized, err := DecodeBase64Image(result.ImageBase64)
	if err != nil {
		t.Fatalf("decode normalized reference: %v", err)
	}
	for y := range 6 {
		for x := range 4 {
			got := color.NRGBAModel.Convert(normalized.At(x, y)).(color.NRGBA)
			want := pixels[(y/2)*2+x/2]
			if got != want {
				t.Fatalf("pixel (%d,%d) = %+v, want %+v", x, y, got, want)
			}
		}
	}
}

func TestNormalizeReferenceDefaultsToSixteenTimesFor48PixelImage(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 48, 48))
	source.SetNRGBA(7, 9, color.NRGBA{R: 20, G: 180, B: 70, A: 200})
	encoded, err := EncodePNGBase64(source)
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}

	result, err := (&processor{}).NormalizeReference(context.Background(), &NormalizeReferenceRequest{ImageBase64: encoded})
	if err != nil {
		t.Fatalf("normalize reference: %v", err)
	}
	if result.Report.Scale != 16 || result.Report.OutputWidth != 768 || result.Report.OutputHeight != 768 {
		t.Fatalf("48px normalization report = %+v", result.Report)
	}
}

func TestNormalizeReferenceDoesNotResampleLargeImage(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 8, 4))
	source.SetNRGBA(3, 2, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
	encoded, err := EncodePNGBase64(source)
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}

	result, err := (&processor{}).NormalizeReference(context.Background(), &NormalizeReferenceRequest{
		ImageBase64: encoded,
		MaxEdge:     8,
	})
	if err != nil {
		t.Fatalf("normalize reference: %v", err)
	}
	if result.Report.Scale != 1 || result.Report.Upscaled || result.Report.OutputWidth != 8 || result.Report.OutputHeight != 4 {
		t.Fatalf("unexpected no-op report: %+v", result.Report)
	}
	if result.ImageBase64 != encoded {
		t.Fatal("large reference was unnecessarily re-encoded")
	}
}
