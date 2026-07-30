//go:build integration

package service_test

import (
	"context"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/config"
	"github.com/1024XEngineer/Holonic-Asset/pkg/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/pkg/viperx"
)

//	go test -tags=integration -run TestGenerate -v ./internal/service

// experimentImageClient 从 config/config.yaml 读 key,构造一个真实的 imageclient。
func experimentImageClient(t *testing.T) imageclient.ImageGenerationService {
	t.Helper()
	var cfg config.Config
	if err := viperx.LoadConfig("../../config/config.yaml", &cfg); err != nil {
		t.Fatalf("load config failed: %v\n"+
			"hint: copy config/config.example.yaml to config/config.yaml and fill qna.apiKey", err)
	}
	if cfg.QNA.APIKey == "" {
		t.Fatalf("qna.apiKey is empty in config/config.yaml")
	}
	return imageclient.NewImageGenerationService(imageclient.NewQNAProvider(imageclient.QNAConfig{
		BaseURL:      cfg.QNA.BaseURL,
		APIKey:       cfg.QNA.APIKey,
		DefaultModel: cfg.QNA.DefaultModel,
	}))
}

// TestGenerateCharacter:编一个角色提示词 → 生图 → (后续在此处理)→ 打印 base64。
func TestGenerateCharacter(t *testing.T) {
	svc := experimentImageClient(t)
	prompt := "a full-body character concept of a young knight with a glowing sword, pixel art style, clean background"

	result, err := svc.Generate(context.Background(), &imageclient.GenerateRequest{
		Prompt: prompt,
		Size:   "1024x1024",
		Params: imageclient.Params{"quality": "high"},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// TODO: 在这里对 result.Images 做后处理(裁剪、缩放、切片成 tile、转码等)。
	// result.Images[i].Base64 是 base64 字符串,result.Images[i].MediaType 是 MIME 类型。

	t.Logf("prompt: %s", prompt)
	if len(result.Images) == 0 {
		t.Fatalf("expected at least one image, got 0")
	}
	for i, img := range result.Images {
		t.Logf("image[%d] mediaType: %s, base64 length: %d", i, img.MediaType, len(img.Base64))
		t.Logf("image[%d] base64: %s", i, img.Base64)
	}
}
