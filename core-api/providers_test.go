package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
)

func TestInitLLMServiceUsesIndependentConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer llm-key" {
			t.Fatalf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		var payload struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Model != "vision-model" {
			t.Fatalf("model = %q, want vision-model", payload.Model)
		}
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"{}"}}]}`))
	}))
	defer server.Close()

	service := InitLLMService(config.LLMClientConfig{
		BaseURL:      server.URL,
		APIKey:       "llm-key",
		DefaultModel: "vision-model",
	})
	_, err := service.Complete(context.Background(), &llmclient.CompletionRequest{
		Prompt: "analyze",
		Images: []llmclient.ImageInput{{URL: "https://cdn.example.test/layer.png"}},
		ResponseSchema: llmclient.JSONSchema{
			Name:   "layout",
			Schema: json.RawMessage(`{"type":"object"}`),
		},
	})
	if err != nil {
		t.Fatalf("complete through initialized service: %v", err)
	}
}
