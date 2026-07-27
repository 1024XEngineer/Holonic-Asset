// Package ioc is the composition root — it wires business handlers to
// infrastructure adapters. Only this layer imports concrete queue
// implementations (River, pgx, etc.); business modules never see these.
package ioc

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/1024XEngineer/Holonic-Asset/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	riveradapter "github.com/1024XEngineer/Holonic-Asset/internal/module/task/river"
)

func InitRiver(
	ctx context.Context,
	cfg config.RiverConfig,
	registry *task.Registry,
) (*river.Client[pgx.Tx], task.Producer, error) {
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("ioc: failed to create pgx pool: %w", err)
	}

	riverCfg := &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {
				MaxWorkers: cfg.MaxWorkers,
			},
		},
	}

	if cfg.ClientTimeout > 0 {
		riverCfg.JobTimeout = cfg.ClientTimeout
	}

	client, err := riveradapter.BuildClient(ctx, dbPool, riverCfg, registry)
	if err != nil {
		dbPool.Close()
		return nil, nil, fmt.Errorf("ioc: failed to build river client: %w", err)
	}

	producer := riveradapter.NewProducer(client)

	return client, producer, nil
}
