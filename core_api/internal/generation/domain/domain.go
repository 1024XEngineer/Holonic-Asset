package domain

import (
	"encoding/json"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/task/domain"
)

type RunID uint
type PlanID uint
type StepID uint
type CandidateID uint

// GenerationRequest captures the business intent accepted by generation.
// Kind-specific parameters remain bounded data interpreted by generation.
type GenerationRequest struct {
	ProjectID              uint
	AssetID                uint
	Kind                   RequestKind
	Prompt                 string
	ReferenceMediaIDs      []string
	TargetAssetResourceIDs []uint
	Parameters             json.RawMessage
}

type Failure struct {
	Code    string
	Message string
}

type GenerationRun struct {
	ID                   RunID
	PlanningTaskID       *uint
	ProjectID            uint
	Request              GenerationRequest
	PlanID               PlanID
	Lifecycle            RunLifecycle
	ConfirmedCandidateID *CandidateID
	Failure              *Failure
}

// GenerationDetail is assembled from generation records and task state.
type GenerationDetail struct {
	Run        GenerationRun
	Steps      []StepDetail
	Candidates []Candidate
}

// StepDetail carries task-owned execution state without persisting a duplicate
// status in the generation module.
type StepDetail struct {
	Step       Step
	TaskStatus *taskdomain.Status
}

type Plan struct {
	ID    PlanID
	RunID RunID
	Steps []Step
}

type Step struct {
	ID           StepID
	TaskID       *uint
	RunID        RunID
	PlanID       PlanID
	Key          string
	Type         string
	Executor     StepExecutor
	Dependencies []StepID
	Parameters   json.RawMessage
}

type StepResult struct {
	MediaIDs []string
	Metadata json.RawMessage
}

type Candidate struct {
	ID       CandidateID
	RunID    RunID
	MediaIDs []string
}

// ConfirmCandidateCommand contains the asset target required to apply an
// accepted candidate without making the asset module query generation state.
type ConfirmCandidateCommand struct {
	RunID                  RunID
	CandidateID            CandidateID
	ProjectID              uint
	AssetID                uint
	Kind                   RequestKind
	TargetAssetResourceIDs []uint
	MediaIDs               []string
}
