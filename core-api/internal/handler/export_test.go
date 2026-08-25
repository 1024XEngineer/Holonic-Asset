package handler_test

import (
	"context"
	"errors"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	exportmodule "github.com/1024XEngineer/Holonic-Asset/internal/module/export"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

type exportManagerStub struct {
	createResponse exportmodule.CreateResponse
	createError    error
	getResponse    exportmodule.ExportResponse
	getError       error
}

func (s exportManagerStub) Create(context.Context, exportmodule.CreateRequest) (exportmodule.CreateResponse, error) {
	return s.createResponse, s.createError
}

func (s exportManagerStub) Get(context.Context, uint) (exportmodule.ExportResponse, error) {
	return s.getResponse, s.getError
}

func (s exportManagerStub) Handle(context.Context, *taskdomain.Task) (any, error) {
	return struct{}{}, nil
}

func TestExportHandlerCreateForwardsSuccess(t *testing.T) {
	manager := exportManagerStub{createResponse: exportmodule.CreateResponse{ExportID: 42, TaskID: 42, Status: "pending"}}
	exportHandler := handler.NewExportHandler(manager)

	response, err := exportHandler.Create(context.Background(), dto.CreateAssetExportRequest{AssetID: 7, Version: 3})
	if err != nil {
		t.Fatalf("create export: %v", err)
	}
	if response.Data.ExportID != 42 || response.Data.TaskID != 42 || response.Data.Status != "pending" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestExportHandlerMapsClientErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
	}{
		{name: "invalid request", err: exportmodule.ErrInvalidRequest, code: 400},
		{name: "unsupported asset", err: exportmodule.ErrUnsupportedAsset, code: 400},
		{name: "missing asset or version", err: errors.Join(exportmodule.ErrNotFound, errors.New("asset 7 version 9")), code: 404},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exportHandler := handler.NewExportHandler(exportManagerStub{createError: test.err})
			response, err := exportHandler.Create(context.Background(), dto.CreateAssetExportRequest{AssetID: 7})
			if response != (dto.SuccessResponse[dto.CreateAssetExportResponse]{}) {
				t.Fatalf("expected empty response, got %+v", response)
			}
			var httpError *echo.HTTPError
			if !errors.As(err, &httpError) {
				t.Fatalf("expected HTTP error, got %v", err)
			}
			if httpError.Code != test.code {
				t.Fatalf("expected status %d, got %d", test.code, httpError.Code)
			}
			if !errors.Is(httpError.Internal, exportmodule.ErrNotFound) && test.code == 404 {
				t.Fatalf("expected internal not-found error, got %v", httpError.Internal)
			}
		})
	}
}

func TestExportHandlerGetPropagatesManagerError(t *testing.T) {
	wantErr := errors.New("task lookup failed")
	exportHandler := handler.NewExportHandler(exportManagerStub{getError: wantErr})

	response, err := exportHandler.Get(context.Background(), 42)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if response != (dto.SuccessResponse[dto.AssetExportResponse]{}) {
		t.Fatalf("expected empty response, got %+v", response)
	}
}
