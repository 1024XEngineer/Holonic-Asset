package imageclient

import (
	"net/http"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

// ProviderType specifies the image client communication protocol.
type ProviderType string

const (
	// ProviderTypeAuto automatically determines protocol from the model name.
	ProviderTypeAuto ProviderType = "auto"
	// ProviderTypeOpenAIImages uses the /v1/images/* endpoint format.
	ProviderTypeOpenAIImages ProviderType = "openai_images"
	// ProviderTypeGeminiChat uses the /v1/chat/completions multimodal endpoint format.
	ProviderTypeGeminiChat ProviderType = "gemini_chat"
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

// NewImageProvider constructs an ImageProvider based on configuration,
// automatically wiring primary and fallback failover providers.
func NewImageProvider(cfg FactoryConfig) ImageProvider {
	primary := createProviderForModel(cfg.Provider, cfg.DefaultModel, cfg)

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
	})

	return NewFailoverImageProvider(FailoverConfig{
		Primary:       primary,
		Fallback:      fallback,
		PrimaryModel:  cfg.DefaultModel,
		FallbackModel: fallbackModel,
		Logger:        cfg.Logger,
	})
}

func createProviderForModel(providerType, model string, cfg FactoryConfig) ImageProvider {
	pt := ProviderType(strings.ToLower(strings.TrimSpace(providerType)))
	if pt == "" || pt == ProviderTypeAuto {
		if IsChatProtocolModel(model) {
			pt = ProviderTypeGeminiChat
		} else {
			pt = ProviderTypeOpenAIImages
		}
	}

	switch pt {
	case ProviderTypeGeminiChat:
		return NewGeminiChatProvider(GeminiChatConfig{
			BaseURL:      cfg.BaseURL,
			APIKey:       cfg.APIKey,
			DefaultModel: model,
			HTTPClient:   cfg.HTTPClient,
			Logger:       cfg.Logger,
		})
	default:
		return NewQNAProvider(QNAConfig{
			BaseURL:      cfg.BaseURL,
			APIKey:       cfg.APIKey,
			DefaultModel: model,
			HTTPClient:   cfg.HTTPClient,
			Logger:       cfg.Logger,
		})
	}
}

// IsChatProtocolModel reports whether modelName targets a chat-based multimodal image model.
func IsChatProtocolModel(modelName string) bool {
	lower := strings.ToLower(strings.TrimSpace(modelName))
	return strings.Contains(lower, "gemini") ||
		strings.Contains(lower, "banana") ||
		strings.Contains(lower, "chat")
}
