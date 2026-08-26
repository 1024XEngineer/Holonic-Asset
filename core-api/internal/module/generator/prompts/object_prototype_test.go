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
		prompts.PrototypeReferenceState{HasProjectReference: true, HasCreatingReference: true},
	)

	for _, expected := range []string{
		"pipeline processing requirements have the highest priority",
		"uniform, solid #00FF00 colour",
		"Do not output transparency or a checkerboard",
		"Reference image 1 is the Project Reference (the project prototype image)",
		"Reference image 2 is the Creating Reference (the user's subject or concept reference)",
		"The Creating Reference is the user's reference for the object or subject being created",
		"is always a strong reference",
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
		"Use the full 48 x 48 logical prototype canvas",
		"Do not reserve internal padding for animation",
		"whole pixels",
		"high-contrast color clusters",
		"medium-size object with 48 x 48 drawable logical pixels",
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
		{name: "zero edge", dimensions: assetdomain.Size{Width: 0, Height: 0}, want: "approximately 20%"},
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

func TestAdaptiveMatteBackgroundPreservesSubjectColours(t *testing.T) {
	constraint := prompts.AdaptiveMatteBackground()
	for _, expected := range []string{
		"strong perceptual colour separation",
		"The subject's correct colours always take precedence",
		"choose a different high-contrast matte colour instead",
		"detect it automatically",
	} {
		if !strings.Contains(constraint, expected) {
			t.Fatalf("expected adaptive matte constraint to contain %q: %s", expected, constraint)
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

func TestObjectPrototypeUsesFullGridFor32pxDetail(t *testing.T) {
	prompt := prompts.ObjectPrototype(
		"game object",
		"Top-Down",
		assetdomain.Size{Width: 32, Height: 32},
		prompts.TransparentBackground(),
		prompts.PrototypeReferenceState{},
	)
	for _, expected := range []string{
		"Choose complexity from this 32 x 32 logical grid",
		"The requested final canvas has a short edge of 32 pixels or less",
		"simplified, continuous, consistently coloured logical-pixel line",
		"simplified continuous one-logical-pixel path",
		"small object with 32 x 32 drawable logical pixels",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected 32px object prompt to contain %q: %s", expected, prompt)
		}
	}
}

func TestObjectPrototypeProtectsElongatedObjectComposition(t *testing.T) {
	prompt := prompts.ObjectPrototype(
		"a long ceremonial weapon with a distinct functional head and handle",
		"Side-On",
		assetdomain.Size{Width: 32, Height: 32},
		prompts.TransparentBackground(),
		prompts.PrototypeReferenceState{},
	)
	for _, expected := range []string{
		"For elongated objects, do not apply that compact square occupancy to both axes",
		"let the long axis use roughly 70-90% of the available drawable length",
		"complete functional end and the grip/shaft as one connected readable silhouette",
		"do not compress the whole design into a thin centred line",
		"Reserve enough short-axis pixels for the functional end",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected elongated-object rule to contain %q: %s", expected, prompt)
		}
	}
}

func TestObjectPrototypeLogicalPixelTiers(t *testing.T) {
	tests := []struct {
		name       string
		dimensions assetdomain.Size
		want       string
	}{
		{name: "16px ultra small full canvas", dimensions: assetdomain.Size{Width: 16, Height: 16}, want: "ultra-small object with only 16 x 16 drawable logical pixels"},
		{name: "32px full canvas", dimensions: assetdomain.Size{Width: 32, Height: 32}, want: "small object with 32 x 32 drawable logical pixels"},
		{name: "64px full canvas", dimensions: assetdomain.Size{Width: 64, Height: 64}, want: "medium-size object with 64 x 64 drawable logical pixels"},
		{name: "128px full canvas", dimensions: assetdomain.Size{Width: 128, Height: 128}, want: "This object has 128 x 128 drawable logical pixels"},
		{name: "256px large full canvas", dimensions: assetdomain.Size{Width: 256, Height: 256}, want: "Even at this larger drawable target, construct the object from deliberate coarse pixel clusters"},
		{name: "extra large (>160) full canvas", dimensions: assetdomain.Size{Width: 512, Height: 512}, want: "Even at this larger drawable target, construct the object from deliberate coarse pixel clusters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := prompts.ObjectPrototype(
				"test object",
				"Top-Down",
				tt.dimensions,
				prompts.TransparentBackground(),
				prompts.PrototypeReferenceState{},
			)
			if !strings.Contains(prompt, tt.want) {
				t.Fatalf("expected prompt to contain %q: %s", tt.want, prompt)
			}
		})
	}
}
