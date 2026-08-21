package prompts

import "fmt"

// PrototypeReferenceState describes which backend-ordered reference roles are
// present in a prototype generation request.
type PrototypeReferenceState struct {
	HasProjectReference  bool
	HasCreatingReference bool
}

func prototypeReferenceImageRoles(
	state PrototypeReferenceState,
	assetKind string,
) string {
	const conflictRule = `- If any reference image conflicts with a specific requirement in the user creative brief, follow the creative brief for that requirement.`
	const projectRole = `Use it only to establish the project's game-wide art style and visual-world conventions: pixel granularity, pixel block size, palette character, outline treatment, contrast, lighting, shading, rendering technique, material language, proportions, and perspective. Do not copy its specific subject, characters, objects, scenery, composition, text, logos, or other recognizable content.`

	switch {
	case state.HasProjectReference && state.HasCreatingReference:
		return fmt.Sprintf(`- Exactly two reference images are supplied in an authoritative order.
- Reference image 1 is the project prototype image and is the Style Reference. %s
- Reference image 2 is the creating reference image.
- The creating reference image is always a strong reference. When the user creative brief asks to preserve, closely match, imitate, or make only specific changes to it, preserve every unmentioned visual attribute and make only the changes explicitly requested in the user creative brief.
- Apply the project Style Reference's visual language to the requested %s.
%s`, projectRole, assetKind, conflictRule)
	case state.HasProjectReference:
		return fmt.Sprintf(`- Exactly one reference image is supplied.
- Reference image 1 is the project prototype image and is the Style Reference. %s
- No creating reference image is supplied. Derive the requested %s from the user creative brief.
%s`, projectRole, assetKind, conflictRule)
	case state.HasCreatingReference:
		return fmt.Sprintf(`- Exactly one reference image is supplied.
- Reference image 1 is the creating reference image.
- The creating reference image is always a strong reference. When the user creative brief asks to preserve, closely match, imitate, or make only specific changes to it, preserve every unmentioned visual attribute and make only the changes explicitly requested in the user creative brief.
- No project Style Reference is supplied.
%s`, conflictRule)
	default:
		return fmt.Sprintf(`- No reference images are supplied. Generate the requested %s from the user creative brief, selected perspective, pipeline processing requirements, and default production guidelines.
- Do not assume that any unstated project or creating reference is available.
%s`, assetKind, conflictRule)
	}
}
