package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

const runReaderBatchLimit = 100

// RunReader builds Generator run projections through the generic Task module.
type RunReader struct {
	tasks taskdomain.Manager
}

func NewRunReader(tasks taskdomain.Manager) *RunReader {
	return &RunReader{tasks: tasks}
}

func (r *RunReader) ListRuns(
	ctx context.Context,
	filter *RunListFilter,
) (*RunListPage, error) {
	if filter == nil {
		return nil, fmt.Errorf("generator: run list filter is required")
	}
	if r == nil || r.tasks == nil {
		return &RunListPage{Runs: []Run{}}, nil
	}

	beforeID, err := decodeRunCursor(filter.Cursor)
	if err != nil {
		return nil, err
	}
	types := filteredTaskTypes(filter.IncludeTaskTypes, filter.ExcludeTaskTypes)
	if len(filter.Statuses) == 0 || len(types) == 0 {
		return &RunListPage{Runs: []Run{}}, nil
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultRunListLimit
	}
	batchLimit := max(runReaderBatchLimit, limit+1)

	runs := make([]Run, 0, limit+1)
	for len(runs) <= limit {
		tasks, listErr := r.tasks.List(ctx, &taskdomain.ListFilter{
			Statuses: filter.Statuses,
			Types:    types,
			BeforeID: beforeID,
			Limit:    batchLimit,
		})
		if listErr != nil {
			return nil, fmt.Errorf("generator: list tasks: %w", listErr)
		}
		if len(tasks) == 0 {
			break
		}

		for _, task := range tasks {
			run, mapErr := taskToRun(task)
			if mapErr != nil {
				return nil, mapErr
			}
			if !matchesRunScope(run, filter) {
				continue
			}
			runs = append(runs, run)
			if len(runs) > limit {
				break
			}
		}
		if len(runs) > limit || len(tasks) < batchLimit {
			break
		}

		nextBeforeID := tasks[len(tasks)-1].ID
		if nextBeforeID == 0 || nextBeforeID == beforeID {
			break
		}
		beforeID = nextBeforeID
	}

	page := &RunListPage{Runs: runs}
	if len(runs) > limit {
		page.Runs = runs[:limit]
		page.NextCursor = strconv.FormatUint(uint64(page.Runs[len(page.Runs)-1].ID), 10)
	}
	return page, nil
}

func decodeRunCursor(cursor string) (uint, error) {
	if cursor == "" {
		return 0, nil
	}
	id, err := strconv.ParseUint(cursor, 10, 0)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: %q", ErrInvalidRunListCursor, cursor)
	}
	return uint(id), nil
}

func filteredTaskTypes(included, excluded []TaskType) []string {
	includeSet := make(map[TaskType]struct{}, len(included))
	for _, taskType := range included {
		includeSet[taskType] = struct{}{}
	}
	excludeSet := make(map[TaskType]struct{}, len(excluded))
	for _, taskType := range excluded {
		excludeSet[taskType] = struct{}{}
	}

	allTypes := TaskTypes()
	types := make([]string, 0, len(allTypes))
	for _, taskType := range allTypes {
		if len(includeSet) > 0 {
			if _, ok := includeSet[taskType]; !ok {
				continue
			}
		}
		if _, ok := excludeSet[taskType]; ok {
			continue
		}
		types = append(types, string(taskType))
	}
	return types
}

func taskToRun(task *taskdomain.Task) (Run, error) {
	var scope struct {
		ProjectID uint  `json:"project_id"`
		AssetID   *uint `json:"asset_id"`
		ParentID  *uint `json:"parent_id"`
	}
	if err := json.Unmarshal(task.Payload, &scope); err != nil {
		return Run{}, fmt.Errorf("generator: decode task %d payload: %w", task.ID, err)
	}

	assetID := scope.ParentID
	if assetID == nil {
		assetID = scope.AssetID
	}
	return Run{
		ID:        RunID(task.ID),
		ProjectID: scope.ProjectID,
		AssetID:   assetID,
		Kind:      TaskType(task.Type),
		Status:    task.Status,
		Result:    task.Result,
		Error:     task.Error,
	}, nil
}

func matchesRunScope(run Run, filter *RunListFilter) bool {
	if run.ProjectID != filter.ProjectID {
		return false
	}
	if filter.AssetID == nil {
		return true
	}
	return run.AssetID != nil && *run.AssetID == *filter.AssetID
}
