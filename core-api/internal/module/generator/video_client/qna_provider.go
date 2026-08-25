package videoclient

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

// ProtocolType identifies a video wire protocol exposed by the QNA gateway.
type ProtocolType string

const (
	// ProtocolTypeFalQueue uses QNA's asynchronous Fal-compatible queue API.
	ProtocolTypeFalQueue ProtocolType = "fal_queue"

	qnaProviderName = "qna"
)

// ModelConfig maps one QNA video model to its wire protocol.
type ModelConfig struct {
	Name     string
	Protocol string
}

// QNAConfig configures the QNA video gateway.
type QNAConfig struct {
	BaseURL      string
	APIKey       string
	Models       []ModelConfig
	PollInterval time.Duration
	PollTimeout  time.Duration
	// MaxRetries defaults to three when zero; use a negative value to disable retries.
	MaxRetries int
	RetryDelay time.Duration
	HTTPClient *http.Client
	Logger     logger.Logger
}

type videoGenerationAdapter interface {
	Generate(context.Context, *ProviderRequest) (*ProviderResult, error)
}

type videoDownloadAdapter interface {
	Download(context.Context, string) ([]byte, error)
}

// QNAProvider owns one QNA gateway and routes each configured model to the
// protocol adapter required by that model.
type QNAProvider struct {
	adapters   map[string]videoGenerationAdapter
	legacy     videoGenerationAdapter
	downloader videoDownloadAdapter
}

// NewQNAProvider creates the QNA gateway provider.
func NewQNAProvider(config QNAConfig) *QNAProvider {
	if config.HTTPClient == nil {
		config.HTTPClient = newDefaultQNAHTTPClient()
	}

	adapterConfig := qnaFalQueueAdapterConfig{
		BaseURL:      config.BaseURL,
		APIKey:       config.APIKey,
		PollInterval: config.PollInterval,
		PollTimeout:  config.PollTimeout,
		MaxRetries:   config.MaxRetries,
		RetryDelay:   config.RetryDelay,
		HTTPClient:   config.HTTPClient,
		Logger:       config.Logger,
	}
	defaultAdapter := newQNAFalQueueAdapter(adapterConfig)
	provider := &QNAProvider{
		adapters:   make(map[string]videoGenerationAdapter, len(config.Models)),
		downloader: defaultAdapter,
	}
	if len(config.Models) == 0 {
		provider.legacy = defaultAdapter
		return provider
	}

	for _, configured := range config.Models {
		model := strings.Trim(strings.TrimSpace(configured.Name), "/")
		if model == "" {
			continue
		}
		key := strings.ToLower(model)
		if _, duplicate := provider.adapters[key]; duplicate {
			provider.adapters[key] = newInvalidVideoProtocolAdapter(
				fmt.Sprintf("model %q is assigned to multiple video protocols", model),
			)
			continue
		}

		switch ProtocolType(strings.ToLower(strings.TrimSpace(configured.Protocol))) {
		case ProtocolTypeFalQueue:
			modelAdapterConfig := adapterConfig
			modelAdapterConfig.CreatePath = path.Join("/queue", model, "image-to-video")
			modelAdapterConfig.ResultPath = path.Join("/queue", model, "requests")
			provider.adapters[key] = newQNAFalQueueAdapter(modelAdapterConfig)
		default:
			provider.adapters[key] = newInvalidVideoProtocolAdapter(
				fmt.Sprintf("unsupported video protocol %q for model %q", configured.Protocol, model),
			)
		}
	}
	return provider
}

// Generate selects a protocol by model and delegates the wire request to its adapter.
func (p *QNAProvider) Generate(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	adapter, routedRequest, err := p.route(request)
	if err != nil {
		return nil, err
	}
	return adapter.Generate(ctx, routedRequest)
}

func (p *QNAProvider) route(
	request *ProviderRequest,
) (videoGenerationAdapter, *ProviderRequest, error) {
	if request == nil {
		return nil, nil, newQNAVideoRoutingError("video request is nil")
	}

	adapter := p.legacy
	model := strings.Trim(strings.TrimSpace(request.Model), "/")
	if adapter == nil {
		if model == "" {
			return nil, nil, newQNAVideoRoutingError("video model is required")
		}
		adapter = p.adapters[strings.ToLower(model)]
		if adapter == nil {
			return nil, nil, newQNAVideoRoutingError(
				fmt.Sprintf("no video protocol is configured for model %q", model),
			)
		}
	}

	routedRequest := *request
	routedRequest.Model = model
	return adapter, &routedRequest, nil
}

// Download delegates generated media downloads to the gateway's shared transport.
func (p *QNAProvider) Download(ctx context.Context, rawURL string) ([]byte, error) {
	return p.downloader.Download(ctx, rawURL)
}

type invalidVideoProtocolAdapter struct {
	message string
}

func newInvalidVideoProtocolAdapter(message string) videoGenerationAdapter {
	return &invalidVideoProtocolAdapter{message: message}
}

func (a *invalidVideoProtocolAdapter) Generate(
	context.Context,
	*ProviderRequest,
) (*ProviderResult, error) {
	return nil, newQNAVideoRoutingError(a.message)
}

func newQNAVideoRoutingError(message string) *ProviderError {
	return &ProviderError{
		Provider:  qnaProviderName,
		Kind:      ErrorKindInvalidRequest,
		Transient: false,
		Message:   message,
	}
}

var _ VideoProvider = (*QNAProvider)(nil)
