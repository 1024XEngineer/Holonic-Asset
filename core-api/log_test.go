package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
