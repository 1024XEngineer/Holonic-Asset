package generator

import (
	"bytes"
	"context"
	"encoding/base64"
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
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const sceneryLiveEnv = "HOLONIC_LLM_INTEGRATION"

func TestLiveSceneryPlanningAndLayoutWithRealLLM(t *testing.T) {
	if strings.TrimSpace(os.Getenv(sceneryLiveEnv)) != "1" {
		t.Skip("set HOLONIC_LLM_INTEGRATION=1 to run real scenery LLM regression test")
	}

	llmConfig, err := loadLiveLLMConfig()
	if err != nil {
		t.Fatalf("load live LLM config: %v", err)
	}

	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      llmConfig.BaseURL,
		APIKey:       llmConfig.APIKey,
		DefaultModel: llmConfig.DefaultModel,
	})
	llmService := llmclient.NewLLMService(provider)

	exec := &executor{
		llm:       llmService,
		processor: imageprocessor.NewProcessor(),
	}

	testCases := []struct {
		name        string
		assetName   string
		brief       string
		style       string
		perspective string
		width       int
		height      int
	}{
		{
			name:        "Forest",
			assetName:   "森林",
			brief:       "A mystical ancient forest at twilight with tall pine silhouettes and misty atmosphere",
			style:       "pixel art",
			perspective: "Side-On",
			width:       640,
			height:      360,
		},
		{
			name:        "Snow",
			assetName:   "冰天雪地",
			brief:       "A frozen winter landscape with snow-covered peaks, icy pine trees, and drifting snowflakes",
			style:       "pixel art",
			perspective: "Side-On",
			width:       640,
			height:      360,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			payload := CreateSceneryPayload{
				ProjectID:     22,
				AssetName:     tc.assetName,
				CreativeBrief: tc.brief,
				Style:         tc.style,
				Perspective:   tc.perspective,
				Dimensions:    assetdomain.Size{Width: uint(tc.width), Height: uint(tc.height)},
				ProjectContext: SceneryProjectContext{
					Name:           "Adventure Game",
					GameType:       "Platformer",
					TargetPlatform: "PC",
					Description:    "A 2D platformer adventure",
				},
			}

			// 1. Live layer planning
			t.Logf("[%s] Calling real QNA LLM planSceneryLayers for %q...", tc.name, tc.assetName)
			plan, err := exec.planSceneryLayers(ctx, payload)
			if err != nil {
				t.Fatalf("[%s] live planSceneryLayers failed: %v", tc.name, err)
			}
			if len(plan) == 0 {
				t.Fatalf("[%s] live planSceneryLayers returned 0 layers", tc.name)
			}
			t.Logf("[%s] live planSceneryLayers returned %d layers:", tc.name, len(plan))
			for i, layer := range plan {
				t.Logf("  Layer %d: Name=%q, CreativeBrief=%q", i+1, layer.Name, layer.CreativeBrief)
			}

			// 2. Prepare test processed layers
			processedLayers := make([]ProcessedSceneryLayer, len(plan))
			for i, p := range plan {
				r := uint8((30 * (i + 1)) & 0xFF)
				dataURI := liveTestPNGDataURI(t, tc.width, tc.height, color.RGBA{R: r, G: 150, B: 50, A: 255})
				data := strings.TrimPrefix(dataURI, "data:image/png;base64,")
				processedLayers[i] = ProcessedSceneryLayer{
					ID:          uint(i + 1),
					Name:        p.Name,
					MediaType:   "image/png",
					ImageBase64: data,
				}
			}

			// 3. Live layout analysis
			t.Logf("[%s] Calling real QNA LLM analyzeSceneryLayout...", tc.name)
			laidOut, err := exec.analyzeSceneryLayout(ctx, payload, processedLayers)
			if err != nil {
				t.Fatalf("[%s] live analyzeSceneryLayout failed: %v", tc.name, err)
			}
			if len(laidOut) != len(processedLayers) {
				t.Fatalf("[%s] laidOut count %d != processedLayers count %d", tc.name, len(laidOut), len(processedLayers))
			}
			t.Logf("[%s] live analyzeSceneryLayout returned %d laid out layers:", tc.name, len(laidOut))
			for _, l := range laidOut {
				t.Logf("  Layer %d (%s): pos=(%.1f, %.1f), scale=(%.2f, %.2f), rotation=%.1f, opacity=%.2f, zIndex=%d",
					l.ID, l.Name, l.Layout.Position.X, l.Layout.Position.Y, l.Layout.Scale.X, l.Layout.Scale.Y,
					l.Layout.Rotation, l.Layout.Opacity, l.Layout.ZIndex)
			}
		})
	}
}

func loadLiveLLMConfig() (config.LLMClientConfig, error) {
	reader := viper.New()
	reader.SetConfigFile("../../config/config.yaml")
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

func liveTestPNGDataURI(t *testing.T, w, h int, c color.RGBA) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}
