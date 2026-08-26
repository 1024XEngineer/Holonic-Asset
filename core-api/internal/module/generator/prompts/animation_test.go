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
		"keep full-frame reference scale/root exactly",
		"matte border is movement room",
		"never resize the subject to force margins",
		"spray, projectiles, particles, trails, glow, and shadows",
		"inside a visible matte edge",
		"pure green #00FF00 by default",
		"pure magenta #FF00FF only when the subject contains substantial colours close to pure green",
		"never recolour, desaturate, gray out, or remove green subject pixels",
		"exactly ONE isolated canonical subject view",
		"exactly ONE complete subject",
		"spritesheet, multiple views, poses, or copies",
		"the system will extract 16 ordered frames later; do not render those frames as a sheet",
		"from the high-resolution prototype or direction sheet",
		"never turn, mirror, switch views",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt does not contain %q:\n%s", expected, prompt)
		}
	}
}

func TestBuildAnimationVideoDescribesIndependentFrameCanvasWithoutRescaling(t *testing.T) {
	prompt := BuildAnimationVideo(AnimationOptions{
		Description:     "hero",
		Action:          "wide sword swing",
		FrameCount:      8,
		PrototypeWidth:  32,
		PrototypeHeight: 48,
		FrameWidth:      64,
		FrameHeight:     72,
	})
	for _, expected := range []string{
		"prototype 32x48 -> frame 64x72",
		"keep reference scale/root exactly",
		"matte border is movement room",
		"never resize the subject",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt does not contain %q:\n%s", expected, prompt)
		}
	}
	for _, unexpected := range []string{"inner 70%", "at least 15%", "inner 64%", "at least 18%"} {
		if strings.Contains(prompt, unexpected) {
			t.Fatalf("prompt still imposes fixed framing %q:\n%s", unexpected, prompt)
		}
	}
}

func TestAnimationVideoFramingRetryPreservesConfiguredSubjectScale(t *testing.T) {
	retry := BuildAnimationVideoRetry("base", "framing")
	for _, expected := range []string{
		"preserve the exact subject scale, root placement, and existing matte border",
		"use the existing matte border as movement room",
		"never invent an arbitrary percentage margin by resizing the subject",
		"projectiles, liquid, spray, particles, trails, glow, and shadows",
		"thin continuous matte line visible at every canvas edge",
	} {
		if !strings.Contains(retry, expected) {
			t.Fatalf("framing retry does not contain %q: %s", expected, retry)
		}
	}
	for _, unexpected := range []string{"inner 64%", "at least 18%", "preserve the smaller scale"} {
		if strings.Contains(retry, unexpected) {
			t.Fatalf("framing retry still imposes fixed framing %q: %s", unexpected, retry)
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
		"start/end inputs are the original unprocessed frames immediately outside",
		"clamped at the animation start or end",
		"matches the start frame exactly",
		"arrives at the end frame exactly",
		"never a contact sheet",
		"LOCAL FRAME EDIT",
		"TARGET OUTPUT SAMPLES: 2, 4",
		"one continuous chronological take",
		"no restart, montage, unrelated motion",
		"ADDITIVE EDIT",
		"PRIMARY REQUIREMENT",
		"requested change must be unmistakably visible",
		"copying original target pixels or making only a token change is invalid",
		"allow local pose, path, and timing adjustments",
		"readable across most target samples",
		"non-target samples are seam context only",
		"do not force target poses to resemble the originals",
		"boundary images may omit an internal action extreme",
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
		"clamped at the animation start or end when necessary",
		"inputs are separate full-frame anchors",
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

func TestBuildAnimationVideoPreservesStoredOriginalActionDuringLocalEdit(t *testing.T) {
	prompt := BuildAnimationVideo(AnimationOptions{
		Description:        "greeter",
		Action:             "put the other hand near the mouth in a shush gesture",
		OriginalAction:     "raise the hat in greeting",
		FrameCount:         10,
		LocalFrameEdit:     true,
		TargetFrameIndices: []int{1, 2, 3, 4, 5, 6, 7, 8},
	})
	for _, expected := range []string{
		"ORIGINAL ACTION — MUST BE PRESERVED: raise the hat in greeting",
		"keep the pre-existing action recognizable",
		"principal phase/extreme",
		"allow local pose, path, and timing adjustments",
		"boundary images may omit an internal action extreme",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("local edit prompt does not contain %q:\n%s", expected, prompt)
		}
	}
}

func TestAnimationVideoRetryRequiresMissingEditToBecomeVisible(t *testing.T) {
	retry := BuildAnimationVideoRetry("base", "edit_application")
	for _, expected := range []string{
		"failed to visibly perform the requested addition",
		"mandatory, not optional",
		"across most target samples",
		"do not prioritize pixel similarity over the requested change",
		"exact subject part, object, pose, or effect named by the user specification",
		"do not return the unedited original motion",
	} {
		if !strings.Contains(retry, expected) {
			t.Fatalf("edit application retry does not contain %q: %s", expected, retry)
		}
	}
}

func TestAnimationVideoRetryRequiresOneChronologicalActionInterval(t *testing.T) {
	retry := BuildAnimationVideoRetry("base", "temporal_coherence")
	for _, expected := range []string{
		"one continuous chronological action interval",
		"no repeated take, restart",
		"chronological phase order",
		"layer the requested change simultaneously",
		"depart visibly from the original target poses",
	} {
		if !strings.Contains(retry, expected) {
			t.Fatalf("temporal coherence retry does not contain %q: %s", expected, retry)
		}
	}
}

func TestAnimationVideoRetryDoesNotSuppressRequestedChange(t *testing.T) {
	expectedByKind := map[string][]string{
		"continuity":          {"target samples do not need to copy", "enough pose freedom", "never shrink, hide, or remove that change"},
		"motion_preservation": {"keep its identity", "local pose, path, and timing adjustments are allowed", "weakening the requested change"},
	}
	for issueKind, expectedValues := range expectedByKind {
		retry := BuildAnimationVideoRetry("base", issueKind)
		for _, expected := range expectedValues {
			if !strings.Contains(retry, expected) {
				t.Fatalf("%s retry does not contain %q: %s", issueKind, expected, retry)
			}
		}
		for _, forbidden := range []string{"full amplitude", "exact trajectory", "copy the original target frames"} {
			if strings.Contains(retry, forbidden) {
				t.Fatalf("%s retry still over-constrains the requested change with %q: %s", issueKind, forbidden, retry)
			}
		}
	}
}

func TestLimitEdgeCases(t *testing.T) {
	if got := limit("some text", 0); got != "" {
		t.Fatalf("expected empty string for maxCharacters=0, got %q", got)
	}
	if got := limit("some text", -5); got != "" {
		t.Fatalf("expected empty string for maxCharacters=-5, got %q", got)
	}
	if got := limit("abc", 1); got != "a" {
		t.Fatalf("expected 'a', got %q", got)
	}
	if got := limit("第一句；第二句！第三句？第四句。", 10); !strings.HasSuffix(got, "…") {
		t.Fatalf("expected punctuation boundary suffix, got %q", got)
	}
}
