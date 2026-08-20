package prompts

import (
	"fmt"
	"strings"
)

// PrototypeReferenceState describes which backend-ordered reference roles are
// present in a prototype generation request.
type PrototypeReferenceState struct {
	HasProjectReference bool
	HasUserReference    bool
	TagReferenceCount   int
}

func prototypeReferenceImageRoles(
	state PrototypeReferenceState,
	assetKind string,
) string {
	const conflictRule = `- If any reference image conflicts with a specific requirement in the user creative brief, follow the creative brief for that requirement.`
	const projectRole = `Use it only to establish the project's game-wide art style and visual-world conventions: pixel granularity, pixel block size, palette character, outline treatment, contrast, lighting, shading, rendering technique, material language, proportions, and perspective. Do not copy its specific subject, characters, objects, scenery, composition, text, logos, or other recognizable content.`
	const userRole = `The user-supplied reference image is always a strong reference. When the user creative brief asks to preserve, closely match, imitate, or make only specific changes to it, preserve every unmentioned visual attribute and make only the changes explicitly requested in the user creative brief.`

	if state.TagReferenceCount > 0 {
		total := state.TagReferenceCount
		if state.HasProjectReference {
			total++
		}
		if state.HasUserReference {
			total++
		}
		lines := []string{fmt.Sprintf("- Exactly %d reference images are supplied in an authoritative order.", total)}
		index := 1
		if state.HasProjectReference {
			lines = append(lines, fmt.Sprintf("- Reference image %d is the project prototype image and is the Style Reference. %s", index, projectRole))
			index++
		}
		if state.HasUserReference {
			lines = append(lines,
				fmt.Sprintf("- Reference image %d is the user-supplied reference image.", index),
				"- "+userRole,
			)
			index++
		}
		for tagIndex := range state.TagReferenceCount {
			if !state.HasProjectReference && tagIndex == 0 {
				lines = append(lines, fmt.Sprintf(
					"- Reference image %d is the highest-ranked same-project Tag asset and is promoted to the Style Reference. %s",
					index,
					projectRole,
				))
			} else {
				lines = append(lines, fmt.Sprintf(
					"- Reference image %d is a same-project Tag asset. Use it only for compatible material treatment, category details, and visual-series consistency; do not copy its subject or composition.",
					index,
				))
			}
			index++
		}
		if state.HasProjectReference {
			lines = append(lines, fmt.Sprintf("- Apply the project Style Reference's visual language to the requested %s.", assetKind))
		} else if !state.HasUserReference {
			lines = append(lines, fmt.Sprintf("- Derive the requested %s from the user creative brief while following the promoted Tag Style Reference.", assetKind))
		}
		lines = append(lines, conflictRule)
		return strings.Join(lines, "\n")
	}

	switch {
	case state.HasProjectReference && state.HasUserReference:
		return fmt.Sprintf(`- Exactly two reference images are supplied in an authoritative order.
- Reference image 1 is the project prototype image and is the Style Reference. %s
- Reference image 2 is the user-supplied reference image.
- %s
- Apply the project Style Reference's visual language to the requested %s.
%s`, projectRole, userRole, assetKind, conflictRule)
	case state.HasProjectReference:
		return fmt.Sprintf(`- Exactly one reference image is supplied.
- Reference image 1 is the project prototype image and is the Style Reference. %s
- No user-supplied reference image is supplied. Derive the requested %s from the user creative brief.
%s`, projectRole, assetKind, conflictRule)
	case state.HasUserReference:
		return fmt.Sprintf(`- Exactly one reference image is supplied.
- Reference image 1 is the user-supplied reference image.
- %s
- No project Style Reference is supplied.
%s`, userRole, conflictRule)
	default:
		return fmt.Sprintf(`- No reference images are supplied. Generate the requested %s from the user creative brief, selected perspective, pipeline processing requirements, and default production guidelines.
- Do not assume that any unstated project or user reference is available.
%s`, assetKind, conflictRule)
	}
}
