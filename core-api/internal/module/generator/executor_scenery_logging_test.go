package generator

import (
	"errors"
	"testing"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type recordedLogEntry struct {
	Level   string
	Message string
	Fields  []logger.Field
}

type recordingLogger struct {
	entries []recordedLogEntry
}

func (l *recordingLogger) Debug(msg string, fields ...logger.Field) {
	l.entries = append(l.entries, recordedLogEntry{Level: "debug", Message: msg, Fields: fields})
}

func (l *recordingLogger) Info(msg string, fields ...logger.Field) {
	l.entries = append(l.entries, recordedLogEntry{Level: "info", Message: msg, Fields: fields})
}

func (l *recordingLogger) Warn(msg string, fields ...logger.Field) {
	l.entries = append(l.entries, recordedLogEntry{Level: "warn", Message: msg, Fields: fields})
}

func (l *recordingLogger) Error(msg string, fields ...logger.Field) {
	l.entries = append(l.entries, recordedLogEntry{Level: "error", Message: msg, Fields: fields})
}

func (l *recordingLogger) Sync() error {
	return nil
}


func TestSceneryLogging(t *testing.T) {
	payload := CreateSceneryPayload{
		ProjectID:  12,
		AssetName:  "Forest Temple",
		Dimensions: assetdomain.Size{Width: 256, Height: 256},
	}

	t.Run("nil logger does not panic", func(t *testing.T) {
		exec := &executor{logger: nil}
		exec.logSceneryStage("stage msg", payload, "plan", time.Now())
		exec.logSceneryFailure(payload, "plan", time.Now(), errors.New("boom"))
	})

	t.Run("log scenery stage", func(t *testing.T) {
		rec := &recordingLogger{}
		exec := &executor{logger: rec}
		exec.logSceneryStage("planning started", payload, "plan", time.Now().Add(-100*time.Millisecond))

		if len(rec.entries) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(rec.entries))
		}
		entry := rec.entries[0]
		if entry.Level != "info" || entry.Message != "planning started" {
			t.Fatalf("unexpected entry: %+v", entry)
		}
	})

	t.Run("log scenery failure with standard error", func(t *testing.T) {
		rec := &recordingLogger{}
		exec := &executor{logger: rec}
		exec.logSceneryFailure(payload, "generate_layer", time.Now(), errors.New("timeout"))

		if len(rec.entries) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(rec.entries))
		}
		entry := rec.entries[0]
		if entry.Level != "error" || entry.Message != "generate scenery stage failed" {
			t.Fatalf("unexpected entry: %+v", entry)
		}
	})

	t.Run("log scenery failure with provider error and cause", func(t *testing.T) {
		rec := &recordingLogger{}
		exec := &executor{logger: rec}
		providerErr := &llmclient.ProviderError{
			Provider:   "qna",
			Kind:       "rate_limit",
			StatusCode: 429,
			Transient:  true,
			Message:    "quota exceeded",
			Cause:      errors.New("tcp reset"),
		}
		exec.logSceneryFailure(payload, "generate_layer", time.Now(), providerErr)

		if len(rec.entries) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(rec.entries))
		}
		entry := rec.entries[0]
		if entry.Level != "error" {
			t.Fatalf("unexpected entry level: %s", entry.Level)
		}
	})
}
