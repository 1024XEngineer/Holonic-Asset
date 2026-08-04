package prompts

import "fmt"

const objectPrototypeTemplate = `Create one production-ready game object asset based on the user requirements.

Priority rules:
- The user requirements have the highest priority.
- Follow every explicit user requirement accurately and completely.
- The general production guidelines below apply only where the user has not provided conflicting instructions.
- If a general guideline conflicts with an explicit user requirement, follow the user requirement.
- Do not weaken, replace, or reinterpret an explicit user requirement to enforce a general guideline.

Default production guidelines:
- Generate one object as the only subject.
- Show the entire object fully inside the canvas.
- Center the object with balanced spacing around all edges.
- Use the specified camera perspective exactly.
- Keep the object's shape, proportions, materials, and details visually coherent.
- Use a clean transparent background.
- Do not include characters, people, hands, creatures, scenery, ground planes, frames, borders, text, labels, logos, watermarks, UI elements, or unrelated objects.
- Do not create a collage, contact sheet, turnaround sheet, multiple variants, or multiple viewing angles.
- Do not crop, cut off, obscure, or overlap any part of the object.
- Preserve the requested visual style without introducing an unrelated art style.
- Make the result suitable for direct isolation and use as a game asset.

User creative brief:
<creative_brief>
%s
</creative_brief>

User-selected perspective:
<perspective>
%s
</perspective>`

// ObjectPrototype combines object defaults with the user's creative inputs.
func ObjectPrototype(creativeBrief string, perspective string) string {
	return fmt.Sprintf(objectPrototypeTemplate, creativeBrief, perspective)
}
