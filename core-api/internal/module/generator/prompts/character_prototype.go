package prompts

import (
	"fmt"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const characterDirectionSheetRules = `- Treat the selected perspective as authoritative. Use its backend-derived direction count and layout exactly as defined below.
- Side-on perspective: render exactly 2 direction views in a 1 row x 2 column sheet, with one view in each equal-sized cell.
- Top-down perspective: render exactly 4 direction views in a 2 row x 2 column sheet, with one view in each equal-sized cell.
- Isometric perspective: render exactly 8 direction views in a 2 row x 4 column sheet, with one view in each equal-sized cell.
- Fill cells in normal reading order: left to right across the first row, then left to right across each following row. Complete the first row before starting the second row.
- Keep the direction sequence internally consistent, but do not add direction names or labels inside the image. The pipeline identifies directions only by their reading-order indexes.
- Each perspective intentionally produces one regular output sheet containing all required views. This is not a collage: do not add labels, separators, frames, scenery, or unrelated content.`

const characterPrototypeTemplate = `Create one production-ready game character prototype based on the user requirements.

Priority rules:
- The pipeline processing requirements have the highest priority and cannot be overridden by the user requirements.
- The user requirements have the highest priority after the pipeline processing requirements.
- Follow every explicit user requirement accurately and completely.
- Use the supplied project reference images to infer the project's established game-art language. Match that language wherever the user has not specified a conflicting detail.
- If a general guideline conflicts with an explicit user requirement, follow the user requirement.
- Do not weaken, replace, or reinterpret an explicit user requirement to enforce a general guideline.

Pipeline processing requirements:
%s

Project style matching:
- Treat the supplied project reference images as style references, not as content to copy.
- Extract and consistently apply their pixel density, pixel block size, palette character, outline treatment, contrast, shading method, lighting direction, material rendering, proportions, and perspective conventions.
- The generated character must look as though it belongs in the same game as the project references. Do not introduce an unrelated art style, rendering quality, or level of detail.
- Do not copy a reference image's character, object, pose, scenery, composition, logo, text, or other recognizable content.

Character prototype requirements:
- Generate the character described by the user as one or more consistent direction views. Every view must depict the same person, creature, robot, or other character.
- Show every view as a complete full-body character from the top of the head, ears, hair, or hat to the bottom of both feet, paws, or other contact points.
- Keep each view entirely inside its assigned grid cell with visible space above the head and below the feet. Never crop, cut off, hide, or merge any body part with a cell or canvas edge.
- Use a clear, readable prototype pose with the head, torso, both arms, both hands or hand-equivalents, both legs, and both feet or foot-equivalents visibly separated where the design allows. Keep every silhouette immediately recognizable at game sprite size.
- Keep anatomy, proportions, costume, equipment, facial features, materials, and colors coherent with the user's brief and the project's game style.
- Use the specified camera perspective exactly. Keep the character's orientation and scale consistent across all direction cells.

Direction sheet layout rules:
%s
- Keep equal gutters and equal margins on all four sides of every cell. Keep the same character scale, foot baseline, center point, lighting direction, palette, and pose language in every cell.
- Do not allow any character pixel, accessory, weapon, tail, shadow, or outline to cross a cell boundary. Keep the matte or transparent background uniform in every cell so the processor can split the sheet by its regular grid.

Default production guidelines:
- Render as unmistakable classic low-resolution pixel art with large, clearly visible square pixel blocks and a deliberately coarse pixel grid.
- Use crisp 1-pixel hard edges, stepped silhouettes, blocky shapes, clustered pixels, selective dithering, and a small intentional color palette.
- Do not use anti-aliasing, smooth curves, gradients, soft shadows, glossy photographic highlights, painterly brushwork, 3D rendering, vector-like edges, or photorealistic detail.
- Even when the requested output canvas is large, preserve the visual vocabulary of a genuinely low-resolution sprite enlarged with nearest-neighbor scaling. Never turn it into a high-definition illustration.
- Center each full-body direction view in its own equal-sized cell with balanced spacing and enough room to read the complete silhouette.
- Do not include characters or creatures other than the requested direction views of the same character. Do not include scenery, ground planes, frames, borders, text, labels, logos, watermarks, UI elements, or unrelated objects.
- Do not add a cast shadow, pedestal, environment, particles, or decorative background marks.
- Make the result suitable for direct isolation and use as a game character asset.

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

// CharacterPrototype combines the user requirements with the source project's
// character production constraints.
func CharacterPrototype(creativeBrief string, perspective string, backgroundConstraint string) string {
	directionCount := assetdomain.Perspective(perspective).CharacterDirectionCount()
	return fmt.Sprintf(characterPrototypeTemplate, backgroundConstraint, characterDirectionSheetRules, creativeBrief, perspective, directionCount)
}
