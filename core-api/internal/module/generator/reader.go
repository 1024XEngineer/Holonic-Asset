package generator

import "context"

// RunReader builds Generator run projections from task records.
type RunReader interface {
	ListRuns(ctx context.Context, filter *RunListFilter) (*RunListPage, error)
}
