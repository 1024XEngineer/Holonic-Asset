package prompts

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBuildAnimationVideoPreservesSemanticActionWithoutClassification(t *testing.T) {
	action := "以左脚为轴完成一圈不规则的仪式动作，然后将发光容器放回腰间"
	prompt := BuildAnimationVideo(AnimationOptions{
		Description: "travelling alchemist",
		Style:       "painted 2D game art",
		Action:      action,
		FrameCount:  16,
	})
	for _, expected := range []string{
		action,
		"interpret the requested action by its actual meaning",
		"every semantically required intermediate stage",
		"complete follow-through and recovery",
		"strict temporal order",
		"do not map it to a generic motion preset",
		"maintain at least 15% uninterrupted empty space",
		"perfectly uniform pure chroma green #00FF00",
		"exactly ONE isolated canonical subject view",
		"exactly ONE complete subject",
		"never show multiple directions, multiple poses",
		"the system will extract 16 ordered frames later; do not render those frames as a sheet",
		"from the high-resolution prototype or direction sheet",
		"never turn, mirror, switch views",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt does not contain %q:\n%s", expected, prompt)
		}
	}
}

func TestBuildAnimationVideoUsesBoundaryFrameReferencesForEdits(t *testing.T) {
	prompt := BuildAnimationVideo(AnimationOptions{
		Description:        "travelling alchemist",
		Style:              "painted 2D game art",
		Action:             "make the flask glow",
		FrameCount:         5,
		LocalFrameEdit:     true,
		TargetFrameIndices: []int{1, 3},
	})
	for _, expected := range []string{
		"BOUNDARY FRAME REFERENCES",
		"start input is the original unprocessed frame immediately before",
		"end input is the original unprocessed frame immediately after",
		"clamped to the animation start",
		"clamped to the animation end",
		"starts exactly from the start frame",
		"arrives exactly at the end frame",
		"never a contact sheet",
		"LOCAL FRAME EDIT",
		"do not restart the full action",
		"target samples can be inserted back into the original animation",
		"TARGET OUTPUT SAMPLES: 2, 4",
		"must be clearly visible there",
		"begins in the first target sample and completes by the last target sample",
		"non-target samples",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("context prompt does not contain %q:\n%s", expected, prompt)
		}
	}
	for _, unexpected := range []string{
		"show one complete cycle from the initial pose",
		"ordered contact sheet",
		"SINGLE-FRAME EDIT REFERENCE",
		"ordered image array",
		"@Image",
	} {
		if strings.Contains(prompt, unexpected) {
			t.Fatalf("frame edit prompt unexpectedly contains %q:\n%s", unexpected, prompt)
		}
	}
}

func TestBuildAnimationVideoDescribesClampedBoundaryFramesForSingleSample(t *testing.T) {
	prompt := BuildAnimationVideo(AnimationOptions{
		Description:        "travelling alchemist",
		Style:              "painted 2D game art",
		Action:             "make the flask glow",
		FrameCount:         1,
		LocalFrameEdit:     true,
		TargetFrameIndices: []int{0},
	})
	for _, expected := range []string{
		"BOUNDARY FRAME REFERENCES",
		"clamped to the animation start when necessary",
		"clamped to the animation end when necessary",
		"two inputs are separate full-frame boundary anchors",
		"LOCAL FRAME EDIT",
		"TARGET OUTPUT SAMPLES: 1",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("single-sample edit prompt does not contain %q:\n%s", expected, prompt)
		}
	}
	if strings.Contains(prompt, "ordered image array") || strings.Contains(prompt, "@Image") {
		t.Fatalf("single-sample edit prompt must not describe an image array:\n%s", prompt)
	}
}

func TestAnimationVideoPromptsStayInsideProviderLimit(t *testing.T) {
	longText := strings.Repeat("长动作描述", 1000)
	prompt := BuildAnimationVideo(AnimationOptions{
		Description: longText,
		Style:       longText,
		Action:      longText,
		FrameCount:  16,
	})
	if got := utf8.RuneCountInString(prompt); got > MaxAnimationVideoCharacters {
		t.Fatalf("video prompt has %d runes", got)
	}
	retry := BuildAnimationVideoRetry(prompt, "framing")
	if got := utf8.RuneCountInString(retry); got > MaxAnimationVideoCharacters {
		t.Fatalf("retry prompt has %d runes", got)
	}
}

func TestAnimationVideoRetryMapsForegroundMediaErrorToSubjectCorrection(t *testing.T) {
	retry := BuildAnimationVideoRetry("base", "foreground")
	if !strings.Contains(retry, "lost the readable subject silhouette") {
		t.Fatalf("foreground error did not select subject correction: %s", retry)
	}
}

func TestBuildAnimationVideoDefaultsContextTargetsToAllSamples(t *testing.T) {
	prompt := BuildAnimationVideo(AnimationOptions{
		Description: "hero", Action: "change pose", FrameCount: 4,
		LocalFrameEdit: true,
	})
	if !strings.Contains(prompt, "TARGET OUTPUT SAMPLES: all 4") {
		t.Fatalf("context prompt did not default to every output sample:\n%s", prompt)
	}
}
