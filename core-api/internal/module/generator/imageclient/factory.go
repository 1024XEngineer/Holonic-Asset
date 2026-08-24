package imageclient

import (
	"net/http"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/qnasdk"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

// ProviderType selects the QNA image API format. The name is retained for
// configuration compatibility; its values describe protocols, not vendors.
type ProviderType string

const (
	// ProviderTypeAuto determines the API format from the configured model name.
	ProviderTypeAuto ProviderType = "auto"
	// ProviderTypeOpenAIImages uses the /v1/images/* endpoint format.
	ProviderTypeOpenAIImages ProviderType = "openai_images"
	// ProviderTypeChatCompletions uses QNA's /v1/chat/completions format.
	ProviderTypeChatCompletions ProviderType = "chat_completions"

	// providerTypeLegacyGeminiChat is accepted for existing deployments. The
	// value is intentionally not used for new configuration or diagnostics.
	providerTypeLegacyGeminiChat ProviderType = "gemini_chat"
)

// FactoryConfig provides parameters to initialize an ImageProvider.
type FactoryConfig struct {
	BaseURL       string
	APIKey        string
	DefaultModel  string
	FallbackModel string
	Provider      string
	HTTPClient    *http.Client
	Logger        logger.Logger
}

// NewImageProvider constructs an ImageProvider for one gateway endpoint. When
// FallbackModel is set, transient primary-model failures fall back to a second
// model through the same endpoint and credentials.
func NewImageProvider(cfg FactoryConfig) ImageProvider {
	sdkClient := qnasdk.NewClient(cfg.BaseURL, cfg.APIKey, cfg.HTTPClient)
	primary := createProviderForModel(cfg.Provider, cfg.DefaultModel, cfg, sdkClient)

	fallbackModel := strings.TrimSpace(cfg.FallbackModel)
	if fallbackModel == "" || strings.EqualFold(fallbackModel, cfg.DefaultModel) {
		return primary
	}

	fallback := createProviderForModel(string(ProviderTypeAuto), fallbackModel, FactoryConfig{
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

func createProviderForModel(protocol, model string, cfg FactoryConfig, sdkClient *qnasdk.Client) ImageProvider {
	selected := ProviderType(strings.ToLower(strings.TrimSpace(protocol)))
	if selected == providerTypeLegacyGeminiChat {
		selected = ProviderTypeChatCompletions
	}
	if selected == "" || selected == ProviderTypeAuto {
		if IsChatProtocolModel(model) {
			selected = ProviderTypeChatCompletions
		} else {
			selected = ProviderTypeOpenAIImages
		}
	}

	switch selected {
	case ProviderTypeChatCompletions:
		return NewQNAChatCompletionsProvider(QNAChatCompletionsConfig{
			BaseURL:      cfg.BaseURL,
			APIKey:       cfg.APIKey,
			DefaultModel: model,
			HTTPClient:   cfg.HTTPClient,
			SDKClient:    sdkClient,
			Logger:       cfg.Logger,
		})
	default:
		return NewQNAImagesProvider(QNAImagesConfig{
			BaseURL:      cfg.BaseURL,
			APIKey:       cfg.APIKey,
			DefaultModel: model,
			HTTPClient:   cfg.HTTPClient,
			SDKClient:    sdkClient,
			Logger:       cfg.Logger,
		})
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
