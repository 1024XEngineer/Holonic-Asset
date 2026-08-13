package generator

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func (e *executor) planSceneryLayers(
	ctx context.Context,
	payload CreateSceneryPayload,
) ([]SceneryLayerDefinition, error) {
	prompt := prompts.SceneryPlan(prompts.SceneryPlanInput{
		AssetName:          payload.AssetName,
		CreativeBrief:      payload.CreativeBrief,
		Style:              payload.Style,
		Perspective:        payload.Perspective,
		ProjectName:        payload.ProjectContext.Name,
		GameType:           payload.ProjectContext.GameType,
		TargetPlatform:     payload.ProjectContext.TargetPlatform,
		ProjectDescription: payload.ProjectContext.Description,
		Width:              payload.Dimensions.Width,
		Height:             payload.Dimensions.Height,
	})
	completion, err := e.llm.Complete(ctx, &llmclient.CompletionRequest{
		Prompt: prompt,
		ResponseSchema: llmclient.JSONSchema{
			Name:   sceneryLayerPlanSchemaName,
			Schema: append([]byte(nil), sceneryLayerPlanJSONSchema...),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("generator: plan scenery layers: %w", err)
	}
	if completion == nil {
		return nil, fmt.Errorf("%w: LLM returned no completion", ErrInvalidSceneryPlan)
	}
	return decodeSceneryLayerPlan(completion.JSON)
}

func (e *executor) generateScenery(ctx context.Context, payload CreateSceneryPayload) (json.RawMessage, error) {
	plan, err := e.planSceneryLayers(ctx, payload)
	if err != nil {
		return nil, err
	}
	layers, err := e.generateSceneryLayers(ctx, payload, plan)
	if err != nil {
		return nil, err
	}
	laidOut, err := e.analyzeSceneryLayout(ctx, payload, layers)
	if err != nil {
		return nil, err
	}
	return e.persistScenery(ctx, payload, laidOut)
}

func (e *executor) generateSceneryLayers(ctx context.Context, payload CreateSceneryPayload, plan []SceneryLayerDefinition) ([]ProcessedSceneryLayer, error) {
	references := []string(nil)
	if payload.Reference != "" {
		reference := payload.Reference
		if e.references != nil {
			resolved, err := e.references.ResolveReference(ctx, reference)
			if err != nil {
				return nil, fmt.Errorf("generator: resolve scenery reference: %w", err)
			}
			reference = resolved
		}
		references = []string{reference}
	}
	layers := make([]ProcessedSceneryLayer, 0, len(plan))
	for _, layer := range plan {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		prompt := prompts.SceneryLayer(prompts.SceneryLayerInput{AssetName: payload.AssetName, CreativeBrief: payload.CreativeBrief, Style: payload.Style, Perspective: payload.Perspective, ProjectName: payload.ProjectContext.Name, GameType: payload.ProjectContext.GameType, TargetPlatform: payload.ProjectContext.TargetPlatform, ProjectDescription: payload.ProjectContext.Description, Width: payload.Dimensions.Width, Height: payload.Dimensions.Height, LayerID: layer.ID, LayerName: layer.Name, LayerCreativeBrief: layer.CreativeBrief, HasReference: len(references) > 0}, prompts.SolidMatteBackground(imageprocessor.DefaultMatteColor))
		generated, err := e.images.Generate(ctx, &imageclient.GenerateRequest{
			Prompt:          prompt,
			ReferenceImages: append([]string(nil), references...),
			Size:            fmt.Sprintf("%dx%d", payload.Dimensions.Width, payload.Dimensions.Height),
		})
		if err != nil {
			return nil, fmt.Errorf("generator: generate scenery layer %d: %w", layer.ID, err)
		}
		if generated == nil || len(generated.Images) != 1 || strings.TrimSpace(generated.Images[0].Base64) == "" {
			return nil, fmt.Errorf("generator: generate scenery layer %d: expected exactly one image", layer.ID)
		}
		removed, err := e.processor.RemoveBackground(ctx, &imageprocessor.RemoveBackgroundRequest{ImageBase64: generated.Images[0].Base64, MatteColor: imageprocessor.DefaultMatteColor})
		if err != nil {
			return nil, fmt.Errorf("generator: remove scenery layer %d background: %w", layer.ID, err)
		}
		if removed == nil || strings.TrimSpace(removed.ImageBase64) == "" {
			return nil, fmt.Errorf("generator: remove scenery layer %d background: empty result", layer.ID)
		}
		resized, err := e.processor.Resize(ctx, &imageprocessor.ResizeRequest{
			ImageBase64: removed.ImageBase64,
			Options: imageprocessor.ResizeOptions{
				Width:       int(payload.Dimensions.Width),
				Height:      int(payload.Dimensions.Height),
				Margin:      0,
				CropContent: false,
				Mode:        imageprocessor.RasterModePixel,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("generator: resize scenery layer %d: %w", layer.ID, err)
		}
		if resized == nil || strings.TrimSpace(resized.ImageBase64) == "" || resized.MIMEType != "image/png" {
			return nil, fmt.Errorf("generator: resize scenery layer %d: invalid PNG result", layer.ID)
		}
		verified, err := e.processor.Verify(ctx, &imageprocessor.VerifyRequest{ImageBase64: resized.ImageBase64, Profile: imageprocessor.ProfileGeneric, ExpectedMatteColor: imageprocessor.DefaultMatteColor})
		if err != nil {
			return nil, fmt.Errorf("generator: verify scenery layer %d: %w", layer.ID, err)
		}
		if verified == nil || !verified.Passed {
			return nil, fmt.Errorf("generator: verify scenery layer %d failed", layer.ID)
		}
		layers = append(layers, ProcessedSceneryLayer{ID: layer.ID, Name: layer.Name, ImageBase64: resized.ImageBase64, MediaType: "image/png"})
	}
	return layers, nil
}

func (e *executor) analyzeSceneryLayout(
	ctx context.Context,
	payload CreateSceneryPayload,
	layers []ProcessedSceneryLayer,
) ([]LaidOutSceneryLayer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(layers) == 0 {
		return nil, fmt.Errorf("%w: at least one processed layer is required", ErrInvalidSceneryLayout)
	}
	promptLayers := make([]prompts.SceneryLayoutLayerInput, len(layers))
	images := make([]llmclient.ImageInput, len(layers))
	for index, layer := range layers {
		promptLayers[index] = prompts.SceneryLayoutLayerInput{ID: layer.ID, Name: layer.Name}
		images[index] = llmclient.ImageInput{URL: "data:" + layer.MediaType + ";base64," + layer.ImageBase64}
	}
	prompt := prompts.SceneryLayoutAnalysis(prompts.SceneryLayoutAnalysisInput{
		AssetName: payload.AssetName, CreativeBrief: payload.CreativeBrief, Style: payload.Style,
		Perspective: payload.Perspective, ProjectName: payload.ProjectContext.Name, GameType: payload.ProjectContext.GameType,
		TargetPlatform: payload.ProjectContext.TargetPlatform, ProjectDescription: payload.ProjectContext.Description,
		Width: payload.Dimensions.Width, Height: payload.Dimensions.Height, Layers: promptLayers,
	})
	completion, err := e.llm.Complete(ctx, &llmclient.CompletionRequest{
		Prompt:         prompt,
		Images:         images,
		ResponseSchema: llmclient.JSONSchema{Name: sceneryLayerLayoutSchemaName, Schema: append([]byte(nil), sceneryLayerLayoutJSONSchema...)},
	})
	if err != nil {
		return nil, fmt.Errorf("generator: analyze scenery layout: %w", err)
	}
	if completion == nil {
		return nil, fmt.Errorf("%w: LLM returned no completion", ErrInvalidSceneryLayout)
	}
	layouts, err := decodeSceneryLayouts(completion.JSON, layers, payload.Dimensions)
	if err != nil {
		return nil, err
	}
	result := make([]LaidOutSceneryLayer, len(layers))
	for index, layer := range layers {
		result[index] = LaidOutSceneryLayer{
			ID: layer.ID, Name: layer.Name, ImageBase64: layer.ImageBase64, MediaType: layer.MediaType, Layout: layouts[layer.ID],
		}
	}
	return result, nil
}

func decodeSceneryLayouts(raw []byte, layers []ProcessedSceneryLayer, dimensions assetdomain.Size) (map[uint]SceneryLayerLayout, error) {
	invalid := func(reason string) error { return fmt.Errorf("%w: %s", ErrInvalidSceneryLayout, reason) }
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var response sceneryLayoutResponse
	if err := decoder.Decode(&response); err != nil {
		return nil, invalid(err.Error())
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, invalid("trailing data")
		}
		return nil, invalid(err.Error())
	}
	if response.Layers == nil {
		return nil, invalid("layers is required")
	}
	if len(*response.Layers) != len(layers) {
		return nil, invalid(fmt.Sprintf("expected %d layer layouts, got %d", len(layers), len(*response.Layers)))
	}
	knownIDs := make(map[uint]struct{}, len(layers))
	for _, layer := range layers {
		if layer.ID == 0 {
			return nil, invalid("processed layer ID must be positive")
		}
		if _, exists := knownIDs[layer.ID]; exists {
			return nil, invalid(fmt.Sprintf("processed layer ID %d is duplicated", layer.ID))
		}
		knownIDs[layer.ID] = struct{}{}
	}
	layouts := make(map[uint]SceneryLayerLayout, len(layers))
	for index, candidate := range *response.Layers {
		layout, id, err := validateSceneryLayoutCandidate(candidate, dimensions)
		if err != nil {
			return nil, invalid(fmt.Sprintf("layer layout %d: %v", index+1, err))
		}
		if _, known := knownIDs[id]; !known {
			return nil, invalid(fmt.Sprintf("unknown layer ID %d", id))
		}
		if _, duplicate := layouts[id]; duplicate {
			return nil, invalid(fmt.Sprintf("layer ID %d is duplicated", id))
		}
		layouts[id] = layout
	}
	for id := range knownIDs {
		if _, present := layouts[id]; !present {
			return nil, invalid(fmt.Sprintf("layer ID %d is missing", id))
		}
	}
	return layouts, nil
}

func validateSceneryLayoutCandidate(candidate sceneryLayoutCandidate, dimensions assetdomain.Size) (SceneryLayerLayout, uint, error) {
	if candidate.ID == nil || *candidate.ID == 0 {
		return SceneryLayerLayout{}, 0, fmt.Errorf("positive id is required")
	}
	position, err := requiredLayoutVector(candidate.Position, "position")
	if err != nil {
		return SceneryLayerLayout{}, 0, err
	}
	scale, err := requiredLayoutVector(candidate.Scale, "scale")
	if err != nil {
		return SceneryLayerLayout{}, 0, err
	}
	if scale.X <= 0 || scale.Y <= 0 {
		return SceneryLayerLayout{}, 0, fmt.Errorf("scale values must be positive")
	}
	if candidate.Rotation == nil || !finite(*candidate.Rotation) {
		return SceneryLayerLayout{}, 0, fmt.Errorf("rotation is required and must be finite")
	}
	if candidate.Opacity == nil || !finite(*candidate.Opacity) || *candidate.Opacity < 0 || *candidate.Opacity > 1 {
		return SceneryLayerLayout{}, 0, fmt.Errorf("opacity is required, finite, and between 0 and 1")
	}
	if candidate.ZIndex == nil {
		return SceneryLayerLayout{}, 0, fmt.Errorf("zIndex is required")
	}
	layout := SceneryLayerLayout{
		Position: position, Scale: scale, Rotation: *candidate.Rotation, Opacity: *candidate.Opacity, ZIndex: *candidate.ZIndex,
	}
	intersects, err := transformedLayerIntersectsCanvas(layout, dimensions)
	if err != nil {
		return SceneryLayerLayout{}, 0, err
	}
	if !intersects {
		return SceneryLayerLayout{}, 0, fmt.Errorf("transformed bounds do not intersect the canvas")
	}
	return layout, *candidate.ID, nil
}

func requiredLayoutVector(candidate *sceneryLayoutVectorCandidate, name string) (SceneryLayoutVector, error) {
	if candidate == nil {
		return SceneryLayoutVector{}, fmt.Errorf("%s is required", name)
	}
	if candidate.X == nil || candidate.Y == nil {
		return SceneryLayoutVector{}, fmt.Errorf("%s x and y are required", name)
	}
	if !finite(*candidate.X) || !finite(*candidate.Y) {
		return SceneryLayoutVector{}, fmt.Errorf("%s values must be finite", name)
	}
	return SceneryLayoutVector{X: *candidate.X, Y: *candidate.Y}, nil
}

func transformedLayerIntersectsCanvas(layout SceneryLayerLayout, dimensions assetdomain.Size) (bool, error) {
	width, height := float64(dimensions.Width)*layout.Scale.X, float64(dimensions.Height)*layout.Scale.Y
	centerX, centerY := layout.Position.X+width/2, layout.Position.Y+height/2
	angle := math.Mod(layout.Rotation, 360) * math.Pi / 180
	halfWidth := math.Abs(math.Cos(angle))*width/2 + math.Abs(math.Sin(angle))*height/2
	halfHeight := math.Abs(math.Sin(angle))*width/2 + math.Abs(math.Cos(angle))*height/2
	for _, value := range []float64{width, height, centerX, centerY, halfWidth, halfHeight} {
		if !finite(value) {
			return false, fmt.Errorf("transformed bounds must be finite")
		}
	}
	return centerX+halfWidth > 0 && centerX-halfWidth < float64(dimensions.Width) &&
		centerY+halfHeight > 0 && centerY-halfHeight < float64(dimensions.Height), nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func (e *executor) persistScenery(
	ctx context.Context,
	payload CreateSceneryPayload,
	layers []LaidOutSceneryLayer,
) (json.RawMessage, error) {
	batchID, err := newSceneryBatchID()
	if err != nil {
		return nil, fmt.Errorf("generator: create scenery resource batch: %w", err)
	}

	persistedKeys := make([]string, 0, len(layers))
	cleanup := func(workflowErr error) error {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), sceneryCleanupTTL)
		defer cancel()
		cleanupErr := e.deleteSceneryResources(cleanupCtx, persistedKeys)
		if cleanupErr == nil {
			return workflowErr
		}
		return errors.Join(workflowErr, cleanupErr)
	}

	contentLayers := make([]assetdomain.SceneryLayer, 0, len(layers))
	for _, layer := range layers {
		if err := ctx.Err(); err != nil {
			return nil, cleanup(err)
		}
		if layer.MediaType != "image/png" {
			return nil, cleanup(fmt.Errorf("generator: persist scenery layer %d: expected image/png", layer.ID))
		}
		data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(layer.ImageBase64))
		if err != nil || len(data) == 0 {
			return nil, cleanup(fmt.Errorf("generator: persist scenery layer %d: invalid base64 PNG", layer.ID))
		}
		objectKey := fmt.Sprintf(
			"projects/%d/scenery/%s/layers/%d.png",
			payload.ProjectID,
			batchID,
			layer.ID,
		)
		if err := e.resources.PutObject(ctx, objectKey, layer.MediaType, data); err != nil {
			return nil, cleanup(fmt.Errorf("generator: persist scenery layer %d: %w", layer.ID, err))
		}
		persistedKeys = append(persistedKeys, objectKey)

		transform, err := json.Marshal(sceneryTransform{
			Scale:    layer.Layout.Scale,
			Rotation: layer.Layout.Rotation,
		})
		if err != nil {
			return nil, cleanup(fmt.Errorf("generator: encode scenery layer %d transform: %w", layer.ID, err))
		}
		contentLayers = append(contentLayers, assetdomain.SceneryLayer{
			ID:        layer.ID,
			Name:      layer.Name,
			Resource:  objectKey,
			Position:  assetdomain.Position{X: layer.Layout.Position.X, Y: layer.Layout.Position.Y},
			Transform: transform,
			Visible:   new(true),
			Opacity:   new(layer.Layout.Opacity),
			ZIndex:    new(layer.Layout.ZIndex),
		})
	}

	asset, err := newSceneryAsset(payload, contentLayers)
	if err != nil {
		return nil, cleanup(err)
	}
	assetID, err := e.assets.CreateSceneryAsset(ctx, asset)
	if err != nil {
		return nil, cleanup(fmt.Errorf("generator: create scenery asset: %w", err))
	}
	if assetID == 0 {
		return nil, cleanup(fmt.Errorf("generator: create scenery asset: empty result"))
	}
	return encodeExecutionResult(ExecutionResult{AssetID: assetID})
}

func newSceneryAsset(payload CreateSceneryPayload, layers []assetdomain.SceneryLayer) (*assetdomain.Asset, error) {
	content := assetdomain.NewAssetContent(assetdomain.AssetTypeScenery)
	content.Layers = layers
	encodedContent, err := assetdomain.EncodeContent(content)
	if err != nil {
		return nil, fmt.Errorf("generator: encode scenery asset content: %w", err)
	}
	encodedDimensions, err := json.Marshal(payload.Dimensions)
	if err != nil {
		return nil, fmt.Errorf("generator: encode scenery asset dimensions: %w", err)
	}
	return &assetdomain.Asset{
		Name:        strings.TrimSpace(payload.AssetName),
		ProjectID:   payload.ProjectID,
		Type:        assetdomain.AssetTypeScenery,
		Description: strings.TrimSpace(payload.CreativeBrief),
		Perspective: assetdomain.Perspective(payload.Perspective),
		Dimensions:  encodedDimensions,
		Content:     encodedContent,
	}, nil
}

func (e *executor) deleteSceneryResources(ctx context.Context, objectKeys []string) error {
	var cleanupErr error
	for _, objectKey := range slices.Backward(objectKeys) {
		if err := e.resources.DeleteObject(ctx, objectKey); err != nil {
			cleanupErr = errors.Join(
				cleanupErr,
				fmt.Errorf("generator: delete unreferenced scenery resource %q: %w", objectKey, err),
			)
		}
	}
	return cleanupErr
}

func newSceneryBatchID() (string, error) {
	value := make([]byte, sceneryBatchIDBytes)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
