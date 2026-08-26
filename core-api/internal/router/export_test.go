package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	exportmodule "github.com/1024XEngineer/Holonic-Asset/internal/module/export"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

type exportRouteManagerStub struct{}

func (exportRouteManagerStub) Create(context.Context, exportmodule.CreateRequest) (exportmodule.CreateResponse, error) {
	return exportmodule.CreateResponse{}, exportmodule.ErrNotFound
}

func (exportRouteManagerStub) Get(context.Context, uint) (exportmodule.ExportResponse, error) {
	return exportmodule.ExportResponse{}, exportmodule.ErrNotFound
}

func (exportRouteManagerStub) Handle(context.Context, *taskdomain.Task) (any, error) {
	return struct{}{}, nil
}

func TestExportRouteReturnsNotFoundForUnavailableAssetVersion(t *testing.T) {
	exportHandler := handler.NewExportHandler(exportRouteManagerStub{})
	server := router.RegisterWithExport(nil, nil, nil, nil, nil, exportHandler)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/asset/export", strings.NewReader(`{"assetId":7,"version":9}`))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, response.Code, response.Body.String())
	}
}
