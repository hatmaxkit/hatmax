package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/adrianpk/hatmax/pkg/lib/hm"
	"github.com/go-chi/chi/v5"

	"github.com/adrianpk/hatmax-ref/services/todo/internal/config"
	"github.com/adrianpk/hatmax-ref/services/todo/internal/sqlite"
	"github.com/adrianpk/hatmax-ref/services/todo/internal/todo"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml", "APP", os.Args)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	logger := hm.NewLogger(cfg.Log.Level)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	xparams := config.XParams{
		Log: logger,
		Cfg: cfg,
	}

	router := chi.NewRouter()

	var deps []any

	ItemRepo := sqlite.NewItemRepo(xparams)
	deps = append(deps, ItemRepo)

	ItemHandler := todo.NewItemHandler(ItemRepo, xparams)
	deps = append(deps, ItemHandler)

	starts, stops := hm.Setup(ctx, router, deps...)

	if err := hm.Start(ctx, starts, stops); err != nil {
		log.Fatal(err)
	}

	hm.Serve(router, hm.ServerOpts{Port: cfg.Server.Port}, stops, logger)
}
