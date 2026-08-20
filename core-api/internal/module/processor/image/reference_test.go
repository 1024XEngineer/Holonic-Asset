package image

import (
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"io"
	"strings"
	"testing"
)

type cancelingReferenceImage struct {
	cancel context.CancelFunc
}

func (i cancelingReferenceImage) ColorModel() color.Model { return color.NRGBAModel }
func (i cancelingReferenceImage) Bounds() image.Rectangle { return image.Rect(0, 0, 2, 2) }
func (i cancelingReferenceImage) At(int, int) color.Color {
	i.cancel()
	return color.NRGBA{R: 1, G: 2, B: 3, A: 255}
}

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

func TestNormalizeReferenceRejectsInvalidRequests(t *testing.T) {
	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name    string
		ctx     context.Context
		request *NormalizeReferenceRequest
		want    string
	}{
		{name: "canceled context", ctx: canceledContext, request: &NormalizeReferenceRequest{}, want: context.Canceled.Error()},
		{name: "missing request", ctx: context.Background(), want: "request is required"},
		{name: "negative max edge", ctx: context.Background(), request: &NormalizeReferenceRequest{MaxEdge: -1}, want: "cannot be negative"},
		{name: "invalid image", ctx: context.Background(), request: &NormalizeReferenceRequest{ImageBase64: "not-base64!"}, want: "decode reference image"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := (&processor{}).NormalizeReference(test.ctx, test.request)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("normalize error = %v, want error containing %q", err, test.want)
			}
		})
	}
}

func TestNormalizeReferenceRejectsEmptyDecodedImage(t *testing.T) {
	const magic = "EMPTY-REFERENCE-IMAGE"
	image.RegisterFormat(
		"empty-reference-test",
		magic,
		func(io.Reader) (image.Image, error) { return image.NewNRGBA(image.Rectangle{}), nil },
		func(io.Reader) (image.Config, error) { return image.Config{}, nil },
	)
	encoded := base64.StdEncoding.EncodeToString([]byte(magic))

	_, err := (&processor{}).NormalizeReference(context.Background(), &NormalizeReferenceRequest{
		ImageBase64: encoded,
		MaxEdge:     8,
	})
	if err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("normalize error = %v, want empty image rejection", err)
	}
}

func TestNormalizeReferenceChecksContextAfterScaling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	const magic = "CANCEL-REFERENCE-IMAGE"
	image.RegisterFormat(
		"cancel-reference-test",
		magic,
		func(io.Reader) (image.Image, error) { return cancelingReferenceImage{cancel: cancel}, nil },
		func(io.Reader) (image.Config, error) {
			return image.Config{ColorModel: color.NRGBAModel, Width: 2, Height: 2}, nil
		},
	)
	encoded := base64.StdEncoding.EncodeToString([]byte(magic))

	_, err := (&processor{}).NormalizeReference(ctx, &NormalizeReferenceRequest{
		ImageBase64: encoded,
		MaxEdge:     4,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("normalize error = %v, want context canceled after scaling", err)
	}
}

func TestIntegerNearestNeighborScaleReturnsInputSizeAtUnitScale(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	source.SetNRGBA(2, 1, color.NRGBA{R: 10, G: 20, B: 30, A: 40})

	result := integerNearestNeighborScale(source, 1)
	if result.Bounds() != source.Bounds() || result.NRGBAAt(2, 1) != source.NRGBAAt(2, 1) {
		t.Fatalf("unit-scale result = bounds %v pixel %+v", result.Bounds(), result.NRGBAAt(2, 1))
	}
}
