package hm

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

// Serve starts an HTTP server and handles graceful shutdown.
func Serve(router *chi.Mux, opts ServerOpts, log Logger) {
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

	// Create a context with a timeout to allow outstanding requests to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error(fmt.Sprintf("server forced to shutdown: %v", err))
	}

	log.Info("Server exited gracefully.")
}

// ServerOpts holds server-related options.
type ServerOpts struct {
	Port string
}
