package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
)

func TestInitLoggerRejectsInvalidConfig(t *testing.T) {
	if _, err := InitLogger(nil); err == nil {
		t.Fatal("expected nil log config to be rejected")
	}
	if _, err := InitLogger(&config.LogConfig{}); err == nil {
		t.Fatal("expected empty log path to be rejected")
	}
}

func TestInitLoggerWritesConfiguredFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "app.log")
	appLogger, err := InitLogger(&config.LogConfig{
		Path:       path,
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
	})
	if err != nil {
		t.Fatalf("initialize logger: %v", err)
	}

	appLogger.Info("logger ready")
	if err := appLogger.Sync(); err != nil {
		t.Fatalf("sync logger: %v", err)
	}

	content, err := os.ReadFile(path) // #nosec G304 -- path is created under t.TempDir for this test.
	if err != nil {
		t.Fatalf("read configured log file: %v", err)
	}
	if !strings.Contains(string(content), "logger ready") {
		t.Fatalf("expected log message in configured file, got %q", content)
	}
}

func TestGetZapLevel(t *testing.T) {
	tests := []struct {
		input string
		want  zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"DEBUG", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"warning", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"dpanic", zapcore.DPanicLevel},
		{"panic", zapcore.PanicLevel},
		{"fatal", zapcore.FatalLevel},
		{"unknown", zapcore.DebugLevel},
		{"", zapcore.DebugLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := getZapLevel(tt.input)
			if got != tt.want {
				t.Errorf("getZapLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPrettyJSONEncoder(t *testing.T) {
	encoderCfg := zap.NewProductionEncoderConfig()
	baseEncoder := zapcore.NewJSONEncoder(encoderCfg)
	prettyEncoder := &PrettyJSONEncoder{Encoder: baseEncoder}

	cloned := prettyEncoder.Clone()
	if cloned == nil {
		t.Fatal("expected non-nil cloned encoder")
	}

	ent := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Message: "pretty log",
	}

	buf, err := prettyEncoder.EncodeEntry(ent, []zapcore.Field{
		zap.String("key", "value"),
	})
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}
	if !strings.Contains(buf.String(), `"key": "value"`) {
		t.Fatalf("expected formatted json in buffer, got %s", buf.String())
	}
}
