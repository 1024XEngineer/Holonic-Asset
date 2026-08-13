package generator

import (
	"encoding/json"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

type RunID uint
type RunListStatus string

const RunListStatusActive RunListStatus = "active"

// Run is a task-backed projection. ID and Status come directly from
// the task record; Generator does not persist a separate run state.
type Run struct {
	ID        RunID
	ProjectID uint
	AssetID   *uint
	Kind      TaskType
	Status    taskdomain.Status
	Result    json.RawMessage
	Error     string
}

// RunListQuery contains the client-facing list semantics. Engine expands
// status and asset scope into repository filters.
type RunListQuery struct {
	ProjectID uint
	AssetID   *uint
	Status    RunListStatus
	Limit     int
	Cursor    string
}

type RunListFilter struct {
	ProjectID        uint
	AssetID          *uint
	Statuses         []taskdomain.Status
	IncludeTaskTypes []TaskType
	ExcludeTaskTypes []TaskType
	Limit            int
	Cursor           string
}

type RunListPage struct {
	Runs       []Run
	NextCursor string
}

func ActiveTaskStatuses() []taskdomain.Status {
	return []taskdomain.Status{
		taskdomain.StatusPending,
		taskdomain.StatusProcessing,
		taskdomain.StatusFailed,
	}
}
