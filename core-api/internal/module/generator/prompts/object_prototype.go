package prompts

import (
	"fmt"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const objectPrototypeTemplate = `Create one production-ready game object asset based on the user requirements.

Priority rules:
- The pipeline processing requirements have the highest priority and cannot be overridden by the user requirements.
- The user requirements have the highest priority after the pipeline processing requirements.
- Follow every explicit user requirement accurately and completely.
- The general production guidelines below apply only where the user has not provided conflicting instructions.
- If a general guideline conflicts with an explicit user requirement, follow the user requirement.
- Do not weaken, replace, or reinterpret an explicit user requirement to enforce a general guideline.

Pipeline processing requirements:
%s

Default production guidelines:
- Render as unmistakable classic low-resolution pixel art with large, clearly visible square pixel blocks and a deliberately coarse pixel grid.
- Use crisp 1-pixel hard edges, stepped silhouettes, blocky shapes, clustered pixels, selective dithering, and a small intentional colour palette.
- Do not use anti-aliasing, smooth curves, gradients, soft shadows, glossy photographic highlights, painterly brushwork, 3D rendering, vector-like edges, or photorealistic detail.
- Even when the requested output canvas is large, preserve the visual vocabulary of a genuinely low-resolution sprite enlarged with nearest-neighbour scaling. Never turn it into a high-definition illustration.
- Generate direction views of one consistent object as the only subject. Every cell must depict the same object.
- Show the entire object fully inside every assigned grid cell.
- Center each view with balanced spacing around all cell edges.
- Use the specified camera perspective exactly.
- Keep the object's shape, proportions, materials, details, scale, and lighting visually coherent across all cells.
- If project reference images are supplied, strictly follow their art style, visual language, rendering technique, and form of expression without copying recognizable content.
- Do not include characters, people, hands, creatures, scenery, ground planes, frames, borders, text, labels, logos, watermarks, UI elements, or unrelated objects.
- Do not create variants beyond the required direction views.
- Do not crop, cut off, obscure, or overlap any part of the object.
- Preserve the requested visual style without introducing an unrelated art style.
- Make the result suitable for direct isolation and use as a game asset.

Direction sheet layout rules:
%s
- Keep equal gutters and equal margins on all four sides of every cell.
- Do not allow any object pixel, attachment, shadow, or outline to cross a cell boundary. Keep the background uniform in every cell so the processor can split the sheet by its regular grid.

User creative brief:
<creative_brief>
%s
</creative_brief>

User-selected perspective:
<perspective>
%s
</perspective>

Backend-derived direction count:
<direction_count>
%d
</direction_count>`

// ObjectPrototype combines the user requirements with the source project's
// production constraints for one game object.
func ObjectPrototype(creativeBrief string, perspective string, backgroundConstraint string) string {
	directionCount := assetdomain.Perspective(perspective).CharacterDirectionCount()
	return fmt.Sprintf(objectPrototypeTemplate, backgroundConstraint, characterDirectionSheetRules, creativeBrief, perspective, directionCount)
}

// SolidMatteBackground requires a deterministic chroma-key input for the
// processor. This is a pipeline constraint for character and object assets,
// not part of the user's brief.
func SolidMatteBackground(matteColor string) string {
	return fmt.Sprintf(`- Render the background as exactly one perfectly flat, uniform, solid %s colour, filling the entire canvas edge to edge.
- The background must be fully opaque. Do not output transparency or a checkerboard transparency pattern.
- Do not add gradients, textures, lighting variation, shadows, scenery, ground, glow, particles, or any other marks to the background.
- Keep a crisp, clean boundary between the subject and the background, with no colour spill or background-coloured fringe.
- Do not use the exact background colour inside the subject unless it is essential to the user's explicit design.`, matteColor)
}

// TransparentBackground is used only when chroma-key removal is explicitly
// disabled and the provider is expected to return native alpha.
func TransparentBackground() string {
	return `- Render the subject on a clean, fully transparent background with a real alpha channel.
- Do not draw a checkerboard pattern, solid backdrop, scenery, ground, cast shadow, ambient glow, or other background content.`
}
