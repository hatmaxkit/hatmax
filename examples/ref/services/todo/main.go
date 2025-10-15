package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"

	"github.com/adrianpk/hatmax-ref/pkg/lib/core"
	"github.com/adrianpk/hatmax-ref/services/todo/internal/config"
	"github.com/adrianpk/hatmax-ref/services/todo/internal/sqlite"
	"github.com/adrianpk/hatmax-ref/services/todo/internal/todo"
)

const (
	name    = "todo"
	version = "0.1.0"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml", "APP", os.Args)
	if err != nil {
		log.Fatalf("Cannot setup %s(%s): %v", name, version, err)
	}

	logger := core.NewLogger(cfg.Log.Level)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	xparams := config.XParams{
		Log: logger,
		Cfg: cfg,
	}

	router := chi.NewRouter()

	var deps []any
	ListRepo := sqlite.NewListSQLiteRepo(xparams)
	deps = append(deps, ListRepo)

	ListHandler := todo.NewListHandler(ListRepo, xparams)
	deps = append(deps, ListHandler)

	starts, stops := core.Setup(ctx, router, deps...)

	if err := core.Start(ctx, starts, stops); err != nil {
		logger.Errorf("Cannot start %s(%s): %v", name, version, err)
		log.Fatal(err)
	}

	logger.Infof("%s(%s) started successfully", name, version)

	go func() {
		core.Serve(router, core.ServerOpts{Port: cfg.Server.Port}, stops, logger)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-stop

	logger.Infof("Shutting down %s(%s)...", name, version)
	cancel()
}
