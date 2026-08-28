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

// AnimationImageDerivationOptions describes a single-reference image edit.
// The reference composite contains the target prototype above the complete
// source-direction action sheet.
type AnimationImageDerivationOptions struct {
	Description       string
	Style             string
	Action            string
	TargetOrientation string
	SourceOrientation string
	FrameCount        int
	Columns           int
	Rows              int
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
	sourceOrientation := strings.TrimSpace(options.SourceOrientation)
	if sourceOrientation == "" {
		sourceOrientation = "the source orientation shown in Image 2"
	}

	var builder strings.Builder
	builder.WriteString("LOCKED STATIC CAMERA, 2D ORTHOGRAPHIC GAME SPRITE FRAMING, NO ZOOM, NO PAN.\n")
	builder.WriteString("Character stays stationary and horizontally centered for the entire animation.\n\n")

	builder.WriteString("CRITICAL ORIENTATION LOCK:\n")
	builder.WriteString("- FRAME 1 MUST reproduce Image 1: same character, equipment, scale, root position, pose, and facing direction.\n")
	fmt.Fprintf(&builder, "- Maintain Image 1's exact facing direction (%s) in EVERY frame.\n", targetOrientation)
	builder.WriteString("- NO TURNING, ROTATING, MIRRORING, side view, three-quarter view, or opposite view.\n\n")

	builder.WriteString("MULTI-REFERENCE CONTRACT:\n")
	builder.WriteString("- Image 1 alone defines TARGET CHARACTER APPEARANCE, opening pose, scale, position, and TARGET FACING ORIENTATION.\n")
	fmt.Fprintf(&builder, "- Image 2 is an animation frame sheet viewed from %s. Read its cells chronologically from left to right, then top to bottom.\n", sourceOrientation)
	builder.WriteString("- Use Image 2 ONLY for SOURCE ACTION PHASES, MOTION AMPLITUDE, VFX SHAPE/SCALE, and CYCLE TIMING.\n")
	builder.WriteString("- NEVER copy Image 2's character orientation, camera, position, identity, sheet/grid layout, multiple copies, or empty cells.\n\n")

	builder.WriteString("TASK DIRECTIVE:\n")
	if action != "" {
		fmt.Fprintf(&builder, "- Animate the character from Image 1 performing the %q action.\n", action)
	} else {
		builder.WriteString("- Animate the character from Image 1 performing the reference action from Image 2.\n")
	}
	builder.WriteString("- Retarget Image 2's movement, limb arcs, and VFX along the target facing axis.\n")
	builder.WriteString("- Match Image 2's effect proportions unless the available canvas is smaller; the canvas boundary rule below always wins.\n")
	builder.WriteString("- Keep the entire subject and every particle/effect inside the canvas with a clear matte safety band at every edge; shorten and contain the effect rather than touching or crossing an edge.\n")
	builder.WriteString("- When shortening an effect for containment, compress ONLY its longitudinal reach along the target facing axis. Preserve its lateral width, particle density, opacity, texture, turbulence, timing, terminal burst/splash, and overall visual intensity; do not uniformly scale down or weaken the effect.\n")
	builder.WriteString("- Complete one cycle: exact Image 1 rest pose, preparation, climax, recovery, then the same rest pose and orientation.\n")
	if options.FrameCount > 0 {
		fmt.Fprintf(&builder, "- Preserve enough distinct action phases for extraction into exactly %d ordered animation frames.\n", options.FrameCount)
	}
	if options.FrameWidth > 0 && options.FrameHeight > 0 {
		fmt.Fprintf(&builder, "- Compose for a %dx%d output frame without changing the Image 1 character scale.\n", options.FrameWidth, options.FrameHeight)
	}
	builder.WriteString("\n")

	if description != "" {
		fmt.Fprintf(&builder, "CHARACTER CONTEXT:\n- %s\n\n", description)
	}

	fmt.Fprintf(&builder, "RENDER STYLE:\n- %s\n- Solid chroma green background (#00FF00). Crisp pixel art game sprite asset.", style)

	return limit(builder.String(), MaxAnimationVideoCharacters)
}

// BuildAnimationDerivationImage asks an image editing model to preserve the
// source sequence cell-for-cell while redrawing it in the target prototype's
// facing direction. The output contract deliberately excludes the prototype
// header from the returned sheet.
func BuildAnimationDerivationImage(options AnimationImageDerivationOptions) string {
	style := limit(options.Style, maxAnimationStyleLength)
	if style == "" {
		style = DefaultAnimationStyle
	}
	targetOrientation := strings.TrimSpace(options.TargetOrientation)
	if targetOrientation == "" {
		targetOrientation = "the target direction shown in the top prototype panel"
	}
	sourceOrientation := strings.TrimSpace(options.SourceOrientation)
	if sourceOrientation == "" {
		sourceOrientation = "the source direction shown in the lower action sheet"
	}
	columns, rows := max(options.Columns, 1), max(options.Rows, 1)

	var builder strings.Builder
	builder.WriteString("EDIT THE SUPPLIED COMPOSITE REFERENCE INTO ONE PRODUCTION ANIMATION FRAME SHEET.\n\n")
	builder.WriteString("REFERENCE COMPOSITE:\n")
	fmt.Fprintf(&builder, "- The TOP panel is the authoritative character/object prototype for %s.\n", targetOrientation)
	fmt.Fprintf(&builder, "- The LOWER panel is a chronological %dx%d action sheet viewed from %s; read left to right, then top to bottom.\n", columns, rows, sourceOrientation)
	builder.WriteString("- Use the top panel for identity, equipment, proportions, colours, scale, and facing direction.\n")
	builder.WriteString("- Use the lower panel only for pose phases, movement arcs, timing, effects, and per-frame placement.\n\n")
	builder.WriteString("REDRAW RULES:\n")
	fmt.Fprintf(&builder, "- Redraw every source action cell in %s without mirroring symbols, asymmetric equipment, clothing, or markings.\n", targetOrientation)
	builder.WriteString("- Preserve exact chronological phase correspondence: output cell N must represent source cell N.\n")
	if action := limit(options.Action, maxAnimationActionLength); action != "" {
		fmt.Fprintf(&builder, "- Preserve the complete %q action and its recovery into the loop pose.\n", action)
	}
	if description := limit(options.Description, maxAnimationDescriptionLength); description != "" {
		fmt.Fprintf(&builder, "- Subject context: %s.\n", description)
	}
	builder.WriteString("- Keep the subject root position, apparent scale, animation timing, and effects consistent across cells.\n")
	builder.WriteString("- Keep every subject and effect fully inside its own cell; never cross a grid boundary.\n\n")
	builder.WriteString("OUTPUT CONTRACT:\n")
	fmt.Fprintf(&builder, "- Output ONLY one %dx%d grid containing exactly %d chronological frames.\n", columns, rows, options.FrameCount)
	if options.FrameWidth > 0 && options.FrameHeight > 0 {
		fmt.Fprintf(&builder, "- Each logical cell represents a %dx%d frame.\n", options.FrameWidth, options.FrameHeight)
	}
	builder.WriteString("- Do not output the top prototype panel, labels, guides, borders, captions, or extra cells.\n")
	builder.WriteString("- Use a flat solid chroma-green (#00FF00) background in every cell.\n")
	fmt.Fprintf(&builder, "- Render style: %s. Crisp pixel-art game sprite asset.", style)
	return builder.String()
}
