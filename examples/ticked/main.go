package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/db"
	auditfeat "hatmax.adrianpk.com/examples/ticked/internal/feat/audit"
	authfeat "hatmax.adrianpk.com/examples/ticked/internal/feat/auth"
	listfeat "hatmax.adrianpk.com/examples/ticked/internal/feat/list"
	tickedweb "hatmax.adrianpk.com/examples/ticked/internal/web"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/middleware"
	"hatmax.adrianpk.com/pubsub/postgres"
	"hatmax.adrianpk.com/ui"
	"hatmax.adrianpk.com/web"
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

	router := app.NewRouter(logger, app.WithPing(), app.WithDebugRoutes())
	router.Use(middleware.DefaultStack()...)

	database := db.New(assetsFS, "postgres", cfg, logger)
	migrator := db.NewMigrator(database, assetsFS, "postgres", logger)
	tmplMgr := web.NewTemplateManager(assetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	broker := postgres.NewBroker(database, postgres.DefaultConfig(), logger)

	auditStore := auditfeat.NewPostgresStore(database)
	listStore := listfeat.NewPostgresStore(database)
	authQueries := authfeat.NewQueries(database)

	auditSvc := auditfeat.NewService(broker, auditStore, logger)
	baseAuthSvc := auth.NewService(authQueries, cfg, logger)
	authSvc := authfeat.NewService(baseAuthSvc, authQueries, logger)
	listSvc := listfeat.NewService(listStore, broker, logger)

	webHandler := tickedweb.NewHandler(assetsFS, authSvc, authQueries, listSvc, auditStore, tmplMgr, logger)

	deps := []any{
		database,
		migrator,
		tmplMgr,
		broker,
		auditStore,
		listStore,
		authQueries,
		auditSvc,
		webHandler,
	}

	starts, stops, registrars := app.Setup(ctx, router, deps...)

	err = app.Start(ctx, logger, starts, stops, registrars, router)
	if err != nil {
		logger.Errorf("Cannot start %s(%s): %v", name, version, err)
		os.Exit(1)
	}

	logger.Infof("%s(%s) started successfully", name, version)

	go func() {
		logger.Infof("Server listening on %s", cfg.Server.Port)

		err = app.Serve(router, cfg.Server.Port)
		if err != nil {
			logger.Errorf("Server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-stop

	logger.Infof("Shutting down %s(%s)...", name, version)
	cancel()

	for i := len(stops) - 1; i >= 0; i-- {
		err = stops[i](context.Background())
		if err != nil {
			logger.Errorf("Error stopping component: %v", err)
		}
	}

	fmt.Println("Goodbye!")
}
