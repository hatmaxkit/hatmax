// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/log"
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
// Components implementing this interface have RegisterRoutes called after startup.
type RouteRegistrar interface {
	RegisterRoutes(chi.Router)
}

// StartupStep pairs a component's Start function with its optional Stop function.
// Stop is nil for a component that does not implement Stoppable. A failing Start
// owns cleanup of its own partial initialization; only completed steps roll back.
type StartupStep struct {
	Start func(context.Context) error
	Stop  func(context.Context) error
}

// Setup discovers component capabilities and builds startup/shutdown pipelines.
// It inspects each component for RouteRegistrar, Startable, and Stoppable interfaces,
// collecting paired startup steps, shutdown functions, and route registrars in order.
//
// Start consumes startup steps and registrars; Shutdown consumes all stop functions,
// including those belonging to components without a Start capability.
func Setup(ctx context.Context, r chi.Router, comps ...any) (
	starts []StartupStep,
	stops []func(context.Context) error,
	registrars []RouteRegistrar,
) {
	for _, c := range comps {
		if rr, ok := c.(RouteRegistrar); ok {
			registrars = append(registrars, rr)
		}

		var stop func(context.Context) error
		if st, ok := c.(Stoppable); ok {
			stop = st.Stop
			stops = append(stops, stop)
		}

		if s, ok := c.(Startable); ok {
			starts = append(starts, StartupStep{Start: s.Start, Stop: stop})
		}
	}

	return
}

// Start executes startup steps in order and registers routes after startup.
// On failure it stops only completed steps with a Stop function, in reverse order,
// using context.Background(). Stop errors are logged; the original Start error is returned.
// The stop slice is retained for existing Setup-based calls but is not used for rollback.
func Start(ctx context.Context, log log.Logger, starts []StartupStep, _ []func(context.Context) error, registrars []RouteRegistrar, router chi.Router) error {
	for i, step := range starts {
		err := step.Start(ctx)
		if err != nil {
			log.Errorf("error starting component #%d: %v", i, err)

			for j := i - 1; j >= 0; j-- {
				if starts[j].Stop == nil {
					continue
				}

				rErr := starts[j].Stop(context.Background())
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

// Serve listens on the caller's server and blocks until shutdown. Pass the same
// instance to Shutdown. Configure it before calling Serve; do not mutate it while
// serving. Zero header and idle timeouts default to 5s and 60s, respectively;
// negative values are rejected. ReadTimeout and WriteTimeout are left unchanged
// so callers retain body and streaming policy. ErrServerClosed is a normal exit.
func Serve(srv *http.Server) error {
	if srv == nil {
		return fmt.Errorf("app: HTTP server is required")
	}

	if srv.ReadHeaderTimeout < 0 || srv.IdleTimeout < 0 {
		return fmt.Errorf("app: HTTP header and idle timeouts must not be negative")
	}

	if srv.ReadHeaderTimeout == 0 {
		srv.ReadHeaderTimeout = 5 * time.Second
	}

	if srv.IdleTimeout == 0 {
		srv.IdleTimeout = time.Minute
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
