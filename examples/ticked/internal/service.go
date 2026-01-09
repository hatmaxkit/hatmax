// Package internal provides the service coordinator for ticked.
package internal

import (
	"context"
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hatmaxkit/hatmax/auth"
	"github.com/hatmaxkit/hatmax/config"
	"github.com/hatmaxkit/hatmax/db"
	adminfeat "github.com/hatmaxkit/hatmax/examples/ticked/internal/feat/admin"
	authfeat "github.com/hatmaxkit/hatmax/examples/ticked/internal/feat/auth"
	listfeat "github.com/hatmaxkit/hatmax/examples/ticked/internal/feat/list"
	"github.com/hatmaxkit/hatmax/examples/ticked/internal/sqlcgen"
	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/middleware"
	"github.com/hatmaxkit/hatmax/web"
)

// Service coordinates all ticked components and manages lifecycle.
type Service struct {
	assetsFS embed.FS
	cfg      *config.Config
	log      log.Logger
	database *db.Database
	tmplMgr  *web.TemplateManager

	authSvc      *auth.Service
	authHandler  *authfeat.Handler
	listHandler  *listfeat.Handler
	adminHandler *adminfeat.Handler
}

// New creates a new Service with the given configuration.
func New(assetsFS embed.FS, cfg *config.Config, log log.Logger) (*Service, error) {
	s := &Service{
		assetsFS: assetsFS,
		cfg:      cfg,
		log:      log,
	}

	s.database = db.New(assetsFS, "postgres", cfg, log)
	s.tmplMgr = web.NewTemplateManager(assetsFS, log)

	return s, nil
}

// Start initializes the service components.
func (s *Service) Start(ctx context.Context) error {
	if err := s.database.Start(ctx); err != nil {
		return err
	}

	if err := s.tmplMgr.Start(ctx); err != nil {
		return err
	}

	sqlcQueries := sqlcgen.New(s.database.GetDB())

	authQueries := authfeat.NewQueries(sqlcQueries)
	s.authSvc = auth.NewService(authQueries, s.cfg, s.log)

	listStore := listfeat.NewPostgresStore(s.database.GetDB())
	listSvc := listfeat.NewService(listStore, s.log)

	s.authHandler = authfeat.NewHandler(s.authSvc, authQueries, s.tmplMgr, s.log)
	s.listHandler = listfeat.NewHandler(listSvc, s.tmplMgr, s.log)
	s.adminHandler = adminfeat.NewHandler(authQueries, s.tmplMgr, s.log)

	s.log.Info("Service started successfully")
	return nil
}

// Stop gracefully shuts down the service.
func (s *Service) Stop(ctx context.Context) error {
	if err := s.database.Stop(ctx); err != nil {
		s.log.Errorf("database stop error: %v", err)
	}

	s.log.Info("Service stopped successfully")
	return nil
}

// RegisterRoutes registers all HTTP routes for the service.
func (s *Service) RegisterRoutes(r chi.Router) {
	staticFS, err := fs.Sub(s.assetsFS, "assets/static")
	if err != nil {
		s.log.Errorf("cannot create static sub-fs: %v", err)
	} else {
		r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	s.authHandler.RegisterRoutes(r)

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(s.authSvc))
		s.listHandler.RegisterRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(s.authSvc))
		r.Use(middleware.RequireAnyRole("superadmin", "admin"))
		s.adminHandler.RegisterRoutes(r)
	})
}
