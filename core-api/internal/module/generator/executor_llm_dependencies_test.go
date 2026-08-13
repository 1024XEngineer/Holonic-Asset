package generator

import (
	"context"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
)

type llmServiceStub struct{}

func (*llmServiceStub) Complete(context.Context, *llmclient.CompletionRequest) (*llmclient.CompletionResult, error) {
	return &llmclient.CompletionResult{}, nil
}

func TestNewExecutorWithDependenciesInjectsLLMService(t *testing.T) {
	llm := &llmServiceStub{}
	created := NewExecutorWithDependencies(nil, nil, nil, ExecutorDependencies{LLM: llm})
	value, ok := created.(*executor)
	if !ok {
		t.Fatalf("executor type = %T, want *executor", created)
	}
	if value.llm != llm {
		t.Fatal("expected LLM service to be injected")
	}
}

var _ llmclient.LLMService = (*llmServiceStub)(nil)
