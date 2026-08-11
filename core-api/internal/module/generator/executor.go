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
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

// Executor owns generation and any resulting asset creation.
type Executor interface {
	Generate(ctx context.Context, taskType TaskType, payload json.RawMessage) (json.RawMessage, error)
}

// AssetWriter is the subset of Workspace asset operations used by generation.
type AssetWriter interface {
	CreateCharacterAsset(context.Context, *assetdomain.Asset) (*assetdomain.Asset, error)
	CreateObjectAsset(context.Context, *assetdomain.Asset) (uint, error)
	CreateSceneryAsset(context.Context, *assetdomain.Asset) (uint, error)
	CreateAnimation(context.Context, uint, string) (uint, error)
	UpdatePrototypeImages(context.Context, uint, []assetdomain.ImageResource) error
	UpdateAnimationFrames(context.Context, uint, uint, []assetdomain.Frame) error
	GetDetail(context.Context, uint) (assetdomain.Asset, error)
}

// ReferenceStore is the object-storage boundary used by the worker. It keeps
// storage-specific credentials and URL formats out of generation logic.
type ReferenceStore interface {
	ResolveReference(context.Context, string) (string, error)
	PersistReference(context.Context, string) (string, error)
}

// ResourceStore publishes generated resources under stable object keys and
// removes them when the enclosing asset workflow cannot be completed.
type ResourceStore interface {
	PutObject(context.Context, string, string, []byte) error
	DeleteObject(context.Context, string) error
}

type executor struct {
	images     imageclient.ImageGenerationService
	llm        llmclient.LLMService
	processor  imageprocessor.Processor
	assets     AssetWriter
	references ReferenceStore
	resources  ResourceStore
}

// NewExecutor creates the image-to-asset workflow used by task handlers.
func NewExecutor(
	images imageclient.ImageGenerationService,
	processor imageprocessor.Processor,
	assets AssetWriter,
	references ...ReferenceStore,
) Executor {
	var referenceStore ReferenceStore
	if len(references) > 0 {
		referenceStore = references[0]
	}
	return NewExecutorWithDependencies(images, processor, assets, ExecutorDependencies{
		References: referenceStore,
	})
}

// ExecutorDependencies contains optional workflow integrations.
type ExecutorDependencies struct {
	References ReferenceStore
	Resources  ResourceStore
	LLM        llmclient.LLMService
}

// NewExecutorWithDependencies creates an executor with explicit optional
// dependencies while preserving NewExecutor for existing callers.
func NewExecutorWithDependencies(
	images imageclient.ImageGenerationService,
	processor imageprocessor.Processor,
	assets AssetWriter,
	dependencies ExecutorDependencies,
) Executor {
	return &executor{
		images:     images,
		llm:        dependencies.LLM,
		processor:  processor,
		assets:     assets,
		references: dependencies.References,
		resources:  dependencies.Resources,
	}
}

type ProcessedSceneryLayer struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	ImageBase64 string `json:"image_base64"`
	MediaType   string `json:"media_type"`
}

const (
	sceneryLayerPlanSchemaName   = "scenery_layer_plan"
	sceneryLayerLayoutSchemaName = "scenery_layer_layout"
	sceneryBatchIDBytes          = 16
	sceneryCleanupTTL            = 15 * time.Second
)

var sceneryLayerPlanJSONSchema = json.RawMessage(`{
  "type": "object",
  "additionalProperties": false,
  "required": ["layers"],
  "properties": {
    "layers": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["name", "creative_brief"],
        "properties": {
          "name": {"type": "string", "minLength": 1},
          "creative_brief": {"type": "string", "minLength": 1}
        }
      }
    }
  }
}`)

var sceneryLayerLayoutJSONSchema = json.RawMessage(`{
  "type":"object","additionalProperties":false,"required":["layers"],"properties":{"layers":{
    "type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,
      "required":["id","position","scale","rotation","opacity","zIndex"],"properties":{
        "id":{"type":"integer","minimum":1},
        "position":{"type":"object","additionalProperties":false,"required":["x","y"],"properties":{"x":{"type":"number"},"y":{"type":"number"}}},
        "scale":{"type":"object","additionalProperties":false,"required":["x","y"],"properties":{"x":{"type":"number","exclusiveMinimum":0},"y":{"type":"number","exclusiveMinimum":0}}},
        "rotation":{"type":"number"},"opacity":{"type":"number","minimum":0,"maximum":1},"zIndex":{"type":"integer"}
      }
    }
  }}
}`)

type sceneryLayerPlanResponse struct {
	Layers *[]sceneryLayerPlanCandidate `json:"layers"`
}

type sceneryLayerPlanCandidate struct {
	Name          *string `json:"name"`
	CreativeBrief *string `json:"creative_brief"`
}

type SceneryLayoutVector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type SceneryLayerLayout struct {
	Position SceneryLayoutVector `json:"position"`
	Scale    SceneryLayoutVector `json:"scale"`
	Rotation float64             `json:"rotation"`
	Opacity  float64             `json:"opacity"`
	ZIndex   int                 `json:"zIndex"`
}

type LaidOutSceneryLayer struct {
	ID          uint               `json:"id"`
	Name        string             `json:"name"`
	ImageBase64 string             `json:"image_base64"`
	MediaType   string             `json:"media_type"`
	Layout      SceneryLayerLayout `json:"layout"`
}

type sceneryLayoutResponse struct {
	Layers *[]sceneryLayoutCandidate `json:"layers"`
}

type sceneryLayoutCandidate struct {
	ID       *uint                         `json:"id"`
	Position *sceneryLayoutVectorCandidate `json:"position"`
	Scale    *sceneryLayoutVectorCandidate `json:"scale"`
	Rotation *float64                      `json:"rotation"`
	Opacity  *float64                      `json:"opacity"`
	ZIndex   *int                          `json:"zIndex"`
}

type sceneryLayoutVectorCandidate struct {
	X *float64 `json:"x"`
	Y *float64 `json:"y"`
}

type sceneryTransform struct {
	Scale    SceneryLayoutVector `json:"scale"`
	Rotation float64             `json:"rotation"`
}

func (e *executor) Generate(
	ctx context.Context,
	taskType TaskType,
	payload json.RawMessage,
) (json.RawMessage, error) {
	if e.images == nil {
		return nil, ErrImageServiceRequired
	}
	if e.assets == nil {
		return nil, ErrAssetWriterRequired
	}
	if e.processor == nil && (taskType == GenerateCharacterProtoType || taskType == GenerateObjectProtoType || taskType == GenerateAnimation || taskType == GenerateScenery) {
		return nil, ErrImageProcessorRequired
	}
	if e.llm == nil && taskType == GenerateScenery {
		return nil, ErrLLMServiceRequired
	}
	if e.resources == nil && taskType == GenerateScenery {
		return nil, ErrResourceStoreRequired
	}

	switch taskType {
	case GenerateCharacterProtoType:
		request := CreateCharacterPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateCharacterPrototype(ctx, request)
	case GenerateObjectProtoType:
		request := CreateObjectPrototypePayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateObjectPrototype(ctx, request)
	case GenerateAnimation:
		request := CreateAnimationPayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		return e.generateAnimation(ctx, taskType, request.ParentID, request.AssetName, request.CreativeBrief)
	case GenerateScenery:
		request := CreateSceneryPayload{}
		if err := decodeExecutionPayload(taskType, payload, &request); err != nil {
			return nil, err
		}
		if err := validateSceneryPayload(request); err != nil {
			return nil, err
		}
		plan, err := e.planSceneryLayers(ctx, request)
		if err != nil {
			return nil, err
		}
		layers, err := e.generateSceneryLayers(ctx, request, plan)
		if err != nil {
			return nil, err
		}
		laidOut, err := e.analyzeSceneryLayout(ctx, request, layers)
		if err != nil {
			return nil, err
		}
		return e.persistScenery(ctx, request, laidOut)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedTaskType, taskType)
	}
}

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

func decodeSceneryLayerPlan(raw []byte) ([]SceneryLayerDefinition, error) {
	invalid := func(reason string) error {
		return fmt.Errorf("%w: %s", ErrInvalidSceneryPlan, reason)
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var response sceneryLayerPlanResponse
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
	if len(*response.Layers) == 0 {
		return nil, invalid("at least one layer is required")
	}

	layers := make([]SceneryLayerDefinition, len(*response.Layers))
	names := make(map[string]struct{}, len(layers))
	for index, candidate := range *response.Layers {
		if candidate.Name == nil {
			return nil, invalid(fmt.Sprintf("layer %d name is required", index+1))
		}
		name := strings.TrimSpace(*candidate.Name)
		if name == "" {
			return nil, invalid(fmt.Sprintf("layer %d name is required", index+1))
		}
		normalizedName := strings.ToLower(name)
		if _, duplicate := names[normalizedName]; duplicate {
			return nil, invalid(fmt.Sprintf("layer name %q is duplicated", name))
		}
		names[normalizedName] = struct{}{}

		if candidate.CreativeBrief == nil {
			return nil, invalid(fmt.Sprintf("layer %d creative brief is required", index+1))
		}
		creativeBrief := strings.TrimSpace(*candidate.CreativeBrief)
		if creativeBrief == "" {
			return nil, invalid(fmt.Sprintf("layer %d creative brief is required", index+1))
		}
		layers[index] = SceneryLayerDefinition{
			ID:            uint(index + 1),
			Name:          name,
			CreativeBrief: creativeBrief,
		}
	}
	return layers, nil
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

func (e *executor) generateCharacterPrototype(
	ctx context.Context,
	payload CreateCharacterPrototypePayload,
) (json.RawMessage, error) {
	perspective, err := parsePerspective(payload.Perspective)
	if err != nil {
		return nil, err
	}
	generated, err := e.generateImages(
		ctx,
		GenerateCharacterProtoType,
		payload.CreativeBrief,
		payload.Dimensions,
		payload.Reference,
	)
	if err != nil {
		return nil, err
	}
	resources, err := e.prototypeResources(ctx, generated)
	if err != nil {
		return nil, err
	}
	value, err := newPrototypeAsset(
		assetdomain.AssetTypeCharacter,
		payload.AssetName,
		payload.ProjectID,
		payload.CreativeBrief,
		perspective,
		payload.Dimensions,
		perspective.CharacterDirectionCount(),
		resources,
	)
	if err != nil {
		return nil, err
	}
	created, err := e.assets.CreateCharacterAsset(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("generator: create character asset: %w", err)
	}
	if created == nil || created.ID == 0 {
		return nil, fmt.Errorf("generator: create character asset: empty result")
	}
	return encodeExecutionResult(ExecutionResult{AssetID: created.ID})
}

func (e *executor) generateObjectPrototype(
	ctx context.Context,
	payload CreateObjectPrototypePayload,
) (json.RawMessage, error) {
	perspective, err := parsePerspective(payload.Perspective)
	if err != nil {
		return nil, err
	}
	generated, err := e.generateImages(
		ctx,
		GenerateObjectProtoType,
		payload.CreativeBrief,
		payload.Dimensions,
		payload.Reference,
	)
	if err != nil {
		return nil, err
	}
	resources, err := e.prototypeResources(ctx, generated)
	if err != nil {
		return nil, err
	}
	value, err := newPrototypeAsset(
		assetdomain.AssetTypeObject,
		payload.AssetName,
		payload.ProjectID,
		payload.CreativeBrief,
		perspective,
		payload.Dimensions,
		0,
		resources,
	)
	if err != nil {
		return nil, err
	}
	assetID, err := e.assets.CreateObjectAsset(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("generator: create object asset: %w", err)
	}
	if assetID == 0 {
		return nil, fmt.Errorf("generator: create object asset: empty result")
	}
	return encodeExecutionResult(ExecutionResult{AssetID: assetID})
}

func (e *executor) generateAnimation(
	ctx context.Context,
	taskType TaskType,
	assetID uint,
	name string,
	prompt string,
) (json.RawMessage, error) {
	asset, err := e.assets.GetDetail(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("generator: load asset %d dimensions: %w", assetID, err)
	}
	if asset.Type != assetdomain.AssetTypeCharacter && asset.Type != assetdomain.AssetTypeObject {
		return nil, fmt.Errorf("generator: animation is unsupported for asset type %q", asset.Type)
	}
	var dimensions assetdomain.Size
	if err := json.Unmarshal(asset.Dimensions, &dimensions); err != nil {
		return nil, fmt.Errorf("generator: decode asset %d dimensions: %w", assetID, err)
	}
	if err := assetdomain.ValidateDimensions(asset.Type, asset.Dimensions); err != nil {
		return nil, err
	}
	generated, err := e.generateImages(ctx, taskType, prompt, dimensions, "")
	if err != nil {
		return nil, err
	}
	frames, err := e.animationFrames(ctx, generated)
	if err != nil {
		return nil, err
	}

	animationID, err := e.assets.CreateAnimation(ctx, assetID, name)
	if err != nil {
		return nil, fmt.Errorf("generator: create animation for asset %d: %w", assetID, err)
	}
	if animationID == 0 {
		return nil, fmt.Errorf("generator: create animation for asset %d: empty result", assetID)
	}
	if err := e.assets.UpdateAnimationFrames(ctx, assetID, animationID, frames); err != nil {
		return nil, fmt.Errorf(
			"generator: update asset %d animation %d frames: %w",
			assetID,
			animationID,
			err,
		)
	}
	return encodeExecutionResult(ExecutionResult{AssetID: assetID, AnimationID: animationID})
}

func (e *executor) generateImages(
	ctx context.Context,
	taskType TaskType,
	prompt string,
	dimensions assetdomain.Size,
	reference string,
) (*imageclient.GenerateResult, error) {
	references := []string(nil)
	if reference != "" {
		if e.references != nil {
			resolved, err := e.references.ResolveReference(ctx, reference)
			if err != nil {
				return nil, fmt.Errorf("generator: resolve %s reference: %w", taskType, err)
			}
			reference = resolved
		}
		references = []string{reference}
	}
	result, err := e.images.Generate(ctx, &imageclient.GenerateRequest{
		Prompt:          prompt,
		ReferenceImages: references,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: generate %s images: %w", taskType, err)
	}
	if result == nil || len(result.Images) == 0 {
		return nil, fmt.Errorf("generator: generate %s images: %w", taskType, ErrImageResultRequired)
	}
	if taskType != GenerateCharacterProtoType && taskType != GenerateObjectProtoType && taskType != GenerateAnimation {
		return result, nil
	}
	if dimensions.Width == 0 || dimensions.Height == 0 {
		return nil, fmt.Errorf("generator: process %s images: dimensions must be positive", taskType)
	}
	processed := &imageclient.GenerateResult{Images: make([]imageclient.GeneratedImage, len(result.Images)), Model: result.Model, Size: result.Size, CreatedAt: result.CreatedAt, Usage: result.Usage}
	for index, generated := range result.Images {
		imageBase64 := generated.Base64
		if taskType == GenerateObjectProtoType {
			backgroundRemoved, processErr := e.processor.RemoveBackground(ctx, &imageprocessor.RemoveBackgroundRequest{ImageBase64: imageBase64})
			if processErr != nil {
				return nil, fmt.Errorf("generator: remove %s image %d background: %w", taskType, index+1, processErr)
			}
			if backgroundRemoved == nil || backgroundRemoved.ImageBase64 == "" {
				return nil, fmt.Errorf("generator: remove %s image %d background: empty result", taskType, index+1)
			}
			imageBase64 = backgroundRemoved.ImageBase64
		}
		resized, resizeErr := e.processor.Resize(ctx, &imageprocessor.ResizeRequest{
			ImageBase64: imageBase64,
			Options:     imageprocessor.DefaultResizeOptions(int(dimensions.Width), int(dimensions.Height)),
		})
		if resizeErr != nil {
			return nil, fmt.Errorf("generator: resize %s image %d: %w", taskType, index+1, resizeErr)
		}
		if resized == nil || resized.ImageBase64 == "" {
			return nil, fmt.Errorf("generator: resize %s image %d: empty result", taskType, index+1)
		}
		processed.Images[index] = imageclient.GeneratedImage{Base64: resized.ImageBase64, MediaType: resized.MIMEType}
	}
	return processed, nil
}

func newPrototypeAsset(
	assetType assetdomain.AssetType,
	name string,
	projectID uint,
	description string,
	perspective assetdomain.Perspective,
	dimensions assetdomain.Size,
	directionCount uint,
	prototype []assetdomain.ImageResource,
) (*assetdomain.Asset, error) {
	content := assetdomain.NewAssetContent(assetType)
	prototypeValue := assetdomain.Prototype(prototype)
	content.Prototype = &prototypeValue
	content.DirectionCount = directionCount
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		return nil, fmt.Errorf("generator: encode prototype asset content: %w", err)
	}
	dimensionsValue, err := json.Marshal(dimensions)
	if err != nil {
		return nil, fmt.Errorf("generator: encode prototype asset dimensions: %w", err)
	}
	return &assetdomain.Asset{
		Name:        name,
		ProjectID:   projectID,
		Type:        assetType,
		Description: description,
		Perspective: perspective,
		Dimensions:  dimensionsValue,
		Content:     encoded,
	}, nil
}

func parsePerspective(perspective string) (assetdomain.Perspective, error) {
	value := assetdomain.Perspective(perspective)
	if !value.Valid() {
		return "", fmt.Errorf("generator: invalid perspective %q", perspective)
	}
	return value, nil
}

func (e *executor) prototypeResources(
	ctx context.Context,
	result *imageclient.GenerateResult,
) ([]assetdomain.ImageResource, error) {
	resources := make([]assetdomain.ImageResource, len(result.Images))
	for index, image := range result.Images {
		url := generatedImageDataURL(image)
		if e.references != nil {
			objectKey, err := e.references.PersistReference(ctx, url)
			if err != nil {
				return nil, fmt.Errorf("generator: persist prototype image %d: %w", index+1, err)
			}
			url = objectKey
		}
		resources[index] = assetdomain.ImageResource{ID: uint(index + 1), URL: &url}
	}
	return resources, nil
}

func (e *executor) animationFrames(
	ctx context.Context,
	result *imageclient.GenerateResult,
) ([]assetdomain.Frame, error) {
	frames := make([]assetdomain.Frame, len(result.Images))
	for index, image := range result.Images {
		url := generatedImageDataURL(image)
		if e.references != nil {
			objectKey, err := e.references.PersistReference(ctx, url)
			if err != nil {
				return nil, fmt.Errorf("generator: persist animation frame %d: %w", index+1, err)
			}
			url = objectKey
		}
		frames[index] = assetdomain.Frame{ID: uint(index + 1), URL: &url}
	}
	return frames, nil
}

func generatedImageDataURL(image imageclient.GeneratedImage) string {
	mediaType := image.MediaType
	if mediaType == "" {
		mediaType = "image/png"
	}
	return "data:" + mediaType + ";base64," + image.Base64
}

var _ Executor = (*executor)(nil)
