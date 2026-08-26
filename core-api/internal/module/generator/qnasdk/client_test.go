package qnasdk_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/qnasdk"
)

func TestClientExecuteSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected auth: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chat-1","choices":[{"message":{"content":"hello"}}]}`))
	}))
	defer server.Close()

	client := qnasdk.NewClient(server.URL, "test-key", nil)
	var resp struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	err := client.Execute(context.Background(), http.MethodPost, "chat/completions", map[string]any{"model": "test"}, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "chat-1" || len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "hello" {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}

func TestClientExecuteErrorExtraction(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantMessage string
	}{
		{
			name:        "openai style error object",
			status:      http.StatusBadRequest,
			body:        `{"error":{"message":"invalid prompt format"}}`,
			wantMessage: "invalid prompt format",
		},
		{
			name:        "top level message",
			status:      http.StatusTooManyRequests,
			body:        `{"message":"API key daily quota exceeded"}`,
			wantMessage: "API key daily quota exceeded",
		},
		{
			name:        "plain text error",
			status:      http.StatusServiceUnavailable,
			body:        "upstream backend unavailable",
			wantMessage: "upstream backend unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := qnasdk.NewClient(server.URL, "test-key", nil)
			var resp map[string]any
			meta, err := client.ExecuteWithMetadata(context.Background(), http.MethodPost, "chat/completions", nil, &resp)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if meta.StatusCode != tt.status {
				t.Fatalf("statusCode = %d, want %d", meta.StatusCode, tt.status)
			}
			var sdkErr *qnasdk.Error
			if !errors.As(err, &sdkErr) {
				t.Fatalf("expected *qnasdk.Error, got %T: %v", err, err)
			}
			if sdkErr.StatusCode != tt.status {
				t.Fatalf("sdkErr.StatusCode = %d, want %d", sdkErr.StatusCode, tt.status)
			}
			if sdkErr.Message != tt.wantMessage {
				t.Fatalf("sdkErr.Message = %q, want %q", sdkErr.Message, tt.wantMessage)
			}
		})
	}
}
