package generator_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
)

func TestExecutorRequiresDependencies(t *testing.T) {
	executor := generator.NewExecutorWithDependencies(nil, nil, nil, generator.ExecutorDependencies{})
	_, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, nil)
	if !errors.Is(err, generator.ErrImageServiceRequired) {
		t.Fatalf("expected image service required error, got %v", err)
	}

	events := []string{}
	executor = generator.NewExecutorWithDependencies(
		&imageGenerationServiceStub{events: &events},
		nil,
		nil,
		generator.ExecutorDependencies{},
	)
	_, err = executor.Generate(context.Background(), generator.GenerateObjectProtoType, nil)
	if !errors.Is(err, generator.ErrAssetWriterRequired) {
		t.Fatalf("expected asset writer required error, got %v", err)
	}

	executor = generator.NewExecutorWithDependencies(
		&imageGenerationServiceStub{events: &events},
		nil,
		&generationAssetWriterStub{events: &events},
		generator.ExecutorDependencies{},
	)
	_, err = executor.Generate(context.Background(), generator.GenerateObjectProtoType, nil)
	if !errors.Is(err, generator.ErrImageProcessorRequired) {
		t.Fatalf("expected image processor required error, got %v", err)
	}

	executor = generator.NewExecutorWithDependencies(
		nil,
		nil,
		&generationAssetWriterStub{events: &events},
		generator.ExecutorDependencies{},
	)
	_, err = executor.Generate(context.Background(), generator.GenerateAnimation, nil)
	if !errors.Is(err, generator.ErrAnimationServiceRequired) {
		t.Fatalf("expected animation service required error, got %v", err)
	}

	executor = generator.NewExecutorWithDependencies(
		nil,
		nil,
		&generationAssetWriterStub{events: &events},
		generator.ExecutorDependencies{
			Animations: &animationGenerationServiceStub{events: &events},
		},
	)
	_, err = executor.Generate(context.Background(), generator.GenerateAnimation, nil)
	if !errors.Is(err, generator.ErrAnimationReferenceStoreRequired) {
		t.Fatalf("expected animation reference store required error, got %v", err)
	}
}

type dependencyLLMStub struct{}

func (*dependencyLLMStub) Complete(context.Context, *llmclient.CompletionRequest) (*llmclient.CompletionResult, error) {
	return &llmclient.CompletionResult{}, nil
}

type dependencyResourceStub struct{}

func (*dependencyResourceStub) PutObject(context.Context, string, string, []byte) error { return nil }
func (*dependencyResourceStub) DeleteObject(context.Context, string) error              { return nil }

func TestExecutorRequiresSceneryDependencies(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{events: &events}
	processor := &imageProcessorStub{events: &events}
	assets := &generationAssetWriterStub{events: &events}
	llm := &dependencyLLMStub{}
	resources := &dependencyResourceStub{}
	tests := []struct {
		name          string
		omitImages    bool
		omitProcessor bool
		omitAssets    bool
		dependencies  generator.ExecutorDependencies
		want          error
	}{
		{name: "images", omitImages: true, dependencies: generator.ExecutorDependencies{LLM: llm, Resources: resources}, want: generator.ErrImageServiceRequired},
		{name: "assets", omitAssets: true, dependencies: generator.ExecutorDependencies{LLM: llm, Resources: resources}, want: generator.ErrAssetWriterRequired},
		{name: "processor", omitProcessor: true, dependencies: generator.ExecutorDependencies{LLM: llm, Resources: resources}, want: generator.ErrImageProcessorRequired},
		{name: "LLM", dependencies: generator.ExecutorDependencies{Resources: resources}, want: generator.ErrLLMServiceRequired},
		{name: "resources", dependencies: generator.ExecutorDependencies{LLM: llm}, want: generator.ErrResourceStoreRequired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var imageService imageclient.ImageGenerationService = images
			var imageProcessor imageprocessor.Processor = processor
			var assetWriter generator.AssetWriter = assets
			if test.omitImages {
				imageService = nil
			}
			if test.omitProcessor {
				imageProcessor = nil
			}
			if test.omitAssets {
				assetWriter = nil
			}
			executor := generator.NewExecutorWithDependencies(imageService, imageProcessor, assetWriter, test.dependencies)
			_, err := executor.Generate(context.Background(), generator.GenerateScenery, nil)
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}

	executor := generator.NewExecutorWithDependencies(images, processor, assets, generator.ExecutorDependencies{LLM: llm, Resources: resources})
	_, err := executor.Generate(context.Background(), generator.GenerateScenery, json.RawMessage(`{`))
	if err == nil || !strings.Contains(err.Error(), "decode generate_scenery execution payload") {
		t.Fatalf("expected malformed scenery payload error, got %v", err)
	}
}

func TestExecutorRejectsMalformedAndUnsupportedTasks(t *testing.T) {
	events := []string{}
	executor := generator.NewExecutorWithDependencies(
		&imageGenerationServiceStub{events: &events},
		&imageProcessorStub{events: &events},
		&generationAssetWriterStub{events: &events},
		generator.ExecutorDependencies{
			Animations: &animationGenerationServiceStub{events: &events},
			References: &executorReferenceStoreStub{},
		},
	)
	for _, taskType := range []generator.TaskType{
		generator.GenerateCharacterProtoType,
		generator.EditCharacterProtoType,
		generator.GenerateObjectProtoType,
		generator.GenerateAnimation,
		generator.EditAnimation,
	} {
		_, err := executor.Generate(context.Background(), taskType, json.RawMessage(`{`))
		if err == nil || !strings.Contains(err.Error(), "decode "+string(taskType)+" execution payload") {
			t.Fatalf("task %s: expected payload error, got %v", taskType, err)
		}
	}
	_, err := executor.Generate(context.Background(), generator.TaskType("unknown"), nil)
	if !errors.Is(err, generator.ErrUnsupportedTaskType) {
		t.Fatalf("expected unsupported task error, got %v", err)
	}
}
