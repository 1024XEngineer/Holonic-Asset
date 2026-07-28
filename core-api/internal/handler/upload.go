package handler

import (
	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type UploadHandler struct {
	service service.UploadService
}

func NewUploadHandler(uploadService service.UploadService) *UploadHandler {
	return &UploadHandler{service: uploadService}
}

func (h *UploadHandler) CreateUploadTarget(
	c *echox.Context,
	request dto.CreateUploadTargetRequest,
) (*dto.UploadTarget, error) {
	target, err := h.service.CreateUploadTarget(c, &service.CreateUploadTargetRequest{
		ContentType:   request.ContentType,
		ContentLength: request.ContentLength,
	})
	if err != nil {
		return nil, err
	}
	return &dto.UploadTarget{
		ObjectKey:   target.ObjectKey,
		ObjectURL:   target.ObjectURL,
		UploadURL:   target.UploadURL,
		UploadToken: target.UploadToken,
	}, nil
}
