// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/db"
	auditfeat "hatmax.adrianpk.com/examples/ticked/internal/feat/audit"
	authfeat "hatmax.adrianpk.com/examples/ticked/internal/feat/auth"
	listfeat "hatmax.adrianpk.com/examples/ticked/internal/feat/list"
	tickedweb "hatmax.adrianpk.com/examples/ticked/internal/web"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/mailer"
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

	err = cfg.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot validate config: %v\n", err)
		os.Exit(1)
	}

	logger := log.NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router := app.NewRouter(
		logger,
		app.WithMiddleware(middleware.DefaultStack()...),
		app.WithPing(),
		app.WithDebugRoutes(),
	)

	database := db.New(assetsFS, db.Postgres, cfg, logger)
	migrator := db.NewMigrator(database, assetsFS, db.Postgres, logger)
	tmplMgr := web.NewTemplateManager(assetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	broker := postgres.NewBroker(database, postgres.DefaultConfig(), logger)

	auditStore := auditfeat.NewPostgresStore(database)
	listStore := listfeat.NewPostgresStore(database)
	authQueries := authfeat.NewQueries(database)

	auditSvc := auditfeat.NewService(broker, auditStore, logger)

	baseAuthSvc, err := auth.NewService(authQueries, cfg, authfeat.NewPasswordChecker(), logger)
	if err != nil {
		logger.Errorf("Cannot initialize authentication: %v", err)
		os.Exit(1)
	}

	enrollmentSvc, err := auth.NewWebAuthnService(baseAuthSvc, authQueries, cfg.Authenticator)
	if err != nil {
		logger.Errorf("Cannot initialize authenticators: %v", err)
		os.Exit(1)
	}

	enrollmentHandler, err := tickedweb.NewEnrollmentHandler(enrollmentSvc, auth.AccessRequirement{Proof: auth.RequirePhishingResistantMFA, Revision: "enrollment-v1", MaxAge: 5 * time.Minute})
	if err != nil {
		logger.Errorf("Cannot initialize enrollment: %v", err)
		os.Exit(1)
	}

	strong := authfeat.PasswordRequirement()
	strong.Proof = auth.RequirePhishingResistantMFA
	strong.MaxAge = 5 * time.Minute

	assertionHandler, err := tickedweb.NewWebAuthnHandler(enrollmentSvc, baseAuthSvc, strong)
	if err != nil {
		logger.Errorf("Cannot initialize WebAuthn: %v", err)
		os.Exit(1)
	}

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
		enrollmentHandler,
		assertionHandler,
	}

	var fallbackSvc *auth.FallbackService

	// Explicit application-owned encryption material enables fallback routes.
	keyID, keyText := os.Getenv("TICKED_TOTP_KEY_ID"), os.Getenv("TICKED_TOTP_KEY")
	if keyID != "" || keyText != "" {
		key, keyErr := base64.StdEncoding.Strict().DecodeString(keyText)
		if keyErr != nil || len(key) != 32 || base64.StdEncoding.EncodeToString(key) != keyText || keyID == "" {
			logger.Errorf("Invalid fallback key configuration")
			os.Exit(1)
		}

		var initErr error

		fallbackSvc, initErr = auth.NewFallbackService(baseAuthSvc, authQueries, config.FallbackConfig{Limits: cfg.Authenticator, Issuer: "Ticked"}, auth.SeedKeys{Active: keyID, Keys: map[string][]byte{keyID: key}})
		clear(key)

		if initErr != nil {
			logger.Errorf("Cannot initialize fallback authentication: %v", initErr)
			os.Exit(1)
		}

		lower := authfeat.PasswordRequirement()
		lower.Proof = auth.RequireMFA
		lower.MaxAge = 5 * time.Minute

		fallbackHandler, initErr := tickedweb.NewFallbackHandler(fallbackSvc, baseAuthSvc, lower, strong)
		if initErr != nil {
			logger.Errorf("Cannot initialize fallback routes: %v", initErr)
			os.Exit(1)
		}

		deps = append(deps, fallbackHandler)
	}

	factorSvc, err := auth.NewFactorService(baseAuthSvc, authQueries, enrollmentSvc.AuthenticatorService, fallbackSvc)
	if err != nil {
		logger.Errorf("Cannot initialize factor management: %v", err)
		os.Exit(1)
	}

	factorHandler, err := tickedweb.NewFactorHandler(factorSvc, auth.FactorPolicy{Management: strong, Access: auth.RequirePassword}, fallbackSvc != nil)
	if err != nil {
		logger.Errorf("Cannot initialize factor routes: %v", err)
		os.Exit(1)
	}

	deps = append(deps, factorHandler)

	// Explicit application-owned origin and active mail enable mailbox routes.
	if origin := os.Getenv("TICKED_RECOVERY_ORIGIN"); origin != "" {
		if !cfg.Mailer.Enabled || cfg.Mailer.Mode != mailer.ModeActive {
			logger.Errorf("Mailbox verification requires active mail delivery")
			os.Exit(1)
		}

		recoverySvc, initErr := auth.NewRecoveryService(baseAuthSvc, authQueries, cfg.Recovery, "mailbox-v1")
		if initErr != nil {
			logger.Errorf("Cannot initialize mailbox verification")
			os.Exit(1)
		}

		delivery, initErr := authfeat.NewMailboxDelivery(recoverySvc, authQueries, mailer.New(cfg, logger), logger, origin, cfg.Authenticator.LocalhostDevelopment)
		if initErr != nil {
			logger.Errorf("Cannot initialize mailbox delivery")
			os.Exit(1)
		}

		handler, initErr := tickedweb.NewMailboxHandler(delivery)
		if initErr != nil {
			logger.Errorf("Cannot initialize mailbox routes")
			os.Exit(1)
		}

		deps = append(deps, handler)
	}

	starts, stops, registrars := app.Setup(ctx, router, deps...)

	err = app.Start(ctx, logger, starts, stops, registrars, router)
	if err != nil {
		logger.Errorf("Cannot start %s(%s): %v", name, version, err)
		os.Exit(1)
	}

	logger.Infof("%s(%s) started successfully", name, version)

	server := &http.Server{Addr: cfg.Server.Port, Handler: router}

	go func() {
		logger.Infof("Server listening on %s", cfg.Server.Port)

		serveErr := app.Serve(server)
		if serveErr != nil {
			logger.Errorf("Server error: %v", serveErr)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-stop

	logger.Infof("Shutting down %s(%s)...", name, version)
	app.Shutdown(server, logger, stops)
}
