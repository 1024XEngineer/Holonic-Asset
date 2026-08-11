package prompts

import (
	"fmt"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const editCharacterPrototypeTemplate = `Edit one existing game character prototype according to the user instructions and the supplied reference images.

Priority rules:
- The pipeline processing requirements have the highest priority and cannot be overridden by the user instructions.
- The user edit instructions have the highest priority after the pipeline processing requirements.
- First determine whether the requested edit is minor, major, or mixed by using the edit-scope rules below.
- Use each reference image only for its stated role. Do not let reference content override an explicit user instruction.
- The general production guidelines apply only where the user has not provided conflicting instructions.
- If a general guideline conflicts with an explicit user instruction, follow the user instruction.
- Do not weaken, replace, or reinterpret an explicit user instruction to preserve the original character.

Pipeline processing requirements:
%s

Reference image roles:
- Reference images 1 through %d are the current character prototype direction views in normal reading order. Together they are the only supplied references and define the character's identity, full-body silhouette, proportions, equipment, current appearance, and visual style from every available direction.
- Treat every supplied reference image as part of the original prototype. No separate user or project reference image is supplied for this edit.
- The reference-image order above is authoritative even if the images contain visually ambiguous content. Never infer or swap their roles.

Original asset description:
<original_description>
%s
</original_description>

Edit-scope rules:
- Minor edit: The request keeps the same character identity, species, core silhouette, proportions, and construction. Examples include changing colors, adding or removing a small accessory, adjusting armor details, changing a weapon, or making another localized alteration.
- For a minor edit, reproduce all original character prototype reference images as faithfully as possible and change only what the user requested. Preserve every unrequested visible feature, including the full-body framing, silhouette, proportions, pose, orientation, anatomy, pixel placement character, outline, materials, textures, shading, highlights, equipment, and decorations.
- Major edit: The request changes the character's species, identity, purpose, core silhouette, main costume structure, or overall construction.
- For a major edit, prioritize the requested new character and appearance over resemblance to the original prototype. Build the form needed to satisfy the user's instructions while retaining compatible visual-style cues from the original prototype reference images.
- Mixed edit: When a request replaces one major part but leaves other parts unchanged, apply the major-edit rule to the replaced structure and the minor-edit rule to every unaffected, still-compatible part.
- If the scope is ambiguous, make the smallest set of visual changes that completely satisfies the user's explicit instructions.

Character framing and style requirements:
%s
- The perspective-derived direction count and grid override any conflicting direction count or layout implied by the original prototype reference images.
- If the original prototype references use a different direction count or order, rebuild the output sheet with the required perspective mapping while preserving the character's identity and compatible visual details.
- Edit every required direction cell consistently when the requested change applies to the character.
- Show the complete character full body in every cell from the top of the head, ears, hair, or hat to the bottom of both feet, paws, or other contact points.
- Keep visible space above the head and below the feet in every cell. Never crop, cut off, hide, or merge any body part with a cell or canvas edge.
- Keep the head, torso, arms, hands or hand-equivalents, legs, and feet or foot-equivalents readable where the design allows. Preserve a clear game-sprite silhouette.
- Use the specified camera perspective exactly. Preserve the same identity, equipment, palette, lighting, proportions, foot baseline, and scale across all direction cells.
- Keep equal gutters and margins in every cell. Do not allow any character pixel, accessory, weapon, tail, shadow, or outline to cross a cell boundary. Keep the background uniform so the processor can split the sheet by its regular grid.
- Match the original prototype's pixel density, palette, outlines, contrast, shading, lighting, materials, and perspective conventions unless the user explicitly requests a change to one of them.
- Render as unmistakable classic low-resolution pixel art with large, clearly visible square pixel blocks and a deliberately coarse pixel grid.
- Use crisp 1-pixel hard edges, stepped silhouettes, blocky shapes, clustered pixels, selective dithering, and a small intentional color palette.
- Do not use anti-aliasing, smooth curves, gradients, soft shadows, glossy photographic highlights, painterly brushwork, 3D rendering, vector-like edges, or photorealistic detail.
- Do not include other characters, people, creatures, scenery, ground planes, frames, borders, text, labels, logos, watermarks, UI elements, cast shadows, or unrelated objects.
- Make the result suitable for direct isolation and use as a game character asset.

User edit instructions:
<edit_instructions>
%s
</edit_instructions>

User-selected perspective:
<perspective>
%s
</perspective>

Backend-derived direction count:
<direction_count>
%d
</direction_count>`

// EditCharacterPrototype combines the original character description and
// direction references with user edit instructions.
func EditCharacterPrototype(
	originalDescription string,
	editInstructions string,
	perspective string,
	originalReferenceCount uint,
	backgroundConstraint string,
) string {
	directionCount := assetdomain.Perspective(perspective).CharacterDirectionCount()
	return fmt.Sprintf(
		editCharacterPrototypeTemplate,
		backgroundConstraint,
		originalReferenceCount,
		originalDescription,
		characterDirectionSheetRules,
		editInstructions,
		perspective,
		directionCount,
	)
}
