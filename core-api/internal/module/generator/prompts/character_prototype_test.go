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
		prompts.PrototypeReferenceState{HasProjectReference: true, HasCreatingReference: true},
	)

	for _, expected := range []string{
		"complete full-body character",
		"Reference image 1 is the Project Reference (the project prototype image)",
		"Reference image 2 is the Creating Reference (the user's subject or concept reference)",
		"The Creating Reference is the user's reference for the object or subject being created",
		"is always a strong reference",
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
		"never render both cells facing the same way",
		"equal gutters and equal margins",
		"one regular output sheet",
		"silver-armored dragon-born interstellar soldier",
		"Side-On",
		"<direction_count>\n2\n</direction_count>",
		"<asset_dimensions>\n{\"width\":48,\"height\":64}\n</asset_dimensions>",
		"Use the full 48 x 64 logical prototype canvas",
		"Do not reserve internal padding for animation",
		"face symbolic and readable",
		"medium-size character with 48 x 64 drawable logical pixels",
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
		{name: "16px full canvas", dimensions: assetdomain.Size{Width: 16, Height: 16}, want: "ultra-small character with only 16 x 16 drawable logical pixels"},
		{name: "32px full canvas", dimensions: assetdomain.Size{Width: 32, Height: 32}, want: "small character with 32 x 32 drawable logical pixels"},
		{name: "48x64 full canvas", dimensions: assetdomain.Size{Width: 48, Height: 64}, want: "medium-size character with 48 x 64 drawable logical pixels"},
		{name: "64px full canvas", dimensions: assetdomain.Size{Width: 64, Height: 64}, want: "medium-size character with 64 x 64 drawable logical pixels"},
		{name: "128px full canvas", dimensions: assetdomain.Size{Width: 128, Height: 128}, want: "This character has 128 x 128 drawable logical pixels"},
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

func TestCharacterPrototypeMakes32pxCharacterUseSmallDetailBudget(t *testing.T) {
	prompt := prompts.CharacterPrototype(
		"player character",
		"Top-Down",
		assetdomain.Size{Width: 32, Height: 32},
		prompts.TransparentBackground(),
		prompts.PrototypeReferenceState{},
	)

	for _, expected := range []string{
		"Choose complexity from this 32 x 32 logical grid",
		"The requested final canvas has a short edge of 32 pixels or less",
		"small character with 32 x 32 drawable logical pixels",
		"prioritize silhouette, head/body separation, and a few broad identity accents",
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
