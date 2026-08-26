package prompts

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	DefaultAnimationStyle         = "finely drawn production-quality 2D game asset art"
	MaxAnimationVideoCharacters   = 2450
	maxAnimationBaseCharacters    = 1700
	maxAnimationDescriptionLength = 240
	maxAnimationStyleLength       = 180
	maxAnimationActionLength      = 320
)

type AnimationOptions struct {
	Description        string
	Style              string
	Action             string
	OriginalAction     string
	FrameCount         int
	PrototypeWidth     int
	PrototypeHeight    int
	FrameWidth         int
	FrameHeight        int
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
	originalAction := limit(options.OriginalAction, maxAnimationActionLength)
	framingInstructions := animationFramingInstructions(options)
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
		originalActionInstruction := "- the stored original action is unavailable; preserve every pre-existing motion implied by the user's edit wording and the boundary poses"
		if originalAction != "" {
			originalActionInstruction = fmt.Sprintf("- ORIGINAL ACTION — MUST BE PRESERVED: %s", originalAction)
		}
		referenceInstructions = `BOUNDARY FRAME REFERENCES:
- start/end inputs are the original unprocessed frames immediately outside the selected interval, clamped at the animation start or end when necessary
- generate one normal full-frame video that matches the start frame exactly and arrives at the end frame exactly
- inputs are separate full-frame anchors, never a contact sheet, collage, grid, storyboard, multi-frame canvas, or spritesheet
- preserve identity, proportions, details, materials, palette, orientation, scale, camera, and root position at both boundaries`
		actionInstructions = fmt.Sprintf(`LOCAL FRAME EDIT — TARGET OUTPUT SAMPLES: %s (1-based positions out of %d):
%s
- begin the change by the first target, keep it readable across most target samples, and complete it by the last
- one continuous chronological take; no restart, montage, unrelated motion, or phase reordering; boundary images may omit an internal action extreme
- ADDITIVE EDIT: keep the pre-existing action recognizable while performing the requested change; never replace it
- PRIMARY REQUIREMENT: the requested change must be unmistakably visible; copying original target pixels or making only a token change is invalid
- retain the original action's identity and principal phase/extreme, but allow local pose, path, and timing adjustments needed to show the change clearly
- non-target samples are seam context only: match entry/exit smoothly, but do not force target poses to resemble the originals`, targetFrames, options.FrameCount, originalActionInstruction)
	}
	sections := fmt.Sprintf("%s\n\n%s", referenceInstructions, actionInstructions)
	if options.LocalFrameEdit {
		// Keep the edit requirements before the boundary-reference details so the
		// provider limit cannot truncate the requested change instructions. A
		// single-sample edit prioritizes the boundary anchors instead.
		if len(options.TargetFrameIndices) == 1 {
			sections = fmt.Sprintf("%s\n\n%s", referenceInstructions, actionInstructions)
		} else {
			sections = fmt.Sprintf("%s\n\n%s", actionInstructions, referenceInstructions)
		}
	}
	prompt := strings.TrimSpace(fmt.Sprintf(`Create one normal single-subject in-place 2D game asset animation video.

CRITICAL OUTPUT FORMAT — NOT A SPRITESHEET:
- every frame contains exactly ONE complete subject and its attached or held props; keep spray, projectiles, particles, trails, glow, and shadows inside a visible matte edge
- normal full-frame video; never a contact sheet, collage, grid, storyboard, spritesheet, multiple views, poses, or copies
- fixed camera/root on uniform chroma: pure green #00FF00 by default; pure magenta #FF00FF only when the subject contains substantial colours close to pure green
%s
- preserve original colours and saturation; never recolour, desaturate, gray out, or remove green subject pixels
- matte only in the background; never use its exact colour inside or over the subject

USER SPECIFICATION — AUTHORITATIVE:
- subject: %s
- style: %s
- requested action: %s
- the system will extract %d ordered frames later; do not render those frames as a sheet

%s

CONTINUITY:
- preserve silhouette, proportions, details, materials, equipment, and attached parts
- fixed root/camera: no sliding, turning, warping, squash, stretch, scale pulsing, or detached parts

FRAMING AND BACKGROUND:
- fixed camera: no pan, tilt, zoom, shake, crop, reframe, or cut
- keep the whole subject, long parts, weapons, tool tips, and every visible effect within the available canvas; this includes projectiles, liquid, spray, particles, trails, glow, and shadows
- keep at least a thin continuous matte line visible at every canvas edge in every frame; shorten the motion or effect rather than resizing the subject or reaching an edge
- background is perfectly uniform in the selected matte; never mix mattes`,
		framingInstructions, description, style, action, options.FrameCount, sections))
	return limit(prompt, MaxAnimationVideoCharacters)
}

func animationFramingInstructions(options AnimationOptions) string {
	if options.PrototypeWidth > 0 && options.PrototypeHeight > 0 && options.FrameWidth > 0 && options.FrameHeight > 0 {
		return fmt.Sprintf("- SCALE: prototype %dx%d -> frame %dx%d; keep reference scale/root exactly; matte border is movement room, never resize the subject",
			options.PrototypeWidth,
			options.PrototypeHeight,
			options.FrameWidth,
			options.FrameHeight,
		)
	}
	return "- SCALE: keep full-frame reference scale/root exactly; matte border is movement room, never resize the subject to force margins"
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
- preserve the exact subject scale, root placement, and existing matte border shown in the supplied reference; never zoom, shrink, crop, or reframe
- keep the complete subject, every attached or held object, and every visible effect within the available frame throughout the entire video
- use the existing matte border as movement room; never invent an arbitrary percentage margin by resizing the subject
- keep at least a thin continuous matte line visible at every canvas edge in every frame
- keep long parts, weapons, tool tips, projectiles, liquid, spray, particles, trails, glow, and shadows within the available canvas; shorten or reduce the motion/effect instead of reaching an edge`
	if issueKind == "subject" || issueKind == "foreground" {
		correction = `Generate a fresh take; the previous take lost the readable subject silhouette.
- output exactly one subject in every frame; never reproduce a multi-direction reference sheet or show multiple views
- preserve the exact subject, attached parts, and opaque silhouette in every frame
- keep the background uniformly pure green #00FF00 by default; use pure magenta #FF00FF only when the subject contains substantial green, with strong colour separation
- do not fade, dissolve, blur away, or merge any body or object part into the background`
	}
	if issueKind == "continuity" {
		correction = `Generate a fresh take; the previous local edit created an abrupt motion discontinuity.
- match the supplied start and end boundary poses at the two seams; target samples do not need to copy their original pixels or exact poses
- keep the pre-existing action recognizable, including its principal phase or extreme, while giving the requested change enough pose freedom to be obvious
- smooth only the entry and exit of the requested change; never shrink, hide, or remove that change merely to resemble the original target frames
- allow no sudden root displacement, scale jump, silhouette pop, or isolated one-frame spike at the edit boundaries
- the requested change remains the primary requirement throughout the target interval`
	}
	if issueKind == "motion_preservation" {
		correction = `Generate a fresh take; the previous local edit erased or weakened motion that already existed in the selected interval.
- restore enough of the pre-existing action to keep its identity and principal phase or extreme recognizable
- treat the requested change as an additive simultaneous layer, never as a replacement for the original action
- local pose, path, and timing adjustments are allowed when needed to make the requested change clearly visible
- do not solve preservation by copying the original target frames or weakening the requested change`
	}
	if issueKind == "temporal_coherence" {
		correction = `Generate exactly one continuous chronological action interval; the previous local edit mixed or reordered motion phases.
- no repeated take, restart, alternate pose sequence, pose montage, unrelated motion, or phase reversal
- keep the pre-existing action recognizable and retain its chronological phase order
- layer the requested change simultaneously onto that action instead of starting a second action
- allow the requested change to depart visibly from the original target poses while progressing smoothly once from the supplied start boundary to the end boundary`
	}
	if issueKind == "edit_application" {
		correction = `Generate a fresh take; the previous local edit reproduced the original animation but failed to visibly perform the requested addition.
- the requested change is mandatory, not optional; make it unmistakably readable across most target samples
- keep the pre-existing action recognizable instead of replacing it, but do not prioritize pixel similarity over the requested change
- visibly transform the exact subject part, object, pose, or effect named by the user specification
- use a clear pose difference, not a token movement; do not return the unedited original motion, a barely changed pose, or a change hidden outside the target interval`
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
