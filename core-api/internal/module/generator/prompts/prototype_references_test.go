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
			name:  "project and user references",
			state: prompts.PrototypeReferenceState{HasProjectReference: true, HasUserReference: true},
			expected: []string{
				"Exactly two reference images are supplied in an authoritative order",
				"Reference image 1 is the project prototype image and is the Style Reference",
				"Reference image 2 is the user-supplied reference image",
				"user-supplied reference image is always a strong reference",
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
				"No user-supplied reference image is supplied",
			},
			forbidden: []string{"Reference image 1 is the user-supplied image"},
		},
		{
			name:  "user reference only",
			state: prompts.PrototypeReferenceState{HasUserReference: true},
			expected: []string{
				"Exactly one reference image is supplied",
				"Reference image 1 is the user-supplied reference image",
				"user-supplied reference image is always a strong reference",
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
				"Do not assume that any unstated project or user reference is available",
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

func TestPrototypePromptsDescribeTagReferenceFallbackRoles(t *testing.T) {
	tests := []struct {
		name      string
		state     prompts.PrototypeReferenceState
		expected  []string
		forbidden []string
	}{
		{
			name:  "project user and tag assets",
			state: prompts.PrototypeReferenceState{HasProjectReference: true, HasUserReference: true, TagReferenceCount: 2},
			expected: []string{
				"Exactly 4 reference images",
				"Reference image 1 is the project prototype image",
				"Reference image 2 is the user-supplied reference image",
				"Reference image 3 is a same-project Tag asset",
				"Reference image 4 is a same-project Tag asset",
			},
			forbidden: []string{"Reference image 3 is the highest-ranked"},
		},
		{
			name:  "user and tag assets",
			state: prompts.PrototypeReferenceState{HasUserReference: true, TagReferenceCount: 2},
			expected: []string{
				"Exactly 3 reference images",
				"Reference image 1 is the user-supplied reference image",
				"Reference image 2 is the highest-ranked same-project Tag asset and is promoted to the Style Reference",
				"Reference image 3 is a same-project Tag asset",
			},
		},
		{
			name:  "tag assets only",
			state: prompts.PrototypeReferenceState{TagReferenceCount: 3},
			expected: []string{
				"Exactly 3 reference images",
				"Reference image 1 is the highest-ranked same-project Tag asset and is promoted to the Style Reference",
				"Reference image 2 is a same-project Tag asset",
				"Reference image 3 is a same-project Tag asset",
				"Derive the requested character from the user creative brief",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prompt := prompts.CharacterPrototype("hero", "Top-Down", prompts.TransparentBackground(), test.state)
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
		})
	}
}
