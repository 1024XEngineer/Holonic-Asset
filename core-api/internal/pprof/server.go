// Package pprof exposes Go runtime profiles on a loopback-only HTTP server.
package pprof

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	netpprof "net/http/pprof"
	"sync"
	"time"
)

const Address = "127.0.0.1:6060"

type Server struct {
	httpServer *http.Server

	mu      sync.Mutex
	started bool
	closed  bool
	wg      sync.WaitGroup
}

func New() *Server {
	return &Server{}
}

func (s *Server) Start(ctx context.Context) error {
	if s == nil {
		return errors.New("pprof: server is nil")
	}
	if ctx == nil {
		return errors.New("pprof: start context is required")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("pprof: start context: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("pprof: server is closed")
	}
	if s.started {
		return errors.New("pprof: server is already started")
	}

	listener, err := net.Listen("tcp", Address)
	if err != nil {
		return fmt.Errorf("pprof: listen on %s: %w", Address, err)
	}

	s.httpServer = &http.Server{
		Handler:           handler(),
		ReadHeaderTimeout: 2 * time.Second,
	}
	s.started = true
	server := s.httpServer
	s.wg.Go(func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("pprof: serve: %v", err)
		}
	})
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.wg.Wait()
		return nil
	}
	s.closed = true
	server := s.httpServer
	s.mu.Unlock()

	var err error
	if server != nil {
		err = server.Shutdown(ctx)
	}
	s.wg.Wait()
	return err
}

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", netpprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", netpprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", netpprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", netpprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", netpprof.Trace)
	for _, name := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		mux.Handle("/debug/pprof/"+name, netpprof.Handler(name))
	}
	return mux
}
