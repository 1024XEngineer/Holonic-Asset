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
auth:
  jwtSecret: test-secret
  tokenExpiry: 2h
log:
  path: ./logs/app.log
  maxSize: 100
  maxBackups: 7
  maxAge: 14
  compress: true
image:
  baseURL: https://images.example.test
  apiKey: test-image-key
  defaultModel: openai/gpt-image-2
  fallbackModel: google/gemini-image
  models:
    - name: openai/gpt-image-2
      protocol: openai_images
    - name: google/gemini-image
      protocol: chat_completions
llm:
  baseURL: https://llm.example.test
  apiKey: test-llm-key
  defaultModel: vision-model
  models:
    - name: vision-model
      protocol: chat_completions
pprof:
  enabled: true
video:
  baseURL: https://video.example.test
  apiKey: test-video-key
  models:
    - name: bytedance/seedance-2.0
      protocol: fal_queue
  pollInterval: 5s
  pollTimeout: 45s
  maxRetries: 3
  retryDelay: 2s
qiniu:
  accessKey: test-ak
  secretKey: test-sk
  bucket: asset-bucket
  domain: cdn.example.com
  uploadTokenExpiry: 45m
  downloadURLExpiry: 20m
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
	if !loaded.Pprof.Enabled {
		t.Fatal("expected pprof to be enabled")
	}
	if loaded.Auth.JWTSecret != "test-secret" || loaded.Auth.TokenExpiry != 2*time.Hour {
		t.Fatalf("unexpected auth config: %+v", loaded.Auth)
	}
	if loaded.Log.Path != "./logs/app.log" || !loaded.Log.Compress {
		t.Fatalf("unexpected log config: %+v", loaded.Log)
	}
	if len(loaded.Image.Models) != 2 ||
		loaded.Image.Models[0].Protocol != "openai_images" ||
		loaded.Image.Models[1].Name != "google/gemini-image" ||
		loaded.Image.Models[1].Protocol != "chat_completions" {
		t.Fatalf("unexpected image models: %+v", loaded.Image.Models)
	}
	if loaded.LLM.BaseURL != "https://llm.example.test" || loaded.LLM.APIKey != "test-llm-key" || loaded.LLM.DefaultModel != "vision-model" {
		t.Fatalf("unexpected LLM config: %+v", loaded.LLM)
	}
	if len(loaded.LLM.Models) != 1 || loaded.LLM.Models[0].Protocol != "chat_completions" {
		t.Fatalf("unexpected LLM models: %+v", loaded.LLM.Models)
	}
	if loaded.Video.BaseURL != "https://video.example.test" || loaded.Video.APIKey != "test-video-key" || len(loaded.Video.Models) != 1 || loaded.Video.Models[0].Protocol != "fal_queue" || loaded.Video.PollInterval != 5*time.Second || loaded.Video.PollTimeout != 45*time.Second || loaded.Video.MaxRetries != 3 || loaded.Video.RetryDelay != 2*time.Second {
		t.Fatalf("unexpected video config: %+v", loaded.Video)
	}
	if loaded.QiNiu.AccessKey != "test-ak" || loaded.QiNiu.SecretKey != "test-sk" || loaded.QiNiu.Bucket != "asset-bucket" || loaded.QiNiu.Domain != "cdn.example.com" || loaded.QiNiu.UploadTokenExpiry != 45*time.Minute || loaded.QiNiu.DownloadURLExpiry != 20*time.Minute {
		t.Fatalf("unexpected qiniu config: %+v", loaded.QiNiu)
	}
}

func TestLoadConfigDecodesExampleConfig(t *testing.T) {
	path := filepath.Join("..", "..", "config", "config.example.yaml")

	var loaded config.Config
	if err := viperx.LoadConfig(path, &loaded); err != nil {
		t.Fatalf("load example config: %v", err)
	}

	if loaded.Pprof.Enabled {
		t.Fatal("expected pprof to be disabled by default in example config")
	}
	if loaded.Image.DefaultModel != "openai/gpt-image-2" {
		t.Fatalf("unexpected image config: %+v", loaded.Image)
	}
	if len(loaded.Image.Models) != 2 || loaded.Image.Models[1].Protocol != "chat_completions" {
		t.Fatalf("unexpected example image models: %+v", loaded.Image.Models)
	}
	if loaded.Auth.TokenExpiry != 24*time.Hour {
		t.Fatalf("unexpected auth config: %+v", loaded.Auth)
	}
	if loaded.LLM.DefaultModel != "google/gemini-3.7-flash" || len(loaded.LLM.Models) != 1 || loaded.LLM.Models[0].Protocol != "chat_completions" {
		t.Fatalf("unexpected example LLM config: %+v", loaded.LLM)
	}
	if len(loaded.Video.Models) != 0 || loaded.Video.PollInterval != 5*time.Second || loaded.Video.PollTimeout != 45*time.Second || loaded.Video.MaxRetries != 3 || loaded.Video.RetryDelay != 2*time.Second {
		t.Fatalf("unexpected example video config: %+v", loaded.Video)
	}
	if loaded.QiNiu.UploadTokenExpiry != time.Hour || loaded.QiNiu.DownloadURLExpiry != 30*time.Minute {
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

func TestExampleConfigDecodes(t *testing.T) {
	path := filepath.Join("..", "..", "config", "config.example.yaml")
	var loaded config.Config
	if err := viperx.LoadConfig(path, &loaded); err != nil {
		t.Fatalf("load example config: %v", err)
	}
	if loaded.QiNiu.UploadTokenExpiry != time.Hour || loaded.QiNiu.DownloadURLExpiry != 30*time.Minute {
		t.Fatalf("unexpected example qiniu config: %+v", loaded.QiNiu)
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
