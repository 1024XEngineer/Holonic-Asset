package executor_test

import (
	"context"
	"errors"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	generatorexecutor "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/executor"
)

func TestExecutorRequiresDependencies(t *testing.T) {
	executor := generatorexecutor.NewExecutor(nil, nil, nil)
	_, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, nil)
	if !errors.Is(err, generator.ErrImageServiceRequired) {
		t.Fatalf("expected image service required error, got %v", err)
	}

	events := []string{}
	executor = generatorexecutor.NewExecutor(&imageGenerationServiceStub{events: &events}, nil, nil)
	_, err = executor.Generate(context.Background(), generator.GenerateObjectProtoType, nil)
	if !errors.Is(err, generator.ErrAssetWriterRequired) {
		t.Fatalf("expected asset writer required error, got %v", err)
	}

	executor = generatorexecutor.NewExecutor(
		&imageGenerationServiceStub{events: &events},
		nil,
		&generationAssetWriterStub{events: &events},
	)
	_, err = executor.Generate(context.Background(), generator.GenerateObjectProtoType, nil)
	if !errors.Is(err, generator.ErrImageProcessorRequired) {
		t.Fatalf("expected image processor required error, got %v", err)
	}

	executor = generatorexecutor.NewExecutor(nil, nil, &generationAssetWriterStub{events: &events})
	_, err = executor.Generate(context.Background(), generator.GenerateAnimation, nil)
	if !errors.Is(err, generator.ErrAnimationServiceRequired) {
		t.Fatalf("expected animation service required error, got %v", err)
	}

	executor = generatorexecutor.NewExecutorWithAnimation(
		nil,
		&animationGenerationServiceStub{events: &events},
		nil,
		&generationAssetWriterStub{events: &events},
	)
	_, err = executor.Generate(context.Background(), generator.GenerateAnimation, nil)
	if !errors.Is(err, generator.ErrAnimationReferenceStoreRequired) {
		t.Fatalf("expected animation reference store required error, got %v", err)
	}
}
