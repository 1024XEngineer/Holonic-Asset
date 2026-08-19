package imageclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

// FailoverConfig configures a primary image provider with an automatic fallback.
type FailoverConfig struct {
	Primary       ImageProvider
	Fallback      ImageProvider
	PrimaryModel  string
	FallbackModel string
	Logger        logger.Logger
}

// FailoverImageProvider executes image operations on the primary provider,
// automatically failing over to the fallback provider when transient failures occur.
type FailoverImageProvider struct {
	primary       ImageProvider
	fallback      ImageProvider
	primaryModel  string
	fallbackModel string
	logger        logger.Logger
}

// NewFailoverImageProvider creates a failover provider wrapping primary and fallback.
func NewFailoverImageProvider(config FailoverConfig) *FailoverImageProvider {
	return &FailoverImageProvider{
		primary:       config.Primary,
		fallback:      config.Fallback,
		primaryModel:  config.PrimaryModel,
		fallbackModel: config.FallbackModel,
		logger:        config.Logger,
	}
}

// Generate executes text-to-image with automatic failover on transient errors.
func (p *FailoverImageProvider) Generate(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	primaryReq := *request
	if primaryReq.Model == "" {
		primaryReq.Model = p.primaryModel
	}

	result, err := p.primary.Generate(ctx, &primaryReq)
	if err == nil {
		return result, nil
	}

	if !p.shouldFailover(ctx, err) {
		return nil, err
	}

	if p.logger != nil {
		p.logger.Warn(
			"primary image generation failed with transient error; failing over to fallback model",
			logger.String("primary_model", primaryReq.Model),
			logger.String("fallback_model", p.fallbackModel),
			logger.Error(err),
		)
	}

	fallbackReq := *request
	fallbackReq.Model = p.fallbackModel
	fbResult, fbErr := p.fallback.Generate(ctx, &fallbackReq)
	if fbErr != nil {
		if p.logger != nil {
			p.logger.Error(
				"fallback image generation also failed",
				logger.String("fallback_model", p.fallbackModel),
				logger.Error(fbErr),
			)
		}
		return nil, fmt.Errorf("failover: primary failed (%w), fallback (%s) failed: %v", err, p.fallbackModel, fbErr)
	}

	return fbResult, nil
}

// Edit executes image-to-image or editing with automatic failover on transient errors.
func (p *FailoverImageProvider) Edit(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	primaryReq := *request
	if primaryReq.Model == "" {
		primaryReq.Model = p.primaryModel
	}

	result, err := p.primary.Edit(ctx, &primaryReq)
	if err == nil {
		return result, nil
	}

	if !p.shouldFailover(ctx, err) {
		return nil, err
	}

	if p.logger != nil {
		p.logger.Warn(
			"primary image edit failed with transient error; failing over to fallback model",
			logger.String("primary_model", primaryReq.Model),
			logger.String("fallback_model", p.fallbackModel),
			logger.Error(err),
		)
	}

	fallbackReq := *request
	fallbackReq.Model = p.fallbackModel
	fbResult, fbErr := p.fallback.Edit(ctx, &fallbackReq)
	if fbErr != nil {
		if p.logger != nil {
			p.logger.Error(
				"fallback image edit also failed",
				logger.String("fallback_model", p.fallbackModel),
				logger.Error(fbErr),
			)
		}
		return nil, fmt.Errorf("failover: primary failed (%w), fallback (%s) failed: %v", err, p.fallbackModel, fbErr)
	}

	return fbResult, nil
}

func (p *FailoverImageProvider) shouldFailover(ctx context.Context, err error) bool {
	if p.fallback == nil || err == nil {
		return false
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return false
	}
	return IsTransient(err)
}
