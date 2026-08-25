package llmclient

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/qnasdk"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

// ProtocolType identifies an LLM wire protocol exposed by the QNA gateway.
type ProtocolType string

const (
	// ProtocolTypeChatCompletions uses QNA's /v1/chat/completions endpoint.
	ProtocolTypeChatCompletions ProtocolType = "chat_completions"

	qnaProviderName = "qna"
)

// ModelConfig maps one QNA model to its wire protocol and endpoint settings.
type ModelConfig struct {
	Name     string
	Protocol string
	BaseURL  string
	APIKey   string
}

// QNAConfig configures the QNA multimodal LLM gateway.
type QNAConfig struct {
	BaseURL      string
	APIKey       string
	DefaultModel string
	Models       []ModelConfig
	HTTPClient   *http.Client
	SDKClient    *qnasdk.Client
	Logger       logger.Logger
}

type llmProtocolAdapter interface {
	Complete(context.Context, *ProviderRequest) (*ProviderResult, error)
}

// QNAProvider owns one QNA gateway and routes each configured model to the
// protocol adapter required by that model.
type QNAProvider struct {
	defaultModel string
	adapters     map[string]llmProtocolAdapter
	legacy       llmProtocolAdapter
}

// NewQNAProvider creates the QNA gateway provider.
func NewQNAProvider(config QNAConfig) *QNAProvider {
	provider := &QNAProvider{
		defaultModel: strings.TrimSpace(config.DefaultModel),
		adapters:     make(map[string]llmProtocolAdapter, len(config.Models)),
	}
	if len(config.Models) == 0 {
		provider.legacy = newQNAChatCompletionsAdapter(config)
		return provider
	}

	for _, configured := range config.Models {
		model := strings.TrimSpace(configured.Name)
		if model == "" {
			continue
		}
		key := strings.ToLower(model)
		if _, duplicate := provider.adapters[key]; duplicate {
			provider.adapters[key] = newInvalidLLMProtocolAdapter(
				fmt.Sprintf("model %q is assigned to multiple LLM protocols", model),
			)
			continue
		}

		baseURL := strings.TrimSpace(configured.BaseURL)
		if baseURL == "" {
			baseURL = strings.TrimSpace(config.BaseURL)
		}
		apiKey := strings.TrimSpace(configured.APIKey)
		if apiKey == "" {
			apiKey = strings.TrimSpace(config.APIKey)
		}
		modelConfig := config
		modelConfig.BaseURL = baseURL
		modelConfig.APIKey = apiKey
		modelConfig.DefaultModel = model
		modelConfig.SDKClient = nil

		switch ProtocolType(strings.ToLower(strings.TrimSpace(configured.Protocol))) {
		case ProtocolTypeChatCompletions:
			provider.adapters[key] = newQNAChatCompletionsAdapter(modelConfig)
		default:
			provider.adapters[key] = newInvalidLLMProtocolAdapter(
				fmt.Sprintf("unsupported LLM protocol %q for model %q", configured.Protocol, model),
			)
		}
	}
	return provider
}

// Complete selects a protocol by model and delegates the wire request to its adapter.
func (p *QNAProvider) Complete(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	adapter, routedRequest, err := p.route(request)
	if err != nil {
		return nil, err
	}
	return adapter.Complete(ctx, routedRequest)
}

func (p *QNAProvider) route(
	request *ProviderRequest,
) (llmProtocolAdapter, *ProviderRequest, error) {
	if request == nil {
		return nil, nil, newQNAError(ErrorKindInvalidRequest, 0, false, "LLM request is nil", nil)
	}
	model := strings.TrimSpace(request.Model)
	if model == "" {
		model = p.defaultModel
	}
	if model == "" {
		return nil, nil, newQNAError(ErrorKindInvalidRequest, 0, false, "LLM model is required", nil)
	}

	adapter := p.legacy
	if adapter == nil {
		adapter = p.adapters[strings.ToLower(model)]
		if adapter == nil {
			return nil, nil, newQNAError(
				ErrorKindInvalidRequest,
				0,
				false,
				fmt.Sprintf("no LLM protocol is configured for model %q", model),
				nil,
			)
		}
	}

	routedRequest := *request
	routedRequest.Model = model
	return adapter, &routedRequest, nil
}

type invalidLLMProtocolAdapter struct {
	message string
}

func newInvalidLLMProtocolAdapter(message string) llmProtocolAdapter {
	return &invalidLLMProtocolAdapter{message: message}
}

func (a *invalidLLMProtocolAdapter) Complete(
	context.Context,
	*ProviderRequest,
) (*ProviderResult, error) {
	return nil, newQNAError(ErrorKindInvalidRequest, 0, false, a.message, nil)
}

var _ LLMProvider = (*QNAProvider)(nil)
