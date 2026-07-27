package service

import (
	"context"
)

// ProjectPreviewUploadService creates presigned R2 upload targets for Project previews.
type ProjectPreviewUploadService interface {
	CreateProjectPreviewUploadTarget(
		ctx context.Context,
		request *CreateProjectPreviewUploadRequest,
	) (*ProjectPreviewUploadTarget, error)
}

// MediaService provides the Project preview upload application skeleton.
type MediaService struct{}

// NewMediaService creates the Media application service used by the HTTP handler.
func NewMediaService() *MediaService {
	return &MediaService{}
}

func (*MediaService) CreateProjectPreviewUploadTarget(
	context.Context,
	*CreateProjectPreviewUploadRequest,
) (*ProjectPreviewUploadTarget, error) {
	return &ProjectPreviewUploadTarget{}, nil
}

var _ ProjectPreviewUploadService = (*MediaService)(nil)
