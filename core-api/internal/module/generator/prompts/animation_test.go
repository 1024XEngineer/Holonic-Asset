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
		"exactly ONE isolated canonical character view",
		"exactly ONE complete character",
		"never show multiple directions, multiple poses",
		"the system will extract 16 ordered frames later; do not render those frames as a sheet",
		"extracted by the backend from the high-resolution direction sheet",
		"never turn, mirror, switch views",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt does not contain %q:\n%s", expected, prompt)
		}
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
