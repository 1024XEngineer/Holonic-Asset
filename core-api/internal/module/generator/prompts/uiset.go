package prompts

import (
	"fmt"
	"strings"
)

type UISetComponentInput struct {
	Index       int
	Name        string
	Description string
}

type UISetPlanInput struct {
	AssetName          string
	CreativeBrief      string
	Style              string
	ProjectName        string
	GameType           string
	TargetPlatform     string
	ProjectDescription string
	Width              uint
	Height             uint
	Components         []UISetComponentInput
}

const uiSetPlanTemplate = `Plan the pixel dimensions of every independently generated Component in one cohesive 2D game UI Set.

Rules:
- Return exactly one entry for every requested Component, in the same order.
- Preserve each supplied Component index. Do not add, remove, merge, split, rename, or reorder Components.
- Assign each Component a positive integer pixel width and height that fits within the complete UI Set canvas.
- Size each Component according to its role, description, hierarchy, readability, and relationship to the other requested Components.
- Each planned size directly defines one complete independently generated Component image.
- A UI Set has no Tiles, Tile size, grid, shape, footprint, occupied cells, atlas, or image-splitting step. Do not derive a Component size from any Tile abstraction.
- Do not choose positions. Layout is handled by a later phase without changing these planned sizes.
- Return only the fields defined by the supplied JSON schema.

UI Set:
<asset_name>%s</asset_name>
<creative_brief>%s</creative_brief>
<style>%s</style>
<canvas width="%d" height="%d" />

Project context:
<project_name>%s</project_name>
<game_type>%s</game_type>
<target_platform>%s</target_platform>
<project_description>%s</project_description>

Requested Components:
%s`

func UISetPlan(input UISetPlanInput) string {
	components := make([]string, len(input.Components))
	for index, component := range input.Components {
		components[index] = fmt.Sprintf(
			`<component index="%d"><name>%s</name><description>%s</description></component>`,
			component.Index,
			strings.TrimSpace(component.Name),
			strings.TrimSpace(component.Description),
		)
	}
	return fmt.Sprintf(
		uiSetPlanTemplate,
		strings.TrimSpace(input.AssetName),
		strings.TrimSpace(input.CreativeBrief),
		strings.TrimSpace(input.Style),
		input.Width,
		input.Height,
		strings.TrimSpace(input.ProjectName),
		strings.TrimSpace(input.GameType),
		strings.TrimSpace(input.TargetPlatform),
		strings.TrimSpace(input.ProjectDescription),
		strings.Join(components, "\n"),
	)
}
