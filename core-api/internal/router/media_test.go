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

func TestProjectPreviewUploadTargetRouteReturnsPlaceholderResponse(t *testing.T) {
	mediaService := service.NewMediaService()
	mediaHandler := handler.NewMediaHandler(mediaService)
	e := router.Register(nil, nil, nil, mediaHandler, nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/media/project-preview/upload-target",
		strings.NewReader(`{"contentType":"image/png"}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "{\"objectKey\":\"\",\"uploadURL\":\"\"}\n" {
		t.Fatalf("unexpected placeholder response: %s", recorder.Body.String())
	}
}

func TestMediaRoutesDoNotExposeUnsupportedOperations(t *testing.T) {
	mediaService := service.NewMediaService()
	mediaHandler := handler.NewMediaHandler(mediaService)
	e := router.Register(nil, nil, nil, mediaHandler, nil)

	routes := []string{
		"/api/v1/media/upload-target",
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
