package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestCharacterPrototypeIncludesFullBodyStyleAndDirectionLayout(t *testing.T) {
	prompt := prompts.CharacterPrototype(
		"a silver-armored dragon-born interstellar soldier",
		"Side-On",
		assetdomain.Size{Width: 48, Height: 64},
		prompts.SolidMatteBackground("#00FF00"),
		prompts.PrototypeReferenceState{HasProjectReference: true, HasUserReference: true},
	)

	for _, expected := range []string{
		"complete full-body character",
		"Reference image 1 is the project prototype image and is the Style Reference",
		"Reference image 2 is the user-supplied reference image",
		"user-supplied reference image is always a strong reference",
		"uniform, solid #00FF00 colour",
		"exactly 2 direction views",
		"1 row x 2 column sheet",
		"SIDE-ON SCALE LOCK",
		"same pixel height",
		"never treat the second cell as an independently framed composition",
		"redraw the mismatched view rather than compensating with a different zoom",
		"normal reading order",
		"Complete the first row before starting the second row",
		"reading-order indexes",
		"zero-based array index is the direction identity",
		"index 0 = left, index 1 = right",
		"equal gutters and equal margins",
		"one regular output sheet",
		"silver-armored dragon-born interstellar soldier",
		"Side-On",
		"<direction_count>\n2\n</direction_count>",
		"<asset_dimensions>\n{\"width\":48,\"height\":64}\n</asset_dimensions>",
		"at most 30 x 46 drawable logical pixels",
		"fixed 9-pixel safety margin",
		"face symbolic and readable",
		"small character with 30 x 46 drawable logical pixels",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected character prompt to contain %q: %s", expected, prompt)
		}
	}
}

func TestCharacterPrototypeDerivesDirectionLayoutFromPerspective(t *testing.T) {
	tests := []struct {
		name        string
		perspective string
		direction   string
		expected    []string
	}{
		{
			name:        "side on",
			perspective: "Side-On",
			direction:   "2",
			expected:    []string{"Side-on perspective", "exactly 2 direction views", "1 row x 2 column sheet"},
		},
		{
			name:        "top down",
			perspective: "Top-Down",
			direction:   "4",
			expected:    []string{"Top-down perspective", "exactly 4 direction views", "2 row x 2 column sheet"},
		},
		{
			name:        "isometric",
			perspective: "Isometric",
			direction:   "8",
			expected:    []string{"Isometric perspective", "exactly 8 direction views", "2 row x 4 column sheet"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prompt := prompts.CharacterPrototype(
				"a readable player character",
				test.perspective,
				assetdomain.Size{Width: 32, Height: 32},
				prompts.TransparentBackground(),
				prompts.PrototypeReferenceState{},
			)
			for _, expected := range test.expected {
				if !strings.Contains(prompt, expected) {
					t.Fatalf("expected %s prompt to contain %q: %s", test.perspective, expected, prompt)
				}
			}
			if !strings.Contains(prompt, "<direction_count>\n"+test.direction+"\n</direction_count>") {
				t.Fatalf("expected %s direction count in prompt: %s", test.direction, prompt)
			}
		})
	}
}

func TestCharacterPrototypeAdaptsRulesToLogicalPixelBudget(t *testing.T) {
	tests := []struct {
		name       string
		dimensions assetdomain.Size
		want       string
	}{
		{name: "16px nominal / 10px drawable", dimensions: assetdomain.Size{Width: 16, Height: 16}, want: "emblem-scale character with only 10 x 10 drawable logical pixels"},
		{name: "32px nominal / 20px drawable", dimensions: assetdomain.Size{Width: 32, Height: 32}, want: "ultra-small character with only 20 x 20 drawable logical pixels"},
		{name: "48x64 nominal / 30x46 drawable", dimensions: assetdomain.Size{Width: 48, Height: 64}, want: "small character with 30 x 46 drawable logical pixels"},
		{name: "64px nominal / 40px drawable", dimensions: assetdomain.Size{Width: 64, Height: 64}, want: "small character with 40 x 40 drawable logical pixels"},
		{name: "128px nominal / 80px drawable", dimensions: assetdomain.Size{Width: 128, Height: 128}, want: "medium-size character with 80 x 80 drawable logical pixels"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prompt := prompts.CharacterPrototype(
				"hero",
				"Top-Down",
				test.dimensions,
				prompts.TransparentBackground(),
				prompts.PrototypeReferenceState{},
			)
			if !strings.Contains(prompt, test.want) {
				t.Fatalf("expected %dx%d prompt to contain %q: %s", test.dimensions.Width, test.dimensions.Height, test.want, prompt)
			}
		})
	}
}

func TestCharacterPrototypeMakesNominal32pxCharacterUseUltraSmallDetailBudget(t *testing.T) {
	prompt := prompts.CharacterPrototype(
		"player character",
		"Top-Down",
		assetdomain.Size{Width: 32, Height: 32},
		prompts.TransparentBackground(),
		prompts.PrototypeReferenceState{},
	)

	for _, expected := range []string{
		"Choose complexity from the 20 x 20 drawable region, not from the nominal 32 x 32 canvas",
		"The requested final canvas has a short edge of 32 pixels or less",
		"ultra-small character with only 20 x 20 drawable logical pixels",
		"few large connected regions",
		"6-8 visually distinct color roles",
		"narrower than three logical pixels",
		"silhouette readability overrides anatomical separation",
		"at most one broad shadow or highlight cluster",
		"Avoid scattered single-pixel highlights",
		"at most one or two intentional high-contrast marks",
		"Avoid one-pixel-wide torsos or limbs",
		"especially in side views",
		"distinguished primarily by silhouette and large color placement",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected 32px character prompt to contain %q: %s", expected, prompt)
		}
	}
}

func TestCharacterPrototypeExplicitlyLowersDetailAtOrBelow32PixelShortEdge(t *testing.T) {
	for _, dimensions := range []assetdomain.Size{
		{Width: 16, Height: 64},
		{Width: 32, Height: 64},
		{Width: 64, Height: 32},
	} {
		prompt := prompts.CharacterPrototype(
			"player character",
			"Top-Down",
			dimensions,
			prompts.TransparentBackground(),
			prompts.PrototypeReferenceState{},
		)
		for _, expected := range []string{
			"short edge of 32 pixels or less",
			"Explicitly lower the design detail level before rendering",
			"Do not create a detailed high-resolution design and expect downscaling or post-processing to simplify it",
			"Simplify facial anatomy, clothing construction, equipment ornament, and limb separation before rendering",
		} {
			if !strings.Contains(prompt, expected) {
				t.Fatalf("expected %dx%d prompt to contain %q: %s", dimensions.Width, dimensions.Height, expected, prompt)
			}
		}
	}

	prompt := prompts.CharacterPrototype(
		"player character",
		"Top-Down",
		assetdomain.Size{Width: 33, Height: 64},
		prompts.TransparentBackground(),
		prompts.PrototypeReferenceState{},
	)
	if strings.Contains(prompt, "short edge of 32 pixels or less") {
		t.Fatalf("33px short edge unexpectedly received the explicit <=32px rule: %s", prompt)
	}
}
