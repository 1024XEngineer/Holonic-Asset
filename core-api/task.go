// Package main is the composition root — it wires business handlers to
// infrastructure modules.
package main

import (
	"context"
	"fmt"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

func InitTask(
	ctx context.Context,
	cfg config.QueueConfig,
	repo task.TaskStore,
) (task.Manager, error) {
	manager, err := task.NewManager(ctx, cfg, repo)
	if err != nil {
		return nil, fmt.Errorf("app: initialize task manager: %w", err)
	}
	return manager, nil
}
