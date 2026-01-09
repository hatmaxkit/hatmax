package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/hatmaxkit/hatmax/app"
	"github.com/hatmaxkit/hatmax/config"
	"github.com/hatmaxkit/hatmax/examples/ticked/internal"
	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/middleware"
)

//go:embed assets/*
var assetsFS embed.FS

const (
	name    = "ticked"
	version = "0.1.0"
)

func main() {
	cfg, err := config.Load("config.yaml", "TICKED_", os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot load config: %v\n", err)
		os.Exit(1)
	}

	logger := log.NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router := chi.NewRouter()
	router.Use(middleware.DefaultStack()...)
	app.ApplyRouterOptions(router, app.WithDebugRoutes())

	var deps []any

	svc, err := internal.New(assetsFS, cfg, logger)
	if err != nil {
		logger.Errorf("Cannot create service: %v", err)
		os.Exit(1)
	}

	deps = append(deps, svc)

	starts, stops, registrars := app.Setup(ctx, router, deps...)

	if err := app.Start(ctx, logger, starts, stops, registrars, router); err != nil {
		logger.Errorf("Cannot start %s(%s): %v", name, version, err)
		os.Exit(1)
	}

	logger.Infof("%s(%s) started successfully", name, version)

	go func() {
		logger.Infof("Server listening on %s", cfg.Server.Port)
		if err := app.Serve(router, cfg.Server.Port); err != nil {
			logger.Errorf("Server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-stop

	logger.Infof("Shutting down %s(%s)...", name, version)
	cancel()

	for i := len(stops) - 1; i >= 0; i-- {
		if err := stops[i](context.Background()); err != nil {
			logger.Errorf("Error stopping component: %v", err)
		}
	}

	fmt.Println("Goodbye!")
}
