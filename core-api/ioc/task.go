// Package ioc is the composition root — it wires business handlers to
// infrastructure modules.
package ioc

import (
	"context"
	"fmt"

	"github.com/1024XEngineer/Holonic-Asset/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

func InitTask(
	ctx context.Context,
	cfg config.QueueConfig,
	repo task.TaskResultStore,
) (task.Queue, error) {
	queue, err := task.NewQueue(ctx, cfg, repo)
	if err != nil {
		return nil, fmt.Errorf("ioc: initialize task queue: %w", err)
	}
	return queue, nil
}
