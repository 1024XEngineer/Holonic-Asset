package generator

import (
	"errors"
	"strings"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

func (e *executor) logSceneryStage(
	message string,
	payload CreateSceneryPayload,
	stage string,
	startedAt time.Time,
	fields ...logger.Field,
) {
	if e.logger == nil {
		return
	}
	base := []logger.Field{
		logger.String("workflow", "generate_scenery"),
		logger.String("stage", stage),
		logger.Int("project_id", int(payload.ProjectID)),
		logger.String("asset_name", strings.TrimSpace(payload.AssetName)),
		logger.Int("width", int(payload.Dimensions.Width)),
		logger.Int("height", int(payload.Dimensions.Height)),
		logger.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	}
	e.logger.Info(message, append(base, fields...)...)
}

func (e *executor) logSceneryFailure(
	payload CreateSceneryPayload,
	stage string,
	startedAt time.Time,
	err error,
	fields ...logger.Field,
) {
	if e.logger == nil {
		return
	}
	base := []logger.Field{
		logger.String("workflow", "generate_scenery"),
		logger.String("stage", stage),
		logger.Int("project_id", int(payload.ProjectID)),
		logger.String("asset_name", strings.TrimSpace(payload.AssetName)),
		logger.Int("width", int(payload.Dimensions.Width)),
		logger.Int("height", int(payload.Dimensions.Height)),
		logger.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		logger.Error(err),
	}
	var providerErr *llmclient.ProviderError
	if errors.As(err, &providerErr) {
		base = append(base,
			logger.String("provider", providerErr.Provider),
			logger.String("error_kind", string(providerErr.Kind)),
			logger.Int("status_code", providerErr.StatusCode),
			logger.Any("transient", providerErr.Transient),
			logger.String("provider_message", providerErr.Message),
		)
		if providerErr.Cause != nil {
			base = append(base, logger.Any("provider_cause", providerErr.Cause))
		}
	}
	e.logger.Error("generate scenery stage failed", append(base, fields...)...)
}
