package imageclient

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/qnasdk"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

// ProtocolType selects the wire format used for one QNA image model.
type ProtocolType string

const (
	// ProtocolTypeAuto determines the API format from the configured model name.
	ProtocolTypeAuto ProtocolType = "auto"
	// ProtocolTypeOpenAIImages uses the /v1/images/* endpoint format.
	ProtocolTypeOpenAIImages ProtocolType = "openai_images"
	// ProtocolTypeChatCompletions uses QNA's /v1/chat/completions format.
	ProtocolTypeChatCompletions ProtocolType = "chat_completions"

	// protocolTypeLegacyGeminiChat is accepted for existing deployments. The
	// value is intentionally not used for new configuration or diagnostics.
	protocolTypeLegacyGeminiChat ProtocolType = "gemini_chat"
)

// ModelConfig assigns one model on the shared gateway to its wire protocol.
type ModelConfig struct {
	Name     string
	Protocol string
}

// FactoryConfig provides parameters to initialize an ImageProvider.
type FactoryConfig struct {
	BaseURL       string
	APIKey        string
	DefaultModel  string
	FallbackModel string
	// Provider is the legacy global protocol override used when Models is empty.
	Provider   string
	Models     []ModelConfig
	HTTPClient *http.Client
	Logger     logger.Logger
}

// NewImageProvider constructs an ImageProvider for one gateway endpoint. When
// FallbackModel is set, transient primary-model failures fall back to a second
// model through the same endpoint and credentials.
func NewImageProvider(cfg FactoryConfig) ImageProvider {
	sdkClient := qnasdk.NewClient(cfg.BaseURL, cfg.APIKey, cfg.HTTPClient)
	if len(cfg.Models) > 0 {
		provider := newQNAProvider(cfg, sdkClient)
		fallbackModel := strings.TrimSpace(cfg.FallbackModel)
		if fallbackModel == "" || strings.EqualFold(fallbackModel, cfg.DefaultModel) {
			return provider
		}
		return NewModelFallbackProvider(ModelFallbackConfig{
			Primary:       provider,
			Fallback:      provider,
			PrimaryModel:  cfg.DefaultModel,
			FallbackModel: fallbackModel,
			Logger:        cfg.Logger,
		})
	}

	primary := createProtocolAdapter(cfg.Provider, cfg.DefaultModel, cfg, sdkClient)

	fallbackModel := strings.TrimSpace(cfg.FallbackModel)
	if fallbackModel == "" || strings.EqualFold(fallbackModel, cfg.DefaultModel) {
		return primary
	}

	fallback := createProtocolAdapter(string(ProtocolTypeAuto), fallbackModel, FactoryConfig{
		BaseURL:      cfg.BaseURL,
		APIKey:       cfg.APIKey,
		DefaultModel: fallbackModel,
		HTTPClient:   cfg.HTTPClient,
		Logger:       cfg.Logger,
	}, sdkClient)

	return NewModelFallbackProvider(ModelFallbackConfig{
		Primary:       primary,
		Fallback:      fallback,
		PrimaryModel:  cfg.DefaultModel,
		FallbackModel: fallbackModel,
		Logger:        cfg.Logger,
	})
}

func createProtocolAdapter(protocol, model string, cfg FactoryConfig, sdkClient *qnasdk.Client) protocolAdapter {
	selected := ProtocolType(strings.ToLower(strings.TrimSpace(protocol)))
	if selected == protocolTypeLegacyGeminiChat {
		selected = ProtocolTypeChatCompletions
	}
	if selected == "" || selected == ProtocolTypeAuto {
		if IsChatProtocolModel(model) {
			selected = ProtocolTypeChatCompletions
		} else {
			selected = ProtocolTypeOpenAIImages
		}
	}

	switch selected {
	case ProtocolTypeChatCompletions:
		return NewQNAChatCompletionsAdapter(QNAChatCompletionsAdapterConfig{
			BaseURL:      cfg.BaseURL,
			APIKey:       cfg.APIKey,
			DefaultModel: model,
			HTTPClient:   cfg.HTTPClient,
			SDKClient:    sdkClient,
			Logger:       cfg.Logger,
		})
	case ProtocolTypeOpenAIImages:
		return NewQNAImagesAdapter(QNAImagesAdapterConfig{
			BaseURL:      cfg.BaseURL,
			APIKey:       cfg.APIKey,
			DefaultModel: model,
			HTTPClient:   cfg.HTTPClient,
			SDKClient:    sdkClient,
			Logger:       cfg.Logger,
		})
	default:
		return newInvalidProtocolAdapter("unsupported image protocol " + strconv.Quote(protocol))
	}
}

// IsChatProtocolModel reports whether modelName targets a chat-based multimodal image model.
func IsChatProtocolModel(modelName string) bool {
	lower := strings.ToLower(strings.TrimSpace(modelName))
	return strings.HasPrefix(lower, "google/") ||
		strings.Contains(lower, "gemini") ||
		strings.Contains(lower, "banana") ||
		strings.Contains(lower, "chat")
}
