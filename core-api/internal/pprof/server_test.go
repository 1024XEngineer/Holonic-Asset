package pprof

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesPprofIndex(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	res := httptest.NewRecorder()

	handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "profile") {
		t.Fatal("expected pprof index to list profiles")
	}
}

func TestServerStartRequiresContext(t *testing.T) {
	var ctx context.Context
	if err := New().Start(ctx); err == nil {
		t.Fatal("expected nil context to be rejected")
	}
}

func TestServerShutdownBeforeStart(t *testing.T) {
	server := New()
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown before start: %v", err)
	}
}
