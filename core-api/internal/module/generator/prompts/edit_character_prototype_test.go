package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
)

func TestEditCharacterPrototypeDefinesReferenceRolesAndDirectionLayout(t *testing.T) {
	prompt := prompts.EditCharacterPrototype(
		"change only the exposed scales to light blue",
		"Side-On",
		prompts.SolidMatteBackground("#00FF00"),
	)

	for _, expected := range []string{
		"Reference image 1, the first supplied image, is always the original character prototype",
		"Reference image 2 and every later supplied image are project reference images",
		"Minor edit",
		"exactly 2 full-body direction views",
		"1 row x 2 column sheet",
		"perspective-derived direction count and grid override",
		"rebuild the output sheet with the required perspective mapping",
		"normal reading order",
		"Complete the first row before starting the second row",
		"Edit every required direction cell consistently",
		"uniform, solid #00FF00 colour",
		"change only the exposed scales to light blue",
		"<direction_count>\n2\n</direction_count>",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected character edit prompt to contain %q: %s", expected, prompt)
		}
	}
}

func TestEditCharacterPrototypeDerivesDirectionLayoutFromPerspective(t *testing.T) {
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
			expected:    []string{"Side-on perspective", "exactly 2 full-body direction views", "1 row x 2 column sheet", "left-facing and right-facing views in that order"},
		},
		{
			name:        "top down",
			perspective: "Top-Down",
			direction:   "4",
			expected:    []string{"Top-down perspective", "exactly 4 full-body direction views", "2 row x 2 column sheet", "front, right, back, and left views in that order"},
		},
		{
			name:        "isometric",
			perspective: "Isometric",
			direction:   "8",
			expected:    []string{"Isometric perspective", "exactly 8 full-body direction views", "2 row x 4 column sheet", "front, front-right, right, back-right, back, back-left, left, and front-left views in that order"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prompt := prompts.EditCharacterPrototype(
				"change the cape to blue",
				test.perspective,
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
