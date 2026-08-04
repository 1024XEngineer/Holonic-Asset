package router_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

func TestOpenAPIIncludesHTTPContract(t *testing.T) {
	server := router.Register(
		handler.NewHandler(nil),
		handler.NewProjectHandler(nil),
		handler.NewGenerationHandler(nil),
		handler.NewUploadHandler(nil),
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	var document struct {
		OpenAPI string `json:"openapi"`
		Paths   map[string]struct {
			Get  json.RawMessage `json:"get"`
			Post json.RawMessage `json:"post"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode OpenAPI document: %v", err)
	}
	if document.OpenAPI != "3.1.0" {
		t.Fatalf("expected OpenAPI 3.1.0, got %q", document.OpenAPI)
	}

	expectedMethods := map[string][]string{
		"/project/create":                        {"post"},
		"/project/reference/generate":            {"post"},
		"/project/list":                          {"get"},
		"/project/detail":                        {"get"},
		"/project/update":                        {"post"},
		"/project/delete":                        {"post"},
		"/projects/{project_id}/generation-runs": {"get", "post"},
		"/generation-runs/{run_id}":              {"get"},
		"/generation-runs/{run_id}/cancel":       {"post"},
		"/uploads":                               {"post"},
		"/projects/{project_id}/assets":          {"get"},
		"/asset/{asset_id}/records":              {"get"},
		"/asset/{asset_id}":                      {"get"},
		"/asset/save":                            {"post"},
		"/asset/copy":                            {"post"},
		"/asset/rollback":                        {"post"},
		"/asset/update":                          {"post"},
	}
	for path, methods := range expectedMethods {
		operation, ok := document.Paths[path]
		if !ok {
			t.Errorf("expected OpenAPI path %q", path)
			continue
		}
		for _, method := range methods {
			if method == "get" && len(operation.Get) == 0 {
				t.Errorf("expected GET operation for %q", path)
			}
			if method == "post" && len(operation.Post) == 0 {
				t.Errorf("expected POST operation for %q", path)
			}
		}
	}
}
