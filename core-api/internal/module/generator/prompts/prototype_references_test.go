package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestPrototypePromptsDescribeReferenceRoles(t *testing.T) {
	builders := []struct {
		name  string
		build func(prompts.PrototypeReferenceState) string
	}{
		{
			name: "character",
			build: func(state prompts.PrototypeReferenceState) string {
				return prompts.CharacterPrototype("a hero", "Top-Down", prompts.TransparentBackground(), state)
			},
		},
		{
			name: "object",
			build: func(state prompts.PrototypeReferenceState) string {
				return prompts.ObjectPrototype(
					"a chest",
					"Top-Down",
					assetdomain.Size{Width: 48, Height: 48},
					prompts.TransparentBackground(),
					state,
				)
			},
		},
	}
	tests := []struct {
		name      string
		state     prompts.PrototypeReferenceState
		expected  []string
		forbidden []string
	}{
		{
			name:  "project and creating references",
			state: prompts.PrototypeReferenceState{HasProjectReference: true, HasCreatingReference: true},
			expected: []string{
				"Exactly two reference images are supplied in an authoritative order",
				"Reference image 1 is the project prototype image and is the Style Reference",
				"Reference image 2 is the creating reference image",
				"creating reference image is always a strong reference",
				"preserve every unmentioned visual attribute",
				"make only the changes explicitly requested in the user creative brief",
			},
			forbidden: []string{"Subject/Concept Reference", "Use it to guide", "intended role"},
		},
		{
			name:  "project reference only",
			state: prompts.PrototypeReferenceState{HasProjectReference: true},
			expected: []string{
				"Exactly one reference image is supplied",
				"Reference image 1 is the project prototype image and is the Style Reference",
				"No creating reference image is supplied",
			},
			forbidden: []string{"Reference image 1 is the user-supplied image"},
		},
		{
			name:  "creating reference only",
			state: prompts.PrototypeReferenceState{HasCreatingReference: true},
			expected: []string{
				"Exactly one reference image is supplied",
				"Reference image 1 is the creating reference image",
				"creating reference image is always a strong reference",
				"preserve every unmentioned visual attribute",
				"make only the changes explicitly requested in the user creative brief",
				"No project Style Reference is supplied",
			},
			forbidden: []string{
				"Reference image 1 is the project prototype image",
				"Subject/Concept Reference",
				"Use it to guide",
				"intended role",
			},
		},
		{
			name:  "no references",
			state: prompts.PrototypeReferenceState{},
			expected: []string{
				"No reference images are supplied",
				"Do not assume that any unstated project or creating reference is available",
			},
			forbidden: []string{"Exactly one reference image", "Exactly two reference images"},
		},
	}

	const conflictRule = "If any reference image conflicts with a specific requirement in the user creative brief, follow the creative brief for that requirement."
	for _, builder := range builders {
		for _, test := range tests {
			t.Run(builder.name+"/"+test.name, func(t *testing.T) {
				prompt := builder.build(test.state)
				for _, expected := range test.expected {
					if !strings.Contains(prompt, expected) {
						t.Fatalf("expected prompt to contain %q: %s", expected, prompt)
					}
				}
				for _, forbidden := range test.forbidden {
					if strings.Contains(prompt, forbidden) {
						t.Fatalf("expected prompt not to contain %q: %s", forbidden, prompt)
					}
				}
				if count := strings.Count(prompt, conflictRule); count != 2 {
					t.Fatalf("expected conflict rule in priority and role sections, got %d: %s", count, prompt)
				}
			})
		}
	}
}
