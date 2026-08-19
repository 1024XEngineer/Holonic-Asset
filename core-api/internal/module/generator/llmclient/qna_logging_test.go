package llmclient_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

func TestQNAProviderLogsTransportCauseAndRequestMetadata(t *testing.T) {
	transportErr := errors.New("dial tcp: lookup api.qnaigc.com: no such host")
	recorder := &llmRecordingLogger{}
	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      "https://api.qnaigc.com",
		DefaultModel: "vision-model",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, transportErr
		})},
		Logger: recorder,
	})

	_, err := provider.Complete(context.Background(), validProviderRequest())
	if err == nil {
		t.Fatal("Complete() error = nil, want transport error")
	}
	if len(recorder.warnings) != 1 {
		t.Fatalf("warning count = %d, want 1", len(recorder.warnings))
	}
	fields := recorder.warnings[0].fields
	if got := fields["error_kind"]; got != string(llmclient.ErrorKindTransport) {
		t.Fatalf("error_kind = %v, want %q", got, llmclient.ErrorKindTransport)
	}
	cause, ok := fields["cause"].(error)
	if !ok || !errors.Is(cause, transportErr) {
		t.Fatalf("cause = %v, want wrapped %v", fields["cause"], transportErr)
	}
	if got := fields["endpoint"]; got != "https://api.qnaigc.com/v1/chat/completions" {
		t.Fatalf("endpoint = %v, want full QNA endpoint", got)
	}
	if got := fields["status_code"]; got != 0 {
		t.Fatalf("status_code = %v, want 0 for transport failure", got)
	}
}

type llmRecordingLogger struct {
	warnings []llmLogRecord
}

type llmLogRecord struct {
	message string
	fields  map[string]any
}

func (*llmRecordingLogger) Debug(string, ...logger.Field) {}
func (*llmRecordingLogger) Info(string, ...logger.Field)  {}
func (l *llmRecordingLogger) Warn(message string, fields ...logger.Field) {
	l.warnings = append(l.warnings, llmLogRecord{message: message, fields: fieldMap(fields)})
}
func (*llmRecordingLogger) Error(string, ...logger.Field) {}
func (*llmRecordingLogger) Sync() error                   { return nil }

func fieldMap(fields []logger.Field) map[string]any {
	result := make(map[string]any, len(fields))
	for _, field := range fields {
		result[field.Key] = field.Val
	}
	return result
}

func TestQNAProviderLogsStructuredResponseDiagnosticsWithoutContent(t *testing.T) {
	const sensitiveContent = "do-not-log-this-model-output"
	recorder := &llmRecordingLogger{}
	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      "https://api.qnaigc.com",
		DefaultModel: "deepseek/deepseek-v4-flash-20260731",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			body := `{"choices":[{"message":{"content":"` + sensitiveContent + `","reasoning_content":"hidden reasoning"},"finish_reason":"stop"}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		})},
		Logger: recorder,
	})

	_, err := provider.Complete(context.Background(), validProviderRequest())
	if err == nil {
		t.Fatal("Complete() error = nil, want invalid structured response")
	}
	var failure *llmLogRecord
	for index := range recorder.warnings {
		if recorder.warnings[index].message == "qna llm request failed" {
			failure = &recorder.warnings[index]
		}
	}
	if failure == nil {
		t.Fatal("missing qna llm request failed log")
	}
	if got := failure.fields["response_format"]; got != "json_object" {
		t.Fatalf("response_format = %v, want json_object", got)
	}
	if got := failure.fields["finish_reason"]; got != "stop" {
		t.Fatalf("finish_reason = %v, want stop", got)
	}
	if got := failure.fields["content_bytes"]; got != len(sensitiveContent) {
		t.Fatalf("content_bytes = %v, want %d", got, len(sensitiveContent))
	}
	if got := failure.fields["reasoning_content_bytes"]; got != len("hidden reasoning") {
		t.Fatalf("reasoning_content_bytes = %v, want %d", got, len("hidden reasoning"))
	}
	for key, value := range failure.fields {
		if strings.Contains(fmt.Sprint(value), sensitiveContent) {
			t.Fatalf("field %q leaked response content", key)
		}
	}
}
