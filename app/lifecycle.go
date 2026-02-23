package app

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hatmaxkit/hatmax/log"
)

// Startable represents a component that can be started.
// Components implementing this interface will have Start called during application startup.
type Startable interface {
	Start(context.Context) error
}

// Stoppable represents a component that can be stopped.
// Components implementing this interface will have Stop called during application shutdown.
type Stoppable interface {
	Stop(context.Context) error
}

// RouteRegistrar represents a component that registers HTTP routes.
// Components implementing this interface will have RegisterRoutes called during setup.
type RouteRegistrar interface {
	RegisterRoutes(chi.Router)
}

// Setup discovers component capabilities and builds startup/shutdown pipelines.
// It inspects each component for RouteRegistrar, Startable, and Stoppable interfaces,
// collecting start/stop functions and route registrars in order.
//
// Returns slices of start functions, stop functions, and route registrars to be executed by Start.
func Setup(ctx context.Context, r chi.Router, comps ...any) (
	starts []func(context.Context) error,
	stops []func(context.Context) error,
	registrars []RouteRegistrar,
) {
	for _, c := range comps {
		if rr, ok := c.(RouteRegistrar); ok {
			registrars = append(registrars, rr)
		}

		if s, ok := c.(Startable); ok {
			starts = append(starts, s.Start)
		}

		if st, ok := c.(Stoppable); ok {
			stops = append(stops, st.Stop)
		}
	}

	return
}

// Start executes startup functions in order with automatic rollback on failure.
// If any start function fails, already-started components are stopped in reverse order.
// After all components start successfully, routes are registered.
//
// This ensures transactional-like behavior: either all components start successfully
// or none remain running.
func Start(ctx context.Context, log log.Logger, starts []func(context.Context) error, stops []func(context.Context) error, registrars []RouteRegistrar, router chi.Router) error {
	for i, start := range starts {
		err := start(ctx)
		if err != nil {
			log.Errorf("error starting component #%d: %v", i, err)

			for j := i - 1; j >= 0; j-- {
				rErr := stops[j](context.Background())
				if rErr != nil {
					log.Errorf("error stopping component #%d during rollback: %v", j, rErr)
				}
			}

			return err
		}
	}

	for _, rr := range registrars {
		rr.RegisterRoutes(router)
	}

	return nil
}

// Serve starts the HTTP server and blocks until it's shut down.
func Serve(router chi.Router, port string) error {
	srv := &http.Server{
		Addr:    port,
		Handler: router,
	}

	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Shutdown performs graceful shutdown of the HTTP server and all components.
// It first attempts graceful server shutdown with a 5-second timeout,
// then stops all components in reverse order (LIFO).
//
// This ensures proper cleanup cascade: server stops accepting requests,
// then components clean up in reverse dependency order.
func Shutdown(srv *http.Server, log log.Logger, stops []func(context.Context) error) {
	log.Info("Shutting down gracefully, press Ctrl+C again to force")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := srv.Shutdown(shutdownCtx)
	if err != nil {
		log.Errorf("server shutdown failed: %v", err)
	}

	for i := len(stops) - 1; i >= 0; i-- {
		err = stops[i](context.Background())
		if err != nil {
			log.Errorf("error stopping component #%d: %v", i, err)
		}
	}
}
