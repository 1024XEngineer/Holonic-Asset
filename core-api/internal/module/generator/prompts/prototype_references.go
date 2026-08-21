package prompts

import (
	"fmt"
	"strings"
)

// PrototypeReferenceState describes which backend-ordered reference roles are
// present in a prototype generation request.
type PrototypeReferenceState struct {
	HasProjectReference  bool
	HasCreatingReference bool
	NexusReferenceCount  int
}

func prototypeReferenceImageRoles(
	state PrototypeReferenceState,
	assetKind string,
) string {
	const conflictRule = `- If any reference image conflicts with a specific requirement in the user creative brief, follow the creative brief for that requirement.`
	const projectRole = `Use the Project Reference only to establish the project's game-wide art style and visual-world conventions: pixel granularity, pixel block size, palette character, outline treatment, contrast, lighting, shading, rendering technique, material language, proportions, and perspective. Do not copy its specific subject, characters, objects, scenery, composition, text, logos, or other recognizable content.`
	const creatingRole = `The Creating Reference is the user's reference for the object or subject being created. When the user creative brief asks to preserve, closely match, imitate, or make only specific changes to it, preserve every unmentioned visual attribute and make only the changes explicitly requested in the user creative brief.`
	const nexusRole = `Use each Nexus Reference only for same-project series consistency, compatible material treatment, and category details. Do not copy its specific subject, identity, pose, layout, or composition, and never treat it as the object the user asked to create.`

	if state.NexusReferenceCount > 0 {
		total := state.NexusReferenceCount
		if state.HasProjectReference {
			total++
		}
		if state.HasCreatingReference {
			total++
		}
		lines := []string{fmt.Sprintf("- Exactly %d reference images are supplied in an authoritative order.", total)}
		index := 1
		if state.HasProjectReference {
			lines = append(lines, fmt.Sprintf("- Reference image %d is the Project Reference (the project prototype image). %s", index, projectRole))
			index++
		}
		if state.HasCreatingReference {
			lines = append(lines,
				fmt.Sprintf("- Reference image %d is the Creating Reference (the user's subject or concept reference).", index),
				"- "+creatingRole,
			)
			index++
		}
		for range state.NexusReferenceCount {
			lines = append(lines, fmt.Sprintf(
				"- Reference image %d is a Nexus Reference selected by matching Tags within the current Project. %s",
				index,
				nexusRole,
			))
			index++
		}
		if state.HasProjectReference {
			lines = append(lines, fmt.Sprintf("- Apply the Project Reference's visual language to the requested %s.", assetKind))
		} else {
			lines = append(lines, fmt.Sprintf("- No Project Reference is supplied; derive the requested %s from the user creative brief and use Nexus References only for series consistency.", assetKind))
		}
		lines = append(lines, conflictRule)
		return strings.Join(lines, "\n")
	}

	switch {
	case state.HasProjectReference && state.HasCreatingReference:
		return fmt.Sprintf(`- Exactly two reference images are supplied in an authoritative order.
- Reference image 1 is the Project Reference (the project prototype image). %s
- Reference image 2 is the Creating Reference (the user's subject or concept reference).
- %s
- Apply the Project Reference's visual language to the requested %s.
%s`, projectRole, creatingRole, assetKind, conflictRule)
	case state.HasProjectReference:
		return fmt.Sprintf(`- Exactly one reference image is supplied.
- Reference image 1 is the Project Reference (the project prototype image). %s
- No Creating Reference is supplied. Derive the requested %s from the user creative brief.
%s`, projectRole, assetKind, conflictRule)
	case state.HasCreatingReference:
		return fmt.Sprintf(`- Exactly one reference image is supplied.
- Reference image 1 is the Creating Reference (the user's subject or concept reference).
- %s
- No project Style Reference is supplied.
%s`, creatingRole, conflictRule)
	default:
		return fmt.Sprintf(`- No reference images are supplied. Generate the requested %s from the user creative brief, selected perspective, pipeline processing requirements, and default production guidelines.
- Do not assume that any unstated Project Reference, Creating Reference, or Nexus Reference is available.
%s`, assetKind, conflictRule)
	}
}
