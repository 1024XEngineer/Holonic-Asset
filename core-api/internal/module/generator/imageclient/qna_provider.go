package imageclient

import (
	"context"
	"fmt"
	"strings"
)

const qnaProviderName = "qna"

type protocolAdapter interface {
	Generate(context.Context, *ProviderRequest) (*ProviderResult, error)
	Edit(context.Context, *ProviderRequest) (*ProviderResult, error)
}

// QNAProvider owns one Modelink/QNA gateway and routes each configured model
// to the protocol adapter required by that model.
type QNAProvider struct {
	defaultModel string
	adapters     map[string]protocolAdapter
}

func newQNAProvider(cfg FactoryConfig) ImageProvider {
	routes := make(map[string]protocolAdapter)
	for _, modelConfig := range cfg.Models {
		model := strings.TrimSpace(modelConfig.Name)
		if model == "" {
			continue
		}
		key := strings.ToLower(model)
		if _, duplicate := routes[key]; duplicate {
			routes[key] = newInvalidProtocolAdapter(
				fmt.Sprintf("model %q is assigned to multiple image protocols", model),
			)
			continue
		}
		routes[key] = createProtocolAdapter(modelConfig.Protocol, model, cfg, modelConfig.BaseURL, modelConfig.APIKey)
	}

	return &QNAProvider{
		defaultModel: strings.TrimSpace(cfg.DefaultModel),
		adapters:     routes,
	}
}

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

func (p *QNAProvider) Edit(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	adapter, routedRequest, err := p.route(request)
	if err != nil {
		return nil, err
	}
	return adapter.Edit(ctx, routedRequest)
}

func (p *QNAProvider) route(
	request *ProviderRequest,
) (protocolAdapter, *ProviderRequest, error) {
	if request == nil {
		return nil, nil, newQNARoutingError("image request is required")
	}

	model := strings.TrimSpace(request.Model)
	if model == "" {
		model = p.defaultModel
	}
	if model == "" {
		return nil, nil, newQNARoutingError("image model is required")
	}

	adapter := p.adapters[strings.ToLower(model)]
	if adapter == nil {
		return nil, nil, newQNARoutingError(
			fmt.Sprintf("no image protocol is configured for model %q", model),
		)
	}

	routedRequest := *request
	routedRequest.Model = model
	return adapter, &routedRequest, nil
}

type invalidProtocolAdapter struct {
	message string
}

func newInvalidProtocolAdapter(message string) protocolAdapter {
	return &invalidProtocolAdapter{message: message}
}

func (p *invalidProtocolAdapter) Generate(
	context.Context,
	*ProviderRequest,
) (*ProviderResult, error) {
	return nil, newQNARoutingError(p.message)
}

func (p *invalidProtocolAdapter) Edit(
	context.Context,
	*ProviderRequest,
) (*ProviderResult, error) {
	return nil, newQNARoutingError(p.message)
}

func newQNARoutingError(message string) *ProviderError {
	return &ProviderError{
		Provider:  qnaProviderName,
		Kind:      ErrorKindInvalidRequest,
		Transient: false,
		Message:   message,
	}
}

var _ ImageProvider = (*QNAProvider)(nil)
