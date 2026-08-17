package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
)

func TestSceneryPromptsContainSharedVisualContract(t *testing.T) {
	plan := prompts.SceneryPlan(prompts.SceneryPlanInput{
		AssetName: "Valley", CreativeBrief: "dawn valley", Style: "pixel art",
		Perspective: "Side-On", ProjectName: "Starbound", GameType: "RPG",
		TargetPlatform: "PC", ProjectDescription: "exploration", Width: 640, Height: 360,
	})
	layer := prompts.SceneryLayer(prompts.SceneryLayerInput{
		AssetName: "Valley", CreativeBrief: "dawn valley", Style: "pixel art",
		Perspective: "Side-On", ProjectName: "Starbound", GameType: "RPG",
		TargetPlatform: "PC", ProjectDescription: "exploration", Width: 640, Height: 360,
		LayerID: 2, LayerName: "Mountains", LayerCreativeBrief: "distant ridge",
	}, "solid matte")
	layout := prompts.SceneryLayoutAnalysis(prompts.SceneryLayoutAnalysisInput{
		AssetName: "Valley", CreativeBrief: "dawn valley", Style: "pixel art",
		Perspective: "Side-On", ProjectName: "Starbound", GameType: "RPG",
		TargetPlatform: "PC", ProjectDescription: "exploration", Width: 640, Height: 360,
		Layers: []prompts.SceneryLayoutLayerInput{{ID: 1, Name: "Sky"}, {ID: 2, Name: "Mountains"}},
	})

	for name, prompt := range map[string]string{"plan": plan, "layer": layer, "layout": layout} {
		for _, required := range []string{
			"classic low-resolution 2D pixel art", "hard aliased edges", "no antialiasing",
			"no characters, people, humanoids, animals, or creatures by default",
			"If and only if the user's creative brief explicitly requests",
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
		!strings.Contains(overlay, "clearly non-matte opaque subject pixels") {
		t.Fatalf("unexpected overlay contract: %s", overlay)
	}
}

func TestSceneryPlanPromptContainsSceneContextAndPlanningRules(t *testing.T) {
	prompt := prompts.SceneryPlan(prompts.SceneryPlanInput{
		AssetName: "Valley", CreativeBrief: "dawn valley", Style: "pixel art",
		Perspective: "Side-On", ProjectName: "Starbound", GameType: "RPG",
		TargetPlatform: "PC", ProjectDescription: "exploration", Width: 640, Height: 360,
	})
	for _, required := range []string{
		"Valley", "dawn valley", "pixel art", "Side-On", "Starbound", "RPG", "PC",
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
		AssetName: "Valley", CreativeBrief: "dawn valley", Style: "pixel art",
		Perspective: "Side-On", ProjectName: "Starbound", GameType: "RPG",
		TargetPlatform: "PC", ProjectDescription: "exploration", Width: 640, Height: 360,
		Layers: []prompts.SceneryLayoutLayerInput{{ID: 1, Name: "Sky"}, {ID: 2, Name: "Mountains"}},
	})
	for _, required := range []string{
		"Valley", "dawn valley", "pixel art", "Side-On", "Starbound", "RPG", "PC", "exploration",
		`<dimensions width="640" height="360" />`, "positive X to the right", "Rotation is clockwise",
		"Return exactly one layout", "already registered to the complete final canvas", "Default to position (0, 0)",
		"first attached image is the authoritative opaque full-canvas backdrop", "unique lowest zIndex",
		`Attached image 1 corresponds to layer ID 1 named "Sky"`,
		`Attached image 2 corresponds to layer ID 2 named "Mountains"`,
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("prompt omitted %q: %s", required, prompt)
		}
	}
}
