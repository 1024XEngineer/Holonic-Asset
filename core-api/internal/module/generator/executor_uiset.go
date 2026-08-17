package generator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
)

func (e *executor) generateUISet(
	ctx context.Context,
	payload CreateUISetPayload,
) (json.RawMessage, error) {
	if _, err := e.planUISetComponents(ctx, payload); err != nil {
		return nil, err
	}
	return nil, nil // Component image generation starts in UI Set phase 3.
}

func (e *executor) planUISetComponents(
	ctx context.Context,
	payload CreateUISetPayload,
) ([]UISetComponentPlan, error) {
	if e.llm == nil {
		return nil, ErrLLMServiceRequired
	}
	if err := validateCreateUISetPayload(&payload); err != nil {
		return nil, err
	}
	components := make([]prompts.UISetComponentInput, len(payload.Components))
	for index, component := range payload.Components {
		components[index] = prompts.UISetComponentInput{
			Index: index, Name: component.Name, Description: component.Description,
		}
	}
	prompt := prompts.UISetPlan(prompts.UISetPlanInput{
		AssetName: payload.AssetName, CreativeBrief: payload.CreativeBrief, Style: payload.Style,
		ProjectName: payload.ProjectContext.Name, GameType: payload.ProjectContext.GameType,
		TargetPlatform: payload.ProjectContext.TargetPlatform, ProjectDescription: payload.ProjectContext.Description,
		Width: payload.Dimensions.Width, Height: payload.Dimensions.Height, Components: components,
	})
	completion, err := e.llm.Complete(ctx, &llmclient.CompletionRequest{
		Prompt: prompt,
		ResponseSchema: llmclient.JSONSchema{
			Name:   uiSetComponentPlanSchemaName,
			Schema: append(json.RawMessage(nil), uiSetComponentPlanJSONSchema...),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("generator: plan UI Set components: %w", err)
	}
	if completion == nil {
		return nil, fmt.Errorf("%w: LLM returned no completion", ErrInvalidUISetPlan)
	}
	return decodeUISetComponentPlan(completion.JSON, payload.Components, payload.Dimensions)
}
