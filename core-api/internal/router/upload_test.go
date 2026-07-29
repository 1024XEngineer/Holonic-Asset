package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
)

func TestUploadsRouteReturnsPlaceholderResponse(t *testing.T) {
	uploadService := service.NewUploadService(nil)
	uploadHandler := handler.NewUploadHandler(uploadService)
	e := router.Register(nil, nil, nil, uploadHandler)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/uploads",
		strings.NewReader(`{"contentType":"image/png","contentLength":8}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "{\"objectKey\":\"\",\"objectURL\":\"\",\"uploadURL\":\"\",\"uploadToken\":\"\"}\n" {
		t.Fatalf("unexpected placeholder response: %s", recorder.Body.String())
	}
}

func TestUploadRoutesDoNotExposeUnsupportedOperations(t *testing.T) {
	uploadService := service.NewUploadService(nil)
	uploadHandler := handler.NewUploadHandler(uploadService)
	e := router.Register(nil, nil, nil, uploadHandler)

	routes := []string{
		"/api/v1/upload",
		"/api/v1/media/upload",
		"/api/v1/media/upload-target",
		"/api/v1/media/project-preview/upload-target",
		"/api/v1/media/project-preview/direct-upload",
		"/api/v1/media/generated-image/upload",
		"/api/v1/media/upload/complete",
		"/api/v1/media/download",
		"/api/v1/media/delete",
	}

	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, route, strings.NewReader("{}"))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			recorder := httptest.NewRecorder()

			e.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusNotFound {
				t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, recorder.Code, recorder.Body.String())
			}
		})
	}
}
