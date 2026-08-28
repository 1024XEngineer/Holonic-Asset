package generator

import (
	"fmt"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
)

func normalizeAnimationGenerationRequest(
	request *AnimationGenerationRequest,
) (AnimationGenerationRequest, error) {
	if request == nil {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation generation request is required")
	}
	value := *request
	value.Description = strings.TrimSpace(value.Description)
	value.Style = strings.TrimSpace(value.Style)
	value.Action = strings.TrimSpace(value.Action)
	value.OriginalAction = strings.TrimSpace(value.OriginalAction)
	value.TargetFrameIndices = append([]int(nil), value.TargetFrameIndices...)
	value.ContextReferenceImages = append([]string(nil), value.ContextReferenceImages...)
	for index := range value.ContextReferenceImages {
		value.ContextReferenceImages[index] = strings.TrimSpace(value.ContextReferenceImages[index])
	}
	value.ReferenceImage = strings.TrimSpace(value.ReferenceImage)
	value.DerivationSourceImage = strings.TrimSpace(value.DerivationSourceImage)
	value.TargetOrientation = strings.TrimSpace(value.TargetOrientation)
	value.SourceOrientation = strings.TrimSpace(value.SourceOrientation)
	value.EndReferenceImage = strings.TrimSpace(value.EndReferenceImage)
	value.Resolution = strings.TrimSpace(value.Resolution)
	value.AspectRatio = strings.TrimSpace(value.AspectRatio)
	if value.Description == "" {
		value.Description = "preserve the supplied subject exactly"
	}
	if value.Style == "" {
		value.Style = prompts.DefaultAnimationStyle
	}
	if value.Action == "" {
		value.Action = "idle"
	}
	if value.ReferenceImage == "" {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation reference image is required")
	}
	if value.FrameCount == 0 {
		value.FrameCount = defaultAnimationFrameCount
	}
	if value.Columns == 0 {
		value.Columns = animationGridColumns(value.FrameCount)
	}
	if value.FrameWidth == 0 {
		value.FrameWidth = defaultAnimationFrameWidth
	}
	if value.FrameHeight == 0 {
		value.FrameHeight = defaultAnimationFrameHeight
	}
	if value.FPS == 0 {
		value.FPS = defaultAnimationFPS
	}
	if value.Resolution == "" {
		value.Resolution = defaultAnimationResolution
	}
	if value.Duration == 0 {
		value.Duration = defaultAnimationDuration
	}
	if value.AspectRatio == "" {
		value.AspectRatio = defaultAnimationAspectRatio
	}
	if value.FrameCount < 1 || value.FrameCount > 32 {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation frame count must be between 1 and 32")
	}
	if value.Columns < 1 || value.Columns > 8 {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation columns must be between 1 and 8")
	}
	if animationRows(value.FrameCount, value.Columns) > 8 {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation grid must not exceed 8 rows")
	}
	if value.FrameWidth < 32 || value.FrameWidth > 1024 || value.FrameHeight < 32 || value.FrameHeight > 1024 {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation frame dimensions must be between 32 and 1024 pixels")
	}
	if value.FPS < 1 || value.FPS > 60 {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation FPS must be between 1 and 60")
	}
	if value.Duration < 4 || value.Duration > 15 {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation video duration must be between 4 and 15 seconds")
	}
	if len(value.TargetFrameIndices) > 0 && !value.ReferenceImageContext {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: target frame indices require an animation context reference")
	}
	previousTarget := -1
	for _, target := range value.TargetFrameIndices {
		if target < 0 || target >= value.FrameCount {
			return AnimationGenerationRequest{}, fmt.Errorf("generator: target frame index %d is outside the %d-frame context", target, value.FrameCount)
		}
		if target <= previousTarget {
			return AnimationGenerationRequest{}, fmt.Errorf("generator: target frame indices must be unique and ordered")
		}
		previousTarget = target
	}
	if value.ReferenceImageContext && value.EndReferenceImage == "" {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation frame edit end reference image is required")
	}
	if value.ReferenceImageContext && len(value.ContextReferenceImages) != value.FrameCount {
		return AnimationGenerationRequest{}, fmt.Errorf(
			"generator: animation frame edit requires %d context reference images; got %d",
			value.FrameCount,
			len(value.ContextReferenceImages),
		)
	}
	if value.ReferenceImageContext && value.DerivationSourceImage != "" {
		return AnimationGenerationRequest{}, fmt.Errorf("generator: animation frame editing and direction derivation cannot be combined")
	}
	for index, reference := range value.ContextReferenceImages {
		if reference == "" {
			return AnimationGenerationRequest{}, fmt.Errorf("generator: animation frame edit context reference image %d is required", index+1)
		}
	}
	return value, nil
}
