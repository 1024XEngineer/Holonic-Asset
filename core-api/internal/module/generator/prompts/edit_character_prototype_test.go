package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestEditCharacterPrototypeDefinesReferenceRolesAndDirectionLayout(t *testing.T) {
	prompt := prompts.EditCharacterPrototype(
		"a red dragon knight with a steel spear",
		"change only the exposed scales to light blue",
		"Side-On",
		2,
		assetdomain.Size{Width: 48, Height: 64},
		prompts.SolidMatteBackground("#00FF00"),
	)

	for _, expected := range []string{
		"Reference images 1 through 2 are the current character prototype direction views",
		"No separate user or project reference image is supplied",
		"a red dragon knight with a steel spear",
		"Minor edit",
		"exactly 2 direction views",
		"1 row x 2 column sheet",
		"SIDE-ON SCALE LOCK",
		"same pixel height",
		"perspective-derived direction count and grid override",
		"rebuild the output sheet with the required perspective mapping",
		"normal reading order",
		"Complete the first row before starting the second row",
		"Edit every required direction cell consistently",
		"uniform, solid #00FF00 colour",
		"change only the exposed scales to light blue",
		"<direction_count>\n2\n</direction_count>",
		"<asset_dimensions>\n{\"width\":48,\"height\":64}\n</asset_dimensions>",
		"Use the full 48 x 64 logical prototype canvas",
		"medium-size character with 48 x 64 drawable logical pixels",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected character edit prompt to contain %q: %s", expected, prompt)
		}
	}
}

func TestEditCharacterPrototypeDerivesDirectionLayoutFromPerspective(t *testing.T) {
	tests := []struct {
		name               string
		perspective        string
		direction          string
		originalReferences uint
		expected           []string
	}{
		{
			name:               "side on",
			perspective:        "Side-On",
			direction:          "2",
			originalReferences: 2,
			expected:           []string{"Side-on perspective", "exactly 2 direction views", "1 row x 2 column sheet"},
		},
		{
			name:               "top down",
			perspective:        "Top-Down",
			direction:          "4",
			originalReferences: 4,
			expected:           []string{"Top-down perspective", "exactly 4 direction views", "2 row x 2 column sheet"},
		},
		{
			name:               "isometric",
			perspective:        "Isometric",
			direction:          "8",
			originalReferences: 8,
			expected:           []string{"Isometric perspective", "exactly 8 direction views", "2 row x 4 column sheet"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prompt := prompts.EditCharacterPrototype(
				"a caped knight",
				"change the cape to blue",
				test.perspective,
				test.originalReferences,
				assetdomain.Size{Width: 32, Height: 32},
				prompts.TransparentBackground(),
			)
			for _, expected := range test.expected {
				if !strings.Contains(prompt, expected) {
					t.Fatalf("expected %s edit prompt to contain %q: %s", test.perspective, expected, prompt)
				}
			}
			if !strings.Contains(prompt, "<direction_count>\n"+test.direction+"\n</direction_count>") {
				t.Fatalf("expected %s direction count in edit prompt: %s", test.direction, prompt)
			}
		})
	}
}

func TestEditCharacterPrototypeUsesFullGridForSmallCharacterDetail(t *testing.T) {
	prompt := prompts.EditCharacterPrototype(
		"a player character",
		"change the shirt color",
		"Top-Down",
		4,
		assetdomain.Size{Width: 32, Height: 32},
		prompts.TransparentBackground(),
	)
	for _, expected := range []string{
		"small character with 32 x 32 drawable logical pixels",
		"prioritize silhouette, head/body separation, and a few broad identity accents",
		"The requested final canvas has a short edge of 32 pixels or less",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected small character edit prompt to contain %q: %s", expected, prompt)
		}
	}
}
