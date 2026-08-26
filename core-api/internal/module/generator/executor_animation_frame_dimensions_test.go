package generator

import (
	"strings"
	"testing"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestResolveAnimationFrameDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		prototype  assetdomain.Size
		width      int
		height     int
		wantWidth  int
		wantHeight int
		wantErr    string
	}{
		{
			name:      "defaults 32 pixel prototype to 48 pixels",
			prototype: assetdomain.Size{Width: 32, Height: 32},
			wantWidth: 48, wantHeight: 48,
		},
		{
			name:      "defaults 64 pixel prototype to 96 pixels",
			prototype: assetdomain.Size{Width: 64, Height: 64},
			wantWidth: 96, wantHeight: 96,
		},
		{
			name:      "rounds odd defaults upward",
			prototype: assetdomain.Size{Width: 33, Height: 35},
			wantWidth: 50, wantHeight: 53,
		},
		{
			name:      "honors explicit dimensions",
			prototype: assetdomain.Size{Width: 64, Height: 48},
			width:     112, height: 80,
			wantWidth: 112, wantHeight: 80,
		},
		{
			name:      "rejects missing height",
			prototype: assetdomain.Size{Width: 32, Height: 32},
			width:     48,
			wantErr:   "both be provided or both be omitted",
		},
		{
			name:      "rejects missing width",
			prototype: assetdomain.Size{Width: 32, Height: 32},
			height:    48,
			wantErr:   "both be provided or both be omitted",
		},
		{
			name:      "rejects dimensions smaller than prototype",
			prototype: assetdomain.Size{Width: 64, Height: 64},
			width:     63, height: 96,
			wantErr: "must not be smaller than prototype dimensions",
		},
		{
			name:      "rejects dimensions below supported minimum",
			prototype: assetdomain.Size{Width: 16, Height: 16},
			width:     24, height: 24,
			wantErr: "between 32 and 1024 pixels",
		},
		{
			name:      "rejects dimensions above supported maximum",
			prototype: assetdomain.Size{Width: 64, Height: 64},
			width:     1025, height: 1024,
			wantErr: "between 32 and 1024 pixels",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			width, height, err := resolveAnimationFrameDimensions(
				test.prototype,
				test.width,
				test.height,
			)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve dimensions: %v", err)
			}
			if width != test.wantWidth || height != test.wantHeight {
				t.Fatalf("dimensions = %dx%d, want %dx%d", width, height, test.wantWidth, test.wantHeight)
			}
		})
	}
}
