package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestObjectPrototypeIncludesInputsStyleAndProcessingConstraints(t *testing.T) {
	background := prompts.SolidMatteBackground("#00FF00")
	prompt := prompts.ObjectPrototype(
		"a wooden chest with two locks",
		"Top-Down",
		assetdomain.Size{Width: 48, Height: 48},
		background,
		prompts.PrototypeReferenceState{HasProjectReference: true, HasUserReference: true},
	)

	for _, expected := range []string{
		"pipeline processing requirements have the highest priority",
		"uniform, solid #00FF00 colour",
		"Do not output transparency or a checkerboard",
		"Reference image 1 is the project prototype image and is the Style Reference",
		"Reference image 2 is the user-supplied reference image",
		"user-supplied reference image is always a strong reference",
		"exactly 4 direction views",
		"2 row x 2 column sheet",
		"normal reading order",
		"reading-order indexes",
		"zero-based array index is the direction identity",
		"index 0 = front, index 1 = right, index 2 = back, index 3 = left",
		"exact centre of its assigned grid cell",
		"approximately 30% of the cell's width",
		"approximately 30% of the cell's height",
		"a wooden chest with two locks",
		"Top-Down",
		"<direction_count>\n4\n</direction_count>",
		"<asset_dimensions>\n{\"width\":48,\"height\":48}\n</asset_dimensions>",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected object prompt to contain %q: %s", expected, prompt)
		}
	}
}

func TestObjectPrototypeDerivesContentPercentageFromTargetDimensions(t *testing.T) {
	tests := []struct {
		name       string
		dimensions assetdomain.Size
		want       string
	}{
		{name: "very small", dimensions: assetdomain.Size{Width: 16, Height: 16}, want: "approximately 20%"},
		{name: "small sprite baseline", dimensions: assetdomain.Size{Width: 48, Height: 48}, want: "approximately 30%"},
		{name: "rectangular couch", dimensions: assetdomain.Size{Width: 188, Height: 128}, want: "approximately 44%"},
		{name: "large asset cap", dimensions: assetdomain.Size{Width: 2048, Height: 1024}, want: "approximately 70%"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prompt := prompts.ObjectPrototype(
				"object",
				"Top-Down",
				test.dimensions,
				prompts.SolidMatteBackground("#00FF00"),
				prompts.PrototypeReferenceState{},
			)
			if !strings.Contains(prompt, test.want+" of the cell's width") ||
				!strings.Contains(prompt, test.want+" of the cell's height") {
				t.Fatalf("prompt does not contain derived content size %q: %s", test.want, prompt)
			}
		})
	}
}

func TestTransparentBackgroundRequiresRealAlpha(t *testing.T) {
	constraint := prompts.TransparentBackground()
	for _, expected := range []string{"real alpha channel", "Do not draw a checkerboard pattern"} {
		if !strings.Contains(constraint, expected) {
			t.Fatalf("expected transparent background constraint to contain %q: %s", expected, constraint)
		}
	}
}

func TestObjectPrototypeSideOnLocksBothViewsToOneScale(t *testing.T) {
	prompt := prompts.ObjectPrototype(
		"a regulation basketball",
		"Side-On",
		assetdomain.Size{Width: 48, Height: 48},
		prompts.TransparentBackground(),
		prompts.PrototypeReferenceState{},
	)
	for _, expected := range []string{
		"SIDE-ON SCALE LOCK",
		"exactly the same camera distance and zoom",
		"same pixel height",
		"identical transparent or matte padding",
		"never treat the second cell as an independently framed composition",
		"redraw the mismatched view rather than compensating with a different zoom",
		"one shared camera, zoom factor, subject scale, and cell-local coordinate system",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected side-on object prompt to contain %q: %s", expected, prompt)
		}
	}
}
