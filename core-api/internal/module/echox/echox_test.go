package echox_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
)

type keyType string

func TestContextPreservesOriginalRequestContextAfterEchoReset(t *testing.T) {
	e := echo.New()
	reqCtx, cancel := context.WithTimeout(context.WithValue(context.Background(), keyType("traceID"), "trace-123"), 5*time.Second)
	defer cancel()

	httpReq := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(reqCtx)
	echoCtx := e.NewContext(httpReq, httptest.NewRecorder())

	handlerExecuted := make(chan struct{})
	wrapped := echox.Wrap(func(c *echox.Context) (map[string]string, error) {
		go func(ctx *echox.Context) {
			defer close(handlerExecuted)
			<-time.After(10 * time.Millisecond)

			if val, ok := ctx.Value(keyType("traceID")).(string); !ok || val != "trace-123" {
				t.Errorf("expected traceID trace-123 after Echo reset, got %v", ctx.Value(keyType("traceID")))
			}
			if _, ok := ctx.Deadline(); !ok {
				t.Error("expected deadline from preserved context")
			}
			if err := ctx.Err(); err != nil {
				t.Errorf("unexpected context error: %v", err)
			}
		}(c)

		return map[string]string{"status": "ok"}, nil
	})

	if err := wrapped(echoCtx); err != nil {
		t.Fatalf("execute wrapped handler: %v", err)
	}

	echoCtx.Reset(httptest.NewRequest(http.MethodGet, "/other", nil), httptest.NewRecorder())
	<-handlerExecuted
}

func TestContextPropagatesCancellationFromOriginalContext(t *testing.T) {
	e := echo.New()
	reqCtx, cancel := context.WithCancel(context.Background())
	httpReq := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(reqCtx)
	echoCtx := e.NewContext(httpReq, httptest.NewRecorder())

	var captured *echox.Context
	wrapped := echox.Wrap(func(c *echox.Context) (map[string]string, error) {
		captured = c
		return map[string]string{"status": "ok"}, nil
	})

	if err := wrapped(echoCtx); err != nil {
		t.Fatalf("execute wrapped handler: %v", err)
	}

	cancel()

	select {
	case <-captured.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected captured context to be done after cancellation")
	}

	if !errors.Is(captured.Err(), context.Canceled) {
		t.Fatalf("expected context.Canceled error, got %v", captured.Err())
	}
}
