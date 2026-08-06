package viperx_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/viperx"
)

func TestLoadConfigDecodesYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
db:
  dsn: postgres://localhost/holonic
  maxIdleConns: 5
  maxOpenConns: 20
  connMaxIdleTime: 15m
  connMaxLifetime: 1h
queue:
  databaseURL: postgres://localhost/holonic
  maxWorkers: 3
  jobTimeout: 30s
log:
  path: ./logs/app.log
  maxSize: 100
  maxBackups: 7
  maxAge: 14
  compress: true
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	var loaded config.Config
	if err := viperx.LoadConfig(path, &loaded); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded.DB.DSN != "postgres://localhost/holonic" || loaded.DB.MaxOpenConns != 20 {
		t.Fatalf("unexpected database config: %+v", loaded.DB)
	}
	if loaded.DB.ConnMaxIdleTime != 15*time.Minute || loaded.Queue.JobTimeout != 30*time.Second {
		t.Fatalf("unexpected duration config: db=%s queue=%s", loaded.DB.ConnMaxIdleTime, loaded.Queue.JobTimeout)
	}
	if loaded.Log.Path != "./logs/app.log" || !loaded.Log.Compress {
		t.Fatalf("unexpected log config: %+v", loaded.Log)
	}
}

func TestLoadConfigDecodesExampleConfig(t *testing.T) {
	path := filepath.Join("..", "..", "config", "config.example.yaml")

	var loaded config.Config
	if err := viperx.LoadConfig(path, &loaded); err != nil {
		t.Fatalf("load example config: %v", err)
	}

	if loaded.Image.DefaultModel != "openai/gpt-image-2" {
		t.Fatalf("unexpected image config: %+v", loaded.Image)
	}
	if loaded.QiNiu.UploadTokenExpiry != time.Hour {
		t.Fatalf("unexpected qiniu config: %+v", loaded.QiNiu)
	}
}

func TestLoadConfigRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("db:\n  unknown: true\n"), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	var loaded config.Config
	if err := viperx.LoadConfig(path, &loaded); err == nil {
		t.Fatal("expected unknown config field to be rejected")
	}
}

func TestLoadConfigValidatesArguments(t *testing.T) {
	var loaded config.Config
	if err := viperx.LoadConfig("", &loaded); err == nil {
		t.Fatal("expected empty path to be rejected")
	}
	if err := viperx.LoadConfig("config.yaml", nil); err == nil {
		t.Fatal("expected nil target to be rejected")
	}
	if err := viperx.LoadConfig("config.yaml", loaded); err == nil {
		t.Fatal("expected non-pointer target to be rejected")
	}
}
