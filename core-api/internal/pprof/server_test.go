package pprof

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerServesPprofEndpoints(t *testing.T) {
	for _, path := range []string{
		"/debug/pprof/",
		"/debug/pprof/allocs",
		"/debug/pprof/block",
		"/debug/pprof/cmdline",
		"/debug/pprof/goroutine?debug=1",
		"/debug/pprof/heap",
		"/debug/pprof/mutex",
		"/debug/pprof/symbol",
		"/debug/pprof/threadcreate",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			res := httptest.NewRecorder()

			handler().ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("unexpected status: %d", res.Code)
			}
		})
	}
}

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

func TestServerStartAndShutdown(t *testing.T) {
	server := New()
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("start server: %v", err)
	}

	if err := waitForPprof(); err != nil {
		_ = server.Shutdown(context.Background())
		t.Fatal(err)
	}

	response, err := http.Get("http://" + Address + "/debug/pprof/")
	if err != nil {
		_ = server.Shutdown(context.Background())
		t.Fatalf("request pprof index: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected pprof status: %d", response.StatusCode)
	}
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Fatalf("read pprof response: %v", err)
	}

	if err := server.Start(context.Background()); err == nil {
		t.Fatal("expected second start to be rejected")
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown server: %v", err)
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown server twice: %v", err)
	}
	if err := server.Start(context.Background()); err == nil {
		t.Fatal("expected start after shutdown to be rejected")
	}
}

func TestServerStartRejectsInvalidContext(t *testing.T) {
	var nilContext context.Context
	if err := New().Start(nilContext); err == nil {
		t.Fatal("expected nil context to be rejected")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := New().Start(ctx); err == nil {
		t.Fatal("expected canceled context to be rejected")
	}
}

func TestServerStartRejectsUnavailableAddress(t *testing.T) {
	listener, err := listenForTest()
	if err != nil {
		t.Fatalf("reserve pprof address: %v", err)
	}
	defer func() { _ = listener.Close() }()

	if err := New().Start(context.Background()); err == nil {
		t.Fatal("expected unavailable address to be rejected")
	}
}

func TestServerShutdownBeforeStart(t *testing.T) {
	server := New()
	var nilContext context.Context
	if err := server.Shutdown(nilContext); err != nil {
		t.Fatalf("shutdown before start: %v", err)
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown before start twice: %v", err)
	}
}

func TestNilServerLifecycle(t *testing.T) {
	var server *Server
	if err := server.Start(context.Background()); err == nil {
		t.Fatal("expected nil server to be rejected")
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown nil server: %v", err)
	}
}

func listenForTest() (net.Listener, error) {
	return net.Listen("tcp", Address)
}

func waitForPprof() error {
	client := &http.Client{Timeout: 100 * time.Millisecond}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get("http://" + Address + "/debug/pprof/")
		if err == nil {
			_ = response.Body.Close()
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return context.DeadlineExceeded
}
