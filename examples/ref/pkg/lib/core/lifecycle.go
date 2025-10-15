package core

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type Startable interface {
	Start(context.Context) error
}

type Stoppable interface {
	Stop(context.Context) error
}

type RouteRegistrar interface {
	RegisterRoutes(chi.Router)
}

func Setup(ctx context.Context, r chi.Router, comps ...any) (
	starts []func(context.Context) error,
	stops []func(context.Context) error,
) {
	for _, c := range comps {
		if rr, ok := c.(RouteRegistrar); ok {
			rr.RegisterRoutes(r)
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

func Start(ctx context.Context, starts []func(context.Context) error, stops []func(context.Context) error) error {
	for i, start := range starts {
		if err := start(ctx); err != nil {
			log.Printf("error starting component #%d: %v", i, err)
			for j := i - 1; j >= 0; j-- {
				if rErr := stops[j](context.Background()); rErr != nil {
					log.Printf("error stopping component #%d during rollback: %v", j, rErr)
				}
			}
			return err
		}
	}
	return nil
}

func Shutdown(srv *http.Server, stops []func(context.Context) error) {
	log.Println("Shutting down gracefully, press Ctrl+C again to force")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %+v", err)
	}

	for i := len(stops) - 1; i >= 0; i-- {
		if err := stops[i](context.Background()); err != nil {
			log.Printf("error stopping component #%d: %v", i, err)
		}
	}
}
