package imageclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

// ModelFallbackConfig configures a primary model with an automatic fallback model.
type ModelFallbackConfig struct {
	Primary       ImageProvider
	Fallback      ImageProvider
	PrimaryModel  string
	FallbackModel string
	Logger        logger.Logger
}

// ModelFallbackProvider retries transient primary-model failures with a fallback model.
type ModelFallbackProvider struct {
	primary       ImageProvider
	fallback      ImageProvider
	primaryModel  string
	fallbackModel string
	logger        logger.Logger
}

// NewModelFallbackProvider creates a model-level fallback wrapper.
func NewModelFallbackProvider(config ModelFallbackConfig) *ModelFallbackProvider {
	return &ModelFallbackProvider{
		primary:       config.Primary,
		fallback:      config.Fallback,
		primaryModel:  config.PrimaryModel,
		fallbackModel: config.FallbackModel,
		logger:        config.Logger,
	}
}

// Generate executes text-to-image with automatic failover on transient errors.
func (p *ModelFallbackProvider) Generate(
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
		return nil, joinFailoverErrors(err, fbErr, p.fallbackModel)
	}

	return fbResult, nil
}

// Edit executes image-to-image or editing with automatic failover on transient errors.
func (p *ModelFallbackProvider) Edit(
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
		return nil, joinFailoverErrors(err, fbErr, p.fallbackModel)
	}

	return fbResult, nil
}

func (p *ModelFallbackProvider) shouldFailover(ctx context.Context, err error) bool {
	if p.fallback == nil || err == nil {
		return false
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return false
	}
	return IsTransient(err)
}

func joinFailoverErrors(primaryErr, fallbackErr error, fallbackModel string) error {
	// Keep fallback first so errors.As-based retry classification reflects the
	// final provider attempt while both provider failures remain discoverable.
	return fmt.Errorf("failover: %w", errors.Join(
		fmt.Errorf("fallback (%s) failed: %w", fallbackModel, fallbackErr),
		fmt.Errorf("primary failed: %w", primaryErr),
	))
}
