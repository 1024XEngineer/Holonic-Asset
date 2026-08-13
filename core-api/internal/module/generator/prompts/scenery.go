package prompts

import (
	"fmt"
	"strings"
)

const sceneryVisualConstraints = `Visual contract:
- The final scene and every independently generated layer MUST use classic low-resolution 2D pixel art, matching the character and object asset pipeline.
- Use deliberate pixel clusters, hard aliased edges, a restricted palette, no antialiasing, no smooth gradients, no painterly brushwork, no vector rendering, no 3D rendering, and no photorealism.
- Keep one consistent pixel density, palette, lighting model, material language, and perspective across every layer.
- Scenery should contain no characters, people, humanoids, animals, or creatures by default. If and only if the user's creative brief explicitly requests one of them, follow that explicit request and ignore this preference.`

const sceneryPlanTemplate = `Plan the independently generated image layers for one complete layered 2D game scenery.

%s

Rules:
- Decide the number of layers needed to express the complete scene. Use the fewest layers that preserve meaningful depth and editability.
- Return layers in back-to-front compositing order.
- Give every layer a short unique name and a self-contained image-generation brief.
- Each layer brief must describe only that layer's visual content and its intended placement, framing, scale, depth, and relationship to the full canvas.
- Coordinate silhouettes, overlaps, palette, lighting, perspective, and level of detail across all layer briefs so separately generated images form one coherent scene.
- Do not add IDs; the backend assigns stable IDs from response order.
- Return only the fields defined by the supplied JSON schema. Do not return explanations, coordinates, resources, or metadata.

Scenery asset:
<asset_name>%s</asset_name>
<creative_brief>%s</creative_brief>
<dimensions width="%d" height="%d" />
<perspective>%s</perspective>
<style>%s</style>

Project visual context:
<project_name>%s</project_name>
<game_type>%s</game_type>
<target_platform>%s</target_platform>
<project_description>%s</project_description>`

type SceneryPlanInput struct {
	AssetName          string
	CreativeBrief      string
	Style              string
	Perspective        string
	ProjectName        string
	GameType           string
	TargetPlatform     string
	ProjectDescription string
	Width              uint
	Height             uint
}

func SceneryPlan(input SceneryPlanInput) string {
	return fmt.Sprintf(
		sceneryPlanTemplate,
		sceneryVisualConstraints,
		strings.TrimSpace(input.AssetName),
		strings.TrimSpace(input.CreativeBrief),
		input.Width,
		input.Height,
		strings.TrimSpace(input.Perspective),
		strings.TrimSpace(input.Style),
		strings.TrimSpace(input.ProjectName),
		strings.TrimSpace(input.GameType),
		strings.TrimSpace(input.TargetPlatform),
		strings.TrimSpace(input.ProjectDescription),
	)
}

const sceneryLayerTemplate = `Create exactly one production-ready image layer for a layered 2D game scenery.

%s

Pipeline requirements:
%s
- Generate only the requested layer. Do not include content assigned to other layers.
- Keep the complete layer artwork inside the canvas with clean separation from the matte background.
- Do not add text, labels, logos, watermarks, borders, frames, UI, or a preview of the assembled scene.
- Match the shared scenery brief, style, perspective, palette, lighting, pixel density, and material treatment so this independently generated layer can be composited with every other layer.
- Treat any supplied reference image as visual-language guidance only. Do not copy its composition or recognizable content.

Scenery asset:
<asset_name>
%s
</asset_name>

Shared creative brief:
<creative_brief>
%s
</creative_brief>

Requested layer:
<layer_id>%d</layer_id>
<layer_name>%s</layer_name>
<layer_creative_brief>
%s
</layer_creative_brief>

Final canvas:
<dimensions width="%d" height="%d" />
<perspective>%s</perspective>
<style>%s</style>

Project visual context:
<project_name>%s</project_name>
<game_type>%s</game_type>
<target_platform>%s</target_platform>
<project_description>%s</project_description>
<reference_supplied>%t</reference_supplied>`

type SceneryLayerInput struct {
	AssetName          string
	CreativeBrief      string
	Style              string
	Perspective        string
	ProjectName        string
	GameType           string
	TargetPlatform     string
	ProjectDescription string
	Width              uint
	Height             uint
	LayerID            uint
	LayerName          string
	LayerCreativeBrief string
	HasReference       bool
}

func SceneryLayer(input SceneryLayerInput, backgroundConstraint string) string {
	return fmt.Sprintf(
		sceneryLayerTemplate,
		sceneryVisualConstraints,
		strings.TrimSpace(backgroundConstraint),
		strings.TrimSpace(input.AssetName),
		strings.TrimSpace(input.CreativeBrief),
		input.LayerID,
		strings.TrimSpace(input.LayerName),
		strings.TrimSpace(input.LayerCreativeBrief),
		input.Width,
		input.Height,
		strings.TrimSpace(input.Perspective),
		strings.TrimSpace(input.Style),
		strings.TrimSpace(input.ProjectName),
		strings.TrimSpace(input.GameType),
		strings.TrimSpace(input.TargetPlatform),
		strings.TrimSpace(input.ProjectDescription),
		input.HasReference,
	)
}

const sceneryLayoutAnalysisTemplate = `Inspect every attached processed image and propose the final layout for one layered 2D game scenery.

%s

Layout rules:
- Return exactly one layout for every supplied layer ID. Do not invent, omit, or duplicate IDs.
- Every attached image is already registered to the complete final canvas at the requested dimensions. Transparent pixels are intentional padding, and visible pixels already express the planned global placement.
- Default to position (0, 0), scale (1, 1), and rotation 0. Change these only when inspection of the actual attached pixels proves a correction is necessary; never transform already-correct content out of its intended canvas region.
- Use canvas pixels with the canvas top-left as (0, 0), positive X to the right, and positive Y downward.
- Position is the top-left of the scaled layer before rotation. Rotation is clockwise in degrees around the scaled layer center.
- Scale X and Y must be finite and greater than zero. Opacity must be from 0 through 1. zIndex must be an integer.
- Keep every transformed layer at least partially intersecting the canvas.
- Use the actual attached pixels, shared creative intent, perspective, and depth relationships to choose placement, scale, rotation, opacity, and stacking order.
- Return only the fields defined by the supplied JSON schema. Do not return names, visibility, resources, metadata, explanations, or revised images.

Scenery asset:
<asset_name>%s</asset_name>
<creative_brief>%s</creative_brief>
<dimensions width="%d" height="%d" />
<perspective>%s</perspective>
<style>%s</style>

Project visual context:
<project_name>%s</project_name>
<game_type>%s</game_type>
<target_platform>%s</target_platform>
<project_description>%s</project_description>

Attached layers:
%s`

type SceneryLayoutLayerInput struct {
	ID   uint
	Name string
}

type SceneryLayoutAnalysisInput struct {
	AssetName          string
	CreativeBrief      string
	Style              string
	Perspective        string
	ProjectName        string
	GameType           string
	TargetPlatform     string
	ProjectDescription string
	Width              uint
	Height             uint
	Layers             []SceneryLayoutLayerInput
}

func SceneryLayoutAnalysis(input SceneryLayoutAnalysisInput) string {
	var layerList strings.Builder
	for index, layer := range input.Layers {
		fmt.Fprintf(&layerList, "Attached image %d corresponds to layer ID %d named %q.\n", index+1, layer.ID, strings.TrimSpace(layer.Name))
	}
	return fmt.Sprintf(
		sceneryLayoutAnalysisTemplate,
		sceneryVisualConstraints,
		strings.TrimSpace(input.AssetName), strings.TrimSpace(input.CreativeBrief), input.Width, input.Height,
		strings.TrimSpace(input.Perspective), strings.TrimSpace(input.Style), strings.TrimSpace(input.ProjectName),
		strings.TrimSpace(input.GameType), strings.TrimSpace(input.TargetPlatform), strings.TrimSpace(input.ProjectDescription),
		strings.TrimSpace(layerList.String()),
	)
}
