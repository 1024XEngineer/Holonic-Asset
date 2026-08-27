package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
)

func TestSceneryPromptsContainSharedVisualContract(t *testing.T) {
	plan := prompts.SceneryPlan(prompts.SceneryPlanInput{
		AssetName: "Valley", CreativeBrief: "dawn valley",
		Perspective: "Side-On", ProjectName: "Starbound", GameType: "RPG",
		TargetPlatform: "PC", ProjectDescription: "exploration", Width: 640, Height: 360,
	})
	layer := prompts.SceneryLayer(prompts.SceneryLayerInput{
		AssetName: "Valley", CreativeBrief: "dawn valley",
		Perspective: "Side-On", ProjectName: "Starbound", GameType: "RPG",
		TargetPlatform: "PC", ProjectDescription: "exploration", Width: 640, Height: 360,
		LayerID: 2, LayerName: "Mountains", LayerCreativeBrief: "distant ridge",
	}, "solid matte")
	layout := prompts.SceneryLayoutAnalysis(prompts.SceneryLayoutAnalysisInput{
		AssetName: "Valley", CreativeBrief: "dawn valley",
		Perspective: "Side-On", ProjectName: "Starbound", GameType: "RPG",
		TargetPlatform: "PC", ProjectDescription: "exploration", Width: 640, Height: 360,
		Layers: []prompts.SceneryLayoutLayerInput{{ID: 1, Name: "Sky"}, {ID: 2, Name: "Mountains"}},
	})

	for name, prompt := range map[string]string{"plan": plan, "layer": layer, "layout": layout} {
		for _, required := range []string{
			"classic low-resolution 2D pixel art", "MUST ALWAYS strictly use a Side-On",
			"provides visual style ONLY", "anchor solidly to the bottom of the frame",
			"Native 1990s 16-bit 2D gameplay scenery", "hard aliased edges", "no antialiasing",
			"never a polished illustration", "modern high-definition pixel artwork",
			"broad connected colour clusters", "12-to-16-colour master palette",
			"two flat tones plus at most one highlight", "one flat light direction",
			"foliage as clustered crowns", "exactly the same palette ramps",
			"no characters, people, humanoids, animals, or creatures by default",
			"If and only if the user's creative brief explicitly requests",
		} {
			if !strings.Contains(prompt, required) {
				t.Fatalf("%s prompt omitted %q: %s", name, required, prompt)
			}
		}
	}
}

func TestSceneryGenerationPromptsUsePixelGridOnlyAsArtDirectionSupport(t *testing.T) {
	plan := prompts.SceneryPlan(prompts.SceneryPlanInput{Width: 1536, Height: 1024})
	layer := prompts.SceneryLayer(
		prompts.SceneryLayerInput{Width: 1536, Height: 1024},
		prompts.SolidMatteBackground("#00ff00"),
	)
	for name, prompt := range map[string]string{"plan": plan, "layer": layer} {
		for _, required := range []string{
			"authored natively on one shared logical 384x256 canvas and presented at 4x nearest-neighbour scale",
			"One logical pixel occupies one crisp 4x4 output block",
			"large shapes, palette discipline, and cluster design must already look finished",
			"grid is only an alignment aid",
			"Do not first create smooth high-resolution artwork and then pixelate",
		} {
			if !strings.Contains(prompt, required) {
				t.Fatalf("%s prompt omitted %q: %s", name, required, prompt)
			}
		}
	}
}

func TestSceneryLayerPromptDistinguishesBackdropAndOverlay(t *testing.T) {
	input := prompts.SceneryLayerInput{
		AssetName: "Valley", CreativeBrief: "dawn", Width: 640, Height: 360,
		LayerID: 1, LayerName: "Sky", LayerCreativeBrief: "full sky",
		IsBackmost: true,
	}
	backdrop := prompts.SceneryLayer(input, prompts.SolidMatteBackground("#00ff00"))
	if !strings.Contains(backdrop, "backmost scenery layer") || !strings.Contains(backdrop, "may be fully opaque") ||
		strings.Contains(backdrop, "continuous matte-only border") {
		t.Fatalf("unexpected backdrop contract: %s", backdrop)
	}
	input.IsBackmost = false
	input.LayerID = 2
	overlay := prompts.SceneryLayer(input, prompts.SolidMatteBackground("#00ff00"))
	if !strings.Contains(overlay, "overlay layer") || !strings.Contains(overlay, "continuous matte-only border") ||
		!strings.Contains(overlay, "clearly non-matte opaque subject pixels") ||
		!strings.Contains(overlay, "never add darker green strokes, grid marks, dithering, shading, or texture") ||
		!strings.Contains(overlay, "Return the exact full-canvas aspect ratio") ||
		!strings.Contains(overlay, "instead of recentering the visible artwork") {
		t.Fatalf("unexpected overlay contract: %s", overlay)
	}
	if !strings.Contains(overlay, "first generated foreground layer and the visual style anchor") ||
		!strings.Contains(overlay, "do not showcase fine detail") {
		t.Fatalf("overlay omitted first-layer style anchor: %s", overlay)
	}
}

func TestSceneryLayerPromptDistinguishesStyleAndGeneratedForegroundReferences(t *testing.T) {
	input := prompts.SceneryLayerInput{
		AssetName: "Valley", CreativeBrief: "dawn", Width: 640, Height: 360,
		LayerID: 2, LayerName: "Mountains", LayerCreativeBrief: "distant ridge",
		HasReference: true,
	}
	styleReference := prompts.SceneryLayer(input, prompts.SolidMatteBackground("#00ff00"))
	if !strings.Contains(styleReference, "visual-language guidance only") ||
		strings.Contains(styleReference, "cumulative transparent preview") {
		t.Fatalf("unexpected style reference contract: %s", styleReference)
	}

	input.HasForegroundReference = true
	foregroundReference := prompts.SceneryLayer(input, prompts.SolidMatteBackground("#00ff00"))
	for _, required := range []string{
		"final supplied reference image is the cumulative transparent preview",
		"exact spatial context",
		"Generate only the requested layer behind the preview",
		"must not override the cumulative preview's established composition",
		"<generated_foreground_reference_supplied>true</generated_foreground_reference_supplied>",
	} {
		if !strings.Contains(foregroundReference, required) {
			t.Fatalf("foreground reference prompt omitted %q: %s", required, foregroundReference)
		}
	}
}

func TestSceneryPlanPromptContainsSceneContextAndPlanningRules(t *testing.T) {
	prompt := prompts.SceneryPlan(prompts.SceneryPlanInput{
		AssetName: "Valley", CreativeBrief: "dawn valley",
		Perspective: "Side-On", ProjectName: "Starbound", GameType: "RPG",
		TargetPlatform: "PC", ProjectDescription: "exploration", Width: 640, Height: 360,
	})
	for _, required := range []string{
		"Assign every scene element, structure, terrain band, and water feature to exactly one layer",
		"full-canvas raster with the exact requested dimensions and aspect ratio",
		"Never plan a square tile, card, panel, or isolated centred illustration",
		"Establish one concise style lock", "Plan broad graphic masses before decorative content",
		"explicitly specify their grounding alignment along the bottom border",
		"Valley", "dawn valley", "Side-On", "Starbound", "RPG", "PC",
		"exploration", `<dimensions width="640" height="360" />`,
		"Decide the number of layers", "back-to-front compositing order",
		"intended placement, framing, scale, depth", "backend assigns stable IDs",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("prompt omitted %q: %s", required, prompt)
		}
	}
}

func TestSceneryLayoutAnalysisPromptContainsContextAndImageMapping(t *testing.T) {
	prompt := prompts.SceneryLayoutAnalysis(prompts.SceneryLayoutAnalysisInput{
		AssetName: "Valley", CreativeBrief: "dawn valley",
		Perspective: "Side-On", ProjectName: "Starbound", GameType: "RPG",
		TargetPlatform: "PC", ProjectDescription: "exploration", Width: 640, Height: 360,
		Layers: []prompts.SceneryLayoutLayerInput{{ID: 1, Name: "Sky"}, {ID: 2, Name: "Mountains"}},
	})
	for _, required := range []string{
		"Valley", "dawn valley", "Side-On", "Starbound", "RPG", "PC", "exploration",
		`<dimensions width="640" height="360" />`, "positive X to the right", "Rotation is clockwise",
		"Return exactly one layout", "already registered to the complete final canvas", "Default to position (0, 0)",
		"first attached image is the authoritative opaque full-canvas backdrop", "unique lowest zIndex",
		`Attached image 1 corresponds to layer ID 1 named "Sky"`,
		`Attached image 2 corresponds to layer ID 2 named "Mountains"`,
		"critically evaluate the overall composition quality",
		"approved to true",
		"approved to false",
		"review_notes",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("prompt omitted %q: %s", required, prompt)
		}
	}
}

func TestSceneryPromptsWithNonGridAlignedDimensions(t *testing.T) {
	for _, dims := range [][2]uint{
		{0, 0},
		{641, 360},
		{640, 361},
	} {
		plan := prompts.SceneryPlan(prompts.SceneryPlanInput{Width: dims[0], Height: dims[1]})
		layer := prompts.SceneryLayer(prompts.SceneryLayerInput{Width: dims[0], Height: dims[1]}, "solid matte")
		for name, prompt := range map[string]string{"plan": plan, "layer": layer} {
			if !strings.Contains(prompt, "Use one visibly chunky, uniform pixel grid across the complete canvas") {
				t.Fatalf("%s prompt with dimensions %dx%d did not contain fallback grid contract: %s", name, dims[0], dims[1], prompt)
			}
		}
	}
}

func TestSceneryPlanPromptIncludesPreviousCritiqueWhenProvided(t *testing.T) {
	promptWithoutCritique := prompts.SceneryPlan(prompts.SceneryPlanInput{
		AssetName: "Valley", CreativeBrief: "dawn valley",
		Perspective: "Side-On", ProjectName: "Starbound", GameType: "RPG",
		TargetPlatform: "PC", ProjectDescription: "exploration", Width: 640, Height: 360,
	})
	if strings.Contains(promptWithoutCritique, "<previous_review_critique>") {
		t.Fatalf("expected no critique section when empty, got %s", promptWithoutCritique)
	}

	promptWithCritique := prompts.SceneryPlan(prompts.SceneryPlanInput{
		AssetName: "Valley", CreativeBrief: "dawn valley",
		Perspective: "Side-On", ProjectName: "Starbound", GameType: "RPG",
		TargetPlatform: "PC", ProjectDescription: "exploration", Width: 640, Height: 360,
		PreviousCritique: "The foreground bridge is floating 50px above the ground.",
	})
	if !strings.Contains(promptWithCritique, "<previous_review_critique>") ||
		!strings.Contains(promptWithCritique, "The foreground bridge is floating 50px above the ground.") {
		t.Fatalf("expected critique section to be included, got %s", promptWithCritique)
	}
}
