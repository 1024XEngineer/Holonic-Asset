package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"strings"
	"unicode"
)

const pngMIMEType = "image/png"

type decodedBase64Image struct {
	image     *image.RGBA
	format    string
	colorType string
	hasAlpha  bool
}

// DecodeBase64Image accepts either raw Base64 or a data URL and decodes it to
// RGBA for local processing.
func DecodeBase64Image(value string) (*image.RGBA, error) {
	decoded, err := decodeBase64Image(value)
	if err != nil {
		return nil, err
	}
	return decoded.image, nil
}

// EncodePNGBase64 encodes an image as a raw (non-data-URL) PNG Base64 string.
func EncodePNGBase64(img image.Image) (string, error) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return "", fmt.Errorf("encode PNG: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

func decodeBase64Image(value string) (decodedBase64Image, error) {
	raw, err := decodeBase64Payload(value)
	if err != nil {
		return decodedBase64Image{}, err
	}
	decoded, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return decodedBase64Image{}, fmt.Errorf("decode image data: %w", err)
	}

	colorType := format
	hasAlpha := hasAlphaFromImage(decoded)
	if format == "png" {
		pngType, pngHasAlpha, pngErr := pngColorType(raw)
		if pngErr == nil {
			colorType = pngType
			hasAlpha = pngHasAlpha
		}
	}

	return decodedBase64Image{
		image:     ToRGBA(decoded),
		format:    format,
		colorType: colorType,
		hasAlpha:  hasAlpha,
	}, nil
}

func decodeBase64Payload(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("image Base64 is required")
	}
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		header, payload, found := strings.Cut(value, ",")
		if !found {
			return nil, fmt.Errorf("invalid image data URL")
		}
		if !strings.Contains(strings.ToLower(header), ";base64") {
			return nil, fmt.Errorf("image data URL must use Base64 encoding")
		}
		value = payload
	}

	value = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)

	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
	}
	return nil, fmt.Errorf("invalid image Base64")
}

func ToRGBA(src image.Image) *image.RGBA {
	bounds := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(out, out.Bounds(), src, bounds.Min, draw.Src)
	return out
}

// PNG color types: 0 grayscale, 2 RGB, 3 indexed, 4 grayscale+alpha, 6 RGBA.
func pngColorType(data []byte) (string, bool, error) {
	if len(data) < 26 || !IsPNGBytes(data) {
		return "", false, fmt.Errorf("not PNG data")
	}
	if string(data[12:16]) != "IHDR" {
		return "", false, fmt.Errorf("PNG data has no IHDR chunk")
	}
	switch colorType := data[25]; colorType {
	case 0:
		return "grayscale", false, nil
	case 2:
		return "rgb", false, nil
	case 3:
		return "indexed", paletteHasAlpha(data), nil
	case 4:
		return "grayscale-alpha", true, nil
	case 6:
		return "rgba", true, nil
	default:
		return fmt.Sprintf("unknown(%d)", colorType), false, nil
	}
}

func paletteHasAlpha(data []byte) bool {
	return bytes.Contains(data, []byte("tRNS"))
}

func hasAlphaFromImage(img image.Image) bool {
	switch typed := img.(type) {
	case *image.NRGBA, *image.NRGBA64, *image.RGBA, *image.RGBA64, *image.Alpha, *image.Alpha16:
		return true
	case *image.Paletted:
		for _, entry := range typed.Palette {
			_, _, _, alpha := entry.RGBA()
			if alpha < 0xffff {
				return true
			}
		}
	}
	return false
}

func IsPNGBytes(data []byte) bool {
	return len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
}
