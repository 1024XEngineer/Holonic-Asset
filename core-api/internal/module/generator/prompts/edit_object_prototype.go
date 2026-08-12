package prompts

import (
	"fmt"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const editObjectPrototypeTemplate = `Edit one existing game object prototype according to the user instructions and the supplied reference images.

Priority rules:
- The pipeline processing requirements have the highest priority and cannot be overridden by the user instructions.
- The user edit instructions have the highest priority after the pipeline processing requirements.
- First determine whether the requested edit is minor, major, or mixed by using the edit-scope rules below.
- Treat every supplied reference image as part of the original object prototype. No user or project reference image is supplied for this edit.
- The general production guidelines apply only where the user has not provided conflicting instructions.
- If a general guideline conflicts with an explicit user instruction, follow the user instruction.
- Do not weaken, replace, or reinterpret an explicit user instruction to preserve the original object.

Pipeline processing requirements:
%s

Reference image roles:
- The backend supplied exactly %d current prototype direction image(s).
- Reference images are the current object prototype direction images, in their supplied order. Together they define the object's identity, silhouette, proportions, construction, orientation, materials, details, and visual style.
- The reference-image order is authoritative. Never infer, swap, omit, or reorder direction images.
- Do not use any unrelated content from a reference image. Every supplied image is an edit target and must remain recognizable as the same object unless the user explicitly requests a major or mixed edit.

Edit-scope rules:
- Minor edit: The request keeps the same object identity, core silhouette, proportions, and construction. Examples include changing a colour, adjusting one material or surface detail, adding or removing a small accessory, changing a small decoration, or making another localized alteration.
- For a minor edit, reproduce all supplied object prototype reference images as faithfully as possible and change only what the user requested. Preserve every unrequested visible feature, including the silhouette, proportions, structure, orientation, pixel placement character, outline, materials, textures, shading, highlights, wear, and decorations.
- Major edit: The request changes the object type, purpose, core silhouette, main structure, or overall construction.
- For a major edit, prioritize the requested new object and appearance over resemblance to the original prototype. Build the form needed to satisfy the user's instructions while retaining compatible visual style cues from the original prototype references.
- Mixed edit: When a request replaces one major part but leaves other parts unchanged, apply the major-edit rule to the replaced structure and the minor-edit rule to every unaffected, still-compatible part.
- If the scope is ambiguous, make the smallest set of visual changes that completely satisfies the user's explicit instructions.

Object framing and style requirements:
%s
- The perspective-derived direction count and grid override any conflicting direction count or layout visible in the supplied references.
- Output one direction sheet containing exactly the backend-derived number of cells in the backend-defined grid.
- Edit every required direction cell consistently when the requested change applies to the object.
- Show the complete object fully inside every cell with balanced spacing on all sides. Never crop, cut off, hide, or merge any part with a cell or canvas edge.
- Use the specified camera perspective exactly. Preserve the same object identity, scale, palette, lighting, proportions, and visual style across all direction cells unless explicitly changed by the user.
- Keep equal gutters and margins in every cell. Do not allow any object pixel, attachment, shadow, or outline to cross a cell boundary. Keep the background uniform so the processor can split the sheet by its regular grid.
- Match the supplied prototype's pixel density, palette, outlines, contrast, shading, lighting, materials, and perspective conventions unless the user explicitly requests a change to one of them.
- Render as unmistakable classic low-resolution pixel art with large, clearly visible square pixel blocks and a deliberately coarse pixel grid.
- Use crisp 1-pixel hard edges, stepped silhouettes, blocky shapes, clustered pixels, selective dithering, and a small intentional colour palette.
- Do not use anti-aliasing, smooth curves, gradients, soft shadows, glossy photographic highlights, painterly brushwork, 3D rendering, vector-like edges, or photorealistic detail.
- Do not include characters, people, hands, creatures, scenery, ground planes, frames, borders, text, labels, logos, watermarks, UI elements, or unrelated objects.
- Do not create variants beyond the required direction views.
- Make the result suitable for direct isolation and use as a game object asset.

Original asset description:
<original_description>
%s
</original_description>

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

// EditObjectPrototype combines the original object description and direction
// references with the user's edit instructions.
func EditObjectPrototype(
	originalDescription string,
	editInstructions string,
	perspective string,
	originalReferenceCount uint,
	backgroundConstraint string,
) string {
	directionCount := assetdomain.Perspective(perspective).CharacterDirectionCount()
	return fmt.Sprintf(
		editObjectPrototypeTemplate,
		backgroundConstraint,
		originalReferenceCount,
		characterDirectionSheetRules,
		originalDescription,
		editInstructions,
		perspective,
		directionCount,
	)
}
