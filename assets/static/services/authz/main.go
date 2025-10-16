package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"

	"github.com/username/repo/pkg/lib/core"
	"github.com/username/repo/services/authz/internal/config"
	"github.com/username/repo/services/authz/internal/mongo"
	"github.com/username/repo/services/authz/internal/authz"
)

const (
	name    = "authz"
	version = "0.1.0"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml", "AUTHZ_", os.Args)
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
	
	// Repository setup
	roleRepo := mongo.NewRoleMongoRepo(xparams)
	deps = append(deps, roleRepo)
	
	grantRepo := mongo.NewGrantMongoRepo(xparams)
	deps = append(deps, grantRepo)
	
	// Policy engine setup
	policyEngine := authz.NewPolicyEngine(roleRepo, grantRepo)
	
	// Handler setup
	roleHandler := authz.NewRoleHandler(roleRepo, xparams)
	deps = append(deps, roleHandler)
	
	grantHandler := authz.NewGrantHandler(grantRepo, roleRepo, xparams)
	deps = append(deps, grantHandler)
	
	policyHandler := authz.NewPolicyHandler(policyEngine, xparams)
	deps = append(deps, policyHandler)

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
