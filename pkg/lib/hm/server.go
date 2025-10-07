package hm

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
)

// Serve starts an HTTP server and handles graceful shutdown.
// It also calls the provided stops functions during shutdown.
func Serve(router *chi.Mux, opts ServerOpts, stops []func(context.Context) error, log Logger) {
	srv := &http.Server{
		Addr:    opts.Port,
		Handler: router,
	}

	// Initializing the server in a goroutine so that it won't block the graceful shutdown handling below
	go func() {
		log.Info(fmt.Sprintf("Starting server on %s...", opts.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("could not listen and serve: %v", err))
		}
	}()

	// Listen for OS signals to perform a graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Use internal hm.Shutdown for complete lifecycle management
	Shutdown(srv, stops)
}

// ServerOpts holds server-related options.
type ServerOpts struct {
	Port string
}
