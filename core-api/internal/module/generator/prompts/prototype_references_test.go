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
				"Reference image 1 is the Project Reference (the project prototype image)",
				"Reference image 2 is the Creating Reference (the user's subject or concept reference)",
				"The Creating Reference is the user's reference for the object or subject being created",
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
				"Reference image 1 is the Project Reference (the project prototype image)",
				"No Creating Reference is supplied",
			},
			forbidden: []string{"Reference image 1 is the user-supplied image"},
		},
		{
			name:  "creating reference only",
			state: prompts.PrototypeReferenceState{HasCreatingReference: true},
			expected: []string{
				"Exactly one reference image is supplied",
				"Reference image 1 is the Creating Reference (the user's subject or concept reference)",
				"The Creating Reference is the user's reference for the object or subject being created",
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
				"Do not assume that any unstated Project Reference, Creating Reference, or Nexus Reference is available",
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

func TestPrototypePromptsDescribeNexusReferenceRoles(t *testing.T) {
	tests := []struct {
		name      string
		state     prompts.PrototypeReferenceState
		expected  []string
		forbidden []string
	}{
		{
			name:  "project creating and nexus references",
			state: prompts.PrototypeReferenceState{HasProjectReference: true, HasCreatingReference: true, NexusReferenceCount: 2},
			expected: []string{
				"Exactly 4 reference images",
				"Reference image 1 is the Project Reference",
				"Reference image 2 is the Creating Reference (the user's subject or concept reference)",
				"Reference image 3 is a Nexus Reference selected by matching Tags within the current Project",
				"Reference image 4 is a Nexus Reference selected by matching Tags within the current Project",
			},
			forbidden: []string{"promoted to the Style Reference", "never treat it as the object the user asked to create.\n- Reference image 4 is the Creating Reference"},
		},
		{
			name:  "creating and nexus references",
			state: prompts.PrototypeReferenceState{HasCreatingReference: true, NexusReferenceCount: 2},
			expected: []string{
				"Exactly 3 reference images",
				"Reference image 1 is the Creating Reference (the user's subject or concept reference)",
				"Reference image 2 is a Nexus Reference selected by matching Tags within the current Project",
				"Reference image 3 is a Nexus Reference selected by matching Tags within the current Project",
				"No Project Reference is supplied",
			},
			forbidden: []string{"promoted to the Style Reference"},
		},
		{
			name:  "project and nexus references without creating reference",
			state: prompts.PrototypeReferenceState{HasProjectReference: true, NexusReferenceCount: 2},
			expected: []string{
				"Exactly 3 reference images",
				"Reference image 1 is the Project Reference",
				"Reference image 2 is a Nexus Reference selected by matching Tags within the current Project",
				"Reference image 3 is a Nexus Reference selected by matching Tags within the current Project",
				"Apply the Project Reference's visual language to the requested character",
			},
			forbidden: []string{"promoted to the Style Reference"},
		},
		{
			name:  "nexus references only",
			state: prompts.PrototypeReferenceState{NexusReferenceCount: 3},
			expected: []string{
				"Exactly 3 reference images",
				"Reference image 1 is a Nexus Reference selected by matching Tags within the current Project",
				"Reference image 2 is a Nexus Reference selected by matching Tags within the current Project",
				"Reference image 3 is a Nexus Reference selected by matching Tags within the current Project",
				"derive the requested character from the user creative brief",
			},
			forbidden: []string{"promoted to the Style Reference"},
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

func TestPrototypePromptsDescribeNexusReferenceRolesForObject(t *testing.T) {
	state := prompts.PrototypeReferenceState{HasProjectReference: true, HasCreatingReference: true, NexusReferenceCount: 1}
	prompt := prompts.ObjectPrototype(
		"magic staff",
		"Side-On",
		assetdomain.Size{Width: 64, Height: 64},
		prompts.TransparentBackground(),
		state,
	)

	for _, expected := range []string{
		"Exactly 3 reference images",
		"Reference image 1 is the Project Reference",
		"Reference image 2 is the Creating Reference (the user's subject or concept reference)",
		"Reference image 3 is a Nexus Reference selected by matching Tags within the current Project",
		"Apply the Project Reference's visual language to the requested object",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected object prompt to contain %q: %s", expected, prompt)
		}
	}
}
