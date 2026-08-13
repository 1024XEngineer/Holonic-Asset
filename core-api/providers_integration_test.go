package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
)

const llmIntegrationEnv = "HOLONIC_LLM_INTEGRATION"

func TestLLMServiceIntegration(t *testing.T) {
	if strings.TrimSpace(os.Getenv(llmIntegrationEnv)) != "1" {
		t.Skip("set HOLONIC_LLM_INTEGRATION=1 to call the configured LLM provider")
	}

	llmConfig, err := loadLLMIntegrationConfig("internal/config/config.yaml")
	if err != nil {
		t.Fatalf("load LLM config: %v", err)
	}
	if strings.TrimSpace(llmConfig.BaseURL) == "" || strings.TrimSpace(llmConfig.APIKey) == "" || strings.TrimSpace(llmConfig.DefaultModel) == "" {
		t.Fatal("LLM baseURL, apiKey, and defaultModel must be configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := InitLLMService(llmConfig).Complete(ctx, &llmclient.CompletionRequest{
		Prompt: "Inspect the test image. Identify the color on its left half and right half. Return only the requested structured result.",
		Images: []llmclient.ImageInput{{URL: integrationTestImageDataURI(t)}},
		ResponseSchema: llmclient.JSONSchema{
			Name: "multimodal_connection_test",
			Schema: json.RawMessage(`{
				"type":"object",
				"additionalProperties":false,
				"required":["imageCount","leftColor","rightColor"],
				"properties":{
					"imageCount":{"type":"integer","const":1},
					"leftColor":{"type":"string","enum":["red"]},
					"rightColor":{"type":"string","enum":["blue"]}
				}
			}`),
		},
	})
	if err != nil {
		t.Fatalf("complete multimodal request: %v", err)
	}

	var observation struct {
		ImageCount int    `json:"imageCount"`
		LeftColor  string `json:"leftColor"`
		RightColor string `json:"rightColor"`
	}
	if err := json.Unmarshal(result.JSON, &observation); err != nil {
		t.Fatalf("decode structured result: %v", err)
	}
	if observation.ImageCount != 1 || observation.LeftColor != "red" || observation.RightColor != "blue" {
		t.Fatalf("unexpected image observation: %+v", observation)
	}
	t.Logf(
		"LLM integration succeeded: model=%q promptTokens=%d completionTokens=%d totalTokens=%d",
		result.Model,
		result.Usage.PromptTokens,
		result.Usage.CompletionTokens,
		result.Usage.TotalTokens,
	)
}

func loadLLMIntegrationConfig(path string) (config.LLMClientConfig, error) {
	reader := viper.New()
	reader.SetConfigFile(path)
	if err := reader.ReadInConfig(); err != nil {
		return config.LLMClientConfig{}, err
	}
	section := reader.Sub("llm")
	if section == nil {
		return config.LLMClientConfig{}, nil
	}
	var value config.LLMClientConfig
	if err := section.UnmarshalExact(&value); err != nil {
		return config.LLMClientConfig{}, err
	}
	return value, nil
}

func integrationTestImageDataURI(t *testing.T) string {
	t.Helper()
	value := image.NewRGBA(image.Rect(0, 0, 64, 32))
	for y := range 32 {
		for x := range 64 {
			pixel := color.RGBA{R: 255, A: 255}
			if x >= 32 {
				pixel = color.RGBA{B: 255, A: 255}
			}
			value.SetRGBA(x, y, pixel)
		}
	}

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, value); err != nil {
		t.Fatalf("encode integration test image: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes())
}
