package prompts

import "fmt"

const editObjectPrototypeTemplate = `Edit one existing game object asset according to the user instructions and the supplied reference images.

Priority rules:
- The pipeline processing requirements have the highest priority and cannot be overridden by the user instructions.
- The user edit instructions have the highest priority after the pipeline processing requirements.
- First determine whether the requested edit is minor, major, or mixed by using the edit-scope rules below.
- Use each reference image only for its stated role. Do not let reference content override an explicit user instruction.
- The general production guidelines apply only where the user has not provided conflicting instructions.
- If a general guideline conflicts with an explicit user instruction, follow the user instruction.
- Do not weaken, replace, or reinterpret an explicit user instruction to preserve the original object.

Pipeline processing requirements:
%s

Reference image roles:
- Reference image 1, the first supplied image, is always the original object prototype and the only edit target. It defines the object's current identity and appearance. Treat this image as the source asset from which the output must be edited or replaced.
- Reference image 2 and every later supplied image are project reference images. They define only the project's visual language, including pixel density, palette character, outlines, shading, lighting, material treatment, and perspective conventions.
- The reference-image order above is authoritative even if the images contain visually ambiguous content. Never infer or swap their roles.
- Never use a project reference image as the edit target, base canvas, primary subject, composition, or scene. Never output a modified or recreated version of a project reference image.
- Do not copy objects, characters, scenery, layouts, or unrelated content from the project reference images. Use them only to keep the object derived from reference image 1 visually consistent with the project.

Edit-scope rules:
- Minor edit: The requested result remains the same object type and retains the same core identity, silhouette, proportions, and construction. Examples include changing a colour, adjusting one material or surface detail, adding or removing a small accessory, changing a small decoration, or making another localized alteration.
- For a minor edit, reproduce reference image 1, the original object prototype, as faithfully as possible and change only what the user requested. Strictly preserve every unrequested visible feature, including the silhouette, proportions, structure, orientation, pixel placement character, outline, materials, textures, shading, highlights, wear, and decorations. Preserve the original perspective unless the user-selected perspective explicitly changes it. Do not redesign, simplify, embellish, clean up, or restyle the rest of the object. Make any new or changed detail fit the project's visual language.
- Major edit: The request changes the object type, purpose, core silhouette, main structure, or overall construction. For example, replacing a chest with a door is a major edit even if the request is phrased as an edit.
- For a major edit, prioritize the requested new object and appearance over resemblance to the original prototype. Build the form needed to satisfy the user's instructions; do not force the old object's category, silhouette, proportions, or construction into the result. Use the project reference images as the primary visual-style guide. Reuse details from the original prototype only when they remain compatible with the new request and do not weaken it.
- Mixed edit: When a request replaces one major part but leaves other parts unchanged, apply the major-edit rule to the replaced structure and the minor-edit rule to every unaffected, still-compatible part.
- If the scope is ambiguous, make the smallest set of visual changes that completely satisfies the user's explicit instructions.

Default production guidelines:
- Render as unmistakable classic low-resolution pixel art with large, clearly visible square pixel blocks and a deliberately coarse pixel grid.
- Use crisp 1-pixel hard edges, stepped silhouettes, blocky shapes, clustered pixels, selective dithering, and a small intentional colour palette.
- Do not use anti-aliasing, smooth curves, gradients, soft shadows, glossy photographic highlights, painterly brushwork, 3D rendering, vector-like edges, or photorealistic detail.
- Even when the requested output canvas is large, preserve the visual vocabulary of a genuinely low-resolution sprite enlarged with nearest-neighbour scaling. Never turn it into a high-definition illustration.
- Generate one edited object as the only subject.
- Show the entire object fully inside the canvas.
- Center the object with balanced spacing around all edges.
- Use the specified camera perspective exactly.
- Keep the object's shape, proportions, materials, and details visually coherent.
- Do not include characters, people, hands, creatures, scenery, ground planes, frames, borders, text, labels, logos, watermarks, UI elements, or unrelated objects.
- Do not create a collage, contact sheet, turnaround sheet, multiple variants, or multiple viewing angles.
- Do not crop, cut off, obscure, or overlap any part of the object.
- Make the result suitable for direct isolation and use as a game asset.

User edit instructions:
<edit_instructions>
%s
</edit_instructions>

User-selected perspective:
<perspective>
%s
</perspective>`

// EditObjectPrototype combines an object edit request with the source project's
// reference-handling and production constraints.
func EditObjectPrototype(editInstructions string, perspective string, backgroundConstraint string) string {
	return fmt.Sprintf(editObjectPrototypeTemplate, backgroundConstraint, editInstructions, perspective)
}
