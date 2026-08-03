package upload

import (
	"context"
	"fmt"
	"strings"
)

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
	if request == nil {
		return nil, fmt.Errorf("%w: request is required", ErrInvalidUploadRequest)
	}
	if strings.TrimSpace(request.ContentType) == "" {
		return nil, fmt.Errorf("%w: content type is required", ErrInvalidUploadRequest)
	}
	if request.ContentLength <= 0 {
		return nil, fmt.Errorf("%w: content length must be positive", ErrInvalidUploadRequest)
	}
	if m.store == nil {
		return &UploadTarget{}, nil
	}

	return m.store.CreateUploadTarget(ctx, UploadRequest{
		ContentType:   request.ContentType,
		ContentLength: request.ContentLength,
	})
}

var _ Manager = (*manager)(nil)
