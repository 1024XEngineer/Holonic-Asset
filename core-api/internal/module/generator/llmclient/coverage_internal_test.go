package llmclient

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

type coverageLLMRoundTripFunc func(*http.Request) (*http.Response, error)

func (f coverageLLMRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type coverageLLMLogger struct{}

func (*coverageLLMLogger) Debug(string, ...logger.Field) {}
func (*coverageLLMLogger) Info(string, ...logger.Field)  {}
func (*coverageLLMLogger) Warn(string, ...logger.Field)  {}
func (*coverageLLMLogger) Error(string, ...logger.Field) {}
func (*coverageLLMLogger) Sync() error                   { return nil }

type coverageLLMProvider struct {
	cancel context.CancelFunc
}

func (p *coverageLLMProvider) Complete(context.Context, *ProviderRequest) (*ProviderResult, error) {
	if p.cancel != nil {
		p.cancel()
	}
	return nil, &ProviderError{Kind: ErrorKindUnavailable, Transient: true, Message: "retry"}
}

func TestLLMCoverageAdapterRequestBoundaries(t *testing.T) {
	logs := &coverageLLMLogger{}
	client := &http.Client{Transport: coverageLLMRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"{}"}}]}`)),
			TLS:        &tls.ConnectionState{},
		}, nil
	})}
	adapter := newQNAChatCompletionsAdapter(QNAConfig{
		BaseURL:      "https://llm.example.test",
		DefaultModel: "default-model",
		HTTPClient:   client,
		Logger:       logs,
	})
	if _, err := adapter.Complete(context.Background(), nil); err == nil {
		t.Fatal("nil adapter request was accepted")
	}
	result, err := adapter.Complete(context.Background(), &ProviderRequest{
		Prompt:         "test",
		ResponseSchema: JSONSchema{Name: "result", Schema: json.RawMessage(`{"type":"object"}`)},
	})
	if err != nil || result.Model != "default-model" {
		t.Fatalf("default-model result = %+v, error = %v", result, err)
	}

	missingModel := newQNAChatCompletionsAdapter(QNAConfig{Logger: logs})
	if _, err := missingModel.Complete(context.Background(), &ProviderRequest{}); err == nil {
		t.Fatal("missing adapter model was accepted")
	}
	missingModel.logRequestSuccess("model", 0, "schema", "/v1/chat/completions", time.Now(), 200, 2, "model", "id", logger.String("extra", "value"))
}

func TestLLMCoverageStructuredJSONBoundaries(t *testing.T) {
	invalidValues := []string{
		"```",
		"```yaml\n{}\n```",
		"```json\n{}",
		"```json\n```nested\n```",
	}
	for _, value := range invalidValues {
		if _, _, err := extractQNAStructuredJSON(value); err == nil {
			t.Fatalf("invalid structured JSON %q was accepted", value)
		}
	}
	cause := errors.New("cause message")
	if got := newQNAError(ErrorKindTransport, 0, true, "", cause).Message; got != cause.Error() {
		t.Fatalf("derived message = %q", got)
	}
}

func TestLLMCoverageRoutingAndFailureClassification(t *testing.T) {
	provider := NewQNAProvider(QNAConfig{
		Models: []ModelConfig{
			{Name: "", Protocol: "chat_completions"},
			{Name: "model", Protocol: "chat_completions"},
		},
	})
	if len(provider.adapters) != 1 {
		t.Fatalf("adapter count = %d, want 1", len(provider.adapters))
	}
	if kind, transient := classifyQNARequestFailure(http.StatusBadGateway, errors.New("status")); kind != ErrorKindUnavailable || !transient {
		t.Fatalf("status failure = (%s, %t)", kind, transient)
	}
	if kind, transient := classifyQNARequestFailure(0, errors.New("transport")); kind != ErrorKindTransport || !transient {
		t.Fatalf("transport failure = (%s, %t)", kind, transient)
	}
}

func TestLLMCoverageCanceledRetryBackoff(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	provider := &coverageLLMProvider{}
	_, err := NewLLMService(provider).Complete(ctx, &CompletionRequest{
		Prompt:      "test",
		MaxAttempts: 2,
		ResponseSchema: JSONSchema{
			Name:   "result",
			Schema: json.RawMessage(`{"type":"object"}`),
		},
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("retry error = %v, want deadline exceeded", err)
	}

	canceled, stop := context.WithCancel(context.Background())
	stop()
	if err := backoffSleep(canceled, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("backoff error = %v", err)
	}
}
