package upload

import "context"

// Manager exposes upload use cases to transports and other modules.
type Manager interface {
	CreateUploadTarget(context.Context, *CreateUploadTargetRequest) (*UploadTarget, error)
}

type manager struct {
	store Store
}

func NewManager(store Store) Manager {
	return &manager{store: store}
}

func (m *manager) CreateUploadTarget(
	ctx context.Context,
	request *CreateUploadTargetRequest,
) (*UploadTarget, error) {
	if m.store == nil {
		return &UploadTarget{}, nil
	}

	return m.store.CreateUploadTarget(ctx, UploadRequest{
		ContentType:   request.ContentType,
		ContentLength: request.ContentLength,
	})
}

var _ Manager = (*manager)(nil)
