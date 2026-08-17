package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
)

func TestUISetPlanPromptDefinesComponentUnitWithoutTiles(t *testing.T) {
	prompt := prompts.UISetPlan(prompts.UISetPlanInput{
		AssetName: "HUD", CreativeBrief: "compact combat HUD", Style: "pixel brass",
		ProjectName: "Moon Forge", GameType: "RPG", TargetPlatform: "PC",
		ProjectDescription: "tactical adventure", Width: 1280, Height: 720,
		Components: []prompts.UISetComponentInput{
			{Index: 0, Name: "Health Bar", Description: "top-left player health"},
			{Index: 1, Name: "Pause Button", Description: "small icon-only control"},
		},
	})
	for _, required := range []string{
		"Return exactly one entry for every requested Component", "positive integer pixel width and height",
		"one complete independently generated Component image", "has no Tiles, Tile size, grid, shape, footprint",
		"Do not choose positions", `<canvas width="1280" height="720" />`,
		`<component index="1"><name>Pause Button</name><description>small icon-only control</description></component>`,
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("UI Set prompt omitted %q: %s", required, prompt)
		}
	}
}
