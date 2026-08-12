package generator_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
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
