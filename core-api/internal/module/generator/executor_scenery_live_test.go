package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
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
		perspective string
		width       int
		height      int
	}{
		{
			name:        "Forest",
			assetName:   "森林",
			brief:       "A mystical ancient forest at twilight with tall pine silhouettes and misty atmosphere",
			perspective: "Side-On",
			width:       640,
			height:      360,
		},
		{
			name:        "Snow",
			assetName:   "冰天雪地",
			brief:       "A frozen winter landscape with snow-covered peaks, icy pine trees, and drifting snowflakes",
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
			plan, err := exec.planSceneryLayers(ctx, payload, "")
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
			approved, notes, laidOut, err := exec.analyzeSceneryLayout(ctx, payload, processedLayers)
			if err != nil {
				t.Fatalf("[%s] live analyzeSceneryLayout failed: %v", tc.name, err)
			}
			t.Logf("[%s] layout review decision: approved=%v, notes=%q", tc.name, approved, notes)
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

func TestLiveSceneryFullGenerationReal16x9(t *testing.T) {
	if strings.TrimSpace(os.Getenv("HOLONIC_SCENERY_GENERATE_16X9")) != "1" {
		t.Skip("set HOLONIC_SCENERY_GENERATE_16X9=1 to run full 16:9 real scenery generation")
	}

	llmConfig, err := loadLiveLLMConfig()
	if err != nil {
		t.Fatalf("load live LLM config: %v", err)
	}
	imgConfig, err := loadLiveImageConfig()
	if err != nil {
		t.Fatalf("load live Image config: %v", err)
	}

	llmModels := make([]llmclient.ModelConfig, 0, len(llmConfig.Models))
	for _, m := range llmConfig.Models {
		llmModels = append(llmModels, llmclient.ModelConfig{
			Name:     m.Name,
			Protocol: m.Protocol,
			BaseURL:  m.BaseURL,
			APIKey:   m.APIKey,
		})
	}
	llmProvider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      llmConfig.BaseURL,
		APIKey:       llmConfig.APIKey,
		DefaultModel: llmConfig.DefaultModel,
		Models:       llmModels,
	})
	llmService := llmclient.NewLLMService(llmProvider)

	imageModels := make([]imageclient.ModelConfig, 0, len(imgConfig.Models))
	for _, m := range imgConfig.Models {
		imageModels = append(imageModels, imageclient.ModelConfig{
			Name:     m.Name,
			Protocol: m.Protocol,
			BaseURL:  m.BaseURL,
			APIKey:   m.APIKey,
		})
	}
	imageProvider := imageclient.NewImageProvider(imageclient.FactoryConfig{
		BaseURL:       imgConfig.BaseURL,
		APIKey:        imgConfig.APIKey,
		DefaultModel:  imgConfig.DefaultModel,
		FallbackModel: imgConfig.FallbackModel,
		Models:        imageModels,
	})
	imageService := imageclient.NewImageGenerationService(imageProvider)

	exec := &executor{
		llm:       llmService,
		images:    imageService,
		processor: imageprocessor.NewProcessor(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	payload := CreateSceneryPayload{
		ProjectID:     101,
		AssetName:     "日落峡谷",
		CreativeBrief: "A nostalgic 16-bit pixel art sunset canyon landscape with layered crimson mountain ridges in the distance, a pine-covered plateau in the midground, and an ancient rocky trail with glowing camp embers in the foreground",
		Perspective:   "Side-On",
		Dimensions:    assetdomain.Size{Width: 1536, Height: 1024},
		ProjectContext: SceneryProjectContext{
			Name:           "Sunset Quest",
			GameType:       "Platformer",
			TargetPlatform: "PC",
			Description:    "16-bit pixel art side-scrolling platformer",
		},
	}

	t.Logf("=== Step 1: Planning Scenery Layers (1536x1024, 16:9) ===")
	plan, err := exec.planSceneryLayers(ctx, payload, "")
	if err != nil {
		t.Fatalf("plan scenery layers failed: %v", err)
	}
	t.Logf("Planner returned %d layers:", len(plan))
	for i, l := range plan {
		t.Logf("  [%d] ID=%d Name=%q Brief=%q", i+1, l.ID, l.Name, l.CreativeBrief)
	}

	t.Logf("=== Step 2: Generating Scenery Layers (Front-to-Back chained references) ===")
	layers, err := exec.generateSceneryLayers(ctx, payload, plan)
	if err != nil {
		t.Fatalf("generate scenery layers failed: %v", err)
	}
	t.Logf("Generated %d processed layers successfully", len(layers))

	t.Logf("=== Step 3: Analyzing Scenery Layout (Gemini Multimodal) ===")
	approved, reviewNotes, laidOut, err := exec.analyzeSceneryLayout(ctx, payload, layers)
	if err != nil {
		t.Fatalf("analyze scenery layout failed: %v", err)
	}
	t.Logf("Layout review decision: approved=%v, notes=%q", approved, reviewNotes)

	outDir := t.TempDir()

	// Save each layer PNG
	for _, l := range laidOut {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(l.ImageBase64))
		if err != nil {
			t.Fatalf("decode layer %d PNG base64: %v", l.ID, err)
		}
		filename := filepath.Join(outDir, fmt.Sprintf("scenery_16x9_layer_%d_%s.png", l.ID, sanitizeFilename(l.Name)))
		if err := os.WriteFile(filename, raw, 0o600); err != nil {
			t.Fatalf("write layer file %s: %v", filename, err)
		}
		t.Logf("Saved Layer %d (%s) -> %s (Layout: pos=(%.1f, %.1f), scale=(%.2f, %.2f), zIndex=%d)",
			l.ID, l.Name, filename, l.Layout.Position.X, l.Layout.Position.Y, l.Layout.Scale.X, l.Layout.Scale.Y, l.Layout.ZIndex)
	}

	// Composite final scenery image ordered by zIndex
	composite := image.NewRGBA(image.Rect(0, 0, int(payload.Dimensions.Width), int(payload.Dimensions.Height)))
	sortedLayers := append([]LaidOutSceneryLayer(nil), laidOut...)
	sort.SliceStable(sortedLayers, func(i, j int) bool {
		return sortedLayers[i].Layout.ZIndex < sortedLayers[j].Layout.ZIndex
	})

	for _, l := range sortedLayers {
		raw, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(l.ImageBase64))
		img, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("decode layer for composite: %v", err)
		}
		draw.Draw(composite, composite.Bounds(), img, image.Point{}, draw.Over)
	}

	var compBuf bytes.Buffer
	if err := png.Encode(&compBuf, composite); err != nil {
		t.Fatalf("encode composite PNG: %v", err)
	}
	compFilename := filepath.Join(outDir, "scenery_16x9_composite.png")
	if err := os.WriteFile(compFilename, compBuf.Bytes(), 0o600); err != nil {
		t.Fatalf("write composite PNG: %v", err)
	}
	t.Logf("Saved Final Scenery Composite -> %s", compFilename)
}

func sanitizeFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else if r > 127 { // allow unicode letters
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}
