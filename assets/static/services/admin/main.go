package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"

	"github.com/adrianpk/hatmax-ref/pkg/lib/core"
	"github.com/adrianpk/hatmax-ref/services/admin/internal/admin"
	"github.com/adrianpk/hatmax-ref/services/admin/internal/config"
)

const (
	name    = "admin"
	version = "0.1.0"
)

//go:embed assets
var assetsFS embed.FS

func main() {
	cfg, err := config.LoadConfig("config.yaml", "ADMIN", os.Args)
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

	fileServer := core.NewFileServer(assetsFS, logger)
	deps = append(deps, fileServer)

	tmplMgr := core.NewTemplateManager(assetsFS, logger)
	deps = append(deps, tmplMgr)

	userRepo := admin.NewFakeUserRepo()
	roleRepo := admin.NewFakeRoleRepo()
	grantRepo := admin.NewFakeGrantRepo(userRepo, roleRepo)

	adminHandler := admin.NewAdminHandler(tmplMgr, userRepo, roleRepo, grantRepo, xparams)
	deps = append(deps, adminHandler)

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
