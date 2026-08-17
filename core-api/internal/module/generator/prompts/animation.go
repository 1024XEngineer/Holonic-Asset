package prompts

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	DefaultAnimationStyle         = "finely drawn production-quality 2D game asset art"
	MaxAnimationVideoCharacters   = 2450
	maxAnimationBaseCharacters    = 2000
	maxAnimationDescriptionLength = 240
	maxAnimationStyleLength       = 180
	maxAnimationActionLength      = 320
)

type AnimationOptions struct {
	Description        string
	Style              string
	Action             string
	FrameCount         int
	LocalFrameEdit     bool
	TargetFrameIndices []int
}

// BuildAnimationVideo converts the user's semantic action specification into
// provider-facing video instructions without classifying it into a fixed set
// of action names. The video model is responsible for understanding which
// phases are required by the described action.
func BuildAnimationVideo(options AnimationOptions) string {
	description := limit(options.Description, maxAnimationDescriptionLength)
	style := limit(options.Style, maxAnimationStyleLength)
	action := limit(options.Action, maxAnimationActionLength)
	if style == "" {
		style = DefaultAnimationStyle
	}
	referenceInstructions := `REFERENCE IMAGE:
- the input is exactly ONE isolated canonical subject view from the high-resolution prototype or direction sheet
- use that subject for identity, scale, and orientation; never turn, mirror, switch views, or invent another direction
- preserve identity, proportions, details, materials, palette, and art style`
	actionInstructions := `ACTION:
- interpret the requested action by its actual meaning; do not map it to a generic motion preset
- show one complete cycle from the initial pose back to the same pose
- include preparation, every semantically required intermediate stage, the main extreme, complete follow-through and recovery
- strict temporal order: begin from the supplied initial pose, perform preparation before the main action, follow through before recovery, and end only after returning to that same pose
- do not start in the middle, reverse the action, jump between phases, repeat the main action before recovery, omit late stages, or freeze at the main pose`
	if options.LocalFrameEdit {
		targetFrames := formatAnimationTargetFrames(options.TargetFrameIndices, options.FrameCount)
		referenceInstructions = `BOUNDARY FRAME REFERENCES:
- the start input is the original unprocessed frame immediately before the selected interval, clamped to the animation start when necessary
- the end input is the original unprocessed frame immediately after the selected interval, clamped to the animation end when necessary
- generate one normal video that starts exactly from the start frame and arrives exactly at the end frame
- the two inputs are separate full-frame boundary anchors, never a contact sheet, collage, grid, storyboard, multi-frame canvas, or spritesheet
- preserve exact identity, proportions, details, materials, palette, orientation, scale, camera, and root position at both boundaries`
		actionInstructions = fmt.Sprintf(`LOCAL FRAME EDIT — TARGET OUTPUT SAMPLES: %s (1-based positions out of %d):
- the user's requested change is authoritative inside those target samples and must be clearly visible there
- schedule and compress the requested motion so it begins in the first target sample and completes by the last target sample; do not place it only before or after them
- preserve timing and pose progression in non-target samples, using them only to enter and exit the edited motion smoothly
- preserve everything not requested; do not restart the full action, invent a new cycle, subject, direction, camera, or canvas layout
- produce one smooth ordered video segment whose target samples can be inserted back into the original animation`, targetFrames, options.FrameCount)
	}
	prompt := strings.TrimSpace(fmt.Sprintf(`Create one normal single-subject in-place 2D game asset animation video from the supplied reference image inputs.

CRITICAL OUTPUT FORMAT — NOT A SPRITESHEET:
- every frame contains exactly ONE complete subject and its attached or held props
- normal full-frame video, never a contact sheet, collage, grid, storyboard, or spritesheet
- never show multiple directions, multiple poses, copies, cells, borders, labels, or the reference layout

USER SPECIFICATION — AUTHORITATIVE:
- subject: %s
- style: %s
- requested action: %s
- the system will extract %d ordered frames later; do not render those frames as a sheet

%s

%s

CONTINUITY:
- preserve silhouette, proportions, details, materials, equipment, and attached-part geometry
- fixed root and camera: no sliding, turning, warping, squash, stretch, scale pulsing, or detached parts

FRAMING AND BACKGROUND:
- fixed camera: no pan, tilt, zoom, shake, crop, reframe, or cut
- keep the whole subject and props inside the inner 70%%; maintain at least 15%% uninterrupted empty space on every side
- keep long parts, weapons, or tool tips inside the central area; reduce amplitude rather than reaching toward an edge
- background is perfectly uniform pure chroma green #00FF00: no floor, shadow, gradient, scenery, lighting change, particles, text, audio, trails, or motion graphics`,
		description, style, action, options.FrameCount, referenceInstructions, actionInstructions))
	return limit(prompt, MaxAnimationVideoCharacters)
}

func formatAnimationTargetFrames(indices []int, frameCount int) string {
	if len(indices) == 0 {
		return fmt.Sprintf("all %d", frameCount)
	}
	values := make([]string, len(indices))
	for index, frameIndex := range indices {
		values[index] = fmt.Sprintf("%d", frameIndex+1)
	}
	return strings.Join(values, ", ")
}

func BuildAnimationVideoRetry(base, issueKind string) string {
	base = limit(base, maxAnimationBaseCharacters)
	correction := `Generate a fresh take; the previous take failed framing checks.
- output exactly one subject in a normal video frame; never reproduce a multi-direction reference sheet or show multiple views
- keep the complete subject and every attached or held object inside the inner 64% of the frame throughout the entire video
- maintain at least 18% uninterrupted pure-green empty space on every side
- preserve the smaller scale of the supplied reference and never zoom, crop, reframe, or push any subject part toward an edge
- keep long parts, weapons, or tool tips inside the central 64%%; use a compact controlled motion instead of a wide edge-reaching swing`
	if issueKind == "subject" || issueKind == "foreground" {
		correction = `Generate a fresh take; the previous take lost the readable subject silhouette.
- output exactly one subject in every frame; never reproduce a multi-direction reference sheet or show multiple views
- preserve the exact subject, attached parts, and opaque silhouette in every frame
- keep the background uniformly #00FF00 with strong colour separation
- do not fade, dissolve, blur away, or merge any body or object part into the background`
	}
	return limit(strings.TrimSpace(base+"\n\nQUALITY RETRY OVERRIDE — FOLLOW THESE MORE STRICTLY:\n"+correction), MaxAnimationVideoCharacters)
}

func limit(value string, maxCharacters int) string {
	value = strings.TrimSpace(value)
	if maxCharacters <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= maxCharacters {
		return value
	}
	runes := []rune(value)
	const suffix = "…"
	cutoff := maxCharacters - utf8.RuneCountInString(suffix)
	if cutoff <= 0 {
		return string(runes[:maxCharacters])
	}
	boundary := -1
	searchStart := cutoff - cutoff/4
	for index := cutoff - 1; index >= searchStart; index-- {
		switch runes[index] {
		case '\n', '.', ';', '。', '；', '！', '？':
			boundary = index + 1
			index = -1
		}
	}
	if boundary > 0 {
		cutoff = boundary
	}
	return strings.TrimSpace(string(runes[:cutoff])) + suffix
}
