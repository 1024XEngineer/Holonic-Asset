package prompts

import (
	"fmt"
	"strings"
)

// AnimationDerivationOptions specifies the universal parameters for deriving
// an animation sequence from a source animation into a target orientation.
type AnimationDerivationOptions struct {
	Description       string
	Style             string
	Action            string
	TargetOrientation string
	SourceOrientation string
	FrameCount        int
	FrameWidth        int
	FrameHeight       int
}

// BuildAnimationDerivationVideo builds the universal multi-reference prompt.
// It enforces:
// 1. Static orthographic camera framing and stationary horizontal character centering.
// 2. Strict orientation locking matching Image 1 throughout all frames (no side turning).
// 3. Action dynamics, cycle timing, and particle VFX scale matching Image 2 retargeted along the target facing axis.
func BuildAnimationDerivationVideo(options AnimationDerivationOptions) string {
	description := limit(options.Description, maxAnimationDescriptionLength)
	style := limit(options.Style, maxAnimationStyleLength)
	action := limit(options.Action, maxAnimationActionLength)
	if style == "" {
		style = DefaultAnimationStyle
	}
	targetOrientation := strings.TrimSpace(options.TargetOrientation)
	if targetOrientation == "" {
		targetOrientation = "the target orientation shown in Image 1"
	}

	var builder strings.Builder
	builder.WriteString("LOCKED STATIC CAMERA, 2D ORTHOGRAPHIC GAME SPRITE FRAMING, NO ZOOM, NO PAN.\n")
	builder.WriteString("The main character MUST remain stationary and positioned squarely at the horizontal center of the frame throughout the entire animation.\n\n")

	builder.WriteString("CRITICAL ORIENTATION LOCK:\n")
	builder.WriteString(fmt.Sprintf("- The character MUST STRICTLY MAINTAIN the exact facing direction of Image 1 (%s) across ALL frames from start to finish.\n", targetOrientation))
	builder.WriteString("- ABSOLUTELY NO TURNING, NO ROTATING, AND NO CHANGING ORIENTATION TO SIDES OR OPPOSITE VIEWS.\n\n")

	builder.WriteString("MULTI-REFERENCE CONTRACT:\n")
	builder.WriteString("- Image 1 defines the TARGET CHARACTER APPEARANCE and TARGET FACING ORIENTATION.\n")
	builder.WriteString("- Image 2 defines the SOURCE ACTION SEQUENCE, MOTION AMPLITUDE, PARTICLE VFX SCALE, and CYCLE TIMING.\n\n")

	builder.WriteString("TASK DIRECTIVE:\n")
	if action != "" {
		builder.WriteString(fmt.Sprintf("- Animate the character from Image 1 performing the %q action.\n", action))
	} else {
		builder.WriteString("- Animate the character from Image 1 performing the reference action from Image 2.\n")
	}
	builder.WriteString("- Retarget all kinetic movements, limb arcs, and visual particle effects from Image 2 directly along the target facing axis.\n")
	builder.WriteString("- The visual effect scale, particle density, and motion amplitude must strictly match the proportions demonstrated in Image 2.\n")
	builder.WriteString("- Execute a complete coherent cycle: start at resting stance, execute preparation, climax, and complete recovery back to the resting pose.\n\n")

	if description != "" {
		builder.WriteString(fmt.Sprintf("CHARACTER CONTEXT:\n- %s\n\n", description))
	}

	builder.WriteString(fmt.Sprintf("RENDER STYLE:\n- %s\n- Solid chroma green background (#00FF00). Crisp pixel art game sprite asset.", style))

	return limit(builder.String(), MaxAnimationVideoCharacters)
}
