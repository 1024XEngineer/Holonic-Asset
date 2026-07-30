package generation

import (
	"encoding/json"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

type RunID uint
type RunListStatus string

const RunListStatusActive RunListStatus = "active"

// GenerationRequest captures the business intent accepted by generation.
// Kind-specific parameters remain bounded data interpreted by generation.
type GenerationRequest struct {
	ProjectID         uint
	AssetID           *uint
	Kind              TaskType
	Prompt            string
	ReferenceMediaIDs []string
	TargetAssetPaths  []string
	Parameters        json.RawMessage
}

// GenerationRun is a task-backed projection. ID and Status come directly from
// the task record; generation does not persist a separate run state.
type GenerationRun struct {
	ID        RunID
	ProjectID uint
	AssetID   *uint
	Kind      TaskType
	Status    taskdomain.Status
	Result    json.RawMessage
	Error     string
}

// RunListQuery contains the client-facing list semantics. The service expands
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
	Runs       []GenerationRun
	NextCursor string
}

func ActiveTaskStatuses() []taskdomain.Status {
	return []taskdomain.Status{
		taskdomain.StatusPending,
		taskdomain.StatusProcessing,
	}
}
