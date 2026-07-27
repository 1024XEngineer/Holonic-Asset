package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
)

func TestTaxonomyRoutesReturnPlaceholderResponses(t *testing.T) {
	discoveryService := service.NewAssetDiscoveryService()
	taxonomyHandler := handler.NewTaxonomyHandler(discoveryService)
	e := router.Register(nil, nil, nil, nil, taxonomyHandler)

	routes := []string{
		"/api/v1/projects/42/assets/search?q=orchard&tags=hero&types=character",
	}

	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, route, nil)
			recorder := httptest.NewRecorder()

			e.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf(
					"expected status %d, got %d: %s",
					http.StatusOK,
					recorder.Code,
					recorder.Body.String(),
				)
			}
			if recorder.Body.String() != "{\"assets\":[]}\n" {
				t.Fatalf("unexpected placeholder response: %s", recorder.Body.String())
			}
		})
	}
}

func TestTaxonomyRoutesDoNotExposeInternalOperations(t *testing.T) {
	discoveryService := service.NewAssetDiscoveryService()
	taxonomyHandler := handler.NewTaxonomyHandler(discoveryService)
	e := router.Register(nil, nil, nil, nil, taxonomyHandler)

	routes := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/taxonomy/assets/search"},
		{method: http.MethodGet, path: "/api/v1/taxonomy/assets/filter"},
		{method: http.MethodGet, path: "/api/v1/taxonomy/assets/related-by-tags"},
		{method: http.MethodGet, path: "/api/v1/taxonomy/assets/related"},
		{method: http.MethodGet, path: "/api/v1/projects/42/assets/9/related"},
		{method: http.MethodPost, path: "/api/v1/taxonomy/tags"},
		{method: http.MethodPost, path: "/api/v1/taxonomy/assets/tags"},
		{method: http.MethodDelete, path: "/api/v1/taxonomy/tags/1"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			request := httptest.NewRequest(route.method, route.path, nil)
			recorder := httptest.NewRecorder()

			e.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusNotFound {
				t.Fatalf(
					"expected status %d, got %d: %s",
					http.StatusNotFound,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestTaxonomyRoutesRejectInvalidIDs(t *testing.T) {
	discoveryService := service.NewAssetDiscoveryService()
	taxonomyHandler := handler.NewTaxonomyHandler(discoveryService)
	e := router.Register(nil, nil, nil, nil, taxonomyHandler)

	routes := []string{
		"/api/v1/projects/0/assets/search",
		"/api/v1/projects/invalid/assets/search",
	}

	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, route, nil)
			recorder := httptest.NewRecorder()

			e.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status %d, got %d: %s",
					http.StatusBadRequest,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}
