package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

func main() {
	output := flag.String("output", "../openapi/openapi.json", "OpenAPI output path")
	flag.Parse()

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
		exitf("generate OpenAPI: unexpected status %d: %s", response.Code, response.Body.String())
	}

	var formatted bytes.Buffer
	if err := jsonIndent(&formatted, response.Body.Bytes()); err != nil {
		exitf("generate OpenAPI: %v", err)
	}
	if err := formatted.WriteByte('\n'); err != nil {
		exitf("generate OpenAPI: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		exitf("create OpenAPI output directory: %v", err)
	}
	if err := os.WriteFile(*output, formatted.Bytes(), 0o644); err != nil {
		exitf("write OpenAPI document: %v", err)
	}
}

func jsonIndent(output *bytes.Buffer, document []byte) error {
	document = bytes.TrimSpace(document)
	return json.Indent(output, document, "", "  ")
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
