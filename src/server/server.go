package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"s3/types"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
	semaphore  chan struct{}
	storage    types.Storage
}

func New(addr string, storage types.Storage, maxConcurr int, logger *slog.Logger) *Server {
	mux := NewRouter(storage, logger)

	semaphore := make(chan struct{}, maxConcurr)

	return &Server{
		semaphore: semaphore,
		storage:   storage,
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      wrapHandler(mux, semaphore, logger),
			ReadTimeout:  30 * time.Minute,
			WriteTimeout: 30 * time.Minute,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

func wrapHandler(h http.Handler, sem chan struct{}, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			h.ServeHTTP(w, r)
		default:
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		}
	})
}

func (s *Server) ListenAndServe() error {
	errChan := make(chan error, 1)
	go func() {
		s.logger.Info("Starting server", "addr", s.httpServer.Addr)
		errChan <- s.httpServer.ListenAndServe()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case <-sigChan:
		s.logger.Info("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}
