package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
)

type UploadHandler struct {
	manager upload.Manager
}

func NewUploadHandler(manager upload.Manager) *UploadHandler {
	return &UploadHandler{manager: manager}
}

func (h *UploadHandler) CreateUploadTarget(
	c context.Context,
	request dto.CreateUploadTargetRequest,
) (dto.SuccessResponse[dto.UploadTarget], error) {
	target, err := h.manager.CreateUploadTarget(c, &upload.CreateUploadTargetRequest{
		ContentType:   request.ContentType,
		ContentLength: request.ContentLength,
	})
	if err != nil {
		return dto.SuccessResponse[dto.UploadTarget]{}, uploadHandlerError(err)
	}
	if target == nil {
		target = &upload.UploadTarget{}
	}
	return dto.NewTypedSuccessResponse(dto.UploadTarget{
		ObjectKey:   target.ObjectKey,
		ObjectURL:   target.ObjectURL,
		UploadURL:   target.UploadURL,
		UploadToken: target.UploadToken,
	}), nil
}

func uploadHandlerError(err error) error {
	if errors.Is(err, upload.ErrInvalidUploadRequest) {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	return err
}
