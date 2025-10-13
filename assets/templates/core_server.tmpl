package core

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
)

// Server represents an HTTP server with graceful shutdown capabilities.
type Server struct {
	Router *chi.Mux
	Opts   ServerOpts
	Logger Logger
	srv    *http.Server
}

// Start starts the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	s.srv = &http.Server{
		Addr:    s.Opts.Port,
		Handler: s.Router,
	}

	s.Logger.Info(fmt.Sprintf("Starting server on %s...", s.Opts.Port))
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("could not listen and serve: %w", err)
	}
	return nil
}

// Stop gracefully stops the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}

	s.Logger.Info("Shutting down server...")
	return s.srv.Shutdown(ctx)
}

// Serve starts an HTTP server and handles graceful shutdown.
// It also calls the provided stops functions during shutdown.
func Serve(router *chi.Mux, opts ServerOpts, stops []func(context.Context) error, log Logger) {
	srv := &http.Server{
		Addr:    opts.Port,
		Handler: router,
	}

	go func() {
		log.Info(fmt.Sprintf("Starting server on %s...", opts.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("could not listen and serve: %v", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	Shutdown(srv, stops)
}

// ServerOpts holds server-related options.
type ServerOpts struct {
	Port string
}
