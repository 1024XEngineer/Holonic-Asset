package prompts

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	DefaultAnimationStyle         = "finely drawn production-quality 2D game character art"
	MaxAnimationVideoCharacters   = 2450
	maxAnimationBaseCharacters    = 2000
	maxAnimationDescriptionLength = 240
	maxAnimationStyleLength       = 180
	maxAnimationActionLength      = 320
)

type AnimationOptions struct {
	Description string
	Style       string
	Action      string
	FrameCount  int
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
	prompt := strings.TrimSpace(fmt.Sprintf(`Create one normal single-subject in-place 2D character animation video from the supplied reference image.

CRITICAL OUTPUT FORMAT — NOT A SPRITESHEET:
- every frame contains exactly ONE complete character and one copy of each held prop
- normal full-frame video, never a contact sheet, turnaround, collage, grid, storyboard, or spritesheet
- never show multiple directions, multiple poses, copies, cells, borders, labels, or the reference layout

USER SPECIFICATION — AUTHORITATIVE:
- character: %s
- style: %s
- requested action: %s
- the system will extract %d ordered frames later; do not render those frames as a sheet

REFERENCE IMAGE:
- the input is exactly ONE isolated canonical character view extracted by the backend from the high-resolution direction sheet
- use that single character as identity, pose scale, and facing; never turn, mirror, switch views, or invent another direction
- preserve identity, proportions, clothing, equipment, palette, and art style

ACTION:
- interpret the requested action by its actual meaning; do not map it to a generic motion preset
- show one complete cycle from the initial pose back to the same pose
- include preparation, every semantically required intermediate stage, the main extreme, complete follow-through and recovery
- strict temporal order: begin from the supplied initial pose, perform preparation before the main action, follow through before recovery, and end only after returning to that same pose
- do not start in the middle, reverse the action, jump between phases, repeat the main action before recovery, omit late stages, or freeze at the main pose

CONTINUITY:
- preserve face, hair, outfit, silhouette, proportions, equipment, and prop geometry
- fixed root and camera: no sliding, turning, warping, squash, stretch, scale pulsing, or detached props

FRAMING AND BACKGROUND:
- fixed camera: no pan, tilt, zoom, shake, crop, reframe, perspective change, or cut
- keep the whole character and every held prop inside the inner 70%%; maintain at least 15%% uninterrupted empty space on every side
- keep long weapon or tool tips inside the central area; reduce amplitude rather than reaching toward an edge
- background is perfectly uniform pure chroma green #00FF00: no floor, shadow, gradient, scenery, lighting change, particles, text, audio, trails, or motion graphics`,
		description, style, action, options.FrameCount))
	return limit(prompt, MaxAnimationVideoCharacters)
}

func BuildAnimationVideoRetry(base, issueKind string) string {
	base = limit(base, maxAnimationBaseCharacters)
	correction := `Generate a fresh take; the previous take failed framing checks.
- output exactly one character in a normal video frame; never reproduce the multi-direction reference sheet or show eight views
- keep the complete character and every held object inside the inner 64% of the frame throughout the entire video
- maintain at least 18% uninterrupted pure-green empty space on every side
- preserve the smaller scale of the supplied reference and never zoom, crop, reframe, or push any body part or object toward an edge
- keep long weapon or tool tips inside the central 64%%; use a compact controlled motion instead of a wide edge-reaching swing`
	if issueKind == "subject" {
		correction = `Generate a fresh take; the previous take lost the readable subject silhouette.
- output exactly one character in every frame; never reproduce the multi-direction reference sheet or show eight views
- preserve the exact character, held objects, and opaque silhouette in every frame
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
