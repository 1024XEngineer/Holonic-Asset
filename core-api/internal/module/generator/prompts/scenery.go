package prompts

import (
	"fmt"
	"strings"
)

const sceneryVisualConstraints = `STYLE LOCK — apply before interpreting subject matter:
- The final scene and every independently generated layer MUST use classic low-resolution 2D pixel art, matching the character and object asset pipeline.
- Perspective contract: Game scenery backgrounds MUST ALWAYS strictly use a Side-On (side-view / 2D horizontal side-scrolling / panoramic) perspective. Never generate top-down, isometric, birds-eye, high-angle, or first-person views for scenery backgrounds.
- Reference contract: Any supplied project or asset reference image provides visual style ONLY (color palette, pixel texture, lighting atmosphere, material mood). Game background scenery MUST NOT inherit or reference any perspective, angle, or composition from project references.
- Grounding contract: Ground-level, terrain, platform, and highway foreground layers MUST occupy the lower region of the canvas and anchor solidly to the bottom of the frame to ground the scene naturally. Do not float terrain or foreground structures in mid-air unless explicitly requested as floating islands.
- Native 1990s 16-bit 2D gameplay scenery assembled from hand-authored tile-scale shapes and sprite-like clusters. It must look like actual low-resolution game art, never a polished illustration, concept painting, 3D render, or modern high-definition pixel artwork.
- Use one shared 12-to-16-colour master palette, two flat tones plus at most one highlight per material, selective one-logical-pixel contours, stepped diagonals, hard aliased edges, and no antialiasing.
- Construct the scene from broad connected colour clusters, simple reusable motifs, strong silhouettes, and quiet untextured areas. Represent foliage as clustered crowns, crops as symbolic grouped rows, and stone, timber, soil, roofs, and water with only a few repeatable marks.
- Use one flat light direction and hard-edged cluster shadows. No smooth gradients, soft shading, ambient occlusion, bevels, glow, bloom, volumetric light, blur, glossy reflections, painterly texture, dense microdetail, or one-output-pixel noise.
- Every layer must use exactly the same palette ramps, contour weight, logical pixel size, light direction, material shorthand, and perspective. Show depth only through overlap, simpler silhouettes, fewer colours, and lower contrast toward the distance.
- If accurate small subject detail conflicts with this STYLE LOCK, simplify or omit the detail. Reject any layer that resembles a high-resolution illustration passed through a pixel filter.
- Scenery should contain no characters, people, humanoids, animals, or creatures by default. If and only if the user's creative brief explicitly requests one of them, follow that explicit request and ignore this preference.`

const sceneryLogicalPixelScale uint = 4

func sceneryPixelGridContract(width, height uint) string {
	if width == 0 || height == 0 || width%sceneryLogicalPixelScale != 0 || height%sceneryLogicalPixelScale != 0 {
		return `Pixel-grid contract:
- Use one visibly chunky, uniform pixel grid across the complete canvas and every layer. Never mix pixel densities or introduce sub-pixel detail.`
	}
	return fmt.Sprintf(`Pixel-craft contract (supports, but does not define, the art direction):
- Compose the final %dx%d file as if it were authored natively on one shared logical %dx%d canvas and presented at %dx nearest-neighbour scale.
- One logical pixel occupies one crisp %dx%d output block. Align silhouettes, contours, shadow clusters, highlights, and intentional dither patterns to that grid, with no sub-grid marks.
- Judge the artwork at the logical resolution: large shapes, palette discipline, and cluster design must already look finished there. The grid is only an alignment aid and must never become a substitute for authentic low-resolution game-art decisions.
- Do not first create smooth high-resolution artwork and then pixelate, sharpen, posterize, downsample, or place a pixel filter over it.`,
		width, height, width/sceneryLogicalPixelScale, height/sceneryLogicalPixelScale,
		sceneryLogicalPixelScale, sceneryLogicalPixelScale, sceneryLogicalPixelScale,
	)
}

const sceneryPlanTemplate = `Plan the independently generated image layers for one complete layered 2D game scenery.

%s

%s

Rules:
- Decide the number of layers needed to express the complete scene. Use the fewest layers that preserve meaningful depth and editability.
- Return layers in back-to-front compositing order.
- Give every layer a short unique name and a self-contained image-generation brief.
- Each layer brief must describe only that layer's visual content and its intended placement, framing, scale, depth, and relationship to the full canvas.
- Assign every scene element, structure, terrain band, and water feature to exactly one layer. Never duplicate content across layer briefs.
- Treat every layer as a full-canvas raster with the exact requested dimensions and aspect ratio. Overlay artwork may occupy only part of that raster, but its transparent padding keeps it registered to the full canvas.
- Never plan a square tile, card, panel, or isolated centred illustration inside a widescreen layer. Anchor visible content to explicit canvas regions and shared horizon lines.
- Coordinate silhouettes, overlaps, palette, lighting, perspective, and level of detail across all layer briefs so separately generated images form one coherent scene.
- Establish one concise style lock covering palette families, contour weight, light direction, shadow method, perspective, and material abstraction, then repeat that same lock in every layer brief. Depth may reduce contrast and detail but must not change the style lock.
- Plan broad graphic masses before decorative content. If a layer would need dense texture to be recognizable, simplify its forms instead.
- For terrain, roads, platforms, or foreground structures, explicitly specify their grounding alignment along the bottom border of the canvas.
- Do not add IDs; the backend assigns stable IDs from response order.
- Return only the fields defined by the supplied JSON schema. Do not return explanations, coordinates, resources, or metadata.

Scenery asset:
<asset_name>%s</asset_name>
<creative_brief>%s</creative_brief>
<dimensions width="%d" height="%d" />
<perspective>%s</perspective>

Project visual context:
<project_name>%s</project_name>
<game_type>%s</game_type>
<target_platform>%s</target_platform>
<project_description>%s</project_description>%s`

type SceneryPlanInput struct {
	AssetName          string
	CreativeBrief      string
	Perspective        string
	ProjectName        string
	GameType           string
	TargetPlatform     string
	ProjectDescription string
	PreviousCritique   string
	Width              uint
	Height             uint
}

func SceneryPlan(input SceneryPlanInput) string {
	critiqueSection := ""
	if critique := strings.TrimSpace(input.PreviousCritique); critique != "" {
		critiqueSection = fmt.Sprintf("\n\nPrevious review critique to address:\n<previous_review_critique>\n%s\n</previous_review_critique>", critique)
	}
	return fmt.Sprintf(
		sceneryPlanTemplate,
		sceneryVisualConstraints,
		sceneryPixelGridContract(input.Width, input.Height),
		strings.TrimSpace(input.AssetName),
		strings.TrimSpace(input.CreativeBrief),
		input.Width,
		input.Height,
		strings.TrimSpace(input.Perspective),
		strings.TrimSpace(input.ProjectName),
		strings.TrimSpace(input.GameType),
		strings.TrimSpace(input.TargetPlatform),
		strings.TrimSpace(input.ProjectDescription),
		critiqueSection,
	)
}

const sceneryLayerTemplate = `Create exactly one production-ready image layer for a layered 2D game scenery.

%s

%s

Pipeline requirements:
%s
%s
- Treat the overall art direction as the acceptance criterion. A recognizable subject rendered as modern high-definition pixel illustration is a failed layer; simplify it until it belongs to the same native low-resolution game-art system as every other layer.
- Generate only the requested layer. Do not include content assigned to other layers.
- Return the exact full-canvas aspect ratio. For overlays, preserve full-canvas registration through matte padding; do not create a square tile, card, panel, or isolated centred illustration.
- Follow the requested global placement instead of recentering the visible artwork. Its scale, horizon, and position must coordinate with the complete canvas and supplied foreground context.
- Keep the complete layer artwork inside the canvas with clean separation from the matte background.
- Do not add text, labels, logos, watermarks, borders, frames, UI, or a preview of the assembled scene.
- Match the shared scenery brief, style, perspective, palette, lighting, pixel density, and material treatment so this independently generated layer can be composited with every other layer.
- Preserve broad value groups and quiet areas. Do not fill available space with extra texture, tiny props, repeated highlights, or ornamental detail merely because the output canvas is large.

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

Project visual context:
<project_name>%s</project_name>
<game_type>%s</game_type>
<target_platform>%s</target_platform>
<project_description>%s</project_description>
<reference_supplied>%t</reference_supplied>
<generated_foreground_reference_supplied>%t</generated_foreground_reference_supplied>`

type SceneryLayerInput struct {
	AssetName              string
	CreativeBrief          string
	Perspective            string
	ProjectName            string
	GameType               string
	TargetPlatform         string
	ProjectDescription     string
	Width                  uint
	Height                 uint
	LayerID                uint
	LayerName              string
	LayerCreativeBrief     string
	HasReference           bool
	HasForegroundReference bool
	IsBackmost             bool
}

func SceneryLayer(input SceneryLayerInput, backgroundConstraint string) string {
	backgroundContract := strings.TrimSpace(backgroundConstraint)
	if input.IsBackmost {
		backgroundContract = `- This is the backmost scenery layer. It may cover the complete canvas edge to edge and may be fully opaque.
- Do not add a chroma-key matte, transparency checkerboard, alpha holes, border, frame, or empty margin to this backmost layer.`
	} else {
		backgroundContract += `
- This is an overlay layer. Every pixel outside the requested artwork must remain the exact solid matte colour.
- Pixel-grid, palette, texture, and dithering instructions apply only to the requested artwork. The matte-only area must remain one exact flat colour in every pixel; never add darker green strokes, grid marks, dithering, shading, or texture to it.
- Leave a continuous matte-only border around the canvas edge. Do not let artwork, glow, shadow, or antialiasing touch the canvas edge.
- Do not return the matte-only input unchanged; the requested layer must contain clearly non-matte opaque subject pixels.`
	}
	referenceContract := "- No reference image is supplied for this layer."
	if input.HasReference {
		referenceContract = "- Treat the supplied reference image as visual-language guidance only. Do not copy its composition or recognizable content."
	}
	if input.HasForegroundReference {
		referenceContract = `- The final supplied reference image is the cumulative transparent preview of the scenery layers already generated in front of this requested layer.
- Use that preview as exact spatial context: align silhouettes, paths, horizons, perspective, palette, and lighting with its existing pixels.
- Generate only the requested layer behind the preview. Do not redraw, copy, or bake any foreground preview content into this layer.
- Any earlier supplied reference image is visual-language guidance only and must not override the cumulative preview's established composition.`
	}
	generationOrderContract := ""
	if !input.IsBackmost && !input.HasForegroundReference {
		generationOrderContract = `- This is the first generated foreground layer and the visual style anchor for the entire scene. Establish the canonical palette, contour weight, cluster size, flat lighting, and material shorthand with the simplest forms that remain readable; do not showcase fine detail.`
	}
	return fmt.Sprintf(
		sceneryLayerTemplate,
		sceneryVisualConstraints,
		sceneryPixelGridContract(input.Width, input.Height),
		backgroundContract,
		strings.TrimSpace(referenceContract+"\n"+generationOrderContract),
		strings.TrimSpace(input.AssetName),
		strings.TrimSpace(input.CreativeBrief),
		input.LayerID,
		strings.TrimSpace(input.LayerName),
		strings.TrimSpace(input.LayerCreativeBrief),
		input.Width,
		input.Height,
		strings.TrimSpace(input.Perspective),
		strings.TrimSpace(input.ProjectName),
		strings.TrimSpace(input.GameType),
		strings.TrimSpace(input.TargetPlatform),
		strings.TrimSpace(input.ProjectDescription),
		input.HasReference,
		input.HasForegroundReference,
	)
}

const sceneryLayoutAnalysisTemplate = `Inspect every attached processed image, critically evaluate the overall composition quality, and propose the final layout for one layered 2D game scenery.

%s

Review and Calibration rules:
- Review the entire composed scene critically for floating ground structures, broken horizontal spans, perspective mismatches, scale inconsistencies, or unnatural gaps.
- If the composition is visually coherent, properly grounded, and ready for production, set approved to true. If the scene has severe layout flaws, floating ground elements, or incompatible layer perspectives, set approved to false.
- Include concise review_notes summarizing your assessment and any specific visual defects observed.
- Return exactly one layout for every supplied layer ID. Do not invent, omit, or duplicate IDs.
- Every attached image is already registered to the complete final canvas at the requested dimensions. Transparent pixels are intentional padding, and visible pixels already express the planned global placement.
- The first attached image is the authoritative opaque full-canvas backdrop. Keep it at position (0, 0), scale (1, 1), rotation 0, opacity 1, and give it the unique lowest zIndex so it can never cover another layer.
- Default to position (0, 0), scale (1, 1), and rotation 0. Change these only when inspection of the actual attached pixels proves a correction is necessary; never transform already-correct content out of its intended canvas region.
- Every layer uses the same final canvas aspect ratio. Scale must be uniform: scale.x and scale.y MUST be identical. Never use a square or other non-uniform scale for a 16:9 scenery layer.
- Use canvas pixels with the canvas top-left as (0, 0), positive X to the right, and positive Y downward.
- Position is the top-left of the scaled layer before rotation. Rotation is clockwise in degrees around the scaled layer center.
- Scale X and Y must be finite and greater than zero. Opacity must be from 0 through 1. zIndex must be an integer.
- Keep every transformed layer at least partially intersecting the canvas.
- Use the actual attached pixels, shared creative intent, perspective, and depth relationships to choose placement, scale, rotation, opacity, and stacking order.
- Return only the fields defined by the supplied JSON schema. Do not return names, visibility, resources, metadata, or revised images.

Scenery asset:
<asset_name>%s</asset_name>
<creative_brief>%s</creative_brief>
<dimensions width="%d" height="%d" />
<perspective>%s</perspective>

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
		strings.TrimSpace(input.Perspective), strings.TrimSpace(input.ProjectName),
		strings.TrimSpace(input.GameType), strings.TrimSpace(input.TargetPlatform), strings.TrimSpace(input.ProjectDescription),
		strings.TrimSpace(layerList.String()),
	)
}
